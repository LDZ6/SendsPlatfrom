package school

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"platform/app/user/types"
	"platform/utils"
	"platform/utils/httpUtil"
)

type OpenIdInfo struct {
	OpenId string `json:"open_id"`
}

func GetGsSession(openid string) (gsSession string, err error) {
	openIdInfo := OpenIdInfo{
		OpenId: openid,
	}
	jsonValue, err := json.Marshal(openIdInfo)
	resp, err := httpUtil.DoPost(httpUtil.GsSession_URL, nil, bytes.NewReader(jsonValue), httpUtil.ScheduleHeaderType)
	if err != nil {
		logrus.Info("[GetGsSessionERROR]:%v\n", err.Error())
		return "", err
	}
	defer resp.Body.Close()
	respJson, _ := ioutil.ReadAll(resp.Body)
	body, err := utils.JSONToMap(string(respJson))
	if err != nil {
		return
	}
	// 打印响应的数据
	if body["code"].(float64) != 200 {
		return "", errors.New("登录失败请绑定桑梓微助手")
	}
	gsSession = body["data"].(string)
	return
}

func ECardLogin(openid string) (info types.ECardCertificate, err error) {
	openIdInfo := OpenIdInfo{
		OpenId: openid,
	}
	jsonValue, err := json.Marshal(openIdInfo)
	resp, err := httpUtil.DoPost(httpUtil.ECard_URL, nil, bytes.NewReader(jsonValue), httpUtil.ScheduleHeaderType)
	if err != nil {
		logrus.Info("[ECardLoginERROR]:%v\n", err.Error())
		return info, err
	}
	defer resp.Body.Close()
	respJson, _ := ioutil.ReadAll(resp.Body)
	body, err := utils.JSONToMap(string(respJson))
	if err != nil {
		return
	}
	// 打印响应的数据
	if body["code"].(float64) != 200 {
		return info, errors.New("登录失败请绑定桑梓微助手")
	}
	info.JsSessionId = body["data"].(map[string]any)["jsessionid"].(string)
	info.HallTicket = body["data"].(map[string]any)["hallticket"].(string)
	return
}
