package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"platform/app/yearBill/database/cache"
	"platform/app/yearBill/database/dao"
	"platform/app/yearBill/database/model"
	"platform/app/yearBill/database/mq"
	"platform/app/yearBill/script"
	"platform/config"
	YearBillPb "platform/idl/pb/yearBill"
	"platform/utils/school"
	"sync"
	"time"
)

// 判断是否有数据，同时统计参与人数
func (*YearBillSrv) InfoCheck(ctx context.Context, req *YearBillPb.InfoCheckRequest) (resp *YearBillPb.InfoCheckResponse, err error) {
	resp = new(YearBillPb.InfoCheckResponse)
	resp.Flag = false
	//查询缓存
	cacheRDB := cache.NewRDBCache(ctx)
	data := cacheRDB.GetRank(req.StuNum)
	if data.ID != 0 {
		resp.Flag = true
		return
	}
	if err != nil && err != redis.Nil {
		return
	}
	return
}

// 获得电子校园卡和教务处凭证
func (*YearBillSrv) GetCertificate(ctx context.Context, req *YearBillPb.GetCertificateRequest) (resp *YearBillPb.GetCertificateResponse, err error) {
	resp = new(YearBillPb.GetCertificateResponse)
	//先查看缓存中有没有凭证（写不下去了）

	wg := sync.WaitGroup{}
	wg.Add(2)
	var mutex sync.Mutex
	var errECard error
	var errJwc error
	go func() {
		defer wg.Done()
		info, errECard := school.ECardLogin(req.Openid)
		if errECard != nil {
			return
		}
		mutex.Lock()
		resp.HallTicket = info.HallTicket
		resp.JsSessionId = info.JsSessionId
		mutex.Unlock()
	}()
	go func() {
		defer wg.Done()
		gssession, errJwc := school.GetGsSession(req.Openid)
		jwc := school.NewJwc()
		jwc.GsSession = gssession
		emaphome_WEU, errJwc := jwc.GetEmaphome_WEU()
		if errJwc != nil {
			return
		}
		mutex.Lock()
		resp.Emaphome_WEU = emaphome_WEU
		resp.GsSession = gssession
		mutex.Unlock()
	}()
	wg.Wait()
	if errJwc != nil {
		err = errJwc
		return
	}
	if errECard != nil {
		err = errECard
		return
	}
	if resp.Emaphome_WEU == "" || resp.HallTicket == "" {
		err = errors.New("获取凭证失败，您未成功绑定桑梓微")
		return
	}
	//没命中校验数据库
	cacheRDB := cache.NewRDBCache(ctx)
	userDao := dao.NewUserDao(ctx)
	users, err := userDao.FindUser(req.StuNum)
	if err != nil {
		return
	}
	//如果数据没被初始化过，开始初始化
	if len(users) == 0 {
		err = userDao.CreateUser(req.StuNum)
		if err != nil {
			return
		}
		mqChan := mq.NewMQConn()
		err = mq.Producer(mqChan, config.YearBillQueue, config.YearBillExchange, &model.DadaInitTask{
			JsSessionId:  resp.JsSessionId,
			HallTicket:   resp.HallTicket,
			GsSession:    resp.GsSession,
			Emaphome_WEU: resp.Emaphome_WEU,
			StuNum:       req.StuNum,
		})
		if err != nil {
			logrus.Info(err.Error())
		}
		return
	} else if users[0].Init == 1 {
		err = cacheRDB.SaveRank(req.StuNum, model.RankCache{
			ID:        uint(users[0].Rank),
			Appraisal: users[0].Appraisal,
		})
		if err != nil {
			return resp, err
		}
		return resp, err
	} else {
		mqChan := mq.NewMQConn()
		err = mq.Producer(mqChan, config.YearBillQueue, config.YearBillExchange, &model.DadaInitTask{
			JsSessionId:  resp.JsSessionId,
			HallTicket:   resp.HallTicket,
			GsSession:    resp.GsSession,
			Emaphome_WEU: resp.Emaphome_WEU,
			StuNum:       req.StuNum,
		})
		if err != nil {
			logrus.Info(err.Error())
		}
		return
	}
	return
}

