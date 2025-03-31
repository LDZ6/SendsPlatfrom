package script

import (
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
	"platform/app/yearBill/database/dao"
	"platform/app/yearBill/types"
	YearBillPb "platform/idl/pb/yearBill"
	"platform/utils/school"
	schoolUtils "platform/utils/school"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 花费数据初始化获取脚本
func PayDataInit(ctx context.Context, req *YearBillPb.PayDataInitRequest) (resp *YearBillPb.PayDataInitResponse, err error) {
	resp = new(YearBillPb.PayDataInitResponse)

	billDao := dao.NewBillDao(ctx)
	cnt, err := billDao.ExistBillByStuNum(req.StuNum)
	if err != nil {
		return
	}
	if cnt != 0 {
		resp.Flag = 1
		return
	}
	eCard := school.NewECard()
	eCard.HallTicket = req.HallTicket
	cardInfo, err := eCard.GetCardInfo()
	if err != nil {
		return
	}
	stuNum := cardInfo["query_card"].(map[string]any)["card"].([]any)[0].(map[string]any)["account"].(string)
	payInfo, err := eCard.GetPay(stuNum, "1")
	if err != nil {
		return
	}

	//确定任务数量，准备ddos
	task := int(payInfo["total"].(float64)/15) + 1
	payData := types.PayData{
		Maps:           make(map[string]float64),
		BestRestaurant: "",
		RestaurantPay:  0,
		EarlyTime:      time.Time{},
		EarlyCount:     math.MaxInt,
		LastTime:       time.Time{},
		LastCount:      0,
		OtherPay:       0,
		LibraryPay:     0,
		Mutex:          sync.Mutex{},
	}
	shanghaiZone, _ := time.LoadLocation("Asia/Shanghai")
	wg := sync.WaitGroup{}
	wg.Add(task)
	for i := 1; i <= task; i++ {
		go func(i int) {
			defer wg.Done()
			payInfo, err := eCard.GetPay(stuNum, strconv.Itoa(i))
			if err != nil {
				return
			}
			body := payInfo["rows"].([]any)
			wgPage := sync.WaitGroup{}
			taskPage := len(body)
			wgPage.Add(taskPage)
			for _, pay := range body {
				go func(pay any) {
					defer wgPage.Done()
					payTimeStr, ok := pay.(map[string]any)["OCCTIME"].(string)
					if !ok {
						return
					}
					payTime, _ := time.ParseInLocation(time.DateTime, payTimeStr, shanghaiZone)
					count := payTime.Hour()*3600 + payTime.Minute()*60 + payTime.Second()
					place, ok := pay.(map[string]any)["MERCNAME"].(string)
					if !ok {
						place = "未知"
					}
					valueFloat, ok := pay.(map[string]any)["TRANAMT"].(float64)
					if !ok {
						return
					}
					value := decimal.NewFromFloat(valueFloat)
					if valueFlot64, _ := value.Float64(); valueFlot64 >= 0 {
						return
					}
					payData.Mutex.Lock()
					if v, ok := payData.Maps[place]; ok {
						payData.Maps[place], _ = decimal.NewFromFloat(v).Sub(value).Float64()
					} else {
						payData.Maps[place], _ = decimal.NewFromFloat(v).Sub(value).Float64()
					}
					if strings.Contains(place, "餐厅") {
						payData.RestaurantPay, _ = decimal.NewFromFloat(payData.RestaurantPay).Sub(value).Float64()
						if payData.Maps[place] > payData.Maps[payData.BestRestaurant] {
							payData.BestRestaurant = place
						}
						if count > payData.LastCount {
							payData.LastCount = count
							payData.LastTime = payTime
						}
						if count < payData.EarlyCount {
							payData.EarlyCount = count
							payData.EarlyTime = payTime
						}
					} else if strings.Contains(place, "图书馆") {
						payData.LibraryPay, _ = decimal.NewFromFloat(payData.LibraryPay).Sub(value).Float64()
					} else {
						payData.OtherPay, _ = decimal.NewFromFloat(payData.OtherPay).Sub(value).Float64()
					}
					payData.Mutex.Unlock()
				}(pay)
			}
			wgPage.Wait()
		}(i)
	}
	wg.Wait()
	resp = &YearBillPb.PayDataInitResponse{
		Flag:                  1,
		FavoriteRestaurant:    strings.TrimSpace(payData.BestRestaurant),
		FavoriteRestaurantPay: payData.Maps[payData.BestRestaurant],
		EarlyTime:             timestamppb.New(payData.EarlyTime),
		LastTime:              timestamppb.New(payData.LastTime),
		OtherPay:              payData.OtherPay,
		RestaurantPay:         payData.RestaurantPay,
		LibraryPay:            payData.LibraryPay,
	}
	return
}

// 有点问题
func BookDataInit(ctx context.Context, req *YearBillPb.BookDataInitRequest) (resp *YearBillPb.BookDataInitResponse, err error) {
	resp = new(YearBillPb.BookDataInitResponse)
	bookDao := dao.NewBookDao(ctx)
	cnt, err := bookDao.ExistBookByStuNum(req.StuNum)
	if err != nil {
		return
	}
	if cnt != 0 {
		resp.Flag = 1
		return
	}
	eCard := school.NewECard()
	eCard.JsSessionId = req.JsSessionid
	//bookData := types.BookData{
	//	Read:     0,
	//	Reading:  0,
	//	BookName: "",
	//	Longest:  time.Time{},
	//}
	bookInfo, err := eCard.GetBook("1")
	fmt.Println(bookInfo)
	return
}

// 教务处数据获取脚本
func LearnDataInit(ctx context.Context, req *YearBillPb.LearnDataInitRequest) (resp *YearBillPb.LearnDataInitResponse, err error) {
	resp = new(YearBillPb.LearnDataInitResponse)
	learnDao := dao.NewLearnDao(ctx)
	cnt, err := learnDao.ExistLearnByStuNum(req.StuNum)
	if err != nil {
		return
	}
	if cnt != 0 {
		resp.Flag = 1
		return
	}
	jwc := schoolUtils.NewJwc()
	jwc.Emaphome_WEU = req.Emaphome_WEU
	jwc.GsSession = req.GsSession
	wg := sync.WaitGroup{}
	learnData := types.LearnData{
		LearnSum:   map[string]int{},
		MostCourse: "",
		Eight:      0,
		Ten:        0,
		Sum:        0,
		Mutex:      sync.Mutex{},
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			semester := ""
			if i == 0 {
				semester = "2023-2024-1"
			} else {
				semester = "2022-2023-2"
			}
			scheduleInfo, err := jwc.GetSchedule(semester)
			if err != nil {
				return
			}
			var wgList sync.WaitGroup
			for _, v := range scheduleInfo.([]any) {
				wgList.Add(1)
				go func(v any) {
					defer wgList.Done()
					var teachWeek string
					var beginTime string
					var endTime string
					var courseName string
					if _, ok := v.(map[string]any)["ZCMC"].(string); !ok {
						teachWeek = ""
					} else {
						teachWeek = v.(map[string]any)["ZCMC"].(string)
					}
					if _, ok := v.(map[string]any)["KSJC_DISPLAY"].(string); !ok {
						beginTime = ""
					} else {
						beginTime = v.(map[string]any)["KSJC_DISPLAY"].(string)
					}
					if _, ok := v.(map[string]any)["KCM"].(string); !ok {
						courseName = ""
					} else {
						courseName = v.(map[string]any)["KCM"].(string)
					}
					if _, ok := v.(map[string]any)["JSJC_DISPLAY"].(string); !ok {
						endTime = ""
					} else {
						endTime = v.(map[string]any)["JSJC_DISPLAY"].(string)
					}

					startWeek := 0
					endWeek := 0
					var errChange error
					for j := 0; j < len(teachWeek); j++ {
						if teachWeek[j:j+1] == "-" {
							startWeek, errChange = strconv.Atoi(teachWeek[:j])
							if errChange != nil {
								break
							}
							if teachWeek[j+2:j+3] < "0" || teachWeek[j+2:j+3] > "9" {
								endWeek, errChange = strconv.Atoi(teachWeek[j+1 : j+2])
							} else {
								endWeek, errChange = strconv.Atoi(teachWeek[j+1 : j+3])
							}
							break
						}
					}
					// 检查是否有转换错误
					if errChange != nil {
						logrus.Info("转换错误:", errChange)
						return
					}
					begin, _ := strconv.Atoi(beginTime[3 : len(beginTime)-3])
					end, _ := strconv.Atoi(endTime[3 : len(endTime)-3])
					sum := (end - begin + 1) * (endWeek - startWeek + 1)
					learnData.Mutex.Lock()
					learnData.LearnSum[courseName] = sum
					if sum > learnData.LearnSum[learnData.MostCourse] {
						learnData.MostCourse = courseName
						learnData.Most = sum
					}
					learnData.Sum += sum
					if begin == 1 {
						learnData.Eight += endWeek - startWeek + 1
					}
					if end == 13 {
						learnData.Ten += endWeek - startWeek + 1
					}
					learnData.Mutex.Unlock()
				}(v)
			}
			wgList.Wait()
		}(i)
	}
	wg.Wait()
	resp = &YearBillPb.LearnDataInitResponse{
		Flag:       1,
		MostCourse: learnData.MostCourse,
		Eight:      int64(learnData.Eight),
		Ten:        int64(learnData.Ten),
		Most:       int64(learnData.Most),
		SumLesson:  int64(learnData.Sum),
	}
	return
}
