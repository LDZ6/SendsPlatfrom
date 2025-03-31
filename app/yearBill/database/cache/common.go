package cache

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"platform/app/yearBill/database/model"
	"platform/app/yearBill/types"
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

func (cache *RDBCache) GetValue(key string) (result any) {
	result, err := cache.Rdb.Get(key).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	// 返回缓存值
	return
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

func (cache *RDBCache) IncrValue(key string) (rank int64, err error) {
	rank, err = cache.Rdb.Incr(key).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	return
}

func (cache *RDBCache) GetECardCertificate(stunum string) (eCardCertificate types.ECardCertificate, err error) {
	date, err := cache.Rdb.HGet("eCardCertificate", stunum).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	err = json.Unmarshal([]byte(date), eCardCertificate)
	if err != nil {
		logrus.Info("[UnmarshalERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) GetJwcCertificate(stunum string) (jwcCertificate types.JwcCertificate, err error) {
	date, err := cache.Rdb.HGet("jwcCertificate", stunum).Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	err = json.Unmarshal([]byte(date), jwcCertificate)
	if err != nil {
		logrus.Info("[UnmarshalERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) SaveECardCertificate(eCardCertificate types.ECardCertificate, stunum string) (err error) {
	data, err := json.Marshal(eCardCertificate)
	if err != nil {
		logrus.Info("[MarshalERROR]:%v", err.Error())
		return
	}
	err = cache.Rdb.HSet("eCardCertificate", stunum, string(data)).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) SaveJwcCertificate(jwcCertificate types.JwcCertificate, stunum string) (err error) {
	data, err := json.Marshal(jwcCertificate)
	if err != nil {
		logrus.Info("[MarshalERROR]:%v", err.Error())
		return
	}
	err = cache.Rdb.HSet("jwcCertificate", stunum, string(data)).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) SaveLearnData(data model.LearnCache, stunum string) (err error) {
	dataJson, err := json.Marshal(data)
	if err != nil {
		logrus.Info("[MarshalERROR]:%v", err.Error())
		return
	}
	err = cache.Rdb.HSet("data:"+stunum, "learn_data", string(dataJson)).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) SavePayData(data model.BillCache, stunum string) (err error) {
	dataJson, err := json.Marshal(data)
	if err != nil {
		logrus.Info("[MarshalERROR]:%v", err.Error())
		return
	}
	err = cache.Rdb.HSet("data:"+stunum, "pay_data", string(dataJson)).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) GetPayData(stunum string) (data model.BillCache, err error) {
	dataStr, err := cache.Rdb.HGet("data:"+stunum, "pay_data").Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	err = json.Unmarshal([]byte(dataStr), &data)
	if err != nil {
		logrus.Info("[UnmarshalERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) GetLearnData(stunum string) (data model.LearnCache, err error) {
	dataStr, err := cache.Rdb.HGet("data:"+stunum, "learn_data").Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	err = json.Unmarshal([]byte(dataStr), &data)
	if err != nil {
		logrus.Info("[UnmarshalERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) SaveRank(stuNum string, data model.RankCache) (err error) {
	dataJson, err := json.Marshal(data)
	if err != nil {
		logrus.Info("[MarshalERROR]:%v", err.Error())
		return
	}
	err = cache.Rdb.HSet("data:"+stuNum, "user_data", string(dataJson)).Err()
	if err != nil {
		logrus.Info("[RDBERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) GetRank(stuNum string) (data model.RankCache) {
	dataStr, err := cache.Rdb.HGet("data:"+stuNum, "user_data").Result()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	err = json.Unmarshal([]byte(dataStr), &data)
	if err != nil {
		logrus.Info("[UnmarshalERROR]:%v", err.Error())
		return
	}
	return
}

func (cache *RDBCache) DelRank(stuNum string) {
	err := cache.Rdb.HDel("data:"+stuNum, "user_data").Err()
	if err != nil {
		if err != redis.Nil {
			logrus.Info("[RDBERROR]:%v", err.Error())
		}
		return
	}
	return
}
