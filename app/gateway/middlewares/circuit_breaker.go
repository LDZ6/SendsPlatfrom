package middlewares

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)

	mutex      sync.Mutex
	state      CircuitBreakerState
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

// State 状态
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// String 状态字符串
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Settings 熔断器设置
type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
}

// DefaultReadyToTrip 默认的熔断条件
func DefaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures >= 5
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := new(CircuitBreaker)

	cb.name = st.Name
	cb.onStateChange = st.OnStateChange

	if st.MaxRequests == 0 {
		cb.maxRequests = 2
	} else {
		cb.maxRequests = st.MaxRequests
	}

	if st.Interval == 0 {
		cb.interval = time.Duration(60) * time.Second
	} else {
		cb.interval = st.Interval
	}

	if st.Timeout == 0 {
		cb.timeout = time.Duration(60) * time.Second
	} else {
		cb.timeout = st.Timeout
	}

	if st.ReadyToTrip == nil {
		cb.readyToTrip = DefaultReadyToTrip
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// Execute 执行操作
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
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
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, fmt.Errorf("circuit breaker is open")
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, fmt.Errorf("circuit breaker is half-open and too many requests")
	}

	cb.counts.onRequest()
	return generation, nil
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

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
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitBreakerState, uint64) {
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

// toNewGeneration 生成新状态
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.clear()

	var zero time.Time
	cb.expiry = zero
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state CircuitBreakerState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, State(prev), State(state))
	}
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess(now time.Time) {
	switch cb.state {
	case StateClosed:
		cb.counts.onSuccess()
	case StateHalfOpen:
		cb.counts.onSuccess()
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure(now time.Time) {
	switch cb.state {
	case StateClosed:
		cb.counts.onFailure()
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
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

// clear 清空计数
func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// CircuitBreakerMiddleware 熔断器中间件
func CircuitBreakerMiddleware(cb *CircuitBreaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := cb.Execute(func() (interface{}, error) {
			c.Next()
			return nil, nil
		})

		if err != nil {
			c.JSON(503, gin.H{
				"success":   false,
				"errorCode": 5030,
				"message":   "Service temporarily unavailable",
				"details":   err.Error(),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
		}
	}
}

// NewDefaultCircuitBreaker 创建默认熔断器
func NewDefaultCircuitBreaker(name string) *CircuitBreaker {
	return NewCircuitBreaker(Settings{
		Name:        name,
		MaxRequests: 2,
		Interval:    60 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from State, to State) {
			// 可以在这里添加日志记录
		},
	})
}
