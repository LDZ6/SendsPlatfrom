package common

import (
	"errors"
	"testing"

	"platform/utils"

	"github.com/stretchr/testify/assert"
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
			name:     "基本错误",
			code:     utils.ErrCodeInvalidParam,
			message:  "参数无效",
			err:      nil,
			expected: "[1001] 参数无效",
		},
		{
			name:     "带详情的错误",
			code:     utils.ErrCodeDatabase,
			message:  "数据库操作失败",
			err:      errors.New("connection timeout"),
			expected: "[4001] 数据库操作失败: connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := utils.NewAppError(tt.code, tt.message, tt.err)
			assert.Equal(t, tt.code, appErr.Code)
			assert.Equal(t, tt.message, appErr.Message)
			assert.Equal(t, tt.expected, appErr.Error())
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("原始错误")
	wrappedErr := utils.WrapError(originalErr, utils.ErrCodeNetwork, "网络错误")

	assert.Equal(t, utils.ErrCodeNetwork, wrappedErr.Code)
	assert.Equal(t, "网络错误", wrappedErr.Message)
	assert.Equal(t, originalErr, wrappedErr.Unwrap())
}

func TestIsAppError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "应用错误",
			err:      utils.NewAppError(utils.ErrCodeInvalidParam, "参数无效", nil),
			expected: true,
		},
		{
			name:     "普通错误",
			err:      errors.New("普通错误"),
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, isAppErr := utils.IsAppError(tt.err)
			assert.Equal(t, tt.expected, isAppErr)
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		code           utils.ErrorCode
		expectedStatus int
	}{
		{"参数错误", utils.ErrCodeInvalidParam, 400},
		{"认证错误", utils.ErrCodeUnauthorized, 401},
		{"业务错误", utils.ErrCodeUserNotFound, 422},
		{"系统错误", utils.ErrCodeDatabase, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := utils.NewAppError(tt.code, "测试错误", nil)
			assert.Equal(t, tt.expectedStatus, appErr.HTTPStatus())
		})
	}
}

func TestValidationErrors(t *testing.T) {
	var ve utils.ValidationErrors

	// 添加验证错误
	ve.Add("username", "用户名不能为空", "")
	ve.Add("email", "邮箱格式不正确", "invalid-email")

	assert.True(t, ve.HasErrors())
	assert.Equal(t, 2, len(ve))

	// 测试错误消息
	expectedMsg := "username: 用户名不能为空; email: 邮箱格式不正确"
	assert.Equal(t, expectedMsg, ve.Error())

	// 测试转换为AppError
	appErr := ve.ToAppError()
	assert.Equal(t, utils.ErrCodeInvalidParam, appErr.Code)
	assert.Equal(t, "参数验证失败", appErr.Message)
}

func TestSafeExecute(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() error
		hasError bool
	}{
		{
			name: "正常执行",
			fn: func() error {
				return nil
			},
			hasError: false,
		},
		{
			name: "返回错误",
			fn: func() error {
				return errors.New("执行失败")
			},
			hasError: true,
		},
		{
			name: "panic恢复",
			fn: func() error {
				panic("测试panic")
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.SafeExecute(tt.fn)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
