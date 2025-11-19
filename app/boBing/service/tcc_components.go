package service

import (
	"context"
	"encoding/json"
	"platform/app/boBing/database/cache"
	"platform/app/boBing/database/dao"
	"platform/app/boBing/pkg"
	"platform/app/common/tcc"
	BoBingPb "platform/idl/pb/boBing"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// BoBingSubmissionTCCComponent 博饼投掷TCC组件
type BoBingSubmissionTCCComponent struct {
	submissionDao *dao.SubmissionDao
	rankDao       *dao.RankDao
	recordDao     *dao.RecordDao
	cache         *cache.RDBCache
	client        *redis.Client
}

func NewBoBingSubmissionTCCComponent(
	submissionDao *dao.SubmissionDao,
	rankDao *dao.RankDao,
	recordDao *dao.RecordDao,
	cache *cache.RDBCache,
	client *redis.Client,
) *BoBingSubmissionTCCComponent {
	return &BoBingSubmissionTCCComponent{
		submissionDao: submissionDao,
		rankDao:       rankDao,
		recordDao:     recordDao,
		cache:         cache,
		client:        client,
	}
}

func (b *BoBingSubmissionTCCComponent) ID() string {
	return "bobing_submission_component"
}

func (b *BoBingSubmissionTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	var submissionReq BoBingPb.BoBingPublishRequest
	if err := json.Unmarshal([]byte(req.Data["request"].(string)), &submissionReq); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 检查是否已经处理过
	key := "tcc_bobing_submission:" + req.TXID + ":" + submissionReq.OpenId
	status, err := b.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 验证投掷结果
	flag, err := strconv.Atoi(submissionReq.Flag)
	if err != nil || flag > 16 || flag < 1 {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 创建博饼消息转换器
	boBingMessage := pkg.NewBoBingMessageTransform()
	boBingMessage.SetType(flag)
	score, types, count := boBingMessage.Transform()

	// 保存临时数据
	tempData := map[string]interface{}{
		"open_id":   submissionReq.OpenId,
		"stu_num":   submissionReq.StuNum,
		"nick_name": submissionReq.NickName,
		"flag":      submissionReq.Flag,
		"check":     submissionReq.Check,
		"score":     score,
		"types":     types,
		"count":     count,
		"timestamp": time.Now(),
	}

	dataBytes, _ := json.Marshal(tempData)
	b.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
	b.client.Set(ctx, key+":status", "tried", 10*time.Minute)

	return &tcc.TCCResp{ACK: true}, nil
}

func (b *BoBingSubmissionTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_bobing_submission:" + txID
	dataStr, err := b.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 执行实际的数据库操作
	openId := tempData["open_id"].(string)
	stuNum := tempData["stu_num"].(string)
	nickName := tempData["nick_name"].(string)
	flag := tempData["flag"].(string)
	check := tempData["check"].(string)
	score := int(tempData["score"].(float64))
	types := tempData["types"].(string)
	count := int(tempData["count"].(float64))

	// 创建请求对象
	req := &BoBingPb.BoBingPublishRequest{
		OpenId:   openId,
		StuNum:   stuNum,
		NickName: nickName,
		Flag:     flag,
		Check:    check,
	}

	// 获取今日提交记录
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	submission, err := b.submissionDao.GetDaySubmissionByOpenId(openId, today)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 更新排名和提交记录
	updatedSubmission, err := b.rankDao.UpdateRank(req, score, count, submission)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 保存记录
	err = b.recordDao.SaveRecord(req, score, types)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 更新缓存
	err = b.cache.SaveDayCount(today, updatedSubmission)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 保存总分和日分
	err = b.cache.SaveToTalScore(nickName, openId, score)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	err = b.cache.SaveDayScore(nickName, openId, score)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果是状元，保存状元记录
	flagInt, _ := strconv.Atoi(flag)
	if flagInt >= 10 && flagInt <= 16 {
		err = b.cache.SaveZhuangYuan(nickName, types, now)
		if err != nil {
			return &TCCResp{ACK: false}, err
		}
	}

	// 清理临时数据
	b.client.Del(ctx, key)
	b.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (b *BoBingSubmissionTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 清理临时数据
	key := "tcc_bobing_submission:" + txID
	b.client.Del(ctx, key)
	b.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

// BoBingRankUpdateTCCComponent 博饼排名更新TCC组件
type BoBingRankUpdateTCCComponent struct {
	rankDao *dao.RankDao
	cache   *cache.RDBCache
	client  *redis.Client
}

func NewBoBingRankUpdateTCCComponent(rankDao *dao.RankDao, cache *cache.RDBCache, client *redis.Client) *BoBingRankUpdateTCCComponent {
	return &BoBingRankUpdateTCCComponent{
		rankDao: rankDao,
		cache:   cache,
		client:  client,
	}
}

func (b *BoBingRankUpdateTCCComponent) ID() string {
	return "bobing_rank_update_component"
}

func (b *BoBingRankUpdateTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	openId := req.Data["open_id"].(string)
	stuNum := req.Data["stu_num"].(string)
	nickName := req.Data["nick_name"].(string)

	// 检查是否已经处理过
	key := "tcc_bobing_rank:" + req.TXID + ":" + openId
	status, err := b.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 检查排名是否存在
	cnt, err := b.rankDao.ExistRankByOpenId(openId)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果排名不存在，创建新排名
	if cnt == 0 {
		err = b.rankDao.CreatRank(openId, stuNum, nickName, 0)
		if err != nil {
			return &TCCResp{ACK: false}, err
		}
	}

	// 保存临时数据
	tempData := map[string]interface{}{
		"open_id":   openId,
		"stu_num":   stuNum,
		"nick_name": nickName,
		"created":   cnt == 0,
	}

	dataBytes, _ := json.Marshal(tempData)
	b.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
	b.client.Set(ctx, key+":status", "tried", 10*time.Minute)

	return &tcc.TCCResp{ACK: true}, nil
}

func (b *BoBingRankUpdateTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_bobing_rank:" + txID
	dataStr, err := b.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果创建了新排名，这里可以进行额外的确认操作
	// 由于排名创建在Try阶段已经完成，这里主要是确认操作

	// 清理临时数据
	b.client.Del(ctx, key)
	b.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (b *BoBingRankUpdateTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_bobing_rank:" + txID
	dataStr, err := b.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &TCCResp{ACK: false}, err
	}

	// 如果创建了新排名，需要删除
	if tempData["created"].(bool) {
		openId := tempData["open_id"].(string)
		// 这里需要实现删除排名的逻辑
		// b.rankDao.DeleteRank(openId)
	}

	// 清理临时数据
	b.client.Del(ctx, key)
	b.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

// BoBingRecordTCCComponent 博饼记录TCC组件
type BoBingRecordTCCComponent struct {
	recordDao *dao.RecordDao
	cache     *cache.RDBCache
	client    *redis.Client
}

func NewBoBingRecordTCCComponent(recordDao *dao.RecordDao, cache *cache.RDBCache, client *redis.Client) *BoBingRecordTCCComponent {
	return &BoBingRecordTCCComponent{
		recordDao: recordDao,
		cache:     cache,
		client:    client,
	}
}

func (b *BoBingRecordTCCComponent) ID() string {
	return "bobing_record_component"
}

func (b *BoBingRecordTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	openId := req.Data["open_id"].(string)
	score := int(req.Data["score"].(float64))
	types := req.Data["types"].(string)

	// 检查是否已经处理过
	key := "tcc_bobing_record:" + req.TXID + ":" + openId
	status, err := b.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 保存临时数据
	tempData := map[string]interface{}{
		"open_id": openId,
		"score":   score,
		"types":   types,
	}

	dataBytes, _ := json.Marshal(tempData)
	b.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
	b.client.Set(ctx, key+":status", "tried", 10*time.Minute)

	return &tcc.TCCResp{ACK: true}, nil
}

func (b *BoBingRecordTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_bobing_record:" + txID
	dataStr, err := b.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 创建请求对象
	req := &BoBingPb.BoBingPublishRequest{
		OpenId: tempData["open_id"].(string),
	}

	// 保存记录
	err = b.recordDao.SaveRecord(req, int(tempData["score"].(float64)), tempData["types"].(string))
	if err != nil {
		return &TCCResp{ACK: false}, err
	}

	// 清理临时数据
	b.client.Del(ctx, key)
	b.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (b *BoBingRecordTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 清理临时数据
	key := "tcc_bobing_record:" + txID
	b.client.Del(ctx, key)
	b.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}
