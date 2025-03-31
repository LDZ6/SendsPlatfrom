package service

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"platform/app/school/database/cache"
	SchoolPb "platform/idl/pb/school"
	schoolUtils "platform/utils/school"
	"strings"
	"sync"
)

type SchoolSrv struct {
	SchoolPb.UnimplementedSchoolServiceServer
}

var SchoolSrvIns *SchoolSrv

var SchoolSrvOnce sync.Once

func GetSchoolSrv() *SchoolSrv {
	SchoolSrvOnce.Do(func() {
		SchoolSrvIns = &SchoolSrv{}
	})
	return SchoolSrvIns
}

func (*SchoolSrv) SchoolPing(ctx context.Context, empty *emptypb.Empty) (resp *SchoolPb.SchoolPingResponse, err error) {
	resp = new(SchoolPb.SchoolPingResponse)
	resp.Message = "School微服务ping通"
	return
}

func (*SchoolSrv) SchoolGpa(ctx context.Context, empty *emptypb.Empty) (resp *SchoolPb.SchoolGpaResponse, err error) {
	resp = new(SchoolPb.SchoolGpaResponse)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	stuNum := strings.Join(md.Get("stu_num"), "")
	cacheRDB := cache.NewRDBCache(ctx)
	jwcCertificate := cacheRDB.GetJwcCertificate(stuNum)
	if jwcCertificate.Emaphome_WEU == "" {
		return nil, errors.New("凭证已过期,请重新进入该界面")
	}
	jwc := schoolUtils.NewJwc()
	jwc.GsSession = jwcCertificate.GsSession
	jwc.Emaphome_WEU = jwcCertificate.Emaphome_WEU
	gpaInfo, err := jwc.GetGpa()
	if err != nil {
		return
	}
	length := len(gpaInfo)
	for i := 0; i < length; i++ {
		var semester string
		var gpa string
		var classRank string
		var majorRank string
		if v, ok := gpaInfo[i].(map[string]any)["XN"].(string); ok {
			semester = v
		} else {
			if v, ok = gpaInfo[i].(map[string]any)["XNXQDM"].(string); ok {
				semester = v
			} else {
				semester = "总计"
			}
		}
		if v, ok := gpaInfo[i].(map[string]any)["GPA"].(float64); ok {
			gpa = fmt.Sprintf("%f", v)
		} else {
			gpa = ""
		}
		if v, ok := gpaInfo[i].(map[string]any)["ZYPM"].(string); ok {
			majorRank = v
		} else {
			majorRank = ""
		}
		if v, ok := gpaInfo[i].(map[string]any)["BJPM"].(string); ok {
			classRank = v
		} else {
			classRank = ""
		}
		resp.Gpa = append(resp.Gpa, &SchoolPb.Gpa{
			Semester:  semester,
			Gpa:       gpa,
			ClassRank: classRank,
			MajorRank: majorRank,
		})
	}
	return
}

