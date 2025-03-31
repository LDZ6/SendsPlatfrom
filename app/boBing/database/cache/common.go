package cache

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"platform/app/boBing/database/models"
	BoBingPb "platform/idl/pb/boBing"
	"strconv"
	"time"
)

type RDBCache struct {
	Rdb *redis.Client
}

func NewRDBCache(ctx context.Context) *RDBCache {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RDBCache{Rdb: NewRDBClient(ctx)}
}

func (cache *RDBCache) GetKey(openid string) (key string, err error) {
	key, err = cache.Rdb.Get(openid).Result()
	if err == redis.Nil {
		//缓存项不存在进行处理
		return "", errors.New("key不存在")
	} else if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return "", err
	}
	// 返回缓存值
	return key, nil
}

func (cache *RDBCache) SaveKey(openId string, key string) (err error) {
	err = cache.Rdb.Set(openId, key, 4*time.Second).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (cache *RDBCache) UnlinkKey(openId string) (err error) {
	err = cache.Rdb.Unlink(openId).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}

type Member struct {
	OpenId   string `json:"open_id"`
	NickName string `json:"nick_name"`
}

type ZhuangYuan struct {
	NickName string    `json:"nick_name"`
	Types    string    `json:"types"`
	Time     time.Time `json:"time"`
}

func (cache *RDBCache) SaveZhuangYuan(nickName string, types string, time time.Time) (err error) {
	zhuangYuan := ZhuangYuan{
		NickName: nickName,
		Types:    types,
		Time:     time,
	}
	json, err := json.Marshal(zhuangYuan)
	if err != nil {
		logrus.Info("[JsonMarshalERROR]:%v\n", err.Error())
		return
	}
	flag := time.Unix()
	err = cache.Rdb.ZIncrBy("tianxuan", float64(flag), string(json)).Err()
	return
}

func (cache *RDBCache) SaveToTalScore(nickName string, openid string, score int) (err error) {
	member := Member{
		OpenId:   openid,
		NickName: nickName,
	}
	json, err := json.Marshal(member)
	if err != nil {
		logrus.Info("[JsonMarshalERROR]:%v\n", err.Error())
		return
	}
	err = cache.Rdb.ZIncrBy("totalrank", float64(score), string(json)).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (cache *RDBCache) SaveDayScore(nickName string, openid string, score int) (err error) {
	member := Member{
		OpenId:   openid,
		NickName: nickName,
	}
	json, err := json.Marshal(member)
	if err != nil {
		logrus.Info("[JsonMarshalERROR]:%v\n", err.Error())
		return
	}
	year, month, day := time.Now().Date()
	var key = strconv.Itoa(year) + ":" + month.String() + ":" + strconv.Itoa(day) + "rank"
	err = cache.Rdb.ZIncrBy(key, float64(score), string(json)).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (cache *RDBCache) GetTop() (resp []*BoBingPb.BoBingRank, err error) {
	z, err := cache.Rdb.ZRevRangeWithScores("totalrank", 0, 29).Result()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	for _, v := range z {
		var member Member
		err = json.Unmarshal([]byte(v.Member.(string)), &member)
		if err != nil {
			logrus.Info("[JsonUnmarshalERROR]:%v\n", err.Error())
			return nil, err
		}
		resp = append(resp, &BoBingPb.BoBingRank{
			NickName: member.NickName,
			Score:    int64(v.Score),
		})
	}
	return
}

func (cache *RDBCache) GetDayTop() (resp []*BoBingPb.BoBingRank, err error) {
	year, month, day := time.Now().Date()
	var key = strconv.Itoa(year) + ":" + month.String() + ":" + strconv.Itoa(day) + "rank"
	z, err := cache.Rdb.ZRevRangeWithScores(key, 0, 29).Result()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	for _, v := range z {
		var member Member
		err = json.Unmarshal([]byte(v.Member.(string)), &member)
		if err != nil {
			logrus.Info("[JsonUnmarshalERROR]:%v\n", err.Error())
			return
		}
		resp = append(resp, &BoBingPb.BoBingRank{
			NickName: member.NickName,
			Score:    int64(v.Score),
		})
	}
	return
}

func (cache *RDBCache) GetMyToTalRank(nickName string, openid string) (rank int64, score float64, err error) {
	memberJson := Member{
		OpenId:   openid,
		NickName: nickName,
	}
	json, err := json.Marshal(memberJson)
	if err != nil {
		logrus.Info("[JsonMarshalERROR]:%v\n", err.Error())
		return
	}
	member := string(json)
	rank, err = cache.Rdb.ZRevRank("totalrank", member).Result()
	if err == redis.Nil {
		//缓存项不存在进行处理
		return 0, 0, nil
	} else if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return 0, 0, err
	}
	score, err = cache.Rdb.ZScore("totalrank", member).Result()
	if err == redis.Nil {
		//缓存项不存在进行处理
		return 0, 0, nil
	} else if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return 0, 0, err
	}
	return
}

func (cache *RDBCache) GetTianXuan() (resp []*BoBingPb.BoBingTianXuan, err error) {
	z, err := cache.Rdb.ZRevRangeWithScores("tianxuan", 0, -1).Result()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	for _, v := range z {
		var member ZhuangYuan
		err = json.Unmarshal([]byte(v.Member.(string)), &member)
		if err != nil {
			logrus.Info("[JsonUnmarshalERROR]:%v\n", err.Error())
			return nil, err
		}
		resp = append(resp, &BoBingPb.BoBingTianXuan{
			NickName: member.NickName,
			Types:    member.Types,
			Time:     timestamppb.New(member.Time),
		})
	}
	return
}

func (cache *RDBCache) GetMyDayRank(nickName string, openid string) (rank int64, score float64, err error) {
	memberJson := Member{
		OpenId:   openid,
		NickName: nickName,
	}
	json, err := json.Marshal(memberJson)
	if err != nil {
		logrus.Info("[JsonMarshalERROR]:%v\n", err.Error())
		return
	}
	member := string(json)
	year, month, day := time.Now().Date()
	var key = strconv.Itoa(year) + ":" + month.String() + ":" + strconv.Itoa(day) + "rank"
	rank, err = cache.Rdb.ZRevRank(key, member).Result()
	if err == redis.Nil {
		//缓存项不存在进行处理
		return 0, 0, nil
	} else if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return 0, 0, err
	}
	score, err = cache.Rdb.ZScore(key, member).Result()
	if err == redis.Nil {
		//缓存项不存在进行处理
		return 0, 0, nil
	} else if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return 0, 0, err
	}
	return
}

