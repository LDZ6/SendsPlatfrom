package middlewares

import (
	"platform/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandler 统一错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err interface{}) {
		logger := utils.GetLogger()

		// 记录错误日志
		logger.WithFields(logrus.Fields{
			"error":  err,
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
			"ip":     c.ClientIP(),
		}).Error("Panic recovered")

		// 检查是否为应用错误
		if appErr, ok := err.(*utils.AppError); ok {
			RespondWithAppError(c, appErr)
		} else {
			// 通用错误响应
			RespondWithInternalError(c, "Internal Server Error")
		}
	})
}

// ErrorResponseMiddleware 错误响应中间件
func ErrorResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// 检查是否为应用错误
			if appErr, ok := err.Err.(*utils.AppError); ok {
				RespondWithAppError(c, appErr)
				return
			}

			// 处理验证错误
			if validationErr, ok := err.Err.(utils.ValidationErrors); ok {
				appErr := validationErr.ToAppError()
				RespondWithAppError(c, appErr)
				return
			}

			// 默认错误处理
			RespondWithInternalError(c, "Internal Server Error")
		}
	}
}
