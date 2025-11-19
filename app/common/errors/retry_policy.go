package errors

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// BackoffStrategy 退避策略接口
type BackoffStrategy interface {
	CalculateDelay(attempt int, baseDelay time.Duration) time.Duration
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	BackoffStrategy BackoffStrategy
	RetryableErrors []error
	Jitter          bool
	Context         context.Context
}

// NewRetryPolicy 创建重试策略
func NewRetryPolicy(maxRetries int, baseDelay time.Duration) *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:      maxRetries,
		BaseDelay:       baseDelay,
		MaxDelay:        baseDelay * 10,
		BackoffStrategy: &ExponentialBackoff{},
		RetryableErrors: []error{},
		Jitter:          true,
		Context:         context.Background(),
	}
}

// SetBackoffStrategy 设置退避策略
func (rp *RetryPolicy) SetBackoffStrategy(strategy BackoffStrategy) {
	rp.BackoffStrategy = strategy
}

// SetRetryableErrors 设置可重试错误
func (rp *RetryPolicy) SetRetryableErrors(errors []error) {
	rp.RetryableErrors = errors
}

// SetJitter 设置抖动
func (rp *RetryPolicy) SetJitter(jitter bool) {
	rp.Jitter = jitter
}

// SetContext 设置上下文
func (rp *RetryPolicy) SetContext(ctx context.Context) {
	rp.Context = ctx
}

// IsRetryable 检查错误是否可重试
func (rp *RetryPolicy) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 检查错误类型
	errorType := ClassifyError(err)
	switch errorType {
	case ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeRateLimit, ErrorTypeCircuitBreaker:
		return true
	case ErrorTypeDatabase, ErrorTypeCache, ErrorTypeExternalAPI:
		return true
	default:
		// 检查是否在可重试错误列表中
		for _, retryableErr := range rp.RetryableErrors {
			if err == retryableErr {
				return true
			}
		}
		return false
	}
}

// Execute 执行重试
func (rp *RetryPolicy) Execute(operation func() error) error {
	var lastError error

	for attempt := 0; attempt <= rp.MaxRetries; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-rp.Context.Done():
			return rp.Context.Err()
		default:
		}

		// 执行操作
		err := operation()
		if err == nil {
			return nil
		}

		lastError = err

		// 检查是否可重试
		if !rp.IsRetryable(err) {
			return err
		}

		// 如果是最后一次尝试，直接返回错误
		if attempt == rp.MaxRetries {
			break
		}

		// 计算延迟时间
		delay := rp.calculateDelay(attempt)

		// 等待
		select {
		case <-rp.Context.Done():
			return rp.Context.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", rp.MaxRetries, lastError)
}

// calculateDelay 计算延迟时间
func (rp *RetryPolicy) calculateDelay(attempt int) time.Duration {
	delay := rp.BackoffStrategy.CalculateDelay(attempt, rp.BaseDelay)

	// 限制最大延迟
	if delay > rp.MaxDelay {
		delay = rp.MaxDelay
	}

	// 添加抖动
	if rp.Jitter {
		jitter := time.Duration(float64(delay) * 0.1)
		delay += time.Duration(rand.Int63n(int64(jitter)))
	}

	return delay
}

// ExponentialBackoff 指数退避策略
type ExponentialBackoff struct {
	Factor float64
}

// NewExponentialBackoff 创建指数退避策略
func NewExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		Factor: 2.0,
	}
}

// CalculateDelay 计算延迟时间
func (eb *ExponentialBackoff) CalculateDelay(attempt int, baseDelay time.Duration) time.Duration {
	delay := float64(baseDelay) * math.Pow(eb.Factor, float64(attempt))
	return time.Duration(delay)
}

// LinearBackoff 线性退避策略
type LinearBackoff struct {
	Increment time.Duration
}

// NewLinearBackoff 创建线性退避策略
func NewLinearBackoff() *LinearBackoff {
	return &LinearBackoff{
		Increment: time.Second,
	}
}

// CalculateDelay 计算延迟时间
func (lb *LinearBackoff) CalculateDelay(attempt int, baseDelay time.Duration) time.Duration {
	return baseDelay + time.Duration(attempt)*lb.Increment
}

