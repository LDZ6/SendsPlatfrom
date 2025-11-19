package utils

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// ErrorCode 错误码类型
type ErrorCode int

const (
	// 通用错误码
	ErrCodeSuccess ErrorCode = 0
	ErrCodeUnknown ErrorCode = 1000

	// 参数错误
	ErrCodeInvalidParam  ErrorCode = 1001
	ErrCodeMissingParam  ErrorCode = 1002
	ErrCodeInvalidFormat ErrorCode = 1003

	// 认证错误
	ErrCodeUnauthorized ErrorCode = 2001
	ErrCodeForbidden    ErrorCode = 2002
	ErrCodeTokenExpired ErrorCode = 2003
	ErrCodeInvalidToken ErrorCode = 2004

	// 业务错误
	ErrCodeUserNotFound       ErrorCode = 3001
	ErrCodeUserExists         ErrorCode = 3002
	ErrCodeInvalidCredentials ErrorCode = 3003
	ErrCodeOperationFailed    ErrorCode = 3004

	// 系统错误
	ErrCodeDatabase           ErrorCode = 4001
	ErrCodeRedis              ErrorCode = 4002
	ErrCodeNetwork            ErrorCode = 4003
	ErrCodeTimeout            ErrorCode = 4004
	ErrCodeServiceUnavailable ErrorCode = 4005
	ErrCodeCacheMiss          ErrorCode = 4006
	ErrCodeCacheWrite         ErrorCode = 4007
	ErrCodeConfigError        ErrorCode = 4008
	ErrCodeExternalAPI        ErrorCode = 4009

	// 博饼游戏错误
	ErrCodeBoBingCheat       ErrorCode = 5001
	ErrCodeBoBingBlacklist   ErrorCode = 5002
	ErrCodeBoBingNoChance    ErrorCode = 5003
	ErrCodeBoBingInvalidFlag ErrorCode = 5004
	ErrCodeBoBingDecrypt     ErrorCode = 5005

	// 用户服务错误
	ErrCodeWXLoginFailed  ErrorCode = 6001
	ErrCodeStuNumNotFound ErrorCode = 6002
	ErrCodeWXInfoFailed   ErrorCode = 6003
	ErrCodeJWCError       ErrorCode = 6004
)

// AppError 应用错误结构
type AppError struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	RequestID string    `json:"requestId,omitempty"`
	File      string    `json:"file,omitempty"`
	Line      int       `json:"line,omitempty"`
	Err       error     `json:"-"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现错误包装
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError 创建应用错误
func NewAppError(code ErrorCode, message string, err error) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Code:    code,
		Message: message,
		Details: getErrorDetails(err),
		File:    file,
		Line:    line,
		Err:     err,
	}
}

// WrapError 包装现有错误
func WrapError(err error, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}

	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Code:    code,
		Message: message,
		Details: getErrorDetails(err),
		File:    file,
		Line:    line,
		Err:     err,
	}
}

// getErrorDetails 获取错误详情
func getErrorDetails(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// HTTPStatus 获取HTTP状态码
func (e *AppError) HTTPStatus() int {
	switch {
	case e.Code >= 1001 && e.Code < 2000:
		return http.StatusBadRequest
	case e.Code >= 2001 && e.Code < 3000:
		return http.StatusUnauthorized
	case e.Code >= 3001 && e.Code < 4000:
		return http.StatusUnprocessableEntity
	case e.Code >= 4001 && e.Code < 5000:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// 预定义错误
var (
	ErrInvalidParam       = NewAppError(ErrCodeInvalidParam, "参数无效", nil)
	ErrMissingParam       = NewAppError(ErrCodeMissingParam, "缺少必要参数", nil)
	ErrUnauthorized       = NewAppError(ErrCodeUnauthorized, "未授权访问", nil)
	ErrForbidden          = NewAppError(ErrCodeForbidden, "禁止访问", nil)
	ErrTokenExpired       = NewAppError(ErrCodeTokenExpired, "令牌已过期", nil)
	ErrInvalidToken       = NewAppError(ErrCodeInvalidToken, "无效令牌", nil)
	ErrUserNotFound       = NewAppError(ErrCodeUserNotFound, "用户不存在", nil)
	ErrUserExists         = NewAppError(ErrCodeUserExists, "用户已存在", nil)
	ErrDatabase           = NewAppError(ErrCodeDatabase, "数据库操作失败", nil)
	ErrRedis              = NewAppError(ErrCodeRedis, "缓存操作失败", nil)
	ErrNetwork            = NewAppError(ErrCodeNetwork, "网络错误", nil)
	ErrTimeout            = NewAppError(ErrCodeTimeout, "请求超时", nil)
	ErrServiceUnavailable = NewAppError(ErrCodeServiceUnavailable, "服务不可用", nil)
	ErrCacheMiss          = NewAppError(ErrCodeCacheMiss, "缓存未命中", nil)
	ErrCacheWrite         = NewAppError(ErrCodeCacheWrite, "缓存写入失败", nil)
	ErrConfigError        = NewAppError(ErrCodeConfigError, "配置错误", nil)
	ErrExternalAPI        = NewAppError(ErrCodeExternalAPI, "外部API调用失败", nil)

	// 博饼游戏错误
	ErrBoBingCheat       = NewAppError(ErrCodeBoBingCheat, "检测到作弊行为，已被封号", nil)
	ErrBoBingBlacklist   = NewAppError(ErrCodeBoBingBlacklist, "用户已被加入黑名单", nil)
	ErrBoBingNoChance    = NewAppError(ErrCodeBoBingNoChance, "今日增加投掷次数已达上限", nil)
	ErrBoBingInvalidFlag = NewAppError(ErrCodeBoBingInvalidFlag, "无效的投掷标志", nil)
	ErrBoBingDecrypt     = NewAppError(ErrCodeBoBingDecrypt, "解密失败", nil)

	// 用户服务错误
	ErrWXLoginFailed  = NewAppError(ErrCodeWXLoginFailed, "微信登录失败", nil)
	ErrStuNumNotFound = NewAppError(ErrCodeStuNumNotFound, "找不到学号，请绑定桑梓微助手", nil)
	ErrWXInfoFailed   = NewAppError(ErrCodeWXInfoFailed, "获取微信用户信息失败", nil)
	ErrJWCError       = NewAppError(ErrCodeJWCError, "教务系统操作失败", nil)
)

// IsAppError 检查是否为应用错误
func IsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}

	return nil, false
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) ErrorCode {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Code
	}
	return ErrCodeUnknown
}

// GetErrorMessage 获取错误消息
func GetErrorMessage(err error) string {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Message
	}
	return err.Error()
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Success   bool      `json:"success"`
	ErrorCode ErrorCode `json:"errorCode"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	RequestID string    `json:"requestId,omitempty"`
	Timestamp int64     `json:"timestamp"`
}

