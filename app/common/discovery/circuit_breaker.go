package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed   CircuitState = iota // 关闭状态
	StateOpen                         // 开启状态
	StateHalfOpen                     // 半开状态
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from, to CircuitState)

	mu         sync.Mutex
	state      CircuitState
	generation uint64
	counts     Counts
	expiry     time.Time
}

// Counts 计数器
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// Settings 熔断器设置
type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from, to CircuitState)
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := new(CircuitBreaker)

	cb.name = st.Name
	cb.onStateChange = st.OnStateChange

	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}

	if st.Interval == 0 {
		cb.interval = time.Duration(0) * time.Second
	} else {
		cb.interval = st.Interval
	}

	if st.Timeout == 0 {
		cb.timeout = time.Duration(20) * time.Second
	} else {
		cb.timeout = st.Timeout
	}

	if st.ReadyToTrip == nil {
		cb.readyToTrip = defaultReadyToTrip
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// Execute 执行函数
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	return cb.ExecuteWithContext(context.Background(), req)
}

// ExecuteWithContext 带上下文的执行函数
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrOpenState
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.onRequest()
	return generation, nil
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(now)
	} else {
		cb.onFailure(now)
	}
}

// currentState 获取当前状态
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitState, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess(now time.Time) {
	cb.counts.onSuccess()
	if cb.state == StateHalfOpen {
		cb.setState(StateClosed, now)
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure(now time.Time) {
	cb.counts.onFailure()
	if cb.readyToTrip(cb.counts) {
		cb.setState(StateOpen, now)
	}
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state CircuitState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// toNewGeneration 生成新代
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// onRequest 请求计数
func (c *Counts) onRequest() {
	c.Requests++
}

// onSuccess 成功计数
func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

// onFailure 失败计数
func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// State 获取当前状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts 获取计数
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	_, _ = cb.currentState(now)
	return cb.counts
}

// defaultReadyToTrip 默认的熔断条件
func defaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures > 5
}

// 错误定义
var (
	ErrOpenState       = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests")
)

// CircuitBreakerManager 熔断器管理器
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerManager 创建熔断器管理器
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetCircuitBreaker 获取熔断器
func (cbm *CircuitBreakerManager) GetCircuitBreaker(name string) *CircuitBreaker {
	cbm.mutex.RLock()
	breaker, exists := cbm.breakers[name]
	cbm.mutex.RUnlock()

	if !exists {
		cbm.mutex.Lock()
		defer cbm.mutex.Unlock()

		// 双重检查
		if breaker, exists = cbm.breakers[name]; !exists {
			breaker = NewCircuitBreaker(Settings{
				Name:        name,
				MaxRequests: 3,
				Interval:    time.Duration(10) * time.Second,
				Timeout:     time.Duration(20) * time.Second,
			})
			cbm.breakers[name] = breaker
		}
	}

	return breaker
}

// RegisterCircuitBreaker 注册熔断器
func (cbm *CircuitBreakerManager) RegisterCircuitBreaker(name string, settings Settings) {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	cbm.breakers[name] = NewCircuitBreaker(settings)
}

// RemoveCircuitBreaker 移除熔断器
func (cbm *CircuitBreakerManager) RemoveCircuitBreaker(name string) {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	delete(cbm.breakers, name)
}

// ListCircuitBreakers 列出所有熔断器
func (cbm *CircuitBreakerManager) ListCircuitBreakers() []string {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	names := make([]string, 0, len(cbm.breakers))
	for name := range cbm.breakers {
		names = append(names, name)
	}
	return names
}

// GetCircuitBreakerStats 获取熔断器统计信息
func (cbm *CircuitBreakerManager) GetCircuitBreakerStats() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	stats := make(map[string]interface{})
	for name, breaker := range cbm.breakers {
		stats[name] = map[string]interface{}{
			"state":  breaker.State(),
			"counts": breaker.Counts(),
		}
	}
	return stats
}

// Get 获取熔断器（兼容增强熔断器接口）
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, error) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	breaker, exists := cbm.breakers[name]
	if !exists {
		return nil, fmt.Errorf("circuit breaker not found: %s", name)
	}
	return breaker, nil
}

// Register 注册熔断器（兼容增强熔断器接口）
func (cbm *CircuitBreakerManager) Register(name string, breaker *CircuitBreaker) {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()
	cbm.breakers[name] = breaker
}

// List 列出所有熔断器（兼容增强熔断器接口）
func (cbm *CircuitBreakerManager) List() []string {
	return cbm.ListCircuitBreakers()
}
