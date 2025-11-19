package service

import (
	"context"
	"encoding/base64"
	"errors"
	"platform/app/boBing/database/cache"
	"platform/app/boBing/database/dao"
	"platform/app/boBing/database/models"
	"platform/app/boBing/pkg"
	BoBingPb "platform/idl/pb/boBing"
	"platform/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type BoBingSrv struct {
	BoBingPb.UnimplementedBoBingServiceServer
}

var BoBingSrvIns *BoBingSrv

var BoBingSrvOnce sync.Once

func GetBoBingSrv() *BoBingSrv {
	BoBingSrvOnce.Do(func() {
		BoBingSrvIns = &BoBingSrv{}
	})
	return BoBingSrvIns
}

func (*BoBingSrv) BoBingPing(ctx context.Context, empty *emptypb.Empty) (resp *BoBingPb.BoBingPingResponse, err error) {
	resp = new(BoBingPb.BoBingPingResponse)
	resp.Message = "BoBing微服务ping通"
	return
}

func (*BoBingSrv) BoBingDayInit(ctx context.Context, req *BoBingPb.BoBingInitRequest) (resp *emptypb.Empty, err error) {
	resp = new(emptypb.Empty)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	submissionDao := dao.NewSubmissionDao(ctx)
	cnt, err := submissionDao.ExistDaySubmissionByOpenId(req.OpenId, today)
	if err != nil {
		return
	}
	if cnt == 0 {
		rankDao := dao.NewRankDao(ctx)
		cnt, err = rankDao.ExistRankByOpenId(req.OpenId)
		if err != nil {
			return
		}
		if cnt == 0 {
			err = rankDao.CreatRank(req.OpenId, req.StuNum, req.NickName, 0)
			if err != nil {
				return
			}
		}
		var rank models.Rank
		rank, err = rankDao.GetRankByOpenId(req.OpenId)
		if err != nil {
			return
		}
		err = submissionDao.CreateDaySubmission(req.OpenId, rank, today)
		if err != nil {
			return
		}
		//cacheRDB := cache.NewRDBCache(ctx)
		//err = cacheRDB.SaveDayCount(today, models.Submission{
		//	RankOpenId: req.OpenId,
		//	Date:       today,
		//	Count:      0,
		//	Condition:  0,
		//	TuiWen:     3,
		//})
		//if err != nil {
		//	return
		//}
	}
	return
}

func (*BoBingSrv) BoBingKey(ctx context.Context, req *BoBingPb.BoBingKeyRequest) (resp *BoBingPb.BoBingKeyResponse, err error) {
	resp = new(BoBingPb.BoBingKeyResponse)
	cache := cache.NewRDBCache(ctx)
	err = cache.UnlinkKey(req.Openid)
	if err != nil {
		return
	}
	key, err := utils.RandString(32)
	if err != nil {
		return
	}
	err = cache.SaveKey(req.Openid, key)
	if err != nil {
		return
	}
	resp.Key = key
	return
}

