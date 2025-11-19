package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// ValidationRule 验证规则
type ValidationRule struct {
	Field    string
	Required bool
	Min      interface{}
	Max      interface{}
	Pattern  string
	Custom   func(interface{}) error
	Message  string
}

// ConfigValidator 配置验证器
type ConfigValidator struct {
	rules map[string][]ValidationRule
}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		rules: make(map[string][]ValidationRule),
	}
}

// AddRule 添加验证规则
func (cv *ConfigValidator) AddRule(configType string, rule ValidationRule) {
	cv.rules[configType] = append(cv.rules[configType], rule)
}

// Validate 验证配置
func (cv *ConfigValidator) Validate(configType string, config interface{}) error {
	rules, exists := cv.rules[configType]
	if !exists {
		return fmt.Errorf("no validation rules found for config type: %s", configType)
	}

	configValue := reflect.ValueOf(config)
	if configValue.Kind() == reflect.Ptr {
		configValue = configValue.Elem()
	}

	if configValue.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a struct")
	}

	for _, rule := range rules {
		if err := cv.validateRule(configValue, rule); err != nil {
			return err
		}
	}

	return nil
}

// validateRule 验证单个规则
func (cv *ConfigValidator) validateRule(configValue reflect.Value, rule ValidationRule) error {
	field := configValue.FieldByName(rule.Field)
	if !field.IsValid() {
		return fmt.Errorf("field %s not found in config", rule.Field)
	}

	// 检查必填字段
	if rule.Required && field.IsZero() {
		return fmt.Errorf("field %s is required", rule.Field)
	}

	// 如果字段为空且不是必填的，跳过其他验证
	if field.IsZero() && !rule.Required {
		return nil
	}

	// 验证最小值
	if rule.Min != nil {
		if err := cv.validateMin(field, rule.Min, rule.Field); err != nil {
			return err
		}
	}

	// 验证最大值
	if rule.Max != nil {
		if err := cv.validateMax(field, rule.Max, rule.Field); err != nil {
			return err
		}
	}

	// 验证正则表达式
	if rule.Pattern != "" {
		if err := cv.validatePattern(field, rule.Pattern, rule.Field); err != nil {
			return err
		}
	}

	// 自定义验证
	if rule.Custom != nil {
		if err := rule.Custom(field.Interface()); err != nil {
			if rule.Message != "" {
				return fmt.Errorf("%s: %w", rule.Message, err)
			}
			return err
		}
	}

	return nil
}

// validateMin 验证最小值
func (cv *ConfigValidator) validateMin(field reflect.Value, min interface{}, fieldName string) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		minVal, ok := min.(int64)
		if !ok {
			return fmt.Errorf("invalid min value type for field %s", fieldName)
		}
		if field.Int() < minVal {
			return fmt.Errorf("field %s must be >= %d", fieldName, minVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		minVal, ok := min.(uint64)
		if !ok {
			return fmt.Errorf("invalid min value type for field %s", fieldName)
		}
		if field.Uint() < minVal {
			return fmt.Errorf("field %s must be >= %d", fieldName, minVal)
		}
	case reflect.Float32, reflect.Float64:
		minVal, ok := min.(float64)
		if !ok {
			return fmt.Errorf("invalid min value type for field %s", fieldName)
		}
		if field.Float() < minVal {
			return fmt.Errorf("field %s must be >= %f", fieldName, minVal)
		}
	case reflect.String:
		minVal, ok := min.(int)
		if !ok {
			return fmt.Errorf("invalid min value type for field %s", fieldName)
		}
		if len(field.String()) < minVal {
			return fmt.Errorf("field %s must be at least %d characters long", fieldName, minVal)
		}
	}

	return nil
}

// validateMax 验证最大值
func (cv *ConfigValidator) validateMax(field reflect.Value, max interface{}, fieldName string) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		maxVal, ok := max.(int64)
		if !ok {
			return fmt.Errorf("invalid max value type for field %s", fieldName)
		}
		if field.Int() > maxVal {
			return fmt.Errorf("field %s must be <= %d", fieldName, maxVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		maxVal, ok := max.(uint64)
		if !ok {
			return fmt.Errorf("invalid max value type for field %s", fieldName)
		}
		if field.Uint() > maxVal {
			return fmt.Errorf("field %s must be <= %d", fieldName, maxVal)
		}
	case reflect.Float32, reflect.Float64:
		maxVal, ok := max.(float64)
		if !ok {
			return fmt.Errorf("invalid max value type for field %s", fieldName)
		}
		if field.Float() > maxVal {
			return fmt.Errorf("field %s must be <= %f", fieldName, maxVal)
		}
	case reflect.String:
		maxVal, ok := max.(int)
		if !ok {
			return fmt.Errorf("invalid max value type for field %s", fieldName)
		}
		if len(field.String()) > maxVal {
			return fmt.Errorf("field %s must be at most %d characters long", fieldName, maxVal)
		}
	}

	return nil
}

