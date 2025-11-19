package utils

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"platform/config"
	"sort"
	"strings"
	"time"
)

type WXLoginResp struct {
	OpenId       string `json:"openid"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type GetAccessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type GetStuNumPost struct {
	OpenId   string `json:"openid"`
	PassWord string `json:"pass_word"`
}

type MessageResp struct {
	Errcode int    `json:"errcode"` //错误码
	Errmsg  string `json:"errmsg"`  //错误消息
	Msgid   int    `json:"msgid"`   //消息id
}

type WeChatUser struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionID    string   `json:"unionid"`
}

type GetStuNumResp struct {
	Msg   string `json:"msg"`
	Stuid string `json:"stuid"`
}

type JsApiTicket struct {
	Ticket    string `json:"ticket"`     // jsapi_ticket 的值
	ExpiresIn int64  `json:"expires_in"` // 有效时间，单位为秒
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
}

// 这个函数以 code 作为输入, 返回调用微信接口得到的对象指针和异常情况
func WXLogin(code string) (*WXLoginResp, error) {
	url := "https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code"

	// 合成url, 这里的appId和secret是在微信公众平台上获取的
	url = fmt.Sprintf(url, config.AppId, config.AppSerect, code)

	// 创建http get请求
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析http请求中body 数据到我们定义的结构体中
	wxResp := WXLoginResp{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&wxResp); err != nil {
		return nil, err
	}

	// 判断微信接口返回的是否是一个异常情况
	if wxResp.ErrCode != 0 {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d  ErrMsg:%s", wxResp.ErrCode, wxResp.ErrMsg))
	}

	return &wxResp, nil
}

func GetWeChatInfo(openid, accessToken string) (*WeChatUser, error) {
	resp, err := http.Get("https://api.weixin.qq.com/sns/userinfo?access_token=" + accessToken + "&openid=" + openid + "&lang=zh_CN")
	defer resp.Body.Close()
	weChatUser := WeChatUser{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&weChatUser); err != nil {
		return nil, err
	}
	return &weChatUser, err
}

func GetStuNum(openId string) (*GetStuNumResp, error) {
	data := make(map[string]interface{})
	data["openid"] = openId
	data["password"] = ""
	bytesData, _ := json.Marshal(data)
	fmt.Println(bytesData)
	resp, _ := http.Post("", "application/json", bytes.NewReader(bytesData))
	defer resp.Body.Close()
	stResp := GetStuNumResp{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&stResp); err != nil {
		return nil, err
	}
	log.Printf(stResp.Msg)
	if stResp.Msg != "success" {
		return nil, errors.New("找不到学号，请绑定桑梓微助手。")
	}
	return &stResp, nil
}

// 这个api不可获取头像、昵称的API
func GetAccessToken() (*GetAccessTokenResp, error) {
	url := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s"

	// 合成url, 这里的appId和secret是在微信公众平台上获取的
	url = fmt.Sprintf(url, config.AppId, config.AppSerect)

	// 创建http get请求
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析http请求中body 数据到我们定义的结构体中
	accessTokenResp := GetAccessTokenResp{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&accessTokenResp); err != nil {
		return nil, err
	}

	// 判断微信接口返回的是否是一个异常情况
	if accessTokenResp.ErrCode != 0 {
		return nil, errors.New(fmt.Sprintf("ErrCode:%s  ErrMsg:%s", accessTokenResp.ErrCode, accessTokenResp.ErrMsg))
	}
	config.ExpiresIn = accessTokenResp.ExpiresIn

	return &accessTokenResp, nil
}

func GetJsApiTicket(accessToken string) (*JsApiTicket, error) {
	url := "https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi"

	// 合成 url
	url = fmt.Sprintf(url, accessToken)

	// 创建 http get 请求
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析 http 请求中 body 数据到我们定义的结构体中
	jsapiTicketResp := JsApiTicket{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&jsapiTicketResp); err != nil {
		return nil, err
	}

	// 判断微信接口返回的是否是一个异常情况
	if jsapiTicketResp.ErrCode != 0 {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d  ErrMsg:%s", jsapiTicketResp.ErrCode, jsapiTicketResp.ErrMsg))
	}

	return &jsapiTicketResp, nil
}

// 生成签名
func GenerateSignature(jsapiTicket string, timestamp int64, nonceStr string, url string) string {
	string1 := makeURLParams(map[string]interface{}{
		"jsapi_ticket": jsapiTicket,
		"noncestr":     nonceStr,
		"timestamp":    timestamp,
		"url":          url,
	})
	signature := GetSHA1(string1)
	return signature
}
func GetSHA1(str ...string) string {
	// 创建一个 SHA1 哈希对象
	h := sha1.New()
	// 遍历给定的字符串切片
	for _, s := range str {
		// 将每个字符串写入哈希对象中
		h.Write([]byte(s))
	}
	// 计算哈希值并转换为十六进制字符串
	return hex.EncodeToString(h.Sum(nil))
}

func makeURLParams(params map[string]interface{}) string {
	keys := make([]string, 0, len(params)) // 创建一个字符串切片，用于存储map的键
	for k := range params {
		keys = append(keys, k) // 将map的键追加到切片中
	}
	sort.Strings(keys) // 对切片按照字典序排序

	pairs := make([]string, 0, len(params)) // 创建一个字符串切片，用于存储键值对
	for _, k := range keys {
		v := fmt.Sprintf("%v", params[k])               // 将map的值转换为字符串
		pairs = append(pairs, k+"="+url.QueryEscape(v)) // 将键和值拼接成键值对，并对值进行URL转义
	}
	return strings.Join(pairs, "&") // 将键值对用&连接，返回字符串
}

func RefreshToken() (string, error) {
	now := time.Now()
	expiresInString := string(config.ExpiresIn) + "s"
	ss, _ := time.ParseDuration(expiresInString)
	ss15 := config.AccessTokenCreatTime.Add(ss)
	if now.After(ss15) {
		accessTokenResp, e := GetAccessToken()
		if e != nil {
			log.Println(e)
			return "", e
		}
		config.AccessToken = accessTokenResp.AccessToken
		config.AccessTokenCreatTime = now
	}
	return config.AccessToken, nil
}

func PassMessage(openId string, songName string, sender string, receiver string, broadcost_date time.Time) (string, error) {
	accessToken, e := RefreshToken()
	if e != nil {
		log.Println(e)
		return "", e
	}
	log.Println(accessToken)
	//url := "http://wx.sends.cc/d46c6Zd6wyEto4gWqyfx/getstuid"
	//创建http post请求
	data := make(map[string]interface{})
	msg := make(map[string]interface{})
	fisrt := make(map[string]interface{})
	keyword1 := make(map[string]interface{})
	keyword2 := make(map[string]interface{})
	keyword3 := make(map[string]interface{})
	keyword4 := make(map[string]interface{})
	remark := make(map[string]interface{})
	data["touser"] = openId
	data["template_id"] = "bZPTkrCWnp6ySHclvbHTdO6-_jS-CFXrWM6-UVpLYHs"
	data["url"] = "songs.sends.cc"
	fisrt["value"] = "您申请的歌曲已通过审核！"
	keyword1["value"] = songName
	keyword2["value"] = sender
	keyword3["value"] = receiver
	keyword4["value"] = broadcost_date.Format("2006-01-02")
	remark["value"] = "记得按时收听广播哦~如果您对目前点歌台的功能有什么建议的话，欢迎加入点歌台反馈群与我们交流！（QQ群号：651565455）"
	msg["first"] = fisrt
	msg["keyword1"] = keyword1
	msg["keyword2"] = keyword2
	msg["keyword3"] = keyword3
	msg["keyword4"] = keyword4
	msg["remark"] = remark
	data["data"] = msg
	bytesData, _ := json.Marshal(data)
	resp, _ := http.Post("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token="+accessToken, "application/json", bytes.NewReader(bytesData))
	defer resp.Body.Close()
	// 解析http请求中body 数据到我们定义的结构体中
	msgResp := MessageResp{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&msgResp); err != nil {
		return "", err
	}
	log.Printf(msgResp.Errmsg)
	if msgResp.Errmsg != "ok" {
		return msgResp.Errmsg, errors.New("模板消息发送失败。")
	}
	return msgResp.Errmsg, nil
}

func NoPassMessage(openId string, songName string, sender string, receiver string, reason string) (string, error) {
	accessToken, e := RefreshToken()
	if e != nil {
		log.Println(e)
		return "", e
	}
	log.Println(accessToken)
	//url := "http://wx.sends.cc/d46c6Zd6wyEto4gWqyfx/getstuid"
	//创建http post请求
	data := make(map[string]interface{})
	msg := make(map[string]interface{})
	fisrt := make(map[string]interface{})
	keyword1 := make(map[string]interface{})
	keyword2 := make(map[string]interface{})
	keyword3 := make(map[string]interface{})
	keyword4 := make(map[string]interface{})
	remark := make(map[string]interface{})
	data["touser"] = openId
	data["template_id"] = "Bt0fO_8QVHDiSKXX_hBynVIDtcqFHq5J6omCVkjVDSk"
	data["url"] = "songs.sends.cc"
	fisrt["value"] = "您申请的歌曲未通过审核！"
	keyword1["value"] = songName
	keyword2["value"] = sender
	keyword3["value"] = receiver
	keyword4["value"] = reason
	remark["value"] = "您可以在申请列表中重新提交歌曲。如果您对目前点歌台的功能有什么建议的话，欢迎加入点歌台反馈群与我们交流！（QQ群号：651565455）"
	msg["first"] = fisrt
	msg["keyword1"] = keyword1
	msg["keyword2"] = keyword2
	msg["keyword3"] = keyword3
	msg["keyword4"] = keyword4
	msg["remark"] = remark
	data["data"] = msg
	bytesData, _ := json.Marshal(data)
	resp, _ := http.Post("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token="+accessToken, "application/json", bytes.NewReader(bytesData))
	defer resp.Body.Close()
	// 解析http请求中body 数据到我们定义的结构体中
	msgResp := MessageResp{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&msgResp); err != nil {
		return "", err
	}
	log.Printf(msgResp.Errmsg)
	if msgResp.Errmsg != "ok" {
		return msgResp.Errmsg, errors.New("模板消息发送失败。")
	}
	return msgResp.Errmsg, nil
}

func NewSongMessage(openId string, songName string, sender string, receiver string, broadcost_date time.Time) (string, error) {
	accessToken, e := RefreshToken()
	if e != nil {
		log.Println(e)
		return "", e
	}
	log.Println(accessToken)
	//url := "http://wx.sends.cc/d46c6Zd6wyEto4gWqyfx/getstuid"
	//创建http post请求
	data := make(map[string]interface{})
	msg := make(map[string]interface{})
	fisrt := make(map[string]interface{})
	keyword1 := make(map[string]interface{})
	keyword2 := make(map[string]interface{})
	keyword3 := make(map[string]interface{})
	keyword4 := make(map[string]interface{})
	remark := make(map[string]interface{})
	data["touser"] = openId
	data["template_id"] = "FPDKi8I-_298XAZffp6VznNfQrhlgtOH-ff-s3Yy8u8"
	data["url"] = "songs.sends.cc/#/admin/home"
	fisrt["value"] = "收到新的点歌申请！"
	keyword1["value"] = songName
	keyword2["value"] = sender
	keyword3["value"] = receiver
	keyword4["value"] = broadcost_date.Format("2006-01-02")
	remark["value"] = "请记得及时审核哦"
	msg["first"] = fisrt
	msg["keyword1"] = keyword1
	msg["keyword2"] = keyword2
	msg["keyword3"] = keyword3
	msg["keyword4"] = keyword4
	msg["remark"] = remark
	data["data"] = msg
	bytesData, _ := json.Marshal(data)
	resp, _ := http.Post("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token="+accessToken, "application/json", bytes.NewReader(bytesData))
	defer resp.Body.Close()
	// 解析http请求中body 数据到我们定义的结构体中
	msgResp := MessageResp{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&msgResp); err != nil {
		return "", err
	}
	log.Printf(msgResp.Errmsg)
	if msgResp.Errmsg != "ok" {
		return msgResp.Errmsg, errors.New("模板消息发送失败。")
	}
	return msgResp.Errmsg, nil
}