// FixedBackoff 固定退避策略
type FixedBackoff struct{}

// NewFixedBackoff 创建固定退避策略
func NewFixedBackoff() *FixedBackoff {
	return &FixedBackoff{}
}

// CalculateDelay 计算延迟时间
func (fb *FixedBackoff) CalculateDelay(attempt int, baseDelay time.Duration) time.Duration {
	return baseDelay
}

// CustomBackoff 自定义退避策略
type CustomBackoff struct {
	Delays []time.Duration
}

// NewCustomBackoff 创建自定义退避策略
func NewCustomBackoff(delays []time.Duration) *CustomBackoff {
	return &CustomBackoff{
		Delays: delays,
	}
}

// CalculateDelay 计算延迟时间
func (cb *CustomBackoff) CalculateDelay(attempt int, baseDelay time.Duration) time.Duration {
	if attempt < len(cb.Delays) {
		return cb.Delays[attempt]
	}
	return baseDelay
}

// RetryExecutor 重试执行器
type RetryExecutor struct {
	policies map[string]*RetryPolicy
	mutex    sync.RWMutex
}

// NewRetryExecutor 创建重试执行器
func NewRetryExecutor() *RetryExecutor {
	return &RetryExecutor{
		policies: make(map[string]*RetryPolicy),
	}
}

// RegisterPolicy 注册重试策略
func (re *RetryExecutor) RegisterPolicy(name string, policy *RetryPolicy) {
	re.mutex.Lock()
	defer re.mutex.Unlock()
	re.policies[name] = policy
}

// GetPolicy 获取重试策略
func (re *RetryExecutor) GetPolicy(name string) (*RetryPolicy, error) {
	re.mutex.RLock()
	defer re.mutex.RUnlock()
	policy, exists := re.policies[name]
	if !exists {
		return nil, fmt.Errorf("retry policy not found: %s", name)
	}
	return policy, nil
}

// Execute 执行重试
func (re *RetryExecutor) Execute(policyName string, operation func() error) error {
	policy, err := re.GetPolicy(policyName)
	if err != nil {
		return err
	}
	return policy.Execute(operation)
}

// ExecuteWithContext 使用上下文执行重试
func (re *RetryExecutor) ExecuteWithContext(ctx context.Context, policyName string, operation func() error) error {
	policy, err := re.GetPolicy(policyName)
	if err != nil {
		return err
	}

	// 创建带上下文的策略副本
	policyWithContext := *policy
	policyWithContext.Context = ctx

	return policyWithContext.Execute(operation)
}

// ListPolicies 列出所有策略
func (re *RetryExecutor) ListPolicies() []string {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	var names []string
	for name := range re.policies {
		names = append(names, name)
	}

	return names
}

// RetryableOperation 可重试操作接口
type RetryableOperation interface {
	Execute() error
	ShouldRetry(err error) bool
	GetRetryPolicy() *RetryPolicy
}

// RetryableFunction 可重试函数
type RetryableFunction struct {
	operation func() error
	policy    *RetryPolicy
}

// NewRetryableFunction 创建可重试函数
func NewRetryableFunction(operation func() error, policy *RetryPolicy) *RetryableFunction {
	return &RetryableFunction{
		operation: operation,
		policy:    policy,
	}
}

// Execute 执行操作
func (rf *RetryableFunction) Execute() error {
	return rf.operation()
}

// ShouldRetry 检查是否应该重试
func (rf *RetryableFunction) ShouldRetry(err error) bool {
	return rf.policy.IsRetryable(err)
}

// GetRetryPolicy 获取重试策略
func (rf *RetryableFunction) GetRetryPolicy() *RetryPolicy {
	return rf.policy
}

// ExecuteWithRetry 执行带重试的操作
func ExecuteWithRetry(operation RetryableOperation) error {
	policy := operation.GetRetryPolicy()
	return policy.Execute(operation.Execute)
}

// ExecuteWithRetryAndContext 使用上下文执行带重试的操作
func ExecuteWithRetryAndContext(ctx context.Context, operation RetryableOperation) error {
	policy := operation.GetRetryPolicy()
	policyWithContext := *policy
	policyWithContext.Context = ctx
	return policyWithContext.Execute(operation.Execute)
}
