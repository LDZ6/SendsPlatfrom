package container

import (
	"context"
	"crypto/md5"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"platform/utils"

	"github.com/redis/go-redis/v9"
)

// CacheStrategy 缓存策略接口
type CacheStrategy interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// AdvancedCache 高级缓存实现
type AdvancedCache struct {
	client         *redis.Client
	localCache     *LocalCache
	penetration    *PenetrationProtection
	avalanche      *AvalancheProtection
	warmup         *WarmupManager
	circuitBreaker *CircuitBreaker
}

// NewAdvancedCache 创建高级缓存
func NewAdvancedCache(client *redis.Client) *AdvancedCache {
	return &AdvancedCache{
		client:         client,
		localCache:     NewLocalCache(),
		penetration:    NewPenetrationProtection(),
		avalanche:      NewAvalancheProtection(),
		warmup:         NewWarmupManager(client),
		circuitBreaker: NewCircuitBreaker(),
	}
}

// Get 获取缓存值
func (ac *AdvancedCache) Get(ctx context.Context, key string) (interface{}, error) {
	// 1. 检查本地缓存
	if value, found := ac.localCache.Get(key); found {
		return value, nil
	}

	// 2. 检查Redis缓存
	value, err := ac.client.Get(ctx, key).Result()
	if err == nil {
		// 回写到本地缓存
		ac.localCache.Set(key, value)
		return value, nil
	}

	// 3. 缓存穿透防护
	if ac.penetration.IsBlocked(key) {
		return nil, utils.WrapError(fmt.Errorf("key %s is blocked", key),
			utils.ErrCodeCacheMiss, "缓存穿透防护")
	}

	// 4. 缓存雪崩防护
	if ac.avalanche.IsAvalanche() {
		return nil, utils.WrapError(fmt.Errorf("cache avalanche detected"),
			utils.ErrCodeServiceUnavailable, "缓存雪崩")
	}

	return nil, utils.WrapError(err, utils.ErrCodeCacheMiss, "缓存未命中")
}

// Set 设置缓存值
func (ac *AdvancedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 1. 设置本地缓存
	ac.localCache.Set(key, value)

	// 2. 设置Redis缓存（添加随机过期时间防止雪崩）
	randomTTL := ac.avalanche.AddRandomTTL(ttl)
	err := ac.client.Set(ctx, key, value, randomTTL).Err()
	if err != nil {
		return utils.WrapRedisError(err, "设置缓存")
	}

	// 3. 记录缓存设置
	ac.penetration.RecordSet(key)

	return nil
}

// Delete 删除缓存
func (ac *AdvancedCache) Delete(ctx context.Context, key string) error {
	// 1. 删除本地缓存
	ac.localCache.Delete(key)

	// 2. 删除Redis缓存
	err := ac.client.Del(ctx, key).Err()
	if err != nil {
		return utils.WrapRedisError(err, "删除缓存")
	}

	return nil
}

// Exists 检查缓存是否存在
func (ac *AdvancedCache) Exists(ctx context.Context, key string) (bool, error) {
	// 1. 检查本地缓存
	if _, found := ac.localCache.Get(key); found {
		return true, nil
	}

	// 2. 检查Redis缓存
	exists, err := ac.client.Exists(ctx, key).Result()
	if err != nil {
		return false, utils.WrapRedisError(err, "检查缓存存在性")
	}

	return exists > 0, nil
}

// Warmup 缓存预热
func (ac *AdvancedCache) Warmup(ctx context.Context, keys []string, loader func(string) (interface{}, error)) error {
	return ac.warmup.Warmup(ctx, keys, loader)
}

// PenetrationProtection 缓存穿透防护
type PenetrationProtection struct {
	blockedKeys map[string]time.Time
	mutex       sync.RWMutex
	blockTime   time.Duration
}

// NewPenetrationProtection 创建穿透防护
func NewPenetrationProtection() *PenetrationProtection {
	return &PenetrationProtection{
		blockedKeys: make(map[string]time.Time),
		blockTime:   5 * time.Minute, // 阻塞5分钟
	}
}

// IsBlocked 检查key是否被阻塞
func (pp *PenetrationProtection) IsBlocked(key string) bool {
	pp.mutex.RLock()
	defer pp.mutex.RUnlock()

	if blockTime, exists := pp.blockedKeys[key]; exists {
		if time.Since(blockTime) < pp.blockTime {
			return true
		}
		// 过期了，删除
		delete(pp.blockedKeys, key)
	}
	return false
}

// BlockKey 阻塞key
func (pp *PenetrationProtection) BlockKey(key string) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	pp.blockedKeys[key] = time.Now()
}

// RecordSet 记录缓存设置
func (pp *PenetrationProtection) RecordSet(key string) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	delete(pp.blockedKeys, key)
}

// AvalancheProtection 缓存雪崩防护
type AvalancheProtection struct {
	lastCheck time.Time
	missCount int
	mutex     sync.RWMutex
	threshold int
	window    time.Duration
}

// NewAvalancheProtection 创建雪崩防护
func NewAvalancheProtection() *AvalancheProtection {
	return &AvalancheProtection{
		threshold: 100,             // 100次缓存未命中
		window:    1 * time.Minute, // 1分钟窗口
	}
}

// IsAvalanche 检查是否发生雪崩
func (ap *AvalancheProtection) IsAvalanche() bool {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	now := time.Now()
	if now.Sub(ap.lastCheck) > ap.window {
		ap.missCount = 0
		ap.lastCheck = now
	}

	ap.missCount++
	return ap.missCount > ap.threshold
}

