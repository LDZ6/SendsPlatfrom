package service

import (
	"context"
	"encoding/json"
	"platform/app/boBing/database/cache"
	"platform/app/boBing/database/dao"
	"platform/app/common/tcc"
	BoBingPb "platform/idl/pb/boBing"
	"time"

	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

// BoBingTCCService BoBing TCC服务
type BoBingTCCService struct {
	txManager *tcc.TXManager
	txStore   *tcc.TXStore
}

// NewBoBingTCCService 创建BoBing TCC服务
func NewBoBingTCCService(db *gorm.DB, redisClient *redis.Client) *BoBingTCCService {
	// 创建TXStore
	txStore := tcc.NewTXStore(db, redisClient)

	// 创建TXManager
	txManager := tcc.NewTXManager(txStore,
		tcc.WithTimeout(30*time.Second),
		tcc.WithMonitorTick(5*time.Second),
	)

	// 创建TCC组件
	submissionDao := dao.NewSubmissionDao(context.Background())
	rankDao := dao.NewRankDao(context.Background())
	recordDao := dao.NewRecordDao(context.Background())
	cacheRDB := cache.NewRDBCache(context.Background())

	submissionComponent := NewBoBingSubmissionTCCComponent(submissionDao, rankDao, recordDao, cacheRDB, redisClient)
	rankUpdateComponent := NewBoBingRankUpdateTCCComponent(rankDao, cacheRDB, redisClient)
	recordComponent := NewBoBingRecordTCCComponent(recordDao, cacheRDB, redisClient)

	// 注册组件
	txManager.Register(submissionComponent)
	txManager.Register(rankUpdateComponent)
	txManager.Register(recordComponent)

	return &BoBingTCCService{
		txManager: txManager,
		txStore:   txStore,
	}
}

// BoBingPublishWithTCC 使用TCC架构进行博饼投掷
func (b *BoBingTCCService) BoBingPublishWithTCC(ctx context.Context, req *BoBingPb.BoBingPublishRequest) error {
	// 将请求数据序列化
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// 创建TCC事务请求
	requests := []*tcc.RequestEntity{
		{
			ComponentID: "bobing_rank_update_component",
			Request: map[string]interface{}{
				"open_id":   req.OpenId,
				"stu_num":   req.StuNum,
				"nick_name": req.NickName,
			},
		},
		{
			ComponentID: "bobing_submission_component",
			Request: map[string]interface{}{
				"request": string(reqJSON),
			},
		},
		{
			ComponentID: "bobing_record_component",
			Request: map[string]interface{}{
				"open_id": req.OpenId,
				"score":   0,  // 这里需要根据实际业务逻辑计算分数
				"types":   "", // 这里需要根据实际业务逻辑计算类型
			},
		},
	}

	// 执行TCC事务
	txID, success, err := b.txManager.Transaction(ctx, requests...)
	if err != nil {
		return err
	}

	if !success {
		return err
	}

	// 记录事务ID用于后续查询
	// 这里可以记录日志或保存到数据库
	return nil
}

// GetTransactionStatus 获取事务状态
func (b *BoBingTCCService) GetTransactionStatus(ctx context.Context, txID string) (*tcc.Transaction, error) {
	return b.txStore.GetTX(ctx, txID)
}

// Stop 停止TCC服务
func (b *BoBingTCCService) Stop() {
	b.txManager.Stop()
}
