package container

import (
	"context"
	"fmt"
	"time"

	"platform/config"
	"platform/utils"

	"github.com/redis/go-redis/v9"
)

// CacheFactory 缓存工厂
type CacheFactory struct {
	config *config.Config
}

// NewCacheFactory 创建缓存工厂
func NewCacheFactory() *CacheFactory {
	return &CacheFactory{
		config: config.Conf,
	}
}

// InitCacheForService 为服务初始化缓存
func (cf *CacheFactory) InitCacheForService(serviceName string) (*redis.Client, error) {
	// 获取Redis配置
	redisConfig, exists := cf.config.Redis[serviceName]
	if !exists {
		return nil, utils.WrapError(fmt.Errorf("redis config for %s not found", serviceName),
			utils.ErrCodeConfigError, "缓存配置未找到")
	}

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Address,
		Password: redisConfig.Password,
		Username: redisConfig.UserName,
		DB:       0, // 默认数据库

		// 优化的连接池配置
		PoolSize:           200,              // 增加连接池大小
		MinIdleConns:       20,               // 增加最小空闲连接
		MaxIdleConns:       100,              // 增加最大空闲连接
		MaxConnAge:         30 * time.Minute, // 减少连接最大存活时间
		PoolTimeout:        30 * time.Second, // 连接池超时
		IdleTimeout:        5 * time.Minute,  // 空闲连接超时
		IdleCheckFrequency: 1 * time.Minute,  // 空闲连接检查频率

		// 重试配置
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		// 超时配置
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, utils.WrapRedisError(err, fmt.Sprintf("连接Redis服务 %s", serviceName))
	}

	return client, nil
}

// InitCacheWithOptions 使用自定义选项初始化缓存
func (cf *CacheFactory) InitCacheWithOptions(serviceName string, options *redis.Options) (*redis.Client, error) {
	// 获取Redis配置
	redisConfig, exists := cf.config.Redis[serviceName]
	if !exists {
		return nil, utils.WrapError(fmt.Errorf("redis config for %s not found", serviceName),
			utils.ErrCodeConfigError, "缓存配置未找到")
	}

	// 设置默认值
	if options.Addr == "" {
		options.Addr = redisConfig.Address
	}
	if options.Password == "" {
		options.Password = redisConfig.Password
	}
	if options.Username == "" {
		options.Username = redisConfig.UserName
	}

	// 创建Redis客户端
	client := redis.NewClient(options)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, utils.WrapRedisError(err, fmt.Sprintf("连接Redis服务 %s", serviceName))
	}

	return client, nil
}

// GetCacheManager 获取缓存管理器
func (cf *CacheFactory) GetCacheManager() *CacheManager {
	return NewCacheManager()
}

// InitAllServiceCaches 初始化所有服务的缓存
func (cf *CacheFactory) InitAllServiceCaches() (map[string]*redis.Client, error) {
	clients := make(map[string]*redis.Client)

	for serviceName := range cf.config.Redis {
		client, err := cf.InitCacheForService(serviceName)
		if err != nil {
			// 关闭已创建的连接
			for _, c := range clients {
				c.Close()
			}
			return nil, utils.WrapError(err, utils.ErrCodeRedis, "初始化所有服务缓存失败")
		}
		clients[serviceName] = client
	}

	return clients, nil
}

// CacheConfig 缓存配置
type CacheConfig struct {
	ServiceName  string
	DB           int
	MaxRetries   int
	PoolSize     int
	MinIdleConns int
	MaxIdleConns int
	MaxConnAge   time.Duration
	PoolTimeout  time.Duration
	IdleTimeout  time.Duration
}

// InitCacheWithConfig 使用配置初始化缓存
func (cf *CacheFactory) InitCacheWithConfig(config CacheConfig) (*redis.Client, error) {
	// 获取Redis配置
	redisConfig, exists := cf.config.Redis[config.ServiceName]
	if !exists {
		return nil, utils.WrapError(fmt.Errorf("redis config for %s not found", config.ServiceName),
			utils.ErrCodeConfigError, "缓存配置未找到")
	}

	// 设置默认值
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.PoolSize == 0 {
		config.PoolSize = 200
	}
	if config.MinIdleConns == 0 {
		config.MinIdleConns = 20
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.MaxConnAge == 0 {
		config.MaxConnAge = 30 * time.Minute
	}
	if config.PoolTimeout == 0 {
		config.PoolTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 5 * time.Minute
	}

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Address,
		Password: redisConfig.Password,
		Username: redisConfig.UserName,
		DB:       config.DB,

		// 连接池配置
		PoolSize:           config.PoolSize,
		MinIdleConns:       config.MinIdleConns,
		MaxIdleConns:       config.MaxIdleConns,
		MaxConnAge:         config.MaxConnAge,
		PoolTimeout:        config.PoolTimeout,
		IdleTimeout:        config.IdleTimeout,
		IdleCheckFrequency: 1 * time.Minute,

		// 重试配置
		MaxRetries:      config.MaxRetries,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		// 超时配置
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, utils.WrapRedisError(err, fmt.Sprintf("连接Redis服务 %s", config.ServiceName))
	}

	return client, nil
}