// validatePattern 验证正则表达式
func (cv *ConfigValidator) validatePattern(field reflect.Value, pattern string, fieldName string) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("pattern validation only supports string fields")
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern for field %s: %w", fieldName, err)
	}

	if !regex.MatchString(field.String()) {
		return fmt.Errorf("field %s does not match pattern %s", fieldName, pattern)
	}

	return nil
}

// Built-in validation functions

// ValidatePort 验证端口号
func ValidatePort(port interface{}) error {
	p, ok := port.(int)
	if !ok {
		return fmt.Errorf("port must be an integer")
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// ValidateURL 验证URL
func ValidateURL(url interface{}) error {
	u, ok := url.(string)
	if !ok {
		return fmt.Errorf("URL must be a string")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	return nil
}

// ValidateEmail 验证邮箱
func ValidateEmail(email interface{}) error {
	e, ok := email.(string)
	if !ok {
		return fmt.Errorf("email must be a string")
	}
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	if !regex.MatchString(e) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidateDuration 验证持续时间
func ValidateDuration(duration interface{}) error {
	d, ok := duration.(string)
	if !ok {
		return fmt.Errorf("duration must be a string")
	}
	_, err := time.ParseDuration(d)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}
	return nil
}

// ValidatePositiveInt 验证正整数
func ValidatePositiveInt(value interface{}) error {
	v, ok := value.(int)
	if !ok {
		return fmt.Errorf("value must be an integer")
	}
	if v <= 0 {
		return fmt.Errorf("value must be positive")
	}
	return nil
}

// ValidateNonEmptyString 验证非空字符串
func ValidateNonEmptyString(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}

// ValidateRange 验证范围
func ValidateRange(min, max interface{}) func(interface{}) error {
	return func(value interface{}) error {
		v, ok := value.(int)
		if !ok {
			return fmt.Errorf("value must be an integer")
		}

		minVal, ok := min.(int)
		if !ok {
			return fmt.Errorf("invalid min value type")
		}

		maxVal, ok := max.(int)
		if !ok {
			return fmt.Errorf("invalid max value type")
		}

		if v < minVal || v > maxVal {
			return fmt.Errorf("value must be between %d and %d", minVal, maxVal)
		}

		return nil
	}
}

// ValidateOneOf 验证值是否在指定选项中
func ValidateOneOf(options ...interface{}) func(interface{}) error {
	return func(value interface{}) error {
		for _, option := range options {
			if reflect.DeepEqual(value, option) {
				return nil
			}
		}
		return fmt.Errorf("value must be one of %v", options)
	}
}

// ValidateConfigFile 验证配置文件
func ValidateConfigFile(filePath string, configType string, config interface{}) error {
	validator := NewConfigValidator()

	// 添加默认验证规则
	validator.addDefaultRules(configType)

	return validator.Validate(configType, config)
}

// addDefaultRules 添加默认验证规则
func (cv *ConfigValidator) addDefaultRules(configType string) {
	switch configType {
	case "database":
		cv.AddRule("database", ValidationRule{
			Field:    "Host",
			Required: true,
			Custom:   ValidateNonEmptyString,
		})
		cv.AddRule("database", ValidationRule{
			Field:    "Port",
			Required: true,
			Custom:   ValidatePort,
		})
		cv.AddRule("database", ValidationRule{
			Field:    "Username",
			Required: true,
			Custom:   ValidateNonEmptyString,
		})
		cv.AddRule("database", ValidationRule{
			Field:    "Password",
			Required: true,
			Custom:   ValidateNonEmptyString,
		})
		cv.AddRule("database", ValidationRule{
			Field:    "Database",
			Required: true,
			Custom:   ValidateNonEmptyString,
		})
	case "redis":
		cv.AddRule("redis", ValidationRule{
			Field:    "Host",
			Required: true,
			Custom:   ValidateNonEmptyString,
		})
		cv.AddRule("redis", ValidationRule{
			Field:    "Port",
			Required: true,
			Custom:   ValidatePort,
		})
		cv.AddRule("redis", ValidationRule{
			Field:    "Password",
			Required: false,
			Custom:   ValidateNonEmptyString,
		})
	case "server":
		cv.AddRule("server", ValidationRule{
			Field:    "Port",
			Required: true,
			Custom:   ValidatePort,
		})
		cv.AddRule("server", ValidationRule{
			Field:    "Host",
			Required: true,
			Custom:   ValidateNonEmptyString,
		})
		cv.AddRule("server", ValidationRule{
			Field:    "ReadTimeout",
			Required: true,
			Custom:   ValidateDuration,
		})
		cv.AddRule("server", ValidationRule{
			Field:    "WriteTimeout",
			Required: true,
			Custom:   ValidateDuration,
		})
	}
}
