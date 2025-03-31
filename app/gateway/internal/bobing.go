package internal

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"net/http"
	"platform/app/gateway/rpc"
	"platform/app/gateway/types"
	boBingPb "platform/idl/pb/boBing"
	"platform/utils"
	"strconv"
	"sync"
	"time"
)

// BoBingPublish
// @Tags 博饼
// @Summary 提交博饼信息
// @Param token header string true "token"
// @Param scoreMessage body types.ScoreMessage true "获取积分及校验,如果是稀有会返回密文，进行广播的校验"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/publish [post]
func BoBingPublish(c *gin.Context) {
	md := metadata.Pairs("open_id", c.Value("open_id").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	err := rpc.BoBingBlacklistRpc(ctx)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}

	json := types.ScoreMessage{}
	c.ShouldBindJSON(&json)
	var openId string
	var nickName string
	if v, exists := c.Get("open_id"); exists {
		openId = v.(string)
	} else {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无openid"))
		return
	}
	if v, exists := c.Get("nick_name"); exists {
		nickName = v.(string)
	} else {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无nickname"))
		return
	}
	req := boBingPb.BoBingPublishRequest{
		NickName: nickName,
		Flag:     json.Points,
		Check:    json.Detail,
		OpenId:   openId,
	}
	resp, err := rpc.BoBingPublishRpc(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "rpc error: code = Unknown desc = 跳猴" {
			types.ResponseSuccessWithMsg(c, "投掷次数-1", resp)
			return
		} else if err.Error() == "rpc error: code = Unknown desc = 加二" {
			types.ResponseSuccessWithMsg(c, "投掷次数+1", resp)
			return
		} else if err.Error() == "rpc error: code = Unknown desc = 加三" {
			types.ResponseSuccessWithMsg(c, "投掷次数+1", resp)
			return
		} else if err.Error() == "rpc error: code = Unknown desc = 加五" {
			types.ResponseSuccessWithMsg(c, "投掷次数+2", resp)
			return
		} else if err.Error() == "rpc error: code = Unknown desc = 加六" {
			types.ResponseSuccessWithMsg(c, "投掷次数+2", resp)
			return
		}
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// BoBingToTalTen
// @Tags 博饼
// @Summary 获取前十及自己的信息（可轮询）
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/top [get]
func BoBingToTalTen(c *gin.Context) {
	md := metadata.Pairs("nick_name", base64.StdEncoding.EncodeToString([]byte(c.Value("nick_name").(string))),
		"open_id", c.Value("open_id").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := rpc.BoBingToTalTenRpc(ctx)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// BoBingKey
// @Tags 博饼
// @Summary 获取密钥
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/key [get]
func BoBingKey(c *gin.Context) {
	var openId string
	if v, exists := c.Get("open_id"); exists {
		openId = v.(string)
	} else {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无openid"))
		return
	}
	req := boBingPb.BoBingKeyRequest{Openid: openId}
	resp, err := rpc.BoBingKeyRpc(c.Request.Context(), &req)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// BoBingDayRank
// @Tags 博饼
// @Summary 获取博饼日榜（可轮询）
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/dayRank [get]
func BoBingDayRank(c *gin.Context) {
	md := metadata.Pairs("nick_name", base64.StdEncoding.EncodeToString([]byte(c.Value("nick_name").(string))),
		"open_id", c.Value("open_id").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := rpc.BoBingDayRankRpc(ctx)
	if err != nil {
		if err.Error() != "rpc error: code = Unknown desc =key不存在" {
			types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
			return
		}
	}
	types.ResponseSuccess(c, resp)
}

var wc = make(map[string]*websocket.Conn)
var mutex sync.Mutex

// BoBingBroadcastMessage
// @Tags 博饼
// @Summary 广播信息（使用websocket）
// @Description Echoes messages sent from the client back to the client
// @Param token header string true "token"
// @Param broadcastMessage body types.BroadcastMessage true "获取要广播的内容和token，当提交稀有的博饼记录时，返回密文用于这里进行一个校验"
// @Accept json
// @Produce json
// @Router /ws/broadcast [get]
func BoBingBroadcastMessage(c *gin.Context) {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 不检查 Origin
		},
		Subprotocols: []string{c.GetHeader("Sec-WebSocket-Protocol")},
	}
	protocol := c.GetHeader("Sec-WebSocket-Protocol")

	// 设置响应头
	header := http.Header{}
	header.Set("Sec-WebSocket-Protocol", protocol)
	conn, err := upgrader.Upgrade(c.Writer, c.Request, header)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统异常：" + err.Error(),
		})
		return
	}
	defer conn.Close()
	uc := c.MustGet("open_id").(string)
	nickName := c.MustGet("nick_name").(string)
	mutex.Lock()
	wc[uc] = conn
	mutex.Unlock()
	for {
		messageBasic := types.BroadcastMessage{}
		err = conn.ReadJSON(&messageBasic)
		if err != nil {
			continue
		}
		user, err := utils.AnalyseUserToken(messageBasic.Token)
		if err != nil {
			logrus.Info("AnalyseUserToken Error:%v\n", err)
			continue
		}
		req := boBingPb.BoBingBroadcastCheckRequest{
			Ciphertext: messageBasic.Ciphertext,
			OpenId:     user.OpenId,
		}
		err = rpc.BoBingBroadcastCheckRpc(c.Request.Context(), &req)
		if err != nil {
			return
		}

		//对所有在线用户进行广播
		for key, client := range wc {
			err := client.WriteMessage(websocket.TextMessage, []byte("恭喜幸运用户"+nickName+"运气大爆发掷出"+messageBasic.Message+"!"))
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				client.Close()
				mutex.Lock()
				delete(wc, key)
				mutex.Unlock()
			}
		}
	}
}

