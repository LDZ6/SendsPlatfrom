package cache

import (
	"context"
	"errors"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"platform/app/school/types"
)

type RDBCache struct {
	Rdb *redis.Client
}

func NewRDBCache(ctx context.Context) *RDBCache {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RDBCache{Rdb: NewRDBClient(ctx)}
}

func (cache *RDBCache) GetKey(key string) (string, error) {
	result, err := cache.Rdb.Get(key).Result()
	if err == redis.Nil {
		//缓存项不存在进行处理
		return "", errors.New("key缓存不存在")
	} else if err != nil {
		// 其他错误处理
		return "", err
	}
	// 返回缓存值
	return result, nil
}

func (cache *RDBCache) GetJwcCertificate(stunum string) (jwcCertificate types.JwcCertificate) {
	maps, err := cache.Rdb.HGetAll("jwcCertificate:" + stunum).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v\n", err.Error())
		}
		return
	}
	jwcCertificate.GsSession = maps["gsSession"]
	jwcCertificate.Emaphome_WEU = maps["emaphome_WEU"]
	return
}
