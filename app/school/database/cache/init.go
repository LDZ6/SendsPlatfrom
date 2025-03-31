package cache

import (
	"context"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"platform/config"
)

var RedisClient *redis.Client

func InitRDB() {
	rConfig := config.Conf.Redis["school"]
	client := redis.NewClient(&redis.Options{
		Addr:     rConfig.Address,
		Password: rConfig.Password,
		DB:       0,
	})
	_, err := client.Ping().Result()
	if err != nil {
		logrus.Info(err)
		panic(err)
	}
	RedisClient = client
}

func NewRDBClient(ctx context.Context) *redis.Client {
	return RedisClient.WithContext(ctx)
}
