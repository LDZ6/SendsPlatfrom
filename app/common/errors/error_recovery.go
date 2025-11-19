package errors

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeNetwork
	ErrorTypeDatabase
	ErrorTypeCache
	ErrorTypeExternalAPI
	ErrorTypeValidation
	ErrorTypeAuthentication
	ErrorTypeAuthorization
	ErrorTypeTimeout
	ErrorTypeRateLimit
	ErrorTypeCircuitBreaker
	ErrorTypeConfiguration
	ErrorTypeBusiness
	ErrorTypeSystem
)

// ErrorSeverity 错误严重程度
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// RecoveryStrategy 恢复策略接口
type RecoveryStrategy interface {
	CanRecover(err error) bool
	Recover(ctx context.Context, err error) error
	GetName() string
	GetPriority() int
}

// RetryStrategy 重试策略
type RetryStrategy struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
}

// NewRetryStrategy 创建重试策略
func NewRetryStrategy(maxRetries int, baseDelay time.Duration) *RetryStrategy {
	return &RetryStrategy{
		MaxRetries:    maxRetries,
		BaseDelay:     baseDelay,
		MaxDelay:      baseDelay * 10,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// CanRecover 检查是否可以恢复
func (rs *RetryStrategy) CanRecover(err error) bool {
	// 检查错误类型是否适合重试
	errorType := ClassifyError(err)
	switch errorType {
	case ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeRateLimit, ErrorTypeCircuitBreaker:
		return true
	case ErrorTypeDatabase, ErrorTypeCache, ErrorTypeExternalAPI:
		return true
	default:
		return false
	}
}

// Recover 执行恢复
func (rs *RetryStrategy) Recover(ctx context.Context, err error) error {
	for attempt := 0; attempt < rs.MaxRetries; attempt++ {
		// 计算延迟时间
		delay := rs.calculateDelay(attempt)

		// 等待
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// 这里应该重新执行原始操作
		// 由于我们不知道原始操作，这里只是模拟
		// 实际使用时需要传入重试函数
		return nil
	}

	return fmt.Errorf("retry failed after %d attempts: %w", rs.MaxRetries, err)
}

// GetName 获取策略名称
func (rs *RetryStrategy) GetName() string {
	return "retry"
}

// GetPriority 获取优先级
func (rs *RetryStrategy) GetPriority() int {
	return 1
}

// calculateDelay 计算延迟时间
func (rs *RetryStrategy) calculateDelay(attempt int) time.Duration {
	delay := rs.BaseDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * rs.BackoffFactor)
		if delay > rs.MaxDelay {
			delay = rs.MaxDelay
		}
	}

	if rs.Jitter {
		// 添加随机抖动
		jitter := time.Duration(float64(delay) * 0.1)
		delay += time.Duration(time.Now().UnixNano() % int64(jitter))
	}

	return delay
}

// CircuitBreakerStrategy 熔断器策略
type CircuitBreakerStrategy struct {
	breaker interface{} // 熔断器实例
}

// NewCircuitBreakerStrategy 创建熔断器策略
func NewCircuitBreakerStrategy(breaker interface{}) *CircuitBreakerStrategy {
	return &CircuitBreakerStrategy{
		breaker: breaker,
	}
}

// CanRecover 检查是否可以恢复
func (cbs *CircuitBreakerStrategy) CanRecover(err error) bool {
	// 检查错误类型是否适合熔断器
	errorType := ClassifyError(err)
	switch errorType {
	case ErrorTypeExternalAPI, ErrorTypeNetwork, ErrorTypeTimeout:
		return true
	default:
		return false
	}
}

// Recover 执行恢复
func (cbs *CircuitBreakerStrategy) Recover(ctx context.Context, err error) error {
	// 这里应该使用熔断器来执行操作
	// 由于我们不知道具体操作，这里只是模拟
	return nil
}

// GetName 获取策略名称
func (cbs *CircuitBreakerStrategy) GetName() string {
	return "circuit_breaker"
}

// GetPriority 获取优先级
func (cbs *CircuitBreakerStrategy) GetPriority() int {
	return 2
}

// FallbackStrategy 降级策略
type FallbackStrategy struct {
	fallbackFunc func(ctx context.Context) (interface{}, error)
}