// ToResponse 转换为响应结构
func (e *AppError) ToResponse() *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		ErrorCode: e.Code,
		Message:   e.Message,
		Details:   e.Details,
		RequestID: e.RequestID,
		Timestamp: 0, // 将在处理时设置
	}
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationErrors 验证错误集合
type ValidationErrors []ValidationError

// Error 实现error接口
func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

// Add 添加验证错误
func (ve *ValidationErrors) Add(field, message, value string) {
	*ve = append(*ve, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors 检查是否有错误
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// ToAppError 转换为应用错误
func (ve ValidationErrors) ToAppError() *AppError {
	return NewAppError(ErrCodeInvalidParam, "参数验证失败", ve)
}

// RecoverPanic 恢复panic并转换为错误
func RecoverPanic() error {
	if r := recover(); r != nil {
		_, file, line, _ := runtime.Caller(2)
		return NewAppError(ErrCodeUnknown, "系统内部错误",
			fmt.Errorf("panic recovered: %v at %s:%d", r, file, line))
	}
	return nil
}

// SafeExecute 安全执行函数，捕获panic
func SafeExecute(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			_, file, line, _ := runtime.Caller(3)
			err = NewAppError(ErrCodeUnknown, "执行失败",
				fmt.Errorf("panic recovered: %v at %s:%d", r, file, line))
		}
	}()

	return fn()
}

// WrapDatabaseError 包装数据库错误
func WrapDatabaseError(err error, operation string) *AppError {
	if err == nil {
		return nil
	}
	return WrapError(err, ErrCodeDatabase, fmt.Sprintf("数据库操作失败: %s", operation))
}

// WrapRedisError 包装Redis错误
func WrapRedisError(err error, operation string) *AppError {
	if err == nil {
		return nil
	}
	return WrapError(err, ErrCodeRedis, fmt.Sprintf("缓存操作失败: %s", operation))
}

// WrapNetworkError 包装网络错误
func WrapNetworkError(err error, operation string) *AppError {
	if err == nil {
		return nil
	}
	return WrapError(err, ErrCodeNetwork, fmt.Sprintf("网络操作失败: %s", operation))
}

// WrapExternalAPIError 包装外部API错误
func WrapExternalAPIError(err error, api string) *AppError {
	if err == nil {
		return nil
	}
	return WrapError(err, ErrCodeExternalAPI, fmt.Sprintf("外部API调用失败: %s", api))
}

// WrapValidationError 包装验证错误
func WrapValidationError(err error, field string) *AppError {
	if err == nil {
		return nil
	}
	return WrapError(err, ErrCodeInvalidParam, fmt.Sprintf("参数验证失败: %s", field))
}

// LogAndReturn 记录错误并返回
func LogAndReturn(err error, logger interface{}) error {
	// 这里可以集成具体的日志库
	// 暂时使用fmt打印
	if err != nil {
		fmt.Printf("Error occurred: %v\n", err)
	}
	return err
}
