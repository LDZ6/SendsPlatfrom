package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// 注册自定义验证器
	registerCustomValidators()
}

// ValidateRequest 通用请求验证中间件
func ValidateRequest[T any]() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T

		// 绑定JSON数据
		if err := c.ShouldBindJSON(&req); err != nil {
			RespondWithValidationError(c, "Invalid request format", err.Error())
			c.Abort()
			return
		}

		// 执行验证
		if err := validate.Struct(req); err != nil {
			validationErrors := parseValidationErrors(err)
			RespondWithValidationError(c, "Validation failed", validationErrors)
			c.Abort()
			return
		}

		// 将验证通过的数据存储到上下文中
		c.Set("validated_request", req)
		c.Next()
	}
}

// ValidateQuery 查询参数验证中间件
func ValidateQuery[T any]() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T

		// 绑定查询参数
		if err := c.ShouldBindQuery(&req); err != nil {
			RespondWithValidationError(c, "Invalid query parameters", err.Error())
			c.Abort()
			return
		}

		// 执行验证
		if err := validate.Struct(req); err != nil {
			validationErrors := parseValidationErrors(err)
			RespondWithValidationError(c, "Query validation failed", validationErrors)
			c.Abort()
			return
		}

		c.Set("validated_query", req)
		c.Next()
	}
}

// parseValidationErrors 解析验证错误
func parseValidationErrors(err error) []map[string]interface{} {
	var errors []map[string]interface{}

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorDetail := map[string]interface{}{
				"field":   fieldError.Field(),
				"tag":     fieldError.Tag(),
				"value":   fieldError.Value(),
				"message": getValidationMessage(fieldError),
			}
			errors = append(errors, errorDetail)
		}
	}

	return errors
}

// getValidationMessage 获取验证错误消息
func getValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "min":
		return fe.Field() + " must be at least " + fe.Param() + " characters"
	case "max":
		return fe.Field() + " must be at most " + fe.Param() + " characters"
	case "len":
		return fe.Field() + " must be exactly " + fe.Param() + " characters"
	case "email":
		return fe.Field() + " must be a valid email address"
	case "oneof":
		return fe.Field() + " must be one of: " + fe.Param()
	case "numeric":
		return fe.Field() + " must be numeric"
	case "alphanum":
		return fe.Field() + " must contain only alphanumeric characters"
	default:
		return fe.Field() + " is invalid"
	}
}

// registerCustomValidators 注册自定义验证器
func registerCustomValidators() {
	// 学号验证器
	validate.RegisterValidation("stu_num", func(fl validator.FieldLevel) bool {
		stuNum := fl.Field().String()
		return len(stuNum) == 10 && isNumeric(stuNum)
	})

	// OpenID验证器
	validate.RegisterValidation("openid", func(fl validator.FieldLevel) bool {
		openId := fl.Field().String()
		return len(openId) >= 20 && len(openId) <= 50
	})

	// 昵称验证器
	validate.RegisterValidation("nickname", func(fl validator.FieldLevel) bool {
		nickName := fl.Field().String()
		return len(nickName) >= 1 && len(nickName) <= 20 && !containsSpecialChars(nickName)
	})
}

// isNumeric 检查字符串是否为数字
func isNumeric(s string) bool {
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// containsSpecialChars 检查是否包含特殊字符
func containsSpecialChars(s string) bool {
	specialChars := []string{"<", ">", "&", "\"", "'", "\\", "/", "=", "(", ")", "[", "]", "{", "}", "|", "`", "~", "!", "@", "#", "$", "%", "^", "*", "+", "?", ":", ";", ",", ".", " "}
	for _, char := range specialChars {
		if strings.Contains(s, char) {
			return true
		}
	}
	return false
}

// GetValidatedRequest 获取验证后的请求数据
func GetValidatedRequest[T any](c *gin.Context) (T, bool) {
	var zero T
	if req, exists := c.Get("validated_request"); exists {
		if typedReq, ok := req.(T); ok {
			return typedReq, true
		}
	}
	return zero, false
}

// GetValidatedQuery 获取验证后的查询参数
func GetValidatedQuery[T any](c *gin.Context) (T, bool) {
	var zero T
	if query, exists := c.Get("validated_query"); exists {
		if typedQuery, ok := query.(T); ok {
			return typedQuery, true
		}
	}
	return zero, false
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidateVar 验证单个变量
func ValidateVar(field interface{}, tag string) error {
	return validate.Var(field, tag)
}