// NewFallbackStrategy 创建降级策略
func NewFallbackStrategy(fallbackFunc func(ctx context.Context) (interface{}, error)) *FallbackStrategy {
	return &FallbackStrategy{
		fallbackFunc: fallbackFunc,
	}
}

// CanRecover 检查是否可以恢复
func (fs *FallbackStrategy) CanRecover(err error) bool {
	// 降级策略通常适用于所有错误
	return true
}

// Recover 执行恢复
func (fs *FallbackStrategy) Recover(ctx context.Context, err error) error {
	if fs.fallbackFunc != nil {
		_, err := fs.fallbackFunc(ctx)
		return err
	}
	return nil
}

// GetName 获取策略名称
func (fs *FallbackStrategy) GetName() string {
	return "fallback"
}

// GetPriority 获取优先级
func (fs *FallbackStrategy) GetPriority() int {
	return 3
}

// ErrorRecovery 错误恢复管理器
type ErrorRecovery struct {
	strategies map[string]RecoveryStrategy
	mutex      sync.RWMutex
	logger     interface{} // 日志接口
}

// NewErrorRecovery 创建错误恢复管理器
func NewErrorRecovery(logger interface{}) *ErrorRecovery {
	return &ErrorRecovery{
		strategies: make(map[string]RecoveryStrategy),
		logger:     logger,
	}
}

// RegisterStrategy 注册恢复策略
func (er *ErrorRecovery) RegisterStrategy(strategy RecoveryStrategy) {
	er.mutex.Lock()
	defer er.mutex.Unlock()
	er.strategies[strategy.GetName()] = strategy
}

// Recover 尝试恢复错误
func (er *ErrorRecovery) Recover(ctx context.Context, err error) error {
	er.mutex.RLock()
	strategies := make([]RecoveryStrategy, 0, len(er.strategies))
	for _, strategy := range er.strategies {
		strategies = append(strategies, strategy)
	}
	er.mutex.RUnlock()

	// 按优先级排序
	for i := 0; i < len(strategies)-1; i++ {
		for j := i + 1; j < len(strategies); j++ {
			if strategies[i].GetPriority() > strategies[j].GetPriority() {
				strategies[i], strategies[j] = strategies[j], strategies[i]
			}
		}
	}

	// 尝试每个策略
	for _, strategy := range strategies {
		if strategy.CanRecover(err) {
			if recoveryErr := strategy.Recover(ctx, err); recoveryErr == nil {
				// 恢复成功
				return nil
			}
		}
	}

	// 所有策略都失败
	return fmt.Errorf("all recovery strategies failed: %w", err)
}

// ErrorClassifier 错误分类器
type ErrorClassifier struct {
	rules map[ErrorType][]ErrorRule
	mutex sync.RWMutex
}

// ErrorRule 错误规则
type ErrorRule struct {
	Pattern     string
	ErrorType   ErrorType
	Severity    ErrorSeverity
	ShouldRetry bool
	ShouldLog   bool
	ShouldAlert bool
}

// NewErrorClassifier 创建错误分类器
func NewErrorClassifier() *ErrorClassifier {
	classifier := &ErrorClassifier{
		rules: make(map[ErrorType][]ErrorRule),
	}

	// 注册默认规则
	classifier.registerDefaultRules()

	return classifier
}

