package internal

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"platform/app/gateway/rpc"
	"platform/app/gateway/types"
	yearBillPb "platform/idl/pb/yearBill"
	"time"
)

// YearBillDataInit
// @Tags 年度账单
// @Summary 数据初始化(websocket)
// @Param token header string true "token"
// @Description Echoes messages sent from the client back to the client
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /yearBill/init [get]
func YearBillDataInit(c *gin.Context) {
	conn, ok := c.Get("wsConn")
	if !ok {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("websocket connection not found"))
		return
	}
	WsConn, ok := conn.(*websocket.Conn)
	if !ok {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("invalid websocket connection"))
		return
	}

	var stuNum, openId string
	if v, exists := c.Get("stu_num"); !exists {
		WsConn.WriteJSON(gin.H{
			"code": types.CodeServerBusy,
			"msg":  "token无stu_num",
		})
		WsConn.Close()
		return
	} else {
		stuNum = v.(string)
	}
	if v, exists := c.Get("open_id"); exists {
		openId = v.(string)
	} else {
		WsConn.WriteJSON(gin.H{
			"code": types.CodeServerBusy,
			"msg":  "token无openid",
		})
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无openid"))
		WsConn.Close()
		return
	}

	//校验是否已经初始化过
	reqInfo := yearBillPb.InfoCheckRequest{StuNum: stuNum}
	respInfo, err := rpc.InfoCheckRpc(c.Request.Context(), &reqInfo)
	if err != nil {
		WsConn.WriteJSON(gin.H{
			"code": types.CodeServerBusy,
			"msg":  err.Error(),
		})
		WsConn.Close()
		return
	}
	if respInfo.Flag == true {
		WsConn.WriteJSON(gin.H{
			"code": types.CodeSuccess,
			"msg":  "数据初始化成功",
		})
		WsConn.Close()
		return
	}

	//无获取凭证
	req := yearBillPb.GetCertificateRequest{
		Openid: openId,
		StuNum: stuNum,
	}
	resp, err := rpc.GetCertificateRpc(c.Request.Context(), &req)
	if err != nil {
		WsConn.WriteJSON(gin.H{
			"code": types.CodeServerBusy,
			"msg":  err.Error(),
		})
		WsConn.Close()
		return
	}

	if resp.HallTicket == "" || resp.Emaphome_WEU == "" || resp.GsSession == "" {
		WsConn.WriteJSON(gin.H{
			"code": types.CodeServerBusy,
			"msg":  "获取凭证失败，请确定你已成功绑定桑梓微助手",
		})
		WsConn.Close()
		return
	}
	WsConn.WriteJSON(gin.H{
		"code": "1004",
		"msg":  "数据正在初始化",
	})

	start := time.Now()
	for {
		time.Sleep(1 * time.Second)
		resp, err := rpc.CheckStateRpc(c, &yearBillPb.CheckStateRequest{StuNum: stuNum})
		if err != nil {
			WsConn.WriteJSON(gin.H{
				"code": types.CodeServerBusy,
				"msg":  err.Error(),
			})
			WsConn.Close()
			return
		}
		if resp.State == true {
			WsConn.WriteJSON(gin.H{
				"code": types.CodeSuccess,
				"msg":  "数据初始化成功",
			})
			WsConn.Close()
			return
		}
		elapsed := time.Since(start)
		if elapsed > 30*time.Second {
			WsConn.WriteJSON(gin.H{
				"code": "1002",
				"msg":  "当前活动火爆，请稍等",
			})
		}
		if elapsed > 4*time.Minute {
			WsConn.WriteJSON(gin.H{
				"code": "1003",
				"msg":  "当前活动太火爆，请稍后再来吧",
			})
			WsConn.Close()
			return
		}
	}
	WsConn.Close()
}

// 异步执行的初始化方法
func YearBillDataInitSync() {
	for {
		rpc.DataInitRpc(context.Background())
	}
}

// GetPayData
// @Tags 年度账单
// @Summary 获得花费数据
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /yearBill/pay [get]
func GetPayData(c *gin.Context) {
	var stuNum string
	if v, exists := c.Get("stu_num"); !exists {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无stu_num"))
		return
	} else {
		stuNum = v.(string)
	}
	req := yearBillPb.GetPayDataRequest{
		StuNum: stuNum,
	}
	resp, err := rpc.GetPayDataRpc(c.Request.Context(), &req)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// GetLearnData
// @Tags 年度账单
// @Summary 获得学习数据
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /yearBill/learn [get]
func GetLearnData(c *gin.Context) {
	var stuNum string
	if v, exists := c.Get("stu_num"); !exists {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无stu_num"))
		return
	} else {
		stuNum = v.(string)
	}
	req := yearBillPb.GetLearnDataRequest{
		StuNum: stuNum,
	}
	resp, err := rpc.GetLearnDataRpc(c.Request.Context(), &req)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// GetRank
// @Tags 年度账单
// @Summary 获得这是第几份年度账单数据
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /yearBill/rank [get]
func GetRank(c *gin.Context) {
	var stuNum string
	if v, exists := c.Get("stu_num"); !exists {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无stu_num"))
		return
	} else {
		stuNum = v.(string)
	}
	req := yearBillPb.GetRankRequest{
		StuNum: stuNum,
	}
	resp, err := rpc.GetRankRpc(c.Request.Context(), &req)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// Appraise
// @Tags 年度账单
// @Summary 对年度账单活动进行评价
// @Param token header string true "token"
// @Param scoreMessage body types.Appraise true "获取积分及校验,如果是稀有会返回密文，进行广播的校验"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /yearBill/appraise [post]
func Appraise(c *gin.Context) {
	json := types.Appraise{}
	c.ShouldBindJSON(&json)
	var stuNum string
	if v, exists := c.Get("stu_num"); !exists {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无stu_num"))
		return
	} else {
		stuNum = v.(string)
	}
	req := yearBillPb.AppraiseRequest{
		StuNum:    stuNum,
		Appraisal: json.Appraisal,
	}
	err := rpc.AppraiseRpc(c.Request.Context(), &req)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, "评价成功")
}
