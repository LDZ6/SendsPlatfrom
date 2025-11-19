package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheConfig 缓存配置
type CacheConfig struct {
	ServiceName string
	ServiceKey  string
}

// InitCache 初始化缓存
func InitCache(config *CacheConfig) (*redis.Client, error) {
	rConfig := config.Conf.Redis[config.ServiceKey]

	rdb := redis.NewClient(&redis.Options{
		Addr:     rConfig.Address,
		Username: rConfig.UserName,
		Password: rConfig.Password,
		DB:       0,
		// 连接池配置
		PoolSize:     100,
		MinIdleConns: 10,
		MaxIdleConns: 50,
		// 连接超时
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		// 连接最大空闲时间
		ConnMaxIdleTime: 5 * time.Minute,
		// 连接最大存活时间
		ConnMaxLifetime: 30 * time.Minute,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	return rdb, nil
}
