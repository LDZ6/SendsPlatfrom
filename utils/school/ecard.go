package school

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"platform/utils"
	"platform/utils/httpUtil"
)

var ECardSrv *ECard

type ECard struct {
	HallTicket  string
	JsSessionId string
}

func init() {
	ECardSrv = NewECard()
}

func NewECard() *ECard {
	return &ECard{}
}

type CardInfo struct {
	Hallticket string `json:"hallticket"`
}

type GetPay struct {
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Account    string `json:"account"`
	Page       string `json:"page"`
	Hallticket string `json:"hallticket"`
}

type GetBook struct {
	PageNo     string `json:"page_no"`
	JSESSIONID string `json:"jsessionid"`
}

func (card *ECard) GetCardInfo() (info map[string]any, err error) {
	cardInfo := CardInfo{
		Hallticket: card.HallTicket,
	}
	jsonValue, _ := json.Marshal(cardInfo)
	resp, err := httpUtil.DoPost(httpUtil.Card_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
	if err != nil {
		logrus.Info("[GetCardInfoERROR]:%v\n", err.Error())
		return
	}
	defer resp.Body.Close()
	respJson, _ := ioutil.ReadAll(resp.Body)
	info, err = utils.JSONToMap(string(respJson))
	info, err = utils.JSONToMap(info["data"].(map[string]any)["Msg"].(string))
	return
}

func (card *ECard) GetPay(account string, page string) (info map[string]any, err error) {
	getPay := GetPay{
		StartDate:  "2023-01-01",
		EndDate:    "2023-12-31",
		Account:    account,
		Page:       page,
		Hallticket: card.HallTicket,
	}
	jsonValue, _ := json.Marshal(getPay)
	resp, err := httpUtil.DoPost(httpUtil.Pay_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
	if err != nil {
		logrus.Info("[GetPayERROR]:%v\n", err.Error())
		return
	}
	defer resp.Body.Close()
	respJson, _ := ioutil.ReadAll(resp.Body)
	info, err = utils.JSONToMap(string(respJson))
	return info["data"].(map[string]any), err
}

func (card *ECard) GetBook(pageNo string) (info map[string]any, err error) {
	getBook := GetBook{
		PageNo:     pageNo,
		JSESSIONID: card.JsSessionId,
	}
	jsonValue, _ := json.Marshal(getBook)
	resp, err := httpUtil.DoPost(httpUtil.Book_URL, nil, bytes.NewReader(jsonValue), httpUtil.NormalHeaderType)
	if err != nil {
		logrus.Info("[GetBookERROR]:%v\n", err.Error())
		return
	}
	defer resp.Body.Close()
	respJson, _ := ioutil.ReadAll(resp.Body)
	info, err = utils.JSONToMap(string(respJson))
	return info, err
}
