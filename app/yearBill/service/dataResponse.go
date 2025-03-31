package service

import (
	"context"
	"errors"
	"github.com/go-redis/redis"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"platform/app/yearBill/database/cache"
	"platform/app/yearBill/database/dao"
	"platform/app/yearBill/database/model"
	YearBillPb "platform/idl/pb/yearBill"
	"strconv"
)

func (*YearBillSrv) GetPayData(ctx context.Context, req *YearBillPb.GetPayDataRequest) (resp *YearBillPb.GetPayDataResponse, err error) {
	resp = new(YearBillPb.GetPayDataResponse)
	cacheRDB := cache.NewRDBCache(ctx)
	data, err := cacheRDB.GetPayData(req.StuNum)
	if err == nil {
		resp = &YearBillPb.GetPayDataResponse{
			FavoriteRestaurant:    data.BestRestaurant,
			EarlyTime:             timestamppb.New(data.EarlyTime),
			LastTime:              timestamppb.New(data.LastTime),
			FavoriteRestaurantPay: data.BestRestaurantPay,
			OtherPay:              data.OtherPay,
			RestaurantPay:         data.RestaurantPay,
			LibraryPay:            data.LibraryPay,
		}
		return
	}
	if err != nil && err != redis.Nil {
		return
	}
	err = nil
	billDao := dao.NewBillDao(ctx)
	bills, err := billDao.GetBill(req.StuNum)
	if err != nil {
		return
	}
	if len(bills) == 0 {
		errors.New("花费数据未初始化")
		return
	}
	bill := bills[0]
	resp = &YearBillPb.GetPayDataResponse{
		FavoriteRestaurant:    bill.BestRestaurant,
		EarlyTime:             timestamppb.New(bill.EarlyTime),
		LastTime:              timestamppb.New(bill.LastTime),
		FavoriteRestaurantPay: bill.BestRestaurantPay,
		OtherPay:              bill.OtherPay,
		RestaurantPay:         bill.RestaurantPay,
		LibraryPay:            bill.LibraryPay,
	}
	err = cacheRDB.SavePayData(model.BillCache{
		BestRestaurant:    bill.BestRestaurant,
		BestRestaurantPay: bill.BestRestaurantPay,
		EarlyTime:         bill.EarlyTime,
		LastTime:          bill.LastTime,
		OtherPay:          bill.OtherPay,
		RestaurantPay:     bill.RestaurantPay,
		LibraryPay:        bill.LibraryPay,
	}, req.StuNum)
	if err != nil {
		return
	}
	return
}

func (*YearBillSrv) GetLearnData(ctx context.Context, req *YearBillPb.GetLearnDataRequest) (resp *YearBillPb.GetLearnDataResponse, err error) {
	resp = new(YearBillPb.GetLearnDataResponse)
	cacheRDB := cache.NewRDBCache(ctx)
	data, err := cacheRDB.GetLearnData(req.StuNum)
	if err == nil {
		resp = &YearBillPb.GetLearnDataResponse{
			MostCourse: data.MostCourse,
			Eight:      data.Eight,
			Ten:        data.Ten,
			SumLesson:  data.SumLesson,
			Most:       data.Most,
		}
		return
	}
	if err != nil && err != redis.Nil {
		return
	}
	err = nil
	learnDao := dao.NewLearnDao(ctx)
	learns, err := learnDao.GetLearn(req.StuNum)
	if err != nil {
		return
	}
	if len(learns) == 0 {
		errors.New("学习数据未初始化")
		return
	}
	learn := learns[0]
	resp = &YearBillPb.GetLearnDataResponse{
		MostCourse: learn.MostCourse,
		Eight:      learn.Eight,
		Ten:        learn.Ten,
		Most:       learn.Most,
		SumLesson:  learn.SumLesson,
	}
	err = cacheRDB.SaveLearnData(model.LearnCache{
		MostCourse: learn.MostCourse,
		Eight:      learn.Eight,
		Ten:        learn.Ten,
		Most:       learn.Most,
		SumLesson:  learn.SumLesson,
	}, req.StuNum)
	return
}

func (*YearBillSrv) GetRank(ctx context.Context, req *YearBillPb.GetRankRequest) (resp *YearBillPb.GetRankResponse, err error) {
	resp = new(YearBillPb.GetRankResponse)
	//查询缓存
	cacheRDB := cache.NewRDBCache(ctx)
	userDao := dao.NewUserDao(ctx)
	data := cacheRDB.GetRank(req.StuNum)
	if data.ID != 0 {
		resp.Index = int64(data.ID)
		resp.Appraisal = data.Appraisal
	} else {
		user, err := userDao.FindUser(req.StuNum)
		if err != nil {
			return nil, err
		}
		if len(user) != 0 {
			resp.Index = user[0].Rank
			resp.Appraisal = user[0].Appraisal
			err = cacheRDB.SaveRank(req.StuNum, model.RankCache{
				ID:        uint(user[0].Rank),
				Appraisal: user[0].Appraisal,
			})
		}
	}
	countAny := cacheRDB.GetValue("count")
	if countAny != nil {
		countStr := countAny.(string)
		count, _ := strconv.Atoi(countStr)
		if count != 0 {
			resp.Count = int64(count)
			return resp, nil
		}
	}
	return
}

func (*YearBillSrv) Appraise(ctx context.Context, req *YearBillPb.AppraiseRequest) (resp *emptypb.Empty, err error) {
	resp = new(emptypb.Empty)
	userDao := dao.NewUserDao(ctx)
	err = userDao.UpdateAppraisal(req.StuNum, req.Appraisal)
	if err != nil {
		return
	}
	cacheRDB := cache.NewRDBCache(ctx)
	cacheRDB.DelRank(req.StuNum)
	return
}
