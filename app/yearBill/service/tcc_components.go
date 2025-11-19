package service

import (
	"context"
	"encoding/json"
	"platform/app/common/tcc"
	"platform/app/yearBill/database/cache"
	"platform/app/yearBill/database/dao"
	"platform/app/yearBill/database/model"
	"platform/app/yearBill/script"
	YearBillPb "platform/idl/pb/yearBill"
	"time"

	"github.com/redis/go-redis/v9"
)

// PayDataTCCComponent 支付数据TCC组件
type PayDataTCCComponent struct {
	dao    *dao.UserDao
	cache  *cache.RDBCache
	client *redis.Client
}

func NewPayDataTCCComponent(dao *dao.UserDao, cache *cache.RDBCache, client *redis.Client) *PayDataTCCComponent {
	return &PayDataTCCComponent{
		dao:    dao,
		cache:  cache,
		client: client,
	}
}

func (p *PayDataTCCComponent) ID() string {
	return "pay_data_component"
}

func (p *PayDataTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	var task model.DadaInitTask
	if err := json.Unmarshal([]byte(req.Data["task"].(string)), &task); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 检查是否已经处理过
	key := "tcc_pay_data:" + req.TXID + ":" + task.StuNum
	status, err := p.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 执行支付数据初始化
	resp, err := script.PayDataInit(ctx, &YearBillPb.PayDataInitRequest{
		StuNum:     task.StuNum,
		HallTicket: task.HallTicket,
	})
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 保存临时数据到缓存
	tempData := map[string]interface{}{
		"favorite_restaurant":     resp.FavoriteRestaurant,
		"favorite_restaurant_pay": resp.FavoriteRestaurantPay,
		"early_time":              resp.EarlyTime.AsTime(),
		"last_time":               resp.LastTime.AsTime(),
		"other_pay":               resp.OtherPay,
		"library_pay":             resp.LibraryPay,
		"restaurant_pay":          resp.RestaurantPay,
	}

	dataBytes, _ := json.Marshal(tempData)
	p.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
	p.client.Set(ctx, key+":status", "tried", 10*time.Minute)

	return &tcc.TCCResp{ACK: true}, nil
}

func (p *PayDataTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_pay_data:" + txID
	dataStr, err := p.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 保存到数据库
	// 这里需要根据实际的数据库操作来实现
	// 由于原代码中的具体实现被省略，这里提供一个框架

	// 清理临时数据
	p.client.Del(ctx, key)
	p.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (p *PayDataTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 清理临时数据
	key := "tcc_pay_data:" + txID
	p.client.Del(ctx, key)
	p.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

// LearnDataTCCComponent 学习数据TCC组件
type LearnDataTCCComponent struct {
	dao    *dao.UserDao
	cache  *cache.RDBCache
	client *redis.Client
}

func NewLearnDataTCCComponent(dao *dao.UserDao, cache *cache.RDBCache, client *redis.Client) *LearnDataTCCComponent {
	return &LearnDataTCCComponent{
		dao:    dao,
		cache:  cache,
		client: client,
	}
}

func (l *LearnDataTCCComponent) ID() string {
	return "learn_data_component"
}

func (l *LearnDataTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	var task model.DadaInitTask
	if err := json.Unmarshal([]byte(req.Data["task"].(string)), &task); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 检查是否已经处理过
	key := "tcc_learn_data:" + req.TXID + ":" + task.StuNum
	status, err := l.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 执行学习数据初始化
	resp, err := script.LearnDataInit(ctx, &YearBillPb.LearnDataInitRequest{
		StuNum:       task.StuNum,
		GsSession:    task.GsSession,
		Emaphome_WEU: task.Emaphome_WEU,
	})
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 保存临时数据到缓存
	tempData := map[string]interface{}{
		"most_course": resp.MostCourse,
		"eight":       resp.Eight,
		"most":        resp.Most,
		"ten":         resp.Ten,
		"sum_lesson":  resp.SumLesson,
	}

	dataBytes, _ := json.Marshal(tempData)
	l.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
	l.client.Set(ctx, key+":status", "tried", 10*time.Minute)

	return &tcc.TCCResp{ACK: true}, nil
}

func (l *LearnDataTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_learn_data:" + txID
	dataStr, err := l.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 保存到数据库
	// 这里需要根据实际的数据库操作来实现

	// 清理临时数据
	l.client.Del(ctx, key)
	l.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (l *LearnDataTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 清理临时数据
	key := "tcc_learn_data:" + txID
	l.client.Del(ctx, key)
	l.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

// RankUpdateTCCComponent 排名更新TCC组件
type RankUpdateTCCComponent struct {
	dao    *dao.UserDao
	cache  *cache.RDBCache
	client *redis.Client
}

func NewRankUpdateTCCComponent(dao *dao.UserDao, cache *cache.RDBCache, client *redis.Client) *RankUpdateTCCComponent {
	return &RankUpdateTCCComponent{
		dao:    dao,
		cache:  cache,
		client: client,
	}
}

func (r *RankUpdateTCCComponent) ID() string {
	return "rank_update_component"
}

func (r *RankUpdateTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	stuNum := req.Data["stu_num"].(string)

	// 检查是否已经处理过
	key := "tcc_rank_update:" + req.TXID + ":" + stuNum
	status, err := r.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 获取当前排名
	index, err := r.cache.IncrValue("count")
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 保存临时排名数据
	tempData := map[string]interface{}{
		"index":   index,
		"stu_num": stuNum,
	}

	dataBytes, _ := json.Marshal(tempData)
	r.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
	r.client.Set(ctx, key+":status", "tried", 10*time.Minute)

	return &tcc.TCCResp{ACK: true}, nil
}

func (r *RankUpdateTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_rank_update:" + txID
	dataStr, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 保存排名到缓存
	stuNum := tempData["stu_num"].(string)
	index := int(tempData["index"].(float64))

	err = r.cache.SaveRank(stuNum, model.RankCache{
		ID:        uint(index),
		Appraisal: 0,
	})
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 清理临时数据
	r.client.Del(ctx, key)
	r.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (r *RankUpdateTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 清理临时数据
	key := "tcc_rank_update:" + txID
	r.client.Del(ctx, key)
	r.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}
