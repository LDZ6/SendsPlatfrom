package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"platform/app/common/container"
)

func TestLocalCache(t *testing.T) {
	cache := container.NewLocalCache()

	// 测试设置和获取
	cache.Set("key1", "value1")
	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}

	// 测试不存在的key
	_, found = cache.Get("key2")
	if found {
		t.Error("Expected not to find key2")
	}

	// 测试删除
	cache.Delete("key1")
	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}

	// 测试清空
	cache.Set("key3", "value3")
	cache.Clear()
	_, found = cache.Get("key3")
	if found {
		t.Error("Expected cache to be cleared")
	}
}

func TestMultiLevelCache(t *testing.T) {
	// 这里需要模拟Redis客户端
	// 由于没有真实的Redis连接，我们只测试本地缓存部分
	localCache := container.NewLocalCache()

	// 测试本地缓存
	localCache.Set("key1", "value1")
	value, found := localCache.Get("key1")
	if !found {
		t.Error("Expected to find key1 in local cache")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}
}

func TestPenetrationProtection(t *testing.T) {
	pp := container.NewPenetrationProtection()

	// 测试初始状态
	if pp.IsBlocked("key1") {
		t.Error("Expected key1 not to be blocked initially")
	}

	// 测试阻塞key
	pp.BlockKey("key1")
	if !pp.IsBlocked("key1") {
		t.Error("Expected key1 to be blocked")
	}

	// 测试记录设置
	pp.RecordSet("key1")
	if pp.IsBlocked("key1") {
		t.Error("Expected key1 not to be blocked after record set")
	}
}

func TestAvalancheProtection(t *testing.T) {
	ap := container.NewAvalancheProtection()

	// 测试初始状态
	if ap.IsAvalanche() {
		t.Error("Expected no avalanche initially")
	}

	// 测试添加随机TTL
	baseTTL := 1 * time.Hour
	randomTTL := ap.AddRandomTTL(baseTTL)
	if randomTTL <= 0 {
		t.Error("Expected random TTL to be positive")
	}

	// 测试雪崩检测（需要模拟多次调用）
	for i := 0; i < 101; i++ {
		ap.IsAvalanche()
	}
	// 注意：由于时间窗口限制，这个测试可能不会触发雪崩
}

func TestBloomFilter(t *testing.T) {
	bf := container.NewBloomFilter(1000, 3)

	// 测试添加元素
	bf.Add("key1")
	bf.Add("key2")

	// 测试包含检查
	if !bf.Contains("key1") {
		t.Error("Expected bloom filter to contain key1")
	}
	if !bf.Contains("key2") {
		t.Error("Expected bloom filter to contain key2")
	}

	// 测试不存在的元素（可能有误报）
	// 注意：布隆过滤器可能有误报，但不会有漏报
}

func TestCacheMetrics(t *testing.T) {
	metrics := container.NewCacheMetrics()

	// 测试记录操作
	metrics.RecordHit()
	metrics.RecordHit()
	metrics.RecordMiss()
	metrics.RecordSet()
	metrics.RecordDelete()
	metrics.RecordError()

	// 测试统计信息
	stats := metrics.GetStats()
	if stats["hits"] != int64(2) {
		t.Errorf("Expected 2 hits, got %v", stats["hits"])
	}
	if stats["misses"] != int64(1) {
		t.Errorf("Expected 1 miss, got %v", stats["misses"])
	}
	if stats["sets"] != int64(1) {
		t.Errorf("Expected 1 set, got %v", stats["sets"])
	}
	if stats["deletes"] != int64(1) {
		t.Errorf("Expected 1 delete, got %v", stats["deletes"])
	}
	if stats["errors"] != int64(1) {
		t.Errorf("Expected 1 error, got %v", stats["errors"])
	}

	// 测试命中率
	hitRate := metrics.GetHitRate()
	expectedHitRate := 2.0 / 3.0 // 2 hits out of 3 total
	if hitRate != expectedHitRate {
		t.Errorf("Expected hit rate %f, got %f", expectedHitRate, hitRate)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := container.NewCircuitBreaker()

	// 测试正常调用
	err := cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试失败调用
	failCount := 0
	for i := 0; i < 6; i++ {
		err := cb.Call(func() error {
			failCount++
			return errors.New("test error")
		})
		if err != nil {
			// 熔断器开启后应该返回错误
			break
		}
	}

	// 验证失败次数
	if failCount < 5 {
		t.Errorf("Expected at least 5 failures, got %d", failCount)
	}
}

func TestCacheWithFallback(t *testing.T) {
	// 创建模拟缓存
	mockCache := &MockCache{
		data: make(map[string]interface{}),
	}

	// 创建降级函数
	fallback := func(key string) (interface{}, error) {
		return "fallback_" + key, nil
	}

	// 创建带降级的缓存
	cache := container.NewCacheWithFallback(mockCache, fallback)

	// 测试缓存未命中时的降级
	value, err := cache.Get(context.Background(), "key1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value != "fallback_key1" {
		t.Errorf("Expected 'fallback_key1', got %v", value)
	}

	// 测试缓存命中
	mockCache.Set("key2", "cached_value2")
	value, err = cache.Get(context.Background(), "key2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value != "cached_value2" {
		t.Errorf("Expected 'cached_value2', got %v", value)
	}
}

// MockCache 模拟缓存实现
type MockCache struct {
	data map[string]interface{}
}

func (mc *MockCache) Get(ctx context.Context, key string) (interface{}, error) {
	value, exists := mc.data[key]
	if !exists {
		return nil, errors.New("key not found")
	}
	return value, nil
}

func (mc *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	mc.data[key] = value
	return nil
}

func (mc *MockCache) Delete(ctx context.Context, key string) error {
	delete(mc.data, key)
	return nil
}

func (mc *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := mc.data[key]
	return exists, nil
}
