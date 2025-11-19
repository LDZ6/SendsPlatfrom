package tcc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// TXStore 事务日志存储接口实现
type TXStore struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewTXStore 创建新的TXStore实例
func NewTXStore(db *gorm.DB, redis *redis.Client) *TXStore {
	return &TXStore{
		db:    db,
		redis: redis,
	}
}

// Transaction 事务记录
type Transaction struct {
	TXID       string                `json:"txID" gorm:"primaryKey"`
	Components []*ComponentTryEntity `json:"components" gorm:"type:json"`
	Status     string                `json:"status"`
	CreatedAt  time.Time             `json:"createdAt"`
	UpdatedAt  time.Time             `json:"updatedAt"`
}

// ComponentTryEntity 组件尝试实体
type ComponentTryEntity struct {
	ComponentID string `json:"componentID"`
	TryStatus   string `json:"tryStatus"`
}

// CreateTX 创建事务记录
func (t *TXStore) CreateTX(ctx context.Context, components ...TCCComponent) (string, error) {
	// 生成事务ID
	txID := generateTXID()

	// 构建组件列表
	componentEntities := make([]*ComponentTryEntity, 0, len(components))
	for _, component := range components {
		componentEntities = append(componentEntities, &ComponentTryEntity{
			ComponentID: component.ID(),
			TryStatus:   "hanging",
		})
	}

	// 创建事务记录
	tx := &Transaction{
		TXID:       txID,
		Components: componentEntities,
		Status:     "hanging",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 保存到数据库
	if err := t.db.WithContext(ctx).Create(tx).Error; err != nil {
		return "", err
	}

	// 保存到Redis缓存
	txJSON, _ := json.Marshal(tx)
	t.redis.Set(ctx, "tx:"+txID, string(txJSON), 24*time.Hour)

	return txID, nil
}

// TXUpdate 更新事务进度
func (t *TXStore) TXUpdate(ctx context.Context, txID string, componentID string, accept bool) error {
	// 从Redis获取事务记录
	txJSON, err := t.redis.Get(ctx, "tx:"+txID).Result()
	if err != nil {
		// 如果Redis中没有，从数据库获取
		var tx Transaction
		if err := t.db.WithContext(ctx).Where("tx_id = ?", txID).First(&tx).Error; err != nil {
			return err
		}
		txJSON, _ = json.Marshal(tx)
	}

	var tx Transaction
	if err := json.Unmarshal([]byte(txJSON), &tx); err != nil {
		return err
	}

	// 更新组件状态
	for _, component := range tx.Components {
		if component.ComponentID == componentID {
			if accept {
				component.TryStatus = "successful"
			} else {
				component.TryStatus = "failure"
			}
			break
		}
	}

	// 更新数据库
	if err := t.db.WithContext(ctx).Model(&Transaction{}).Where("tx_id = ?", txID).Updates(map[string]interface{}{
		"components": tx.Components,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return err
	}

	// 更新Redis缓存
	txJSON, _ = json.Marshal(tx)
	t.redis.Set(ctx, "tx:"+txID, string(txJSON), 24*time.Hour)

	return nil
}

// TXSubmit 提交事务最终状态
func (t *TXStore) TXSubmit(ctx context.Context, txID string, success bool) error {
	status := "failure"
	if success {
		status = "successful"
	}

	// 更新数据库
	if err := t.db.WithContext(ctx).Model(&Transaction{}).Where("tx_id = ?", txID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return err
	}

	// 更新Redis缓存
	txJSON, err := t.redis.Get(ctx, "tx:"+txID).Result()
	if err == nil {
		var tx Transaction
		if err := json.Unmarshal([]byte(txJSON), &tx); err == nil {
			tx.Status = status
			tx.UpdatedAt = time.Now()
			txJSON, _ = json.Marshal(tx)
			t.redis.Set(ctx, "tx:"+txID, string(txJSON), 24*time.Hour)
		}
	}

	return nil
}

// GetHangingTXs 获取所有未完成的事务
func (t *TXStore) GetHangingTXs(ctx context.Context) ([]*Transaction, error) {
	var txs []*Transaction

	// 从数据库查询未完成的事务
	if err := t.db.WithContext(ctx).Where("status = ?", "hanging").Find(&txs).Error; err != nil {
		return nil, err
	}

	// 检查是否有超时的事务
	timeout := time.Now().Add(-30 * time.Minute) // 30分钟超时
	var expiredTXs []*Transaction
	for _, tx := range txs {
		if tx.CreatedAt.Before(timeout) {
			expiredTXs = append(expiredTXs, tx)
		}
	}

	// 将超时的事务标记为失败
	for _, tx := range expiredTXs {
		t.TXSubmit(ctx, tx.TXID, false)
	}

	// 重新查询未完成的事务
	if err := t.db.WithContext(ctx).Where("status = ?", "hanging").Find(&txs).Error; err != nil {
		return nil, err
	}

	return txs, nil
}

// GetTX 获取指定事务
func (t *TXStore) GetTX(ctx context.Context, txID string) (*Transaction, error) {
	// 先从Redis获取
	txJSON, err := t.redis.Get(ctx, "tx:"+txID).Result()
	if err == nil {
		var tx Transaction
		if err := json.Unmarshal([]byte(txJSON), &tx); err == nil {
			return &tx, nil
		}
	}

	// 如果Redis中没有，从数据库获取
	var tx Transaction
	if err := t.db.WithContext(ctx).Where("tx_id = ?", txID).First(&tx).Error; err != nil {
		return nil, err
	}

	// 保存到Redis缓存
	txJSON, _ = json.Marshal(tx)
	t.redis.Set(ctx, "tx:"+txID, string(txJSON), 24*time.Hour)

	return &tx, nil
}

// Lock 锁定TXStore模块
func (t *TXStore) Lock(ctx context.Context, expireDuration time.Duration) error {
	lockKey := "txstore:lock"
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	// 尝试获取分布式锁
	result := t.redis.SetNX(ctx, lockKey, lockValue, expireDuration)
	if result.Err() != nil {
		return result.Err()
	}

	if !result.Val() {
		return fmt.Errorf("failed to acquire lock")
	}

	return nil
}

// Unlock 解锁TXStore模块
func (t *TXStore) Unlock(ctx context.Context) error {
	lockKey := "txstore:lock"
	return t.redis.Del(ctx, lockKey).Err()
}

// generateTXID 生成事务ID
func generateTXID() string {
	return fmt.Sprintf("tx_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}