// 发送在线人数
func SendOnlineCount() {
	for {
		time.Sleep(25 * time.Second)
		onlineCount := len(wc)
		message := fmt.Sprintf("当前在线博饼人数：%d", onlineCount+8)
		resp, _ := rpc.BoBingGetNumberRp(context.Background())
		for key, coon := range wc {
			err := coon.WriteMessage(websocket.TextMessage, []byte(message))
			err = coon.WriteMessage(websocket.TextMessage, []byte("活动总参与人数为："+strconv.Itoa(int(resp.Number))))
			if err != nil {
				coon.Close()
				mutex.Lock()
				delete(wc, key)
				mutex.Unlock()
			}
		}
	}
}

// BoBingDayInit
// @Tags 博饼
// @Summary 博饼每次进入后的初始化操作
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/init [get]
func BoBingDayInit(c *gin.Context) {
	var openId string
	var nickName string
	if v, exists := c.Get("open_id"); exists {
		openId = v.(string)
	} else {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无openid"))
		return
	}
	if v, exists := c.Get("nick_name"); exists {
		nickName = v.(string)
	} else {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无nickname"))
		return
	}
	req := boBingPb.BoBingInitRequest{
		NickName: nickName,
		OpenId:   openId,
	}
	err := rpc.BoBingDayInitRpc(c.Request.Context(), &req)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, "初始化成功")
}

// BoBingTianXuan
// @Tags 博饼
// @Summary 博饼天选榜(可轮询)
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/tianXuan [get]
func BoBingTianXuan(c *gin.Context) {
	resp, err := rpc.BoBingTianXuanRpc(c.Request.Context())
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// BoBingGetCount
// @Tags 博饼
// @Summary 博饼获取当天剩余次数
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/getCount [get]
func BoBingGetCount(c *gin.Context) {
	md := metadata.Pairs(
		"open_id", c.Value("open_id").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := rpc.BoBingGetCountRpc(ctx)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// BoBingRetransmission
// @Tags 博饼
// @Summary 转发阅读推文增加次数
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/addCount [get]
func BoBingRetransmission(c *gin.Context) {
	md := metadata.Pairs(
		"open_id", c.Value("open_id").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	err := rpc.BoBingRetransmissionRpc(ctx)
	if err != nil {
		if err.Error() == "rpc error: code = Unknown desc = 今日增加投掷次数已达上限" {
			types.ResponseSuccess(c, "今日增加投掷次数已达上限")
			return
		}
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, "次数增加成功")
}

// BoBingRecord
// @Tags 博饼
// @Summary 查看自己的博饼记录
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /boBing/record [get]
func BoBingRecord(c *gin.Context) {
	md := metadata.Pairs(
		"open_id", c.Value("open_id").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := rpc.BoBingRecordRpc(ctx)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}