func (*SchoolSrv) SchoolGrade(ctx context.Context, empty *emptypb.Empty) (resp *SchoolPb.SchoolGradeResponse, err error) {
	resp = new(SchoolPb.SchoolGradeResponse)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	stuNum := strings.Join(md.Get("stu_num"), "")
	cacheRDB := cache.NewRDBCache(ctx)
	jwcCertificate := cacheRDB.GetJwcCertificate(stuNum)
	if jwcCertificate.Emaphome_WEU == "" {
		return nil, errors.New("凭证已过期,请重新进入该界面")
	}
	jwc := schoolUtils.NewJwc()
	jwc.GsSession = jwcCertificate.GsSession
	jwc.Emaphome_WEU = jwcCertificate.Emaphome_WEU
	gradeResponse, err := jwc.GetGrade()
	if err != nil {
		return
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, v := range gradeResponse.([]any) {
		wg.Add(1)
		go func(v any) {
			defer wg.Done()
			var zcj float64
			var pscj string
			var qmcj string
			var xufen float64
			var xszcjmc string
			if _, ok := v.(map[string]any)["XF"].(float64); !ok {
				xufen = 0
			} else {
				xufen = v.(map[string]any)["XF"].(float64)
			}
			if _, ok := v.(map[string]any)["ZCJ"].(float64); !ok {
				zcj = 0
			} else {
				zcj = v.(map[string]any)["ZCJ"].(float64)
			}
			if _, ok := v.(map[string]any)["PSCJ"].(string); !ok {
				pscj = ""
			} else {
				pscj = fmt.Sprint(v.(map[string]any)["PSCJ"].(string))
			}
			if _, ok := v.(map[string]any)["QMCJ"].(string); !ok {
				qmcj = ""
			} else {
				qmcj = fmt.Sprint(fmt.Sprint(v.(map[string]any)["QMCJ"].(string)))
			}
			if _, ok := v.(map[string]any)["DJCJMC"].(string); !ok {
				xszcjmc = ""
			} else {
				xszcjmc = fmt.Sprint(v.(map[string]any)["DJCJMC"].(string))
			}
			mu.Lock()
			resp.Grade = append(resp.Grade, &SchoolPb.Grade{
				XNXQDM:  fmt.Sprint(v.(map[string]any)["XNXQDM"].(string)),
				XF:      xufen,
				XSKCM:   fmt.Sprint(v.(map[string]any)["XSKCM"].(string)),
				XSZCJMC: xszcjmc,
				ZCJ:     zcj,
				QMCJ:    qmcj,
				PSCJ:    pscj,
			})
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	return
}

func (*SchoolSrv) SchoolXuefen(ctx context.Context, empty *emptypb.Empty) (resp *SchoolPb.SchoolXuefenResponse, err error) {
	resp = new(SchoolPb.SchoolXuefenResponse)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	stuNum := strings.Join(md.Get("stu_num"), "")
	cacheRDB := cache.NewRDBCache(ctx)
	jwcCertificate := cacheRDB.GetJwcCertificate(stuNum)
	if jwcCertificate.Emaphome_WEU == "" {
		return nil, errors.New("凭证已过期,请重新进入该界面")
	}
	jwc := schoolUtils.NewJwc()
	jwc.GsSession = jwcCertificate.GsSession
	jwc.Emaphome_WEU = jwcCertificate.Emaphome_WEU
	xuefenResponse, err := jwc.GetXuefen()
	if err != nil {
		return
	}
	for _, v := range xuefenResponse.([]any) {
		var yxxf float64
		var yhxf float64
		var xnxqdm string
		var bjgxf float64
		var wlcjxf float64
		if value, ok := v.(map[string]any)["YXXF"].(float64); ok {
			yxxf = value
		} else {
			yxxf = 0
		}
		if value, ok := v.(map[string]any)["YHXF"].(float64); ok {
			yhxf = value
		} else {
			yhxf = 0
		}
		if value, ok := v.(map[string]any)["XNXQDM"].(string); ok {
			xnxqdm = value
		} else {
			xnxqdm = ""
		}
		if value, ok := v.(map[string]any)["BJGXF"].(float64); ok {
			bjgxf = value
		} else {
			bjgxf = 0
		}
		if value, ok := v.(map[string]any)["WLCJXF"].(float64); ok {
			wlcjxf = value
		} else {
			wlcjxf = 0
		}
		resp.Xuefen = append(resp.Xuefen, &SchoolPb.Xytj{
			YXXF:   yxxf,
			YHXF:   yhxf,
			XNXQDM: xnxqdm,
			BJGXF:  bjgxf,
			WLCJXF: wlcjxf,
		})
	}
	return
}

func (*SchoolSrv) SchoolSchedule(ctx context.Context, empty *emptypb.Empty) (resp *SchoolPb.SchoolScheduleResponse, err error) {
	resp = new(SchoolPb.SchoolScheduleResponse)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("信息为空")
	}
	semester := strings.Join(md.Get("semester"), "")
	stuNum := strings.Join(md.Get("stu_num"), "")
	cacheRDB := cache.NewRDBCache(ctx)
	jwcCertificate := cacheRDB.GetJwcCertificate(stuNum)
	if jwcCertificate.Emaphome_WEU == "" {
		return nil, errors.New("凭证已过期,请重新进入该界面")
	}
	jwc := schoolUtils.NewJwc()
	jwc.GsSession = jwcCertificate.GsSession
	jwc.Emaphome_WEU = jwcCertificate.Emaphome_WEU

	scheduleInfo, err := jwc.GetSchedule(semester)
	if err != nil {
		return
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, v := range scheduleInfo.([]any) {
		wg.Add(1)
		go func(v any) {
			defer wg.Done()
			var teachWeek string
			var lessonName string
			var week string
			var beginTime string
			var endTime string
			var address string
			var teacherName string
			if _, ok := v.(map[string]any)["ZCMC"].(string); !ok {
				teachWeek = ""
			} else {
				teachWeek = v.(map[string]any)["ZCMC"].(string)
			}

			if _, ok := v.(map[string]any)["KCM"].(string); !ok {
				lessonName = ""
			} else {
				lessonName = v.(map[string]any)["KCM"].(string)
			}

			if _, ok := v.(map[string]any)["SKXQ_DISPLAY"].(string); !ok {
				week = ""
			} else {
				week = v.(map[string]any)["SKXQ_DISPLAY"].(string)
			}

			if _, ok := v.(map[string]any)["KSJC_DISPLAY"].(string); !ok {
				beginTime = ""
			} else {
				beginTime = v.(map[string]any)["KSJC_DISPLAY"].(string)
			}

			if _, ok := v.(map[string]any)["JSJC_DISPLAY"].(string); !ok {
				endTime = ""
			} else {
				endTime = v.(map[string]any)["JSJC_DISPLAY"].(string)
			}

			if _, ok := v.(map[string]any)["JASMC"].(string); !ok {
				address = ""
			} else {
				address = v.(map[string]any)["JASMC"].(string)
			}

			if _, ok := v.(map[string]any)["SKJS"].(string); !ok {
				teacherName = ""
			} else {
				teacherName = v.(map[string]any)["SKJS"].(string)
			}
			schedule := SchoolPb.Schedule{
				TeachWeek:   teachWeek,
				LessonName:  lessonName,
				Week:        week,
				BeginTime:   beginTime,
				EndTime:     endTime,
				Address:     address,
				TeacherName: teacherName,
			}
			mu.Lock()
			resp.Schedules = append(resp.Schedules, &schedule)
			mu.Unlock()
		}(v)

	}
	wg.Wait()
	return
}
