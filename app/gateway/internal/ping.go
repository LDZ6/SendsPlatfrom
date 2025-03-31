package internal

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"platform/app/gateway/rpc"
)

func UserPing(c *gin.Context) {
	resp, err := rpc.UserPingRpc(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": "-1",
			"msg":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": "200",
		"msg":  resp.Message,
	})
}

func BoBingPing(c *gin.Context) {
	resp, err := rpc.BoBingPingRpc(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": "-1",
			"msg":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": "200",
		"msg":  resp.Message,
	})
}

func YearBillPing(c *gin.Context) {
	resp, err := rpc.YearBillPingRpc(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": "-1",
			"msg":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": "200",
		"msg":  resp.Message,
	})
}