func (*BoBingSrv) BoBingPublish(ctx context.Context, req *BoBingPb.BoBingPublishRequest) (resp *BoBingPb.BoBingPublishResponse, err error) {
	resp = new(BoBingPb.BoBingPublishResponse)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	//初始化校验
	submissionDao := dao.NewSubmissionDao(ctx)
	cnt, err := submissionDao.ExistDaySubmissionByOpenId(req.OpenId, today)
	if err != nil {
		return
	}
	if cnt == 0 {
		rankDao := dao.NewRankDao(ctx)
		cnt, err = rankDao.ExistRankByOpenId(req.OpenId)
		if err != nil {
			return
		}
		if cnt == 0 {
			err = rankDao.CreatRank(req.OpenId, req.StuNum, req.NickName, 0)
			if err != nil {
				return
			}
		}
		var rank models.Rank
		rank, err = rankDao.GetRankByOpenId(req.OpenId)
		if err != nil {
			return
		}
		err = submissionDao.CreateDaySubmission(req.OpenId, rank, today)
		if err != nil {
			return
		}
	}
	//检查今日是否还有次数，没有直接跳出
	submission, err := submissionDao.GetDaySubmissionByOpenId(req.OpenId, today)
	if err != nil {
		return
	}
	if submission.Count >= submission.Condition+10 {
		return
	}
	cacheRDB := cache.NewRDBCache(ctx)
	//验证解密是否成功
	key, err := cacheRDB.GetKey(req.OpenId)
	if err != nil {
		return
	}
	//销毁
	err = cacheRDB.UnlinkKey(req.OpenId)
	if err != nil {
		return
	}
	aes := utils.NewEncryption()
	aes.SetKey(key)
	err, req.Flag = aes.AesDecoding(req.Flag)
	if err != nil {
		logrus.Info("[AESDecodingERROR]:%v\n", err.Error())
		return
	}
	err, req.Check = aes.AesDecoding(req.Check)
	if err != nil {
		logrus.Info("[AESDecodingERROR]:%v\n", err.Error())
		return
	}

	//检验是否有作弊行为
	valid, err, quaternions := pkg.ValidateParams(req.OpenId, []byte(req.Check))
	if !valid {
		err = cacheRDB.SaveBlacklist(req.OpenId)
		if err != nil {
			return nil, utils.WrapRedisError(err, "保存黑名单")
		}
		return nil, utils.ErrBoBingCheat
	}
	if err != nil {
		return
	}

	err = cacheRDB.ExistQuaternion(quaternions)
	if err != nil {
		err = cacheRDB.SaveBlacklist(req.OpenId)
		if err != nil {
			return nil, utils.WrapRedisError(err, "保存黑名单")
		}
		return nil, utils.ErrBoBingCheat
	}
	err = cacheRDB.SaveQuaternion(quaternions)
	if err != nil {
		return
	}

	//获取对应的投掷信息
	flag, err := strconv.Atoi(req.Flag)
	if err != nil {
		return
	}
	if flag > 16 || flag < 1 {
		err = cacheRDB.SaveBlacklist(req.OpenId)
		if err != nil {
			return nil, utils.WrapRedisError(err, "保存黑名单")
		}
		return nil, utils.ErrBoBingInvalidFlag
	}
	boBingMessage := pkg.NewBoBingMessageTransform()
	boBingMessage.SetType(flag)
	score, types, count := boBingMessage.Transform()

	recordDao := dao.NewRecordDao(ctx)
	rankDao := dao.NewRankDao(ctx)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		//更新排名记录和次数
		submission, err = rankDao.UpdateRank(req, score, count, submission)
		if err != nil {
			return
		}
		err = cacheRDB.SaveDayCount(today, submission)
		if err != nil {
			if err.Error() == "检测到你有作弊行为，已进入黑名单" {
				err = cacheRDB.SaveBlacklist(req.OpenId)
				if err != nil {
					return
				}
				err = errors.New("检测到你有作弊行为，已被封号")
				return
			}
			return
		}
	}()
	go func() {
		defer wg.Done()
		//将这条记录写入保存用于之后校验
		err = recordDao.SaveRecord(req, score, types)
		if err != nil {
			return
		}
	}()
	wg.Wait()
	if err != nil {
		return
	}

	if flag >= 10 && flag <= 16 {
		//是状元时写入redis
		err = cacheRDB.SaveZhuangYuan(req.NickName, types, now)
		if err != nil {
			return
		}
	}
	err = cacheRDB.SaveToTalScore(req.NickName, req.OpenId, score)
	if err != nil {
		return
	}
	err = cacheRDB.SaveDayScore(req.NickName, req.OpenId, score)
	if err != nil {
		return
	}

	if flag == 1 {
		err = errors.New("跳猴")
		return
	}
	//只有稀有才发放密文
	if flag >= 4 {
		key, err = utils.RandString(16)
		if err != nil {
			return
		}
		err = cacheRDB.SaveKey(req.OpenId, key)
		if err != nil {
			return
		}
		aes.SetKey(key)
		resp.Ciphertext = aes.AesEncoding(utils.AcquiesceKey)
	}

	if flag == 10 {
		err = errors.New("加二")
	}
	if flag >= 11 && flag <= 13 {
		err = errors.New("加三")
	}

	if flag >= 14 && flag <= 15 {
		err = errors.New("加五")
	}

	if flag == 16 {
		err = errors.New("加六")
	}
	return
}

