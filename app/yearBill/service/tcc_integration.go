package service

import (
	"context"
	"encoding/json"
	"platform/app/common/tcc"
	"platform/app/yearBill/database/cache"
	"platform/app/yearBill/database/dao"
	"platform/app/yearBill/database/model"
	"platform/utils"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// YearBillTCCService YearBill TCC服务
type YearBillTCCService struct {
	txManager *tcc.TXManager
	txStore   *tcc.TXStore
}

// NewYearBillTCCService 创建YearBill TCC服务
func NewYearBillTCCService(db *gorm.DB, redisClient *redis.Client) *YearBillTCCService {
	// 创建TXStore
	txStore := tcc.NewTXStore(db, redisClient)

	// 创建TXManager
	txManager := tcc.NewTXManager(txStore,
		tcc.WithTimeout(30*time.Second),
		tcc.WithMonitorTick(5*time.Second),
	)

	// 创建TCC组件
	userDao := dao.NewUserDao(context.Background())
	cacheRDB := cache.NewRDBCache(context.Background())

	payDataComponent := NewPayDataTCCComponent(userDao, cacheRDB, redisClient)
	learnDataComponent := NewLearnDataTCCComponent(userDao, cacheRDB, redisClient)
	rankUpdateComponent := NewRankUpdateTCCComponent(userDao, cacheRDB, redisClient)

	// 注册组件
	txManager.Register(payDataComponent)
	txManager.Register(learnDataComponent)
	txManager.Register(rankUpdateComponent)

	return &YearBillTCCService{
		txManager: txManager,
		txStore:   txStore,
	}
}

// DataInitWithTCC 使用TCC架构进行数据初始化
func (y *YearBillTCCService) DataInitWithTCC(ctx context.Context, task model.DadaInitTask) error {
	// 将任务数据序列化
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// 创建TCC事务请求
	requests := []*tcc.RequestEntity{
		{
			ComponentID: "pay_data_component",
			Request: map[string]interface{}{
				"task": string(taskJSON),
			},
		},
		{
			ComponentID: "learn_data_component",
			Request: map[string]interface{}{
				"task": string(taskJSON),
			},
		},
		{
			ComponentID: "rank_update_component",
			Request: map[string]interface{}{
				"stu_num": task.StuNum,
			},
		},
	}

	// 执行TCC事务
	txID, success, err := y.txManager.Transaction(ctx, requests...)
	if err != nil {
		return err
	}

	if !success {
		return err
	}

	// 记录事务ID用于后续查询
	utils.Info("YearBill data init TCC transaction completed, txID: " + txID)
	return nil
}

// GetTransactionStatus 获取事务状态
func (y *YearBillTCCService) GetTransactionStatus(ctx context.Context, txID string) (*tcc.Transaction, error) {
	return y.txStore.GetTX(ctx, txID)
}

// Stop 停止TCC服务
func (y *YearBillTCCService) Stop() {
	y.txManager.Stop()
}
