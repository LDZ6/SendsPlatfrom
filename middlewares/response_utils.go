package middlewares

import (
	"net/http"
	"time"

	"platform/utils"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 统一错误响应结构
type ErrorResponse struct {
	Success   bool        `json:"success"`
	ErrorCode int         `json:"errorCode"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// SuccessResponse 统一成功响应结构
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// RespondWithError 统一错误响应
func RespondWithError(c *gin.Context, statusCode int, errorCode int, message string, details interface{}) {
	c.JSON(statusCode, ErrorResponse{
		Success:   false,
		ErrorCode: errorCode,
		Message:   message,
		Details:   details,
		Timestamp: time.Now().Unix(),
	})
}

// RespondWithSuccess 统一成功响应
func RespondWithSuccess(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().Unix(),
	})
}

// RespondWithAppError 响应应用错误
func RespondWithAppError(c *gin.Context, appErr *utils.AppError) {
	c.JSON(appErr.HTTPStatus(), appErr.ToResponse())
}

// RespondWithValidationError 响应验证错误
func RespondWithValidationError(c *gin.Context, message string, details interface{}) {
	RespondWithError(c, http.StatusBadRequest, int(utils.ErrCodeInvalidParam), message, details)
}

// RespondWithUnauthorized 响应未授权错误
func RespondWithUnauthorized(c *gin.Context, message string) {
	RespondWithError(c, http.StatusUnauthorized, int(utils.ErrCodeUnauthorized), message, nil)
}

// RespondWithForbidden 响应禁止访问错误
func RespondWithForbidden(c *gin.Context, message string) {
	RespondWithError(c, http.StatusForbidden, int(utils.ErrCodeForbidden), message, nil)
}

// RespondWithInternalError 响应内部服务器错误
func RespondWithInternalError(c *gin.Context, message string) {
	RespondWithError(c, http.StatusInternalServerError, int(utils.ErrCodeUnknown), message, nil)
}

// RespondWithTooManyRequests 响应请求过多错误
func RespondWithTooManyRequests(c *gin.Context, message string) {
	RespondWithError(c, http.StatusTooManyRequests, 429, message, nil)
}

// RespondWithMethodNotAllowed 响应方法不允许错误
func RespondWithMethodNotAllowed(c *gin.Context, message string) {
	RespondWithError(c, http.StatusMethodNotAllowed, 405, message, nil)
}

// RespondWithUnsupportedMediaType 响应不支持的媒体类型错误
func RespondWithUnsupportedMediaType(c *gin.Context, message string) {
	RespondWithError(c, http.StatusUnsupportedMediaType, 415, message, nil)
}
