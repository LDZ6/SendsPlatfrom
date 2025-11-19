package common

import (
	"errors"
	"testing"

	"platform/utils"
)

func TestAppError(t *testing.T) {
	tests := []struct {
		name     string
		code     utils.ErrorCode
		message  string
		err      error
		expected string
	}{
		{
			name:     "basic error",
			code:     utils.ErrCodeInvalidParam,
			message:  "参数无效",
			err:      nil,
			expected: "[1001] 参数无效",
		},
		{
			name:     "error with details",
			code:     utils.ErrCodeDatabase,
			message:  "数据库操作失败",
			err:      errors.New("connection timeout"),
			expected: "[4001] 数据库操作失败: connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := utils.NewAppError(tt.code, tt.message, tt.err)
			if appErr.Error() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, appErr.Error())
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     utils.ErrorCode
		message  string
		expected bool
	}{
		{
			name:     "wrap nil error",
			err:      nil,
			code:     utils.ErrCodeDatabase,
			message:  "数据库操作失败",
			expected: false,
		},
		{
			name:     "wrap existing error",
			err:      errors.New("connection failed"),
			code:     utils.ErrCodeDatabase,
			message:  "数据库操作失败",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := utils.WrapError(tt.err, tt.code, tt.message)
			if (appErr != nil) != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, appErr != nil)
			}
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		code     utils.ErrorCode
		expected int
	}{
		{
			name:     "invalid param",
			code:     utils.ErrCodeInvalidParam,
			expected: 400,
		},
		{
			name:     "unauthorized",
			code:     utils.ErrCodeUnauthorized,
			expected: 401,
		},
		{
			name:     "user not found",
			code:     utils.ErrCodeUserNotFound,
			expected: 422,
		},
		{
			name:     "database error",
			code:     utils.ErrCodeDatabase,
			expected: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := utils.NewAppError(tt.code, "test", nil)
			if appErr.HTTPStatus() != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, appErr.HTTPStatus())
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	var ve utils.ValidationErrors

	// 添加验证错误
	ve.Add("username", "不能为空", "")
	ve.Add("email", "格式不正确", "invalid-email")

	// 测试错误消息
	expected := "username: 不能为空; email: 格式不正确"
	if ve.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, ve.Error())
	}

	// 测试是否有错误
	if !ve.HasErrors() {
		t.Error("Expected validation errors")
	}

	// 测试转换为AppError
	appErr := ve.ToAppError()
	if appErr.Code != utils.ErrCodeInvalidParam {
		t.Errorf("Expected %d, got %d", utils.ErrCodeInvalidParam, appErr.Code)
	}
}

func TestSafeExecute(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() error
		expected bool
	}{
		{
			name: "normal execution",
			fn: func() error {
				return nil
			},
			expected: false,
		},
		{
			name: "panic recovery",
			fn: func() error {
				panic("test panic")
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.SafeExecute(tt.fn)
			hasError := (err != nil)
			if hasError != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, hasError)
			}
		})
	}
}

func TestWrapDatabaseError(t *testing.T) {
	err := errors.New("connection timeout")
	appErr := utils.WrapDatabaseError(err, "查询用户")

	if appErr.Code != utils.ErrCodeDatabase {
		t.Errorf("Expected %d, got %d", utils.ErrCodeDatabase, appErr.Code)
	}

	if appErr.Message != "数据库操作失败: 查询用户" {
		t.Errorf("Expected '数据库操作失败: 查询用户', got %s", appErr.Message)
	}
}

func TestWrapRedisError(t *testing.T) {
	err := errors.New("connection failed")
	appErr := utils.WrapRedisError(err, "设置缓存")

	if appErr.Code != utils.ErrCodeRedis {
		t.Errorf("Expected %d, got %d", utils.ErrCodeRedis, appErr.Code)
	}

	if appErr.Message != "缓存操作失败: 设置缓存" {
		t.Errorf("Expected '缓存操作失败: 设置缓存', got %s", appErr.Message)
	}
}

func TestWrapNetworkError(t *testing.T) {
	err := errors.New("timeout")
	appErr := utils.WrapNetworkError(err, "HTTP请求")

	if appErr.Code != utils.ErrCodeNetwork {
		t.Errorf("Expected %d, got %d", utils.ErrCodeNetwork, appErr.Code)
	}

	if appErr.Message != "网络操作失败: HTTP请求" {
		t.Errorf("Expected '网络操作失败: HTTP请求', got %s", appErr.Message)
	}
}

func TestWrapExternalAPIError(t *testing.T) {
	err := errors.New("API unavailable")
	appErr := utils.WrapExternalAPIError(err, "微信登录")

	if appErr.Code != utils.ErrCodeExternalAPI {
		t.Errorf("Expected %d, got %d", utils.ErrCodeExternalAPI, appErr.Code)
	}

	if appErr.Message != "外部API调用失败: 微信登录" {
		t.Errorf("Expected '外部API调用失败: 微信登录', got %s", appErr.Message)
	}
}

func TestWrapValidationError(t *testing.T) {
	err := errors.New("invalid format")
	appErr := utils.WrapValidationError(err, "email")

	if appErr.Code != utils.ErrCodeInvalidParam {
		t.Errorf("Expected %d, got %d", utils.ErrCodeInvalidParam, appErr.Code)
	}

	if appErr.Message != "参数验证失败: email" {
		t.Errorf("Expected '参数验证失败: email', got %s", appErr.Message)
	}
}

func TestIsAppError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "AppError",
			err:      utils.NewAppError(utils.ErrCodeInvalidParam, "test", nil),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, isAppError := utils.IsAppError(tt.err)
			if isAppError != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, isAppError)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected utils.ErrorCode
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: utils.ErrCodeUnknown,
		},
		{
			name:     "AppError",
			err:      utils.NewAppError(utils.ErrCodeInvalidParam, "test", nil),
			expected: utils.ErrCodeInvalidParam,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: utils.ErrCodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := utils.GetErrorCode(tt.err)
			if code != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, code)
			}
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "AppError",
			err:      utils.NewAppError(utils.ErrCodeInvalidParam, "test message", nil),
			expected: "test message",
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: "regular error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := utils.GetErrorMessage(tt.err)
			if message != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, message)
			}
		})
	}
}

func TestToResponse(t *testing.T) {
	appErr := utils.NewAppError(utils.ErrCodeInvalidParam, "参数无效", nil)
	appErr.RequestID = "test-request-id"

	response := appErr.ToResponse()

	if response.Success != false {
		t.Error("Expected Success to be false")
	}

	if response.ErrorCode != utils.ErrCodeInvalidParam {
		t.Errorf("Expected %d, got %d", utils.ErrCodeInvalidParam, response.ErrorCode)
	}

	if response.Message != "参数无效" {
		t.Errorf("Expected '参数无效', got %s", response.Message)
	}

	if response.RequestID != "test-request-id" {
		t.Errorf("Expected 'test-request-id', got %s", response.RequestID)
	}
}