// registerDefaultRules 注册默认规则
func (ec *ErrorClassifier) registerDefaultRules() {
	// 网络错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "network",
		ErrorType:   ErrorTypeNetwork,
		Severity:    SeverityMedium,
		ShouldRetry: true,
		ShouldLog:   true,
		ShouldAlert: false,
	})

	// 数据库错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "database",
		ErrorType:   ErrorTypeDatabase,
		Severity:    SeverityHigh,
		ShouldRetry: true,
		ShouldLog:   true,
		ShouldAlert: true,
	})

	// 缓存错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "cache",
		ErrorType:   ErrorTypeCache,
		Severity:    SeverityLow,
		ShouldRetry: true,
		ShouldLog:   true,
		ShouldAlert: false,
	})

	// 外部API错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "external",
		ErrorType:   ErrorTypeExternalAPI,
		Severity:    SeverityMedium,
		ShouldRetry: true,
		ShouldLog:   true,
		ShouldAlert: false,
	})

	// 验证错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "validation",
		ErrorType:   ErrorTypeValidation,
		Severity:    SeverityLow,
		ShouldRetry: false,
		ShouldLog:   true,
		ShouldAlert: false,
	})

	// 认证错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "authentication",
		ErrorType:   ErrorTypeAuthentication,
		Severity:    SeverityHigh,
		ShouldRetry: false,
		ShouldLog:   true,
		ShouldAlert: true,
	})

	// 授权错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "authorization",
		ErrorType:   ErrorTypeAuthorization,
		Severity:    SeverityHigh,
		ShouldRetry: false,
		ShouldLog:   true,
		ShouldAlert: true,
	})

	// 超时错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "timeout",
		ErrorType:   ErrorTypeTimeout,
		Severity:    SeverityMedium,
		ShouldRetry: true,
		ShouldLog:   true,
		ShouldAlert: false,
	})

	// 限流错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "rate limit",
		ErrorType:   ErrorTypeRateLimit,
		Severity:    SeverityLow,
		ShouldRetry: true,
		ShouldLog:   true,
		ShouldAlert: false,
	})

	// 熔断器错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "circuit breaker",
		ErrorType:   ErrorTypeCircuitBreaker,
		Severity:    SeverityMedium,
		ShouldRetry: true,
		ShouldLog:   true,
		ShouldAlert: false,
	})

	// 配置错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "configuration",
		ErrorType:   ErrorTypeConfiguration,
		Severity:    SeverityCritical,
		ShouldRetry: false,
		ShouldLog:   true,
		ShouldAlert: true,
	})

	// 业务错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "business",
		ErrorType:   ErrorTypeBusiness,
		Severity:    SeverityMedium,
		ShouldRetry: false,
		ShouldLog:   true,
		ShouldAlert: false,
	})

	// 系统错误规则
	ec.AddRule(ErrorRule{
		Pattern:     "system",
		ErrorType:   ErrorTypeSystem,
		Severity:    SeverityCritical,
		ShouldRetry: false,
		ShouldLog:   true,
		ShouldAlert: true,
	})
}

// AddRule 添加规则
func (ec *ErrorClassifier) AddRule(rule ErrorRule) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.rules[rule.ErrorType] = append(ec.rules[rule.ErrorType], rule)
}

// ClassifyError 分类错误
func (ec *ErrorClassifier) ClassifyError(err error) (ErrorType, ErrorSeverity, bool, bool, bool) {
	if err == nil {
		return ErrorTypeUnknown, SeverityLow, false, false, false
	}

	errorMsg := err.Error()

	ec.mutex.RLock()
	defer ec.mutex.RUnlock()

	// 遍历所有规则
	for errorType, rules := range ec.rules {
		for _, rule := range rules {
			if contains(errorMsg, rule.Pattern) {
				return errorType, rule.Severity, rule.ShouldRetry, rule.ShouldLog, rule.ShouldAlert
			}
		}
	}

	// 默认分类
	return ErrorTypeUnknown, SeverityMedium, false, true, false
}

// contains 检查字符串是否包含子字符串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

// containsSubstring 检查字符串是否包含子字符串
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ClassifyError 全局错误分类函数
func ClassifyError(err error) ErrorType {
	classifier := NewErrorClassifier()
	errorType, _, _, _, _ := classifier.ClassifyError(err)
	return errorType
}

// ShouldRetry 检查是否应该重试
func ShouldRetry(err error) bool {
	classifier := NewErrorClassifier()
	_, _, shouldRetry, _, _ := classifier.ClassifyError(err)
	return shouldRetry
}

// GetRecoveryStrategy 获取恢复策略
func GetRecoveryStrategy(err error) RecoveryStrategy {
	errorType := ClassifyError(err)

	switch errorType {
	case ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeRateLimit:
		return NewRetryStrategy(3, time.Second)
	case ErrorTypeExternalAPI:
		return NewCircuitBreakerStrategy(nil)
	default:
		return NewFallbackStrategy(nil)
	}
}
