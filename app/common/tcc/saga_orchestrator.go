package tcc

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SagaStep Saga步骤
type SagaStep struct {
	ID           string
	Name         string
	Action       func(ctx context.Context, data interface{}) error
	Compensation func(ctx context.Context, data interface{}) error
	Timeout      time.Duration
	RetryCount   int
	Data         interface{}
}

// CompensationStep 补偿步骤
type CompensationStep struct {
	StepID       string
	Compensation func(ctx context.Context, data interface{}) error
	Data         interface{}
}

// SagaState Saga状态
type SagaState int

const (
	SagaStatePending SagaState = iota
	SagaStateRunning
	SagaStateCompleted
	SagaStateCompensating
	SagaStateCompensated
	SagaStateFailed
)

// String 状态字符串
func (s SagaState) String() string {
	switch s {
	case SagaStatePending:
		return "pending"
	case SagaStateRunning:
		return "running"
	case SagaStateCompleted:
		return "completed"
	case SagaStateCompensating:
		return "compensating"
	case SagaStateCompensated:
		return "compensated"
	case SagaStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// SagaExecution Saga执行记录
type SagaExecution struct {
	ID          string
	Steps       []SagaStep
	State       SagaState
	StartTime   time.Time
	EndTime     *time.Time
	Error       error
	Compensated bool
	mutex       sync.RWMutex
}

// StateManager 状态管理器接口
type StateManager interface {
	SaveExecution(execution *SagaExecution) error
	GetExecution(id string) (*SagaExecution, error)
	UpdateExecution(execution *SagaExecution) error
	DeleteExecution(id string) error
	ListExecutions(state SagaState) ([]*SagaExecution, error)
}

// SagaOrchestrator Saga编排器
type SagaOrchestrator struct {
	steps         []SagaStep
	compensations []CompensationStep
	stateManager  StateManager
	timeout       time.Duration
	retryCount    int
	mutex         sync.RWMutex
}

// NewSagaOrchestrator 创建Saga编排器
func NewSagaOrchestrator(stateManager StateManager) *SagaOrchestrator {
	return &SagaOrchestrator{
		stateManager: stateManager,
		timeout:      30 * time.Second,
		retryCount:   3,
	}
}

// AddStep 添加步骤
func (so *SagaOrchestrator) AddStep(step SagaStep) {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	so.steps = append(so.steps, step)
}

// AddCompensation 添加补偿步骤
func (so *SagaOrchestrator) AddCompensation(compensation CompensationStep) {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	so.compensations = append(so.compensations, compensation)
}

// Execute 执行Saga
func (so *SagaOrchestrator) Execute(ctx context.Context, sagaID string) error {
	so.mutex.RLock()
	steps := make([]SagaStep, len(so.steps))
	copy(steps, so.steps)
	so.mutex.RUnlock()

	// 创建执行记录
	execution := &SagaExecution{
		ID:        sagaID,
		Steps:     steps,
		State:     SagaStatePending,
		StartTime: time.Now(),
	}

	// 保存执行记录
	if err := so.stateManager.SaveExecution(execution); err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(ctx, so.timeout)
	defer cancel()

	// 执行步骤
	execution.State = SagaStateRunning
	if err := so.stateManager.UpdateExecution(execution); err != nil {
		return fmt.Errorf("failed to update execution state: %w", err)
	}

	var executedSteps []SagaStep
	var lastError error

	for _, step := range steps {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			execution.State = SagaStateFailed
			execution.Error = ctx.Err()
			so.stateManager.UpdateExecution(execution)
			return ctx.Err()
		default:
		}

		// 执行步骤
		if err := so.executeStep(ctx, step); err != nil {
			lastError = err
			execution.State = SagaStateFailed
			execution.Error = err
			so.stateManager.UpdateExecution(execution)

			// 执行补偿
			if err := so.compensate(ctx, executedSteps); err != nil {
				return fmt.Errorf("saga failed and compensation failed: %w, compensation error: %v", lastError, err)
			}

			execution.State = SagaStateCompensated
			so.stateManager.UpdateExecution(execution)
			return lastError
		}

		executedSteps = append(executedSteps, step)
	}

	// 所有步骤执行成功
	execution.State = SagaStateCompleted
	now := time.Now()
	execution.EndTime = &now
	so.stateManager.UpdateExecution(execution)

	return nil
}

// executeStep 执行单个步骤
func (so *SagaOrchestrator) executeStep(ctx context.Context, step SagaStep) error {
	var lastError error

	for attempt := 0; attempt <= step.RetryCount; attempt++ {
		// 设置步骤超时
		stepCtx := ctx
		if step.Timeout > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
			defer cancel()
		}

		// 执行步骤
		if err := step.Action(stepCtx, step.Data); err != nil {
			lastError = err
			if attempt < step.RetryCount {
				// 等待后重试
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return err
		}

		// 执行成功
		return nil
	}

	return lastError
}

// compensate 执行补偿
func (so *SagaOrchestrator) compensate(ctx context.Context, executedSteps []SagaStep) error {
	so.mutex.RLock()
	compensations := make([]CompensationStep, len(so.compensations))
	copy(compensations, so.compensations)
	so.mutex.RUnlock()

	// 按相反顺序执行补偿
	for i := len(executedSteps) - 1; i >= 0; i-- {
		step := executedSteps[i]

		// 查找对应的补偿步骤
		var compensation *CompensationStep
		for _, comp := range compensations {
			if comp.StepID == step.ID {
				compensation = &comp
				break
			}
		}

		// 如果没有找到补偿步骤，使用步骤自带的补偿函数
		if compensation == nil && step.Compensation != nil {
			compensation = &CompensationStep{
				StepID:       step.ID,
				Compensation: step.Compensation,
				Data:         step.Data,
			}
		}

		// 执行补偿
		if compensation != nil {
			if err := compensation.Compensation(ctx, compensation.Data); err != nil {
				// 补偿失败，记录错误但继续执行其他补偿
				// 这里可以添加日志记录
				continue
			}
		}
	}

	return nil
}

// GetExecution 获取执行记录
func (so *SagaOrchestrator) GetExecution(id string) (*SagaExecution, error) {
	return so.stateManager.GetExecution(id)
}

// ListExecutions 列出执行记录
func (so *SagaOrchestrator) ListExecutions(state SagaState) ([]*SagaExecution, error) {
	return so.stateManager.ListExecutions(state)
}

// SetTimeout 设置超时时间
func (so *SagaOrchestrator) SetTimeout(timeout time.Duration) {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	so.timeout = timeout
}

// SetRetryCount 设置重试次数
func (so *SagaOrchestrator) SetRetryCount(retryCount int) {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	so.retryCount = retryCount
}

// MemoryStateManager 内存状态管理器
type MemoryStateManager struct {
	executions map[string]*SagaExecution
	mutex      sync.RWMutex
}

// NewMemoryStateManager 创建内存状态管理器
func NewMemoryStateManager() *MemoryStateManager {
	return &MemoryStateManager{
		executions: make(map[string]*SagaExecution),
	}
}

// SaveExecution 保存执行记录
func (msm *MemoryStateManager) SaveExecution(execution *SagaExecution) error {
	msm.mutex.Lock()
	defer msm.mutex.Unlock()
	msm.executions[execution.ID] = execution
	return nil
}

// GetExecution 获取执行记录
func (msm *MemoryStateManager) GetExecution(id string) (*SagaExecution, error) {
	msm.mutex.RLock()
	defer msm.mutex.RUnlock()
	execution, exists := msm.executions[id]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	return execution, nil
}

// UpdateExecution 更新执行记录
func (msm *MemoryStateManager) UpdateExecution(execution *SagaExecution) error {
	msm.mutex.Lock()
	defer msm.mutex.Unlock()
	msm.executions[execution.ID] = execution
	return nil
}

// DeleteExecution 删除执行记录
func (msm *MemoryStateManager) DeleteExecution(id string) error {
	msm.mutex.Lock()
	defer msm.mutex.Unlock()
	delete(msm.executions, id)
	return nil
}

// ListExecutions 列出执行记录
func (msm *MemoryStateManager) ListExecutions(state SagaState) ([]*SagaExecution, error) {
	msm.mutex.RLock()
	defer msm.mutex.RUnlock()

	var executions []*SagaExecution
	for _, execution := range msm.executions {
		if execution.State == state {
			executions = append(executions, execution)
		}
	}

	return executions, nil
}

// SagaManager Saga管理器
type SagaManager struct {
	orchestrators map[string]*SagaOrchestrator
	stateManager  StateManager
	mutex         sync.RWMutex
}

// NewSagaManager 创建Saga管理器
func NewSagaManager(stateManager StateManager) *SagaManager {
	return &SagaManager{
		orchestrators: make(map[string]*SagaOrchestrator),
		stateManager:  stateManager,
	}
}

// RegisterOrchestrator 注册编排器
func (sm *SagaManager) RegisterOrchestrator(name string, orchestrator *SagaOrchestrator) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.orchestrators[name] = orchestrator
}

// GetOrchestrator 获取编排器
func (sm *SagaManager) GetOrchestrator(name string) (*SagaOrchestrator, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	orchestrator, exists := sm.orchestrators[name]
	if !exists {
		return nil, fmt.Errorf("orchestrator not found: %s", name)
	}
	return orchestrator, nil
}

// ExecuteSaga 执行Saga
func (sm *SagaManager) ExecuteSaga(ctx context.Context, orchestratorName, sagaID string) error {
	orchestrator, err := sm.GetOrchestrator(orchestratorName)
	if err != nil {
		return err
	}

	return orchestrator.Execute(ctx, sagaID)
}

// ListOrchestrators 列出所有编排器
func (sm *SagaManager) ListOrchestrators() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var names []string
	for name := range sm.orchestrators {
		names = append(names, name)
	}

	return names
}