func (*BoBingSrv) BoBingToTalTen(ctx context.Context, empty *emptypb.Empty) (resp *BoBingPb.BoBingToTalTenResponse, err error) {
	resp = new(BoBingPb.BoBingToTalTenResponse)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	openid := strings.Join(md.Get("open_id"), "")

	//对中文进行编码
	nickName := strings.Join(md.Get("nick_name"), "")
	bytes, _ := base64.StdEncoding.DecodeString(nickName)
	nickName = string(bytes)
	cacheRDB := cache.NewRDBCache(ctx)
	total, err := cacheRDB.GetTop()
	if err != nil {
		return
	}
	rank, score, err := cacheRDB.GetMyToTalRank(nickName, openid)
	if err != nil {
		return
	}
	resp.BoBingRank = total
	resp.BingMyRank = new(BoBingPb.BoBingMyRank)
	resp.BingMyRank.Rank = rank + 1
	resp.BingMyRank.Score = int64(score)
	resp.BingMyRank.NickName = nickName
	return
}

func (*BoBingSrv) BoBingDayRank(ctx context.Context, empty *emptypb.Empty) (resp *BoBingPb.BoBingDayRankResponse, err error) {
	resp = new(BoBingPb.BoBingDayRankResponse)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	openid := strings.Join(md.Get("open_id"), "")

	//对中文进行编码
	nickName := strings.Join(md.Get("nick_name"), "")
	bytes, _ := base64.StdEncoding.DecodeString(nickName)
	nickName = string(bytes)
	logrus.Info(openid)
	logrus.Info(nickName)
	cacheRDB := cache.NewRDBCache(ctx)
	rank, score, err := cacheRDB.GetMyDayRank(nickName, openid)
	if err != nil {
		return
	}
	resp.BingMyRank = new(BoBingPb.BoBingMyRank)
	resp.BingMyRank.Rank = rank + 1
	resp.BingMyRank.Score = int64(score)
	resp.BingMyRank.NickName = nickName
	resp.BoBingRank, err = cacheRDB.GetDayTop()
	if err != nil {
		return
	}
	return
}

func (*BoBingSrv) BoBingTianXuan(ctx context.Context, empty *emptypb.Empty) (resp *BoBingPb.BoBingTianXuanResponse, err error) {
	resp = new(BoBingPb.BoBingTianXuanResponse)
	cacheRDB := cache.NewRDBCache(ctx)
	resp.BoBingTianXuan, err = cacheRDB.GetTianXuan()

	if err != nil {
		return
	}
	return
}

func (*BoBingSrv) BoBingGetCount(ctx context.Context, empty *emptypb.Empty) (resp *BoBingPb.BoBingGetCountResponse, err error) {
	resp = new(BoBingPb.BoBingGetCountResponse)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	openid := strings.Join(md.Get("open_id"), "")
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	//初始化校验
	cacheRDB := cache.NewRDBCache(ctx)
	count, err := cacheRDB.GetDayCount(today, openid)
	if err != nil {
		if err == redis.Nil {
			submissionDao := dao.NewSubmissionDao(ctx)
			cnt, err := submissionDao.ExistDaySubmissionByOpenId(openid, today)
			if err != nil {
				return resp, err
			}
			if cnt == 0 {
				rankDao := dao.NewRankDao(ctx)
				cnt, err = rankDao.ExistRankByOpenId(openid)
				if err != nil {
					return resp, err
				}
				if cnt == 0 {
					return nil, errors.New("初始化错误")
				}
				var rank models.Rank
				rank, err = rankDao.GetRankByOpenId(openid)
				if err != nil {
					return resp, err
				}
				err = submissionDao.CreateDaySubmission(openid, rank, today)
				if err != nil {
					return resp, err
				}
			}
			submission, err := submissionDao.GetDaySubmissionByOpenId(openid, today)
			if err != nil {
				return resp, err
			}
			err = cacheRDB.SaveDayCount(today, submission)
			if err != nil {
				return resp, err
			}
			resp.Count = int64(submission.Condition + 10 - submission.Count)
			return resp, nil
		} else {
			return
		}
	}
	resp.Count = int64(count)
	return
}

