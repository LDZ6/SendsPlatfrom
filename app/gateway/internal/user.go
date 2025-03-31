package internal

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"platform/app/gateway/rpc"
	"platform/app/gateway/types"
	userPb "platform/idl/pb/user"
)

// UserLogin
// @Tags 用户
// @Summary 用户登陆
// @Param codeInfo body types.CodeInfo true "微信获取code"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /user/login [post]
func UserLogin(c *gin.Context) {

	//-----------当前接入的是群众登录接口

	json := types.CodeInfo{}
	c.BindJSON(&json)
	code := json.Code
	logrus.Info(code)
	userReq := userPb.UserLoginRequest{Code: code}
	//resp, err := rpc.UserLogin(c, &userReq)
	resp, err := rpc.MassesLogin(c, &userReq)
	if err != nil {
		if err.Error() == "rpc error: code = Unknown desc =找不到学号，请绑定桑梓微助手！" {
			types.ResponseErrorWithMsg(c, http.StatusForbidden, errors.New("找不到学号，请绑定桑梓微助手！"))
			return
		}
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp.Token)
}

// SchoolUserLogin
// @Tags 用户
// @Summary 校内服务用户登陆
// @Param codeInfo body types.CodeInfo true "微信获取code"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /user/school_login [post]
func SchoolUserLogin(c *gin.Context) {
	json := types.CodeInfo{}
	c.BindJSON(&json)
	code := json.Code
	userReq := userPb.UserLoginRequest{Code: code}
	//resp, err := rpc.UserLogin(c, &userReq)
	resp, err := rpc.SchoolUserLoginRpc(c, &userReq)
	if err != nil {
		if err.Error() == "rpc error: code = Unknown desc =登录失败请绑定桑梓微助手" {
			types.ResponseErrorWithMsg(c, http.StatusForbidden, errors.New("找不到学号，请绑定桑梓微助手！"))
			return
		}
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp.Token)
}

// WxJSSDK
// @Tags 用户
// @Summary JS-SDk权限验证配置
// @Param codeInfo body types.UrlInfo true "获取url"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /user/jssdk [post]
func WxJSSDK(c *gin.Context) {
	json := types.UrlInfo{}
	c.ShouldBindJSON(&json)
	req := userPb.WxJSSDKRequest{Url: json.Url}
	resp, err := rpc.WxJSSDKRpc(c, &req)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err)
	}
	types.ResponseSuccess(c, resp)
}

// YearBillLogin
// @Tags 用户
// @Summary 年度账单用户登陆
// @Param codeInfo body types.CodeInfo true "微信获取code"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /user/bill_login [post]
func YearBillLogin(c *gin.Context) {
	json := types.CodeInfo{}
	c.ShouldBindJSON(&json)
	code := json.Code
	logrus.Info(code)
	userReq := userPb.UserLoginRequest{Code: code}
	resp, err := rpc.YearBillLoginRpc(c, &userReq)
	if err != nil {
		if err.Error() == "rpc error: code = Unknown desc =找不到学号，请绑定桑梓微助手！" {
			types.ResponseErrorWithMsg(c, http.StatusForbidden, errors.New("找不到学号，请绑定桑梓微助手！"))
			return
		}
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, "请绑定桑梓微助手！")
		return
	}
	types.ResponseSuccess(c, resp)
}
