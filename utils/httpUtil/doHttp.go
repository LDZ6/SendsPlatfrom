package httpUtil

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

func DoPost(Url string, cookies map[string]string, jsonValue io.Reader, headerType string) (resp *http.Response, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", Url, jsonValue)
	for k, v := range cookies {
		cookie := &http.Cookie{
			Name:    k,
			Value:   v,
			Expires: time.Now().Add(10 * time.Second),
		}
		req.AddCookie(cookie)
	}
	if err != nil {
		fmt.Println("创建请求失败:", err)
		return nil, errors.New("创建请求失败")
	}
	req.Header.Set("Content-Type", headerType)
	// 发送请求
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return resp, nil
}

func DoGet(Url string, cookies map[string]string, headerType string) (resp *http.Response, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", Url, nil)
	for k, v := range cookies {
		cookie := &http.Cookie{
			Name:    k,
			Value:   v,
			Expires: time.Now().Add(10 * time.Second),
		}
		req.AddCookie(cookie)
	}
	req.Header.Set("Content-Type", headerType)
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return resp, nil
}
