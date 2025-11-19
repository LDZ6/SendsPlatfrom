package performance

import (
	"context"
	"time"
)

// Semaphore 信号量
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore 创建信号量
func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, capacity),
	}
}

// Acquire 获取信号量
func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

// TryAcquire 尝试获取信号量
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// AcquireWithTimeout 带超时的获取信号量
func (s *Semaphore) AcquireWithTimeout(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case s.ch <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

// Release 释放信号量
func (s *Semaphore) Release() {
	<-s.ch
}

// Capacity 获取容量
func (s *Semaphore) Capacity() int {
	return cap(s.ch)
}

// Available 获取可用数量
func (s *Semaphore) Available() int {
	return cap(s.ch) - len(s.ch)
}

// Acquired 获取已获取数量
func (s *Semaphore) Acquired() int {
	return len(s.ch)
}

// RateLimiter 速率限制器
type RateLimiter struct {
	semaphore *Semaphore
	interval  time.Duration
	ticker    *time.Ticker
	quit      chan bool
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		semaphore: NewSemaphore(rate),
		interval:  interval,
		quit:      make(chan bool),
	}

	// 启动令牌桶
	rl.startTokenBucket()

	return rl
}

// startTokenBucket 启动令牌桶
func (rl *RateLimiter) startTokenBucket() {
	rl.ticker = time.NewTicker(rl.interval)
	go func() {
		for {
			select {
			case <-rl.ticker.C:
				// 释放一个令牌
				select {
				case <-rl.semaphore.ch:
				default:
					// 如果没有令牌被占用，跳过
				}
			case <-rl.quit:
				return
			}
		}
	}()
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	return rl.semaphore.TryAcquire()
}

// Wait 等待获取令牌
func (rl *RateLimiter) Wait() {
	rl.semaphore.Acquire()
}

// WaitWithTimeout 带超时的等待
func (rl *RateLimiter) WaitWithTimeout(timeout time.Duration) bool {
	return rl.semaphore.AcquireWithTimeout(timeout)
}

// Stop 停止速率限制器
func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
	rl.quit <- true
}

// GetStats 获取统计信息
func (rl *RateLimiter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"capacity":  rl.semaphore.Capacity(),
		"available": rl.semaphore.Available(),
		"acquired":  rl.semaphore.Acquired(),
		"interval":  rl.interval.String(),
	}
}

// AdaptiveRateLimiter 自适应速率限制器
type AdaptiveRateLimiter struct {
	baseRate     int
	currentRate  int
	maxRate      int
	minRate      int
	interval     time.Duration
	successCount int
	errorCount   int
	lastUpdate   time.Time
	rateLimiter  *RateLimiter
	mutex        chan struct{}
}

// NewAdaptiveRateLimiter 创建自适应速率限制器
func NewAdaptiveRateLimiter(baseRate, maxRate, minRate int, interval time.Duration) *AdaptiveRateLimiter {
	arl := &AdaptiveRateLimiter{
		baseRate:    baseRate,
		currentRate: baseRate,
		maxRate:     maxRate,
		minRate:     minRate,
		interval:    interval,
		lastUpdate:  time.Now(),
		mutex:       make(chan struct{}, 1),
	}

	arl.rateLimiter = NewRateLimiter(arl.currentRate, arl.interval)

	// 启动自适应调整
	go arl.adaptiveAdjust()

	return arl
}

// adaptiveAdjust 自适应调整
func (arl *AdaptiveRateLimiter) adaptiveAdjust() {
	ticker := time.NewTicker(arl.interval * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			arl.adjustRate()
		}
	}
}

// adjustRate 调整速率
func (arl *AdaptiveRateLimiter) adjustRate() {
	arl.mutex <- struct{}{}
	defer func() { <-arl.mutex }()

	// 计算成功率
	total := arl.successCount + arl.errorCount
	if total == 0 {
		return
	}

	successRate := float64(arl.successCount) / float64(total)

	// 根据成功率调整速率
	if successRate > 0.9 {
		// 成功率高，增加速率
		arl.currentRate = min(arl.currentRate+1, arl.maxRate)
	} else if successRate < 0.7 {
		// 成功率低，减少速率
		arl.currentRate = max(arl.currentRate-1, arl.minRate)
	}

	// 重新创建速率限制器
	arl.rateLimiter.Stop()
	arl.rateLimiter = NewRateLimiter(arl.currentRate, arl.interval)

	// 重置计数器
	arl.successCount = 0
	arl.errorCount = 0
	arl.lastUpdate = time.Now()
}

// RecordSuccess 记录成功
func (arl *AdaptiveRateLimiter) RecordSuccess() {
	arl.mutex <- struct{}{}
	arl.successCount++
	<-arl.mutex
}

// RecordError 记录错误
func (arl *AdaptiveRateLimiter) RecordError() {
	arl.mutex <- struct{}{}
	arl.errorCount++
	<-arl.mutex
}

// Allow 检查是否允许请求
func (arl *AdaptiveRateLimiter) Allow() bool {
	return arl.rateLimiter.Allow()
}

// Wait 等待获取令牌
func (arl *AdaptiveRateLimiter) Wait() {
	arl.rateLimiter.Wait()
}

// WaitWithTimeout 带超时的等待
func (arl *AdaptiveRateLimiter) WaitWithTimeout(timeout time.Duration) bool {
	return arl.rateLimiter.WaitWithTimeout(timeout)
}

// Stop 停止自适应速率限制器
func (arl *AdaptiveRateLimiter) Stop() {
	arl.rateLimiter.Stop()
}

// GetStats 获取统计信息
func (arl *AdaptiveRateLimiter) GetStats() map[string]interface{} {
	arl.mutex <- struct{}{}
	defer func() { <-arl.mutex }()

	stats := arl.rateLimiter.GetStats()
	stats["baseRate"] = arl.baseRate
	stats["currentRate"] = arl.currentRate
	stats["maxRate"] = arl.maxRate
	stats["minRate"] = arl.minRate
	stats["successCount"] = arl.successCount
	stats["errorCount"] = arl.errorCount
	stats["lastUpdate"] = arl.lastUpdate

	return stats
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