func (*BoBingSrv) BoBingRetransmission(ctx context.Context, empty *emptypb.Empty) (resp *emptypb.Empty, err error) {
	resp = new(emptypb.Empty)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	openid := strings.Join(md.Get("open_id"), "")
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	//初始化校验
	submissionDao := dao.NewSubmissionDao(ctx)
	cnt, err := submissionDao.ExistDaySubmissionByOpenId(openid, today)
	if err != nil {
		return
	}
	if cnt == 0 {
		rankDao := dao.NewRankDao(ctx)
		cnt, err = rankDao.ExistRankByOpenId(openid)
		if err != nil {
			return
		}
		if cnt == 0 {
			return nil, errors.New("初始化错误")
		}
		var rank models.Rank
		rank, err = rankDao.GetRankByOpenId(openid)
		if err != nil {
			return
		}
		err = submissionDao.CreateDaySubmission(openid, rank, today)
		if err != nil {
			return
		}
	}

	submission, err := submissionDao.GetDaySubmissionByOpenId(openid, today)
	if err != nil {
		return
	}
	if submission.TuiWen > 0 {
		submission.TuiWen -= 1
		submission.Condition++
		err = submissionDao.UpdateSubmission(submission)
		if err != nil {
			return
		}
	} else {
		return nil, utils.ErrBoBingNoChance
	}
	cacheRDB := cache.NewRDBCache(ctx)
	err = cacheRDB.SaveDayCount(today, submission)
	if err != nil {
		if err.Error() == "检测到你有作弊行为，已进入黑名单" {
			err = cacheRDB.SaveBlacklist(openid)
			if err != nil {
				return nil, errors.New("检测到你有作弊行为，已被封号")
			}
			return nil, errors.New("检测到你有作弊行为，已被封号")
		}
		return
	}
	return
}

func (*BoBingSrv) BoBingBroadcastCheck(ctx context.Context, req *BoBingPb.BoBingBroadcastCheckRequest) (resp *emptypb.Empty, err error) {
	resp = new(emptypb.Empty) //验证解密是否成功
	cacheRDB := cache.NewRDBCache(ctx)
	key, err := cacheRDB.GetKey(req.OpenId)
	if err != nil {
		return
	}
	//销毁
	err = cacheRDB.UnlinkKey(req.OpenId)
	if err != nil {
		return
	}
	aes := utils.NewEncryption()
	aes.SetKey(key)
	err, reqKey := aes.AesDecoding(req.Ciphertext)
	if err != nil {
		return
	}
	if reqKey != utils.AcquiesceKey {
		return nil, errors.New("信息不符合")
	}

	return
}

func (*BoBingSrv) BoBingRecord(ctx context.Context, empty *emptypb.Empty) (resp *BoBingPb.BoBingRecordResponse, err error) {
	resp = new(BoBingPb.BoBingRecordResponse)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	openid := strings.Join(md.Get("open_id"), "")
	recordDao := dao.NewRecordDao(ctx)
	resp.Maps = make(map[string]int64)
	records, err := recordDao.GetRecordByOpenId(openid)
	if err != nil {
		return
	}
	for _, v := range records {
		resp.Maps[v.Types]++
	}
	return
}

func (*BoBingSrv) BoBingBlacklist(ctx context.Context, req *emptypb.Empty) (resp *emptypb.Empty, err error) {
	resp = new(emptypb.Empty)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	openid := strings.Join(md.Get("open_id"), "")
	nickName := strings.Join(md.Get("nick_name"), "")
	bytes, _ := base64.StdEncoding.DecodeString(nickName)
	nickName = string(bytes)
	cacheRDB := cache.NewRDBCache(ctx)
	_, score, err := cacheRDB.GetMyDayRank(nickName, openid)
	if score >= 100 {
		err = cacheRDB.SaveBlacklist(openid)
		if err != nil {
			return nil, errors.New("检测到你有作弊行为，已被封号")
		}
		return nil, errors.New("检测到你有作弊行为，已被封号")
	}
	err = cacheRDB.ExistBlacklist(openid)
	if err != nil {
		return
	}
	return
}

func (*BoBingSrv) BoBingGetNumber(ctx context.Context, req *emptypb.Empty) (resp *BoBingPb.BoBingGetNumberResponse, err error) {
	resp = new(BoBingPb.BoBingGetNumberResponse)
	rankDao := dao.NewRankDao(ctx)
	cnt, err := rankDao.CountRank()
	if err != nil {
		return
	}
	resp.Number = cnt
	return
}
