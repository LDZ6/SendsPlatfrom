package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"platform/app/gateway/types"
	"platform/utils"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源的连接
		return true
	},
}

func WebsocketMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		protocol := c.GetHeader("Sec-WebSocket-Protocol")
		header := http.Header{}
		header.Set("Sec-WebSocket-Protocol", protocol)
		conn, err := upgrader.Upgrade(c.Writer, c.Request, header)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		auth := c.GetHeader("Sec-WebSocket-Protocol")
		userClaims, err := utils.AnalyseUserToken(auth)
		if err != nil {
			c.Abort()
			conn.WriteJSON(gin.H{
				"code": types.CodeServerBusy,
				"msg":  "身份校验错误",
			})
			conn.Close()
			return
		}
		c.Set("open_id", userClaims.OpenId)
		c.Set("stu_num", userClaims.StuNum)
		c.Set("wsConn", conn)
		c.Next()
	}
}

func WsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Sec-WebSocket-Protocol")
		userClaims, err := utils.AnalyseMassesToken(auth)
		if err != nil {
			c.Abort()
			types.ResponseErrorWithMsg(c, http.StatusUnauthorized, "Unauthorized")
			return
		}
		c.Set("open_id", userClaims.OpenId)
		c.Set("nick_name", userClaims.NickName)
		c.Next()
	}
}
