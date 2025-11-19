package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"platform/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	GetLimit() int
	GetWindow() time.Duration
}

// TokenBucketRateLimiter 令牌桶限流器
type TokenBucketRateLimiter struct {
	capacity     int
	refillRate   int
	refillPeriod time.Duration
	redis        *redis.Client
	keyPrefix    string
}

// NewTokenBucketRateLimiter 创建令牌桶限流器
func NewTokenBucketRateLimiter(redis *redis.Client, capacity, refillRate int, refillPeriod time.Duration) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		capacity:     capacity,
		refillRate:   refillRate,
		refillPeriod: refillPeriod,
		redis:        redis,
		keyPrefix:    "rate_limit:token_bucket:",
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucketRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := tb.keyPrefix + key

	// 使用Lua脚本实现原子操作
	script := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refillRate = tonumber(ARGV[2])
		local refillPeriod = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		
		local bucket = redis.call('HMGET', key, 'tokens', 'lastRefill')
		local tokens = tonumber(bucket[1]) or capacity
		local lastRefill = tonumber(bucket[2]) or now
		
		-- 计算需要补充的令牌数
		local timePassed = now - lastRefill
		local tokensToAdd = math.floor(timePassed / refillPeriod) * refillRate
		
		-- 更新令牌数
		tokens = math.min(capacity, tokens + tokensToAdd)
		lastRefill = now
		
		-- 检查是否有足够的令牌
		if tokens >= 1 then
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'lastRefill', lastRefill)
			redis.call('EXPIRE', key, 3600) -- 设置过期时间
			return {1, tokens}
		else
			redis.call('HMSET', key, 'tokens', tokens, 'lastRefill', lastRefill)
			redis.call('EXPIRE', key, 3600)
			return {0, tokens}
		end
	`

	result, err := tb.redis.Eval(ctx, script, []string{redisKey},
		tb.capacity, tb.refillRate, int64(tb.refillPeriod.Seconds()), time.Now().Unix()).Result()
	if err != nil {
		return false, err
	}

	results := result.([]interface{})
	allowed := results[0].(int64) == 1
	remainingTokens := results[1].(int64)

	// 设置响应头
	ctx.Value("rate_limit_remaining").(*int) = int(remainingTokens)

	return allowed, nil
}

// GetLimit 获取限制数量
func (tb *TokenBucketRateLimiter) GetLimit() int {
	return tb.capacity
}

// GetWindow 获取时间窗口
func (tb *TokenBucketRateLimiter) GetWindow() time.Duration {
	return tb.refillPeriod
}

// SlidingWindowRateLimiter 滑动窗口限流器
type SlidingWindowRateLimiter struct {
	limit     int
	window    time.Duration
	redis     *redis.Client
	keyPrefix string
}

// NewSlidingWindowRateLimiter 创建滑动窗口限流器
func NewSlidingWindowRateLimiter(redis *redis.Client, limit int, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		limit:     limit,
		window:    window,
		redis:     redis,
		keyPrefix: "rate_limit:sliding_window:",
	}
}

// Allow 检查是否允许请求
func (sw *SlidingWindowRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := sw.keyPrefix + key
	now := time.Now()
	windowStart := now.Add(-sw.window)

	// 使用Lua脚本实现原子操作
	script := `
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local windowStart = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		-- 清理过期的记录
		redis.call('ZREMRANGEBYSCORE', key, 0, windowStart)
		
		-- 获取当前窗口内的请求数
		local current = redis.call('ZCARD', key)
		
		if current < limit then
			-- 添加当前请求
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, 3600) -- 设置过期时间
			return {1, limit - current - 1}
		else
			return {0, 0}
		end
	`

	result, err := sw.redis.Eval(ctx, script, []string{redisKey},
		sw.limit, windowStart.Unix(), now.Unix()).Result()
	if err != nil {
		return false, err
	}

	results := result.([]interface{})
	allowed := results[0].(int64) == 1
	remaining := results[1].(int64)

	// 设置响应头
	ctx.Value("rate_limit_remaining").(*int) = int(remaining)

	return allowed, nil
}

// GetLimit 获取限制数量
func (sw *SlidingWindowRateLimiter) GetLimit() int {
	return sw.limit
}

// GetWindow 获取时间窗口
func (sw *SlidingWindowRateLimiter) GetWindow() time.Duration {
	return sw.window
}

// FixedWindowRateLimiter 固定窗口限流器
type FixedWindowRateLimiter struct {
	limit     int
	window    time.Duration
	redis     *redis.Client
	keyPrefix string
}

// NewFixedWindowRateLimiter 创建固定窗口限流器
func NewFixedWindowRateLimiter(redis *redis.Client, limit int, window time.Duration) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		limit:     limit,
		window:    window,
		redis:     redis,
		keyPrefix: "rate_limit:fixed_window:",
	}
}

// Allow 检查是否允许请求
func (fw *FixedWindowRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	windowStart := now.Truncate(fw.window)
	redisKey := fw.keyPrefix + key + ":" + strconv.FormatInt(windowStart.Unix(), 10)

	// 使用Lua脚本实现原子操作
	script := `
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		
		local current = redis.call('GET', key)
		if current == false then
			current = 0
		else
			current = tonumber(current)
		end
		
		if current < limit then
			redis.call('INCR', key)
			redis.call('EXPIRE', key, window)
			return {1, limit - current - 1}
		else
			return {0, 0}
		end
	`

	result, err := fw.redis.Eval(ctx, script, []string{redisKey},
		fw.limit, int64(fw.window.Seconds())).Result()
	if err != nil {
		return false, err
	}

	results := result.([]interface{})
	allowed := results[0].(int64) == 1
	remaining := results[1].(int64)

	// 设置响应头
	ctx.Value("rate_limit_remaining").(*int) = int(remaining)

	return allowed, nil
}

// GetLimit 获取限制数量
func (fw *FixedWindowRateLimiter) GetLimit() int {
	return fw.limit
}

// GetWindow 获取时间窗口
func (fw *FixedWindowRateLimiter) GetWindow() time.Duration {
	return fw.window
}

// RateLimiterManager 限流器管理器
type RateLimiterManager struct {
	limiters map[string]RateLimiter
	mutex    sync.RWMutex
}

// NewRateLimiterManager 创建限流器管理器
func NewRateLimiterManager() *RateLimiterManager {
	return &RateLimiterManager{
		limiters: make(map[string]RateLimiter),
	}
}

// RegisterRateLimiter 注册限流器
func (rlm *RateLimiterManager) RegisterRateLimiter(name string, limiter RateLimiter) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.limiters[name] = limiter
}

// GetRateLimiter 获取限流器
func (rlm *RateLimiterManager) GetRateLimiter(name string) (RateLimiter, bool) {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()
	limiter, exists := rlm.limiters[name]
	return limiter, exists
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(limiterName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取限流器管理器（从上下文或全局变量获取）
		rlm := c.MustGet("rate_limiter_manager").(*RateLimiterManager)

		limiter, exists := rlm.GetRateLimiter(limiterName)
		if !exists {
			c.Next()
			return
		}

		// 生成限流键
		key := generateRateLimitKey(c)

		// 检查是否允许请求
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			// 限流器错误，记录日志但允许请求通过
			utils.GetLogger().WithError(err).Error("Rate limiter error")
			c.Next()
			return
		}

		if !allowed {
			// 请求被限流
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success":   false,
				"errorCode": 429,
				"message":   "Rate limit exceeded",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 设置响应头
		if remaining, ok := c.Get("rate_limit_remaining"); ok {
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining.(int)))
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.GetLimit()))
		c.Header("X-RateLimit-Window", limiter.GetWindow().String())

		c.Next()
	}
}

// generateRateLimitKey 生成限流键
func generateRateLimitKey(c *gin.Context) string {
	// 可以根据需要自定义键的生成策略
	// 例如：IP + UserID + Path
	ip := c.ClientIP()
	userID := c.GetString("user_id")
	path := c.Request.URL.Path

	if userID != "" {
		return fmt.Sprintf("%s:%s:%s", ip, userID, path)
	}
	return fmt.Sprintf("%s:%s", ip, path)
}

// AdaptiveRateLimiter 自适应限流器
type AdaptiveRateLimiter struct {
	baseLimiter    RateLimiter
	adaptiveFactor float64
	redis          *redis.Client
	keyPrefix      string
}

// NewAdaptiveRateLimiter 创建自适应限流器
func NewAdaptiveRateLimiter(baseLimiter RateLimiter, redis *redis.Client) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseLimiter:    baseLimiter,
		adaptiveFactor: 1.0,
		redis:          redis,
		keyPrefix:      "rate_limit:adaptive:",
	}
}

// Allow 检查是否允许请求
func (ar *AdaptiveRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// 获取系统负载
	load, err := ar.getSystemLoad(ctx)
	if err != nil {
		// 如果获取负载失败，使用基础限流器
		return ar.baseLimiter.Allow(ctx, key)
	}

	// 根据负载调整限流因子
	if load > 0.8 {
		ar.adaptiveFactor = 0.5 // 高负载时降低限流阈值
	} else if load < 0.3 {
		ar.adaptiveFactor = 1.5 // 低负载时提高限流阈值
	} else {
		ar.adaptiveFactor = 1.0 // 正常负载
	}

	// 使用调整后的限流器
	return ar.baseLimiter.Allow(ctx, key)
}

// getSystemLoad 获取系统负载
func (ar *AdaptiveRateLimiter) getSystemLoad(ctx context.Context) (float64, error) {
	// 这里可以实现获取系统负载的逻辑
	// 例如：CPU使用率、内存使用率、请求响应时间等
	// 简化实现，返回固定值
	return 0.5, nil
}

// GetLimit 获取限制数量
func (ar *AdaptiveRateLimiter) GetLimit() int {
	return int(float64(ar.baseLimiter.GetLimit()) * ar.adaptiveFactor)
}

// GetWindow 获取时间窗口
func (ar *AdaptiveRateLimiter) GetWindow() time.Duration {
	return ar.baseLimiter.GetWindow()
}