// 数据初始化（从mq中读入进行削峰）
func (*YearBillSrv) DataInit(ctx context.Context, req *emptypb.Empty) (empty *emptypb.Empty, err error) {
	empty = new(emptypb.Empty)
	ch := mq.NewMQConn()
	msgs, err := mq.Consumer(ch, config.YearBillQueue)
	if err != nil {
		return nil, err
	}
	for d := range msgs {
		time.Sleep(100 * time.Millisecond)
		ctxTask, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		go func() {
			d.Ack(false)
			RDBCache := cache.NewRDBCache(ctx)
			var task model.DadaInitTask
			errTask := json.Unmarshal(d.Body, &task)
			if errTask != nil {
				logrus.Info(errTask.Error())
				return
			}
			data := RDBCache.GetRank(task.StuNum)
			if data.ID != 0 {
				return
			}
			time.Sleep(500 * time.Millisecond)
			wgTask := sync.WaitGroup{}
			wgTask.Add(2)
			var resp_pay *YearBillPb.PayDataInitResponse
			var resp_learn *YearBillPb.LearnDataInitResponse
			// 创建两个通道
			payCh := make(chan *YearBillPb.PayDataInitResponse, 1)
			learnCh := make(chan *YearBillPb.LearnDataInitResponse, 1)
			go func() {
				defer wgTask.Done()
				resp_pay, _ = script.PayDataInit(ctx, &YearBillPb.PayDataInitRequest{
					StuNum:     task.StuNum,
					HallTicket: task.HallTicket,
				})
				payCh <- resp_pay
			}()
			go func() {
				defer wgTask.Done()
				resp_learn, _ = script.LearnDataInit(ctx, &YearBillPb.LearnDataInitRequest{
					StuNum:       task.StuNum,
					GsSession:    task.GsSession,
					Emaphome_WEU: task.Emaphome_WEU,
				})
				learnCh <- resp_learn
			}()
			wgTask.Wait()
			close(payCh)
			close(learnCh)

			resp_pay = <-payCh
			resp_learn = <-learnCh

			userDao := dao.NewUserDao(ctx)
			index, errTask := RDBCache.IncrValue("count")
			if errTask != nil {
				return
			}
			errTask = RDBCache.SaveRank(task.StuNum, model.RankCache{
				ID:        uint(index),
				Appraisal: 0,
			})
			if errTask != nil {
				return
			}
			errTask = userDao.StorageData(task.StuNum, index, model.Bill{
				StuNum:            task.StuNum,
				BestRestaurant:    resp_pay.FavoriteRestaurant,
				BestRestaurantPay: resp_pay.FavoriteRestaurantPay,
				EarlyTime:         resp_pay.EarlyTime.AsTime(),
				LastTime:          resp_pay.LastTime.AsTime(),
				OtherPay:          resp_pay.OtherPay,
				LibraryPay:        resp_pay.LibraryPay,
				RestaurantPay:     resp_pay.RestaurantPay,
			}, model.Learn{
				StuNum:     task.StuNum,
				MostCourse: resp_learn.MostCourse,
				Eight:      resp_learn.Eight,
				Most:       resp_learn.Most,
				Ten:        resp_learn.Ten,
				SumLesson:  resp_learn.SumLesson,
			})
			if errTask != nil {
				return
			}

			errTask = RDBCache.SavePayData(model.BillCache{
				BestRestaurant:    resp_pay.FavoriteRestaurant,
				BestRestaurantPay: resp_pay.FavoriteRestaurantPay,
				EarlyTime:         resp_pay.EarlyTime.AsTime(),
				LastTime:          resp_pay.LastTime.AsTime(),
				OtherPay:          resp_pay.OtherPay,
				RestaurantPay:     resp_pay.RestaurantPay,
				LibraryPay:        resp_pay.LibraryPay,
			}, task.StuNum)
			if errTask != nil {
				return
			}
			errTask = RDBCache.SaveLearnData(model.LearnCache{
				MostCourse: resp_learn.MostCourse,
				Eight:      resp_learn.Eight,
				Ten:        resp_learn.Ten,
				Most:       resp_learn.Most,
				SumLesson:  resp_learn.SumLesson,
			}, task.StuNum)
		}()
		select {
		case <-ctxTask.Done():
			logrus.Info("获取数据失败")
			cancel()
			break
		}
		cancel()
		//释放资源
	}
	ch.Close()
	return

}

func (*YearBillSrv) CheckState(ctx context.Context, req *YearBillPb.CheckStateRequest) (resp *YearBillPb.CheckStateResponse, err error) {
	resp = new(YearBillPb.CheckStateResponse)
	cacheRDB := cache.NewRDBCache(ctx)
	data := cacheRDB.GetRank(req.StuNum)
	if data.ID != 0 {
		resp.State = true
		return
	}
	return
}
