package tcc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
)

// TransactionCoordinator 分布式事务协调器
type TransactionCoordinator struct {
	client      *clientv3.Client
	components  map[string]TCCComponent
	store       TransactionStore
	monitor     TransactionMonitor
	compensator Compensator
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// TransactionStore 事务存储接口
type TransactionStore interface {
	SaveTransaction(tx *Transaction) error
	GetTransaction(txID string) (*Transaction, error)
	UpdateTransaction(tx *Transaction) error
	DeleteTransaction(txID string) error
	ListTransactions(status TransactionStatus) ([]*Transaction, error)
}

// TransactionMonitor 事务监控接口
type TransactionMonitor interface {
	OnTransactionStarted(tx *Transaction)
	OnTransactionCompleted(tx *Transaction)
	OnTransactionFailed(tx *Transaction, err error)
	OnTransactionTimeout(tx *Transaction)
}

// Compensator 补偿器接口
type Compensator interface {
	Compensate(ctx context.Context, tx *Transaction) error
	CanCompensate(tx *Transaction) bool
}

// Transaction 事务结构
type Transaction struct {
	ID         string                 `json:"id"`
	Status     TransactionStatus      `json:"status"`
	Components []string               `json:"components"`
	StartTime  time.Time              `json:"startTime"`
	EndTime    time.Time              `json:"endTime"`
	Timeout    time.Duration          `json:"timeout"`
	RetryCount int                    `json:"retryCount"`
	MaxRetries int                    `json:"maxRetries"`
	Metadata   map[string]interface{} `json:"metadata"`
	Error      string                 `json:"error,omitempty"`
}

// TransactionStatus 事务状态
type TransactionStatus int

const (
	StatusPending TransactionStatus = iota
	StatusTrying
	StatusConfirming
	StatusConfirmed
	StatusCancelling
	StatusCancelled
	StatusFailed
	StatusTimeout
)

// NewTransactionCoordinator 创建事务协调器
func NewTransactionCoordinator(client *clientv3.Client, store TransactionStore, monitor TransactionMonitor, compensator Compensator) *TransactionCoordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &TransactionCoordinator{
		client:      client,
		components:  make(map[string]TCCComponent),
		store:       store,
		monitor:     monitor,
		compensator: compensator,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// RegisterComponent 注册TCC组件
func (tc *TransactionCoordinator) RegisterComponent(component TCCComponent) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.components[component.ID()] = component
}

// StartTransaction 开始事务
func (tc *TransactionCoordinator) StartTransaction(components []string, timeout time.Duration, metadata map[string]interface{}) (*Transaction, error) {
	txID := generateTransactionID()

	tx := &Transaction{
		ID:         txID,
		Status:     StatusPending,
		Components: components,
		StartTime:  time.Now(),
		Timeout:    timeout,
		MaxRetries: 3,
		Metadata:   metadata,
	}

	// 保存事务
	if err := tc.store.SaveTransaction(tx); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	// 通知监控器
	if tc.monitor != nil {
		tc.monitor.OnTransactionStarted(tx)
	}

	// 异步执行事务
	go tc.executeTransaction(tx)

	return tx, nil
}

// executeTransaction 执行事务
func (tc *TransactionCoordinator) executeTransaction(tx *Transaction) {
	ctx, cancel := context.WithTimeout(tc.ctx, tx.Timeout)
	defer cancel()

	// 更新状态为尝试中
	tx.Status = StatusTrying
	tc.store.UpdateTransaction(tx)

	// 执行Try阶段
	if err := tc.executeTryPhase(ctx, tx); err != nil {
		tc.handleTransactionFailure(tx, err)
		return
	}

	// 执行Confirm阶段
	if err := tc.executeConfirmPhase(ctx, tx); err != nil {
		tc.handleTransactionFailure(tx, err)
		return
	}

	// 事务成功
	tx.Status = StatusConfirmed
	tx.EndTime = time.Now()
	tc.store.UpdateTransaction(tx)

	if tc.monitor != nil {
		tc.monitor.OnTransactionCompleted(tx)
	}
}

// executeTryPhase 执行Try阶段
func (tc *TransactionCoordinator) executeTryPhase(ctx context.Context, tx *Transaction) error {
	for _, componentID := range tx.Components {
		component, exists := tc.components[componentID]
		if !exists {
			return fmt.Errorf("component not found: %s", componentID)
		}

		req := &TCCReq{
			ComponentID: componentID,
			TXID:        tx.ID,
			Data:        tx.Metadata,
		}

		resp, err := component.Try(ctx, req)
		if err != nil {
			return fmt.Errorf("try phase failed for component %s: %w", componentID, err)
		}

		if !resp.ACK {
			return fmt.Errorf("try phase not acknowledged by component %s", componentID)
		}
	}

	return nil
}

// executeConfirmPhase 执行Confirm阶段
func (tc *TransactionCoordinator) executeConfirmPhase(ctx context.Context, tx *Transaction) error {
	tx.Status = StatusConfirming
	tc.store.UpdateTransaction(tx)

	for _, componentID := range tx.Components {
		component, exists := tc.components[componentID]
		if !exists {
			return fmt.Errorf("component not found: %s", componentID)
		}

		resp, err := component.Confirm(ctx, tx.ID)
		if err != nil {
			return fmt.Errorf("confirm phase failed for component %s: %w", componentID, err)
		}

		if !resp.ACK {
			return fmt.Errorf("confirm phase not acknowledged by component %s", componentID)
		}
	}

	return nil
}

// handleTransactionFailure 处理事务失败
func (tc *TransactionCoordinator) handleTransactionFailure(tx *Transaction, err error) {
	tx.Status = StatusFailed
	tx.Error = err.Error()
	tx.EndTime = time.Now()
	tc.store.UpdateTransaction(tx)

	// 执行补偿
	if tc.compensator != nil && tc.compensator.CanCompensate(tx) {
		go tc.executeCompensation(tx)
	}

	if tc.monitor != nil {
		tc.monitor.OnTransactionFailed(tx, err)
	}
}

// executeCompensation 执行补偿
func (tc *TransactionCoordinator) executeCompensation(tx *Transaction) {
	ctx, cancel := context.WithTimeout(tc.ctx, tx.Timeout)
	defer cancel()

	tx.Status = StatusCancelling
	tc.store.UpdateTransaction(tx)

	// 执行Cancel阶段
	for _, componentID := range tx.Components {
		component, exists := tc.components[componentID]
		if !exists {
			continue
		}

		// 尝试执行Cancel
		_, err := component.Cancel(ctx, tx.ID)
		if err != nil {
			// 记录错误但继续执行其他组件的Cancel
			continue
		}
	}

	tx.Status = StatusCancelled
	tx.EndTime = time.Now()
	tc.store.UpdateTransaction(tx)
}

// GetTransaction 获取事务
func (tc *TransactionCoordinator) GetTransaction(txID string) (*Transaction, error) {
	return tc.store.GetTransaction(txID)
}

// ListTransactions 列出事务
func (tc *TransactionCoordinator) ListTransactions(status TransactionStatus) ([]*Transaction, error) {
	return tc.store.ListTransactions(status)
}

// RetryTransaction 重试事务
func (tc *TransactionCoordinator) RetryTransaction(txID string) error {
	tx, err := tc.store.GetTransaction(txID)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	if tx.RetryCount >= tx.MaxRetries {
		return fmt.Errorf("max retries exceeded for transaction %s", txID)
	}

	tx.RetryCount++
	tx.Status = StatusPending
	tc.store.UpdateTransaction(tx)

	// 异步重试
	go tc.executeTransaction(tx)

	return nil
}

// Stop 停止事务协调器
func (tc *TransactionCoordinator) Stop() {
	tc.cancel()
}

// generateTransactionID 生成事务ID
func generateTransactionID() string {
	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}

// EtcdTransactionStore etcd事务存储实现
type EtcdTransactionStore struct {
	client    *clientv3.Client
	keyPrefix string
}

// NewEtcdTransactionStore 创建etcd事务存储
func NewEtcdTransactionStore(client *clientv3.Client, keyPrefix string) *EtcdTransactionStore {
	return &EtcdTransactionStore{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// SaveTransaction 保存事务
func (ets *EtcdTransactionStore) SaveTransaction(tx *Transaction) error {
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	key := ets.keyPrefix + tx.ID
	_, err = ets.client.Put(context.Background(), key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	return nil
}

// GetTransaction 获取事务
func (ets *EtcdTransactionStore) GetTransaction(txID string) (*Transaction, error) {
	key := ets.keyPrefix + txID

	resp, err := ets.client.Get(context.Background(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("transaction not found: %s", txID)
	}

	var tx Transaction
	if err := json.Unmarshal(resp.Kvs[0].Value, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	return &tx, nil
}

// UpdateTransaction 更新事务
func (ets *EtcdTransactionStore) UpdateTransaction(tx *Transaction) error {
	return ets.SaveTransaction(tx)
}

// DeleteTransaction 删除事务
func (ets *EtcdTransactionStore) DeleteTransaction(txID string) error {
	key := ets.keyPrefix + txID

	_, err := ets.client.Delete(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	return nil
}

// ListTransactions 列出事务
func (ets *EtcdTransactionStore) ListTransactions(status TransactionStatus) ([]*Transaction, error) {
	resp, err := ets.client.Get(context.Background(), ets.keyPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	var transactions []*Transaction
	for _, kv := range resp.Kvs {
		var tx Transaction
		if err := json.Unmarshal(kv.Value, &tx); err != nil {
			continue
		}

		if status == -1 || tx.Status == status {
			transactions = append(transactions, &tx)
		}
	}

	return transactions, nil
}

// DefaultTransactionMonitor 默认事务监控器
type DefaultTransactionMonitor struct {
	logger interface{} // 日志接口
}

// NewDefaultTransactionMonitor 创建默认事务监控器
func NewDefaultTransactionMonitor(logger interface{}) *DefaultTransactionMonitor {
	return &DefaultTransactionMonitor{
		logger: logger,
	}
}

// OnTransactionStarted 事务开始
func (dtm *DefaultTransactionMonitor) OnTransactionStarted(tx *Transaction) {
	// 记录日志
	fmt.Printf("Transaction started: %s\n", tx.ID)
}

// OnTransactionCompleted 事务完成
func (dtm *DefaultTransactionMonitor) OnTransactionCompleted(tx *Transaction) {
	// 记录日志
	fmt.Printf("Transaction completed: %s\n", tx.ID)
}

// OnTransactionFailed 事务失败
func (dtm *DefaultTransactionMonitor) OnTransactionFailed(tx *Transaction, err error) {
	// 记录日志
	fmt.Printf("Transaction failed: %s, error: %v\n", tx.ID, err)
}

// OnTransactionTimeout 事务超时
func (dtm *DefaultTransactionMonitor) OnTransactionTimeout(tx *Transaction) {
	// 记录日志
	fmt.Printf("Transaction timeout: %s\n", tx.ID)
}

// DefaultCompensator 默认补偿器
type DefaultCompensator struct {
	coordinator *TransactionCoordinator
}

// NewDefaultCompensator 创建默认补偿器
func NewDefaultCompensator(coordinator *TransactionCoordinator) *DefaultCompensator {
	return &DefaultCompensator{
		coordinator: coordinator,
	}
}

// Compensate 执行补偿
func (dc *DefaultCompensator) Compensate(ctx context.Context, tx *Transaction) error {
	// 执行Cancel阶段
	for _, componentID := range tx.Components {
		component, exists := dc.coordinator.components[componentID]
		if !exists {
			continue
		}

		_, err := component.Cancel(ctx, tx.ID)
		if err != nil {
			// 记录错误但继续执行
			continue
		}
	}

	return nil
}

// CanCompensate 是否可以补偿
func (dc *DefaultCompensator) CanCompensate(tx *Transaction) bool {
	// 只有在Try阶段成功的情况下才能补偿
	return tx.Status == StatusTrying || tx.Status == StatusConfirming
}
