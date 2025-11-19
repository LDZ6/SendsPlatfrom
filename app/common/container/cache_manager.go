package container

import (
	"context"
	"fmt"
	"sync"
	"time"

	"platform/config"

	"github.com/redis/go-redis/v9"
)

// CacheManager 缓存管理器
type CacheManager struct {
	clients map[string]*redis.Client
	mutex   sync.RWMutex
	config  *config.Config
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	return &CacheManager{
		clients: make(map[string]*redis.Client),
		config:  config.Conf,
	}
}

// GetClient 获取缓存客户端
func (cm *CacheManager) GetClient(name string) (*redis.Client, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if client, exists := cm.clients[name]; exists {
		return client, nil
	}
	return nil, fmt.Errorf("cache client %s not found", name)
}

// AddClient 添加缓存客户端
func (cm *CacheManager) AddClient(name string, client *redis.Client) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.clients[name] = client
}

// InitClient 初始化缓存客户端
func (cm *CacheManager) InitClient(name string) (*redis.Client, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 检查是否已存在
	if client, exists := cm.clients[name]; exists {
		return client, nil
	}

	// 使用缓存工厂创建客户端
	factory := NewCacheFactory()
	client, err := factory.InitCacheForService(name)
	if err != nil {
		return nil, err
	}

	cm.clients[name] = client
	return client, nil
}

// GetOrCreateClient 获取或创建缓存客户端
func (cm *CacheManager) GetOrCreateClient(name string) (*redis.Client, error) {
	// 先尝试获取现有客户端
	if client, err := cm.GetClient(name); err == nil {
		return client, nil
	}

	// 如果不存在，创建新客户端
	return cm.InitClient(name)
}

// Close 关闭所有缓存连接
func (cm *CacheManager) Close() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	var errors []error
	for name, client := range cm.clients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close cache %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing caches: %v", errors)
	}

	return nil
}

// HealthCheck 健康检查
func (cm *CacheManager) HealthCheck() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	health := make(map[string]interface{})
	ctx := context.Background()

	for name, client := range cm.clients {
		// 测试连接
		if err := client.Ping(ctx).Err(); err != nil {
			health[name] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			// 获取Redis信息
			info, err := client.Info(ctx).Result()
			if err != nil {
				health[name] = map[string]interface{}{
					"status": "healthy",
					"error":  "failed to get info: " + err.Error(),
				}
			} else {
				health[name] = map[string]interface{}{
					"status": "healthy",
					"info":   info,
				}
			}
		}
	}

	return health
}

// GetStats 获取缓存统计信息
func (cm *CacheManager) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := make(map[string]interface{})
	ctx := context.Background()

	for name, client := range cm.clients {
		// 获取Redis统计信息
		info, err := client.Info(ctx, "stats").Result()
		if err != nil {
			stats[name] = map[string]interface{}{
				"error": err.Error(),
			}
		} else {
			stats[name] = map[string]interface{}{
				"info": info,
			}
		}
	}

	return stats
}

// MultiLevelCache 多级缓存策略
type MultiLevelCache struct {
	localCache *LocalCache
	redisCache *redis.Client
	ttl        time.Duration
}

// LocalCache 本地缓存
type LocalCache struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// NewLocalCache 创建本地缓存
func NewLocalCache() *LocalCache {
	return &LocalCache{
		data: make(map[string]interface{}),
	}
}

// Get 获取值
func (lc *LocalCache) Get(key string) (interface{}, bool) {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()
	value, exists := lc.data[key]
	return value, exists
}

// Set 设置值
func (lc *LocalCache) Set(key string, value interface{}) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	lc.data[key] = value
}

// Delete 删除值
func (lc *LocalCache) Delete(key string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	delete(lc.data, key)
}

// Clear 清空缓存
func (lc *LocalCache) Clear() {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	lc.data = make(map[string]interface{})
}

// NewMultiLevelCache 创建多级缓存
func NewMultiLevelCache(redisClient *redis.Client, ttl time.Duration) *MultiLevelCache {
	return &MultiLevelCache{
		localCache: NewLocalCache(),
		redisCache: redisClient,
		ttl:        ttl,
	}
}

// Get 多级缓存获取
func (mlc *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, error) {
	// L1: 本地缓存
	if value, found := mlc.localCache.Get(key); found {
		return value, nil
	}

	// L2: Redis缓存
	if value, err := mlc.redisCache.Get(ctx, key).Result(); err == nil {
		// 回写到本地缓存
		mlc.localCache.Set(key, value)
		return value, nil
	}

	return nil, fmt.Errorf("key not found: %s", key)
}

// Set 多级缓存设置
func (mlc *MultiLevelCache) Set(ctx context.Context, key string, value interface{}) error {
	// 设置本地缓存
	mlc.localCache.Set(key, value)

	// 设置Redis缓存
	return mlc.redisCache.Set(ctx, key, value, mlc.ttl).Err()
}

// Delete 多级缓存删除
func (mlc *MultiLevelCache) Delete(ctx context.Context, key string) error {
	// 删除本地缓存
	mlc.localCache.Delete(key)

	// 删除Redis缓存
	return mlc.redisCache.Del(ctx, key).Err()
}

// CreateMultiLevelCache 创建多级缓存实例
func (cm *CacheManager) CreateMultiLevelCache(name string, ttl time.Duration) (*MultiLevelCache, error) {
	client, err := cm.GetOrCreateClient(name)
	if err != nil {
		return nil, err
	}
	return NewMultiLevelCache(client, ttl), nil
}
