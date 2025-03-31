package cache

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"platform/app/user/types"
	"time"
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
func (cache *RDBCache) GetValue(key string) (string, error) {
	result, err := cache.Rdb.Get(key).Result()
	if err == redis.Nil {
		//缓存项不存在进行处理
		return "", err
	} else if err != nil {
		// 其他错误处理
		return "", err
	}
	// 返回缓存值
	return result, nil
}

func (cache *RDBCache) SaveValue(key string, value any, time time.Duration) {
	err := cache.Rdb.Set(key, value, time).Err()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	// 返回缓存值
	return
}

func (cache *RDBCache) SaveGeSession(openid string, geSession string) (err error) {
	err = cache.Rdb.Set("hqu"+openid, geSession, 10*time.Minute).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (cache *RDBCache) GetECardCertificate(stunum string) (eCardCertificate types.ECardCertificate, err error) {
	date, err := cache.Rdb.HGet("eCardCertificate", stunum).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v\n", err.Error())
		}
		return
	}
	err = json.Unmarshal([]byte(date), eCardCertificate)
	if err != nil {
		logrus.Info("[UnmarshalERROR]:%v\n", err.Error())
		return
	}
	return
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

func (cache *RDBCache) SaveECardCertificate(eCardCertificate types.ECardCertificate, stunum string) (err error) {
	data, err := json.Marshal(eCardCertificate)
	if err != nil {
		logrus.Info("[MarshalERROR]:%v\n", err.Error())
		return
	}
	err = cache.Rdb.HSet("eCardCertificate", stunum, string(data)).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (cache *RDBCache) SaveJwcCertificate(jwcCertificate types.JwcCertificate, stunum string) (err error) {
	maps := make(map[string]any)
	maps["gsSession"] = jwcCertificate.GsSession
	maps["emaphome_WEU"] = jwcCertificate.Emaphome_WEU
	err = cache.Rdb.HMSet("jwcCertificate:"+stunum, maps).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	err = cache.Rdb.Expire("jwcCertificate:"+stunum, 8*time.Minute).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v\n", err.Error())
		return
	}
	return
}