func (cache *RDBCache) SaveBlacklist(openid string) (err error) {
	err = cache.Rdb.SAdd("blacklist", openid).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (cache *RDBCache) ExistBlacklist(openid string) (err error) {
	isExist := cache.Rdb.SIsMember("blacklist", openid).Val()
	if isExist {
		return errors.New("检测到你有作弊行为，已进入黑名单")
	}
	return
}

func (cache *RDBCache) SaveQuaternion(quaternion []string) (err error) {
	err = cache.Rdb.SAdd("quaternions", quaternion[0], quaternion[1], quaternion[2], quaternion[3], quaternion[4], quaternion[5]).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (cache *RDBCache) ExistQuaternion(quaternion []string) (err error) {
	for _, v := range quaternion {
		isExist := cache.Rdb.SIsMember("quaternions", v).Val()
		if isExist {
			return errors.New("检测到你有作弊行为，已进入黑名单")
		}
	}
	return
}

func (cache *RDBCache) SaveDayCount(today time.Time, submission models.Submission) (err error) {
	if submission.Condition >= 12 {
		return errors.New("检测到你有作弊行为，已进入黑名单")
	}
	year, month, day := today.Date()
	var key = strconv.Itoa(year) + ":" + month.String() + ":" + strconv.Itoa(day) + "count"
	err = cache.Rdb.ZRem(key, submission.RankOpenId).Err()
	if err != nil && err != redis.Nil {
		return err
	}
	err = cache.Rdb.ZAdd(key, redis.Z{
		Score:  float64(10 + submission.Condition - submission.Count),
		Member: submission.RankOpenId,
	}).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return err
	}
	return
}

func (cache *RDBCache) GetDayCount(today time.Time, openid string) (count float64, err error) {
	year, month, day := today.Date()
	var key = strconv.Itoa(year) + ":" + month.String() + ":" + strconv.Itoa(day) + "count"
	count, err = cache.Rdb.ZScore(key, openid).Result()
	if err != nil {
		if err == redis.Nil {
			return
		}
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}
