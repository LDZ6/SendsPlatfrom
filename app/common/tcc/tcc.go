package tcc

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TCCPhase TCC阶段
type TCCPhase int

const (
	PhaseTry TCCPhase = iota
	PhaseConfirm
	PhaseCancel
)

// String 阶段字符串
func (p TCCPhase) String() string {
	switch p {
	case PhaseTry:
		return "try"
	case PhaseConfirm:
		return "confirm"
	case PhaseCancel:
		return "cancel"
	default:
		return "unknown"
	}
}

// TCCStatus TCC状态
type TCCStatus int

const (
	StatusPending TCCStatus = iota
	StatusTrying
	StatusConfirmed
	StatusCancelled
	StatusFailed
	StatusTimeout
)

// String 状态字符串
func (s TCCStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusTrying:
		return "trying"
	case StatusConfirmed:
		return "confirmed"
	case StatusCancelled:
		return "cancelled"
	case StatusFailed:
		return "failed"
	case StatusTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// TCCTransaction TCC事务
type TCCTransaction struct {
	ID         string
	Components []TCCComponent
	Status     TCCStatus
	Phase      TCCPhase
	StartTime  time.Time
	EndTime    *time.Time
	Timeout    time.Duration
	RetryCount int
	MaxRetries int
	Error      error
	mutex      sync.RWMutex
}

// TCCCoordinator TCC协调器
type TCCCoordinator struct {
	transactions map[string]*TCCTransaction
	components   map[string]TCCComponent
	timeout      time.Duration
	retryCount   int
	mutex        sync.RWMutex
}

// NewTCCCoordinator 创建TCC协调器
func NewTCCCoordinator() *TCCCoordinator {
	return &TCCCoordinator{
		transactions: make(map[string]*TCCTransaction),
		components:   make(map[string]TCCComponent),
		timeout:      30 * time.Second,
		retryCount:   3,
	}
}

// RegisterComponent 注册组件
func (tc *TCCCoordinator) RegisterComponent(component TCCComponent) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.components[component.ID()] = component
}

// BeginTransaction 开始事务
func (tc *TCCCoordinator) BeginTransaction(ctx context.Context, txID string, componentIDs []string) (*TCCTransaction, error) {
	// 检查组件是否存在
	var components []TCCComponent
	for _, id := range componentIDs {
		component, exists := tc.components[id]
		if !exists {
			return nil, fmt.Errorf("component not found: %s", id)
		}
		components = append(components, component)
	}

	// 创建事务
	transaction := &TCCTransaction{
		ID:         txID,
		Components: components,
		Status:     StatusPending,
		Phase:      PhaseTry,
		StartTime:  time.Now(),
		Timeout:    tc.timeout,
		RetryCount: 0,
		MaxRetries: tc.retryCount,
	}

	// 保存事务
	tc.mutex.Lock()
	tc.transactions[txID] = transaction
	tc.mutex.Unlock()

	return transaction, nil
}

// ExecuteTransaction 执行事务
func (tc *TCCCoordinator) ExecuteTransaction(ctx context.Context, txID string) error {
	tc.mutex.RLock()
	transaction, exists := tc.transactions[txID]
	tc.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("transaction not found: %s", txID)
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(ctx, transaction.Timeout)
	defer cancel()

	// 执行Try阶段
	if err := tc.executeTryPhase(ctx, transaction); err != nil {
		// Try阶段失败，执行Cancel阶段
		tc.executeCancelPhase(ctx, transaction)
		return err
	}

	// Try阶段成功，执行Confirm阶段
	return tc.executeConfirmPhase(ctx, transaction)
}

// executeTryPhase 执行Try阶段
func (tc *TCCCoordinator) executeTryPhase(ctx context.Context, transaction *TCCTransaction) error {
	transaction.mutex.Lock()
	transaction.Status = StatusTrying
	transaction.Phase = PhaseTry
	transaction.mutex.Unlock()

	var errors []error
	var successfulComponents []TCCComponent

	// 执行所有组件的Try操作
	for _, component := range transaction.Components {
		req := &TCCReq{
			ComponentID: component.ID(),
			TXID:        transaction.ID,
			Data:        make(map[string]interface{}),
		}

		var err error
		for attempt := 0; attempt <= transaction.MaxRetries; attempt++ {
			_, err = component.Try(ctx, req)
			if err == nil {
				break
			}
			if attempt < transaction.MaxRetries {
				time.Sleep(time.Duration(attempt+1) * time.Second)
			}
		}

		if err != nil {
			errors = append(errors, fmt.Errorf("component %s try failed: %w", component.ID(), err))
			// 对已成功的组件执行Cancel
			for _, successfulComponent := range successfulComponents {
				tc.cancelComponent(ctx, successfulComponent, transaction.ID)
			}
			break
		}

		successfulComponents = append(successfulComponents, component)
	}

	if len(errors) > 0 {
		transaction.mutex.Lock()
		transaction.Status = StatusFailed
		transaction.Error = fmt.Errorf("try phase failed: %v", errors)
		transaction.mutex.Unlock()
		return transaction.Error
	}

	return nil
}

