package service

import (
	"context"
	"errors"
	"fmt"
	userPb "platform/idl/pb/user"
	"platform/utils"
	"sync"
)

func (*UserSrv) YearBillLogin(ctx context.Context, req *userPb.UserLoginRequest) (resp *userPb.UserLoginResponse, err error) {
	resp = new(userPb.UserLoginResponse)
	wxLoginResp, err := utils.WXLogin(req.Code)
	if err != nil || wxLoginResp.OpenId == "" {
		return
	}
	openid := wxLoginResp.OpenId
	var stResp *utils.GetStuNumResp
	wxInfoResp := &utils.WeChatUser{}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		stResp, err = utils.GetStuNum(openid)
		if err != nil {
			err = errors.New("找不到学号，请绑定桑梓微助手！")
			return
		}
	}()
	go func() {
		defer wg.Done()
		wxInfoResp, err = utils.GetWeChatInfo(openid, wxLoginResp.AccessToken)
		if err != nil {
			err = fmt.Errorf("%w %w", err, errors.New("获取头像昵称错误！"))
		}
		return
	}()
	wg.Wait()
	token := utils.GenerateUserToken(openid, stResp.Stuid)
	resp.Token = token
	resp.NickName = wxInfoResp.Nickname
	resp.Avatar = wxInfoResp.HeadImgURL
	return
}
