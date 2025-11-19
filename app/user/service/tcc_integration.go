package service

import (
	"context"
	"platform/app/common/tcc"
	"platform/app/user/database/cache"
	"platform/app/user/database/dao"
	userPb "platform/idl/pb/user"
	"time"

	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

// UserTCCService User TCC服务
type UserTCCService struct {
	txManager *tcc.TXManager
	txStore   *tcc.TXStore
}

// NewUserTCCService 创建User TCC服务
func NewUserTCCService(db *gorm.DB, redisClient *redis.Client) *UserTCCService {
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

	userLoginComponent := NewUserLoginTCCComponent(userDao, cacheRDB, redisClient)
	schoolUserLoginComponent := NewSchoolUserLoginTCCComponent(userDao, cacheRDB, redisClient)
	massesLoginComponent := NewMassesLoginTCCComponent(userDao, cacheRDB, redisClient)

	// 注册组件
	txManager.Register(userLoginComponent)
	txManager.Register(schoolUserLoginComponent)
	txManager.Register(massesLoginComponent)

	return &UserTCCService{
		txManager: txManager,
		txStore:   txStore,
	}
}

// UserLoginWithTCC 使用TCC架构进行用户登录
func (u *UserTCCService) UserLoginWithTCC(ctx context.Context, req *userPb.UserLoginRequest) error {
	// 创建TCC事务请求
	requests := []*tcc.RequestEntity{
		{
			ComponentID: "user_login_component",
			Request: map[string]interface{}{
				"code":    req.Code,
				"open_id": "", // 这里需要从微信登录响应中获取
			},
		},
	}

	// 执行TCC事务
	txID, success, err := u.txManager.Transaction(ctx, requests...)
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

// SchoolUserLoginWithTCC 使用TCC架构进行学校用户登录
func (u *UserTCCService) SchoolUserLoginWithTCC(ctx context.Context, req *userPb.UserLoginRequest) error {
	// 创建TCC事务请求
	requests := []*tcc.RequestEntity{
		{
			ComponentID: "school_user_login_component",
			Request: map[string]interface{}{
				"code":    req.Code,
				"open_id": "", // 这里需要从微信登录响应中获取
			},
		},
	}

	// 执行TCC事务
	txID, success, err := u.txManager.Transaction(ctx, requests...)
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

// MassesLoginWithTCC 使用TCC架构进行群众登录
func (u *UserTCCService) MassesLoginWithTCC(ctx context.Context, req *userPb.UserLoginRequest) error {
	// 创建TCC事务请求
	requests := []*tcc.RequestEntity{
		{
			ComponentID: "masses_login_component",
			Request: map[string]interface{}{
				"code":    req.Code,
				"open_id": "", // 这里需要从微信登录响应中获取
			},
		},
	}

	// 执行TCC事务
	txID, success, err := u.txManager.Transaction(ctx, requests...)
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
func (u *UserTCCService) GetTransactionStatus(ctx context.Context, txID string) (*tcc.Transaction, error) {
	return u.txStore.GetTX(ctx, txID)
}

// Stop 停止TCC服务
func (u *UserTCCService) Stop() {
	u.txManager.Stop()
}