// executeConfirmPhase 执行Confirm阶段
func (tc *TCCCoordinator) executeConfirmPhase(ctx context.Context, transaction *TCCTransaction) error {
	transaction.mutex.Lock()
	transaction.Status = StatusConfirmed
	transaction.Phase = PhaseConfirm
	transaction.mutex.Unlock()

	var errors []error

	// 执行所有组件的Confirm操作
	for _, component := range transaction.Components {
		var err error
		for attempt := 0; attempt <= transaction.MaxRetries; attempt++ {
			_, err = component.Confirm(ctx, transaction.ID)
			if err == nil {
				break
			}
			if attempt < transaction.MaxRetries {
				time.Sleep(time.Duration(attempt+1) * time.Second)
			}
		}

		if err != nil {
			errors = append(errors, fmt.Errorf("component %s confirm failed: %w", component.ID(), err))
		}
	}

	if len(errors) > 0 {
		transaction.mutex.Lock()
		transaction.Status = StatusFailed
		transaction.Error = fmt.Errorf("confirm phase failed: %v", errors)
		transaction.mutex.Unlock()
		return transaction.Error
	}

	// 设置结束时间
	now := time.Now()
	transaction.mutex.Lock()
	transaction.EndTime = &now
	transaction.mutex.Unlock()

	return nil
}

// executeCancelPhase 执行Cancel阶段
func (tc *TCCCoordinator) executeCancelPhase(ctx context.Context, transaction *TCCTransaction) error {
	transaction.mutex.Lock()
	transaction.Status = StatusCancelled
	transaction.Phase = PhaseCancel
	transaction.mutex.Unlock()

	var errors []error

	// 执行所有组件的Cancel操作
	for _, component := range transaction.Components {
		if err := tc.cancelComponent(ctx, component, transaction.ID); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		transaction.mutex.Lock()
		transaction.Status = StatusFailed
		transaction.Error = fmt.Errorf("cancel phase failed: %v", errors)
		transaction.mutex.Unlock()
		return transaction.Error
	}

	// 设置结束时间
	now := time.Now()
	transaction.mutex.Lock()
	transaction.EndTime = &now
	transaction.mutex.Unlock()

	return nil
}

// cancelComponent 取消组件
func (tc *TCCCoordinator) cancelComponent(ctx context.Context, component TCCComponent, txID string) error {
	var err error
	for attempt := 0; attempt <= tc.retryCount; attempt++ {
		_, err = component.Cancel(ctx, txID)
		if err == nil {
			break
		}
		if attempt < tc.retryCount {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("component %s cancel failed: %w", component.ID(), err)
	}

	return nil
}

// GetTransaction 获取事务
func (tc *TCCCoordinator) GetTransaction(txID string) (*TCCTransaction, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	transaction, exists := tc.transactions[txID]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", txID)
	}

	return transaction, nil
}

// ListTransactions 列出事务
func (tc *TCCCoordinator) ListTransactions(status TCCStatus) []*TCCTransaction {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	var transactions []*TCCTransaction
	for _, transaction := range tc.transactions {
		if transaction.Status == status {
			transactions = append(transactions, transaction)
		}
	}

	return transactions
}

// SetTimeout 设置超时时间
func (tc *TCCCoordinator) SetTimeout(timeout time.Duration) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.timeout = timeout
}

// SetRetryCount 设置重试次数
func (tc *TCCCoordinator) SetRetryCount(retryCount int) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.retryCount = retryCount
}

// CleanupTransactions 清理事务
func (tc *TCCCoordinator) CleanupTransactions(olderThan time.Duration) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for txID, transaction := range tc.transactions {
		if transaction.StartTime.Before(cutoff) {
			delete(tc.transactions, txID)
		}
	}
}

// TCCManager TCC管理器
type TCCManager struct {
	coordinators map[string]*TCCCoordinator
	mutex        sync.RWMutex
}

// NewTCCManager 创建TCC管理器
func NewTCCManager() *TCCManager {
	return &TCCManager{
		coordinators: make(map[string]*TCCCoordinator),
	}
}

// RegisterCoordinator 注册协调器
func (tm *TCCManager) RegisterCoordinator(name string, coordinator *TCCCoordinator) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tm.coordinators[name] = coordinator
}

// GetCoordinator 获取协调器
func (tm *TCCManager) GetCoordinator(name string) (*TCCCoordinator, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	coordinator, exists := tm.coordinators[name]
	if !exists {
		return nil, fmt.Errorf("coordinator not found: %s", name)
	}
	return coordinator, nil
}

// ListCoordinators 列出所有协调器
func (tm *TCCManager) ListCoordinators() []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var names []string
	for name := range tm.coordinators {
		names = append(names, name)
	}

	return names
}