// AddRandomTTL 添加随机TTL防止雪崩
func (ap *AvalancheProtection) AddRandomTTL(baseTTL time.Duration) time.Duration {
	// 添加±10%的随机时间
	randomFactor := 0.8 + rand.Float64()*0.4 // 0.8-1.2
	return time.Duration(float64(baseTTL) * randomFactor)
}

// WarmupManager 缓存预热管理器
type WarmupManager struct {
	client *redis.Client
	mutex  sync.RWMutex
}

// NewWarmupManager 创建预热管理器
func NewWarmupManager(client *redis.Client) *WarmupManager {
	return &WarmupManager{
		client: client,
	}
}

// Warmup 执行缓存预热
func (wm *WarmupManager) Warmup(ctx context.Context, keys []string, loader func(string) (interface{}, error)) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	for _, key := range keys {
		// 检查是否已存在
		exists, err := wm.client.Exists(ctx, key).Result()
		if err != nil {
			continue
		}
		if exists > 0 {
			continue
		}

		// 加载数据
		value, err := loader(key)
		if err != nil {
			continue
		}

		// 设置缓存
		wm.client.Set(ctx, key, value, time.Hour)
	}

	return nil
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	failures    int
	lastFailure time.Time
	mutex       sync.RWMutex
	threshold   int
	timeout     time.Duration
	state       string // "closed", "open", "half-open"
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		threshold: 5,                // 5次失败
		timeout:   30 * time.Second, // 30秒超时
		state:     "closed",
	}
}

// Call 执行调用
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// 检查熔断器状态
	if cb.state == "open" {
		if time.Since(cb.lastFailure) < cb.timeout {
			return utils.WrapError(fmt.Errorf("circuit breaker is open"),
				utils.ErrCodeServiceUnavailable, "熔断器开启")
		}
		cb.state = "half-open"
	}

	// 执行调用
	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.threshold {
			cb.state = "open"
		}
		return err
	}

	// 成功，重置计数器
	if cb.state == "half-open" {
		cb.state = "closed"
	}
	cb.failures = 0

	return nil
}

// CacheWithFallback 带降级的缓存
type CacheWithFallback struct {
	cache    CacheStrategy
	fallback func(string) (interface{}, error)
}

// NewCacheWithFallback 创建带降级的缓存
func NewCacheWithFallback(cache CacheStrategy, fallback func(string) (interface{}, error)) *CacheWithFallback {
	return &CacheWithFallback{
		cache:    cache,
		fallback: fallback,
	}
}

// Get 获取缓存，失败时使用降级
func (cf *CacheWithFallback) Get(ctx context.Context, key string) (interface{}, error) {
	// 尝试从缓存获取
	value, err := cf.cache.Get(ctx, key)
	if err == nil {
		return value, nil
	}

	// 缓存失败，使用降级
	if cf.fallback != nil {
		value, err := cf.fallback(key)
		if err != nil {
			return nil, err
		}
		// 异步回写缓存
		go func() {
			cf.cache.Set(context.Background(), key, value, time.Hour)
		}()
		return value, nil
	}

	return nil, err
}

// BloomFilter 布隆过滤器（简化版）
type BloomFilter struct {
	bitset []bool
	size   int
	hashes int
	mutex  sync.RWMutex
}

// NewBloomFilter 创建布隆过滤器
func NewBloomFilter(size, hashes int) *BloomFilter {
	return &BloomFilter{
		bitset: make([]bool, size),
		size:   size,
		hashes: hashes,
	}
}

// Add 添加元素
func (bf *BloomFilter) Add(key string) {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	for i := 0; i < bf.hashes; i++ {
		hash := bf.hash(key, i)
		bf.bitset[hash%bf.size] = true
	}
}

// Contains 检查元素是否存在
func (bf *BloomFilter) Contains(key string) bool {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	for i := 0; i < bf.hashes; i++ {
		hash := bf.hash(key, i)
		if !bf.bitset[hash%bf.size] {
			return false
		}
	}
	return true
}

// hash 计算哈希值
func (bf *BloomFilter) hash(key string, seed int) int {
	h := md5.New()
	h.Write([]byte(key))
	h.Write([]byte{byte(seed)})
	hash := h.Sum(nil)

	result := 0
	for _, b := range hash {
		result = result*256 + int(b)
	}
	if result < 0 {
		result = -result
	}
	return result
}

// CacheMetrics 缓存指标
type CacheMetrics struct {
	Hits    int64
	Misses  int64
	Sets    int64
	Deletes int64
	Errors  int64
	mutex   sync.RWMutex
}

// NewCacheMetrics 创建缓存指标
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{}
}

// RecordHit 记录命中
func (cm *CacheMetrics) RecordHit() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Hits++
}

// RecordMiss 记录未命中
func (cm *CacheMetrics) RecordMiss() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Misses++
}

// RecordSet 记录设置
func (cm *CacheMetrics) RecordSet() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Sets++
}

// RecordDelete 记录删除
func (cm *CacheMetrics) RecordDelete() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Deletes++
}

// RecordError 记录错误
func (cm *CacheMetrics) RecordError() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Errors++
}

// GetHitRate 获取命中率
func (cm *CacheMetrics) GetHitRate() float64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	total := cm.Hits + cm.Misses
	if total == 0 {
		return 0
	}
	return float64(cm.Hits) / float64(total)
}

// GetStats 获取统计信息
func (cm *CacheMetrics) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return map[string]interface{}{
		"hits":     cm.Hits,
		"misses":   cm.Misses,
		"sets":     cm.Sets,
		"deletes":  cm.Deletes,
		"errors":   cm.Errors,
		"hit_rate": cm.GetHitRate(),
	}
}
