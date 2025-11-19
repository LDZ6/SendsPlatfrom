package school

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"platform/app/school/types"
	"platform/utils"
	"platform/utils/httpUtil"
	"sync"
)

var JwcSrv *Jwc

type Jwc struct {
	GsSession    string
	Emaphome_WEU string
}

type GpaInfo struct {
	Gssession string `json:"gssession"`
	WEU       string `json:"weu"`
	StuNum    string `json:"stu_num"`
}

func init() {
	JwcSrv = NewJwc()
}

func NewJwc() *Jwc {
	return &Jwc{}
}

func (jwc *Jwc) GetEmaphome_WEU() (emaphome_WEU string, err error) {
	cookieInfo := types.CookieInfo{
		Cookie: jwc.GsSession,
	}
	jsonValue, _ := json.Marshal(cookieInfo)
	resp, err := httpUtil.DoPost(httpUtil.Emaphome_WEU_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
	if err != nil {
		logrus.Info("[GetEmaphome_WEUERROR]:%v\n", err.Error())
		return "", err
	}
	defer resp.Body.Close()
	respJson, _ := ioutil.ReadAll(resp.Body)
	body, err := utils.JSONToMap(string(respJson))
	emaphome_WEU = body["data"].(string)
	return
}

func (jwc *Jwc) GetSchedule(semester string) (scheduleInfo any, err error) {
	schedule := types.ScheduleInfo{
		Gssession: jwc.GsSession,
		WEU:       jwc.Emaphome_WEU,
		Semester:  semester,
	}
	jsonValue, _ := json.Marshal(schedule)
	resp, err := httpUtil.DoPost(httpUtil.Schedule_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
	if err != nil {
		logrus.Info("[GetScheduleERROR]:%v\n", err.Error())
	}
	defer resp.Body.Close()
	respJson, _ := ioutil.ReadAll(resp.Body)
	body, err := utils.JSONToMap(string(respJson))
	if err != nil {
		return
	}
	// 打印响应的数据
	scheduleInfo = body["data"].([]any)
	return
}

func (j *Jwc) GetGpa() (gpa []any, err error) {
	gapInfo := GpaInfo{
		Gssession: j.GsSession,
		WEU:       j.Emaphome_WEU,
	}
	jsonValue, _ := json.Marshal(gapInfo)
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		resp, err := httpUtil.DoPost(httpUtil.Gpa_All_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
		if err != nil {
			logrus.Info("[GetGpaERROR]:%v\n", err.Error())
		}
		defer resp.Body.Close()
		respJson, _ := ioutil.ReadAll(resp.Body)
		body, err := utils.JSONToMap(string(respJson))
		if err != nil {
			return
		}
		// 打印响应的数据
		info, ok := body["data"].(map[string]any)["rows"].([]any)
		if !ok {
			return
		}
		mutex.Lock()
		gpa = append(gpa, info...)
		mutex.Unlock()
	}()
	go func() {
		defer wg.Done()
		resp, err := httpUtil.DoPost(httpUtil.Gpa_Semester_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
		if err != nil {
			logrus.Info("[GetGpaERROR]:%v\n", err.Error())
		}
		defer resp.Body.Close()
		respJson, _ := ioutil.ReadAll(resp.Body)
		body, err := utils.JSONToMap(string(respJson))
		if err != nil {
			return
		}
		// 打印响应的数据
		info, ok := body["data"].(map[string]any)["rows"].([]any)
		if !ok {
			return
		}
		mutex.Lock()
		gpa = append(gpa, info...)
		mutex.Unlock()
	}()
	go func() {
		defer wg.Done()
		resp, err := httpUtil.DoPost(httpUtil.Gpa_Year_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
		if err != nil {
			logrus.Info("[GetGpaERROR]:%v\n", err.Error())
		}
		defer resp.Body.Close()
		respJson, _ := ioutil.ReadAll(resp.Body)
		body, err := utils.JSONToMap(string(respJson))
		if err != nil {
			return
		}
		// 打印响应的数据
		info, ok := body["data"].(map[string]any)["rows"].([]any)
		if !ok {
			return
		}
		mutex.Lock()
		gpa = append(gpa, info...)
		mutex.Unlock()
	}()
	wg.Wait()
	return
}

func (j *Jwc) GetGrade() (grade any, err error) {
	semester := "1"
	gradeInfo := types.GradeInfo{
		Gssession: j.GsSession,
		WEU:       j.Emaphome_WEU,
		Semester:  semester,
	}
	jsonValue, _ := json.Marshal(gradeInfo)
	// 创建一个 HTTP 请求
	resp, err := httpUtil.DoPost(httpUtil.Grade_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
	if err != nil {
		logrus.Info("[GetGradeERROR]:%v\n", err.Error())
	}
	defer resp.Body.Close()
	respJson, _ := ioutil.ReadAll(resp.Body)
	body, err := utils.JSONToMap(string(respJson))
	if err != nil {
		return
	}
	// 打印响应的数据
	grade = body["data"].([]any)
	return
}

func (j *Jwc) GetXuefen() (xuefen any, err error) {
	semester := "1"
	xuefenInfo := types.XueFenInfo{
		Gssession: j.GsSession,
		WEU:       j.Emaphome_WEU,
		Semester:  semester,
	}
	jsonValue, _ := json.Marshal(xuefenInfo)
	// 创建一个 HTTP 请求
	resp, err := httpUtil.DoPost(httpUtil.Xuefen_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
	if err != nil {
		logrus.Info("[GetXueFenERROR]:%v\n", err.Error())
		return
	}
	defer resp.Body.Close()
	respJson, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	fmt.Println(string(respJson))
	body, err := utils.JSONToMap(string(respJson))
	if err != nil {
		return
	}
	// 打印响应的数据
	xuefen, ok := body["data"].([]any)
	if !ok {
		return nil, nil
	}
	return
}
