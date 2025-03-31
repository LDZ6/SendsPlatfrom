package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"platform/app/gateway/types"
	"platform/app/user/database/cache"
	"platform/utils"
)

func AuthUserCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("token")
		userClaims, err := utils.AnalyseUserToken(auth)
		if err != nil {
			c.Abort()
			types.ResponseErrorWithMsg(c, http.StatusUnauthorized, "Unauthorized")
			return
		}
		c.Set("open_id", userClaims.OpenId)
		c.Set("stu_num", userClaims.StuNum)
		c.Next()
	}
}

func AuthAdminCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("token")
		adminClaims, err := utils.AnalyseAdminToken(auth)
		if err != nil {
			c.Abort()
			types.ResponseErrorWithMsg(c, http.StatusUnauthorized, "Unauthorized")
			return
		}
		c.Set("organization", adminClaims.Organization)
		c.Next()
	}
}

func AuthMassesCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("token")
		massesClaims, err := utils.AnalyseMassesToken(auth)
		if err != nil {
			c.Abort()
			types.ResponseErrorWithMsg(c, http.StatusUnauthorized, "Unauthorized")
			return
		}
		err = utils.ParseToken(auth)
		if err != nil {
			c.Abort()
			types.ResponseError(c, types.AuthenticationTimeout)
			err = cache.NewRDBCache(c).Rdb.Unlink(massesClaims.OpenId).Err()
			if err != nil {
				types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
				return
			}
			return
		}
		c.Set("open_id", massesClaims.OpenId)
		c.Set("nick_name", massesClaims.NickName)
		c.Next()
	}
}
