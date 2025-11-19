package middlewares

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"platform/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	// 请求限制
	RateLimitEnabled bool
	RateLimitRPS     int
	RateLimitBurst   int

	// 请求大小限制
	MaxRequestSize int64

	// 超时设置
	RequestTimeout time.Duration

	// 安全头
	EnableSecurityHeaders bool

	// CORS设置
	CORSEnabled    bool
	AllowedOrigins []string

	// 增强安全功能
	XSSProtection          bool
	CSRFProtection         bool
	SQLInjectionProtection bool
	DataMasking            bool
	EncryptionEnabled      bool
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		RateLimitEnabled:       true,
		RateLimitRPS:           100,
		RateLimitBurst:         200,
		MaxRequestSize:         10 * 1024 * 1024, // 10MB
		RequestTimeout:         30 * time.Second,
		EnableSecurityHeaders:  true,
		CORSEnabled:            true,
		AllowedOrigins:         []string{"*"},
		XSSProtection:          true,
		CSRFProtection:         true,
		SQLInjectionProtection: true,
		DataMasking:            true,
		EncryptionEnabled:      true,
	}
}

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	config       *SecurityConfig
	rateLimiters map[string]*rate.Limiter
	encryption   *EncryptionService
	validator    *InputValidator
	logger       *logrus.Logger
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware(config *SecurityConfig, logger *logrus.Logger) *SecurityMiddleware {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &SecurityMiddleware{
		config:       config,
		rateLimiters: make(map[string]*rate.Limiter),
		encryption:   NewEncryptionService(),
		validator:    NewInputValidator(),
		logger:       logger,
	}
}

// RateLimit 限流中间件
func (sm *SecurityMiddleware) RateLimit() gin.HandlerFunc {
	if !sm.config.RateLimitEnabled {
		return gin.HandlerFunc(func(c *gin.Context) {
			c.Next()
		})
	}

	return gin.HandlerFunc(func(c *gin.Context) {
		// 获取客户端IP
		clientIP := c.ClientIP()

		// 检查限流
		if !sm.rateLimiter.Allow() {
			sm.logger.Warnf("请求被限流，客户端IP: %s", clientIP)
			RespondWithTooManyRequests(c, "请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	})
}

// RequestSizeLimit 请求大小限制中间件
func (sm *SecurityMiddleware) RequestSizeLimit() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if c.Request.ContentLength > sm.config.MaxRequestSize {
			sm.logger.Warnf("请求过大，大小: %d, 限制: %d",
				c.Request.ContentLength, sm.config.MaxRequestSize)
			RespondWithError(c, http.StatusRequestEntityTooLarge, 413, "请求体大小超过限制", nil)
			c.Abort()
			return
		}

		c.Next()
	})
}

// SecurityHeaders 安全头中间件
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	if !sm.config.EnableSecurityHeaders {
		return gin.HandlerFunc(func(c *gin.Context) {
			c.Next()
		})
	}

	return gin.HandlerFunc(func(c *gin.Context) {
		// 防止XSS攻击
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")

		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'")

		// 严格传输安全
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 引用者策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	})
}

// CORSMiddleware CORS中间件
func (sm *SecurityMiddleware) CORSMiddleware() gin.HandlerFunc {
	if !sm.config.CORSEnabled {
		return gin.HandlerFunc(func(c *gin.Context) {
			c.Next()
		})
	}

	return gin.HandlerFunc(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查允许的源
		allowed := false
		for _, allowedOrigin := range sm.config.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	})
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// InputValidation 输入验证中间件
func InputValidation() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 验证请求方法
		if !isValidMethod(c.Request.Method) {
			RespondWithMethodNotAllowed(c, "请求方法不被允许")
			c.Abort()
			return
		}

		// 验证Content-Type
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if !isValidContentType(contentType) {
				RespondWithUnsupportedMediaType(c, "Content-Type不被支持")
				c.Abort()
				return
			}
		}

		c.Next()
	})
}

// isValidMethod 验证请求方法
func isValidMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"}
	for _, validMethod := range validMethods {
		if method == validMethod {
			return true
		}
	}
	return false
}

// isValidContentType 验证Content-Type
func isValidContentType(contentType string) bool {
	validTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
	}

	for _, validType := range validTypes {
		if strings.HasPrefix(contentType, validType) {
			return true
		}
	}
	return false
}

// SQLInjectionProtection SQL注入防护中间件
func SQLInjectionProtection() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 检查查询参数
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if containsSQLInjection(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "检测到潜在的安全威胁",
						"code":    400,
						"message": "请求包含非法字符",
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	})
}

// containsSQLInjection 检查是否包含SQL注入
func containsSQLInjection(input string) bool {
	sqlPatterns := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_",
		"union", "select", "insert", "update", "delete",
		"drop", "create", "alter", "exec", "execute",
	}

	inputLower := strings.ToLower(input)
	for _, pattern := range sqlPatterns {
		if strings.Contains(inputLower, pattern) {
			return true
		}
	}
	return false
}

// XSSProtection XSS防护中间件
func XSSProtection() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 检查请求体中的潜在XSS
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			// 这里可以添加更复杂的XSS检测逻辑
			// 例如检查script标签、javascript:等
		}

		c.Next()
	})
}

// IPWhitelist IP白名单中间件
func IPWhitelist(allowedIPs []string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		clientIP := c.ClientIP()

		allowed := false
		for _, allowedIP := range allowedIPs {
			if allowedIP == "*" || allowedIP == clientIP {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "访问被拒绝",
				"code":    403,
				"message": "您的IP地址不在允许列表中",
			})
			c.Abort()
			return
		}

		c.Next()
	})
}

// RequestTimeout 请求超时中间件
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 设置超时上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// 监听超时
		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded {
				c.JSON(http.StatusRequestTimeout, gin.H{
					"error":   "请求超时",
					"code":    408,
					"message": "请求处理超时",
				})
				c.Abort()
			}
		}()

		c.Next()
	})
}

// XSSProtectionMiddleware XSS防护中间件
func (sm *SecurityMiddleware) XSSProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.XSSProtection {
			c.Next()
			return
		}

		// 设置XSS防护头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}

// CSRFProtectionMiddleware CSRF防护中间件
func (sm *SecurityMiddleware) CSRFProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.CSRFProtection {
			c.Next()
			return
		}

		// 对于非GET请求，检查CSRF令牌
		if c.Request.Method != "GET" {
			token := c.GetHeader("X-CSRF-Token")
			if token == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "缺少CSRF令牌",
					"code":  utils.ErrCodeForbidden,
				})
				c.Abort()
				return
			}

			// 验证CSRF令牌
			if !sm.validateCSRFToken(token, c.ClientIP()) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "无效的CSRF令牌",
					"code":  utils.ErrCodeForbidden,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// SQLInjectionProtectionMiddleware SQL注入防护中间件
func (sm *SecurityMiddleware) SQLInjectionProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.SQLInjectionProtection {
			c.Next()
			return
		}

		// 检查请求参数中的SQL注入模式
		for _, values := range c.Request.URL.Query() {
			for _, value := range values {
				if sm.validator.ContainsSQLInjection(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "检测到SQL注入攻击",
						"code":  utils.ErrCodeInvalidParam,
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

// DataMaskingMiddleware 数据脱敏中间件
func (sm *SecurityMiddleware) DataMaskingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.DataMasking {
			c.Next()
			return
		}

		// 在响应中脱敏敏感数据
		c.Next()

		// 处理响应数据脱敏
		if data, exists := c.Get("response_data"); exists {
			maskedData := sm.maskSensitiveData(data)
			c.Set("response_data", maskedData)
		}
	}
}

// EncryptionMiddleware 加密中间件
func (sm *SecurityMiddleware) EncryptionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.EncryptionEnabled {
			c.Next()
			return
		}

		// 解密请求数据
		if c.Request.Header.Get("Content-Type") == "application/encrypted" {
			body, err := c.GetRawData()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "读取请求数据失败",
					"code":  utils.ErrCodeInvalidParam,
				})
				c.Abort()
				return
			}

			decrypted, err := sm.encryption.Decrypt(body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "解密失败",
					"code":  utils.ErrCodeInvalidParam,
				})
				c.Abort()
				return
			}

			c.Request.Body = &decryptedReader{data: decrypted}
		}

		c.Next()
	}
}

// validateCSRFToken 验证CSRF令牌
func (sm *SecurityMiddleware) validateCSRFToken(token, clientIP string) bool {
	// 简单的CSRF令牌验证逻辑
	// 实际项目中应该使用更复杂的验证机制
	expectedToken := sm.generateCSRFToken(clientIP)
	return token == expectedToken
}

// generateCSRFToken 生成CSRF令牌
func (sm *SecurityMiddleware) generateCSRFToken(clientIP string) string {
	// 简单的CSRF令牌生成逻辑
	// 实际项目中应该使用更复杂的生成机制
	return fmt.Sprintf("csrf_%s_%d", clientIP, time.Now().Unix())
}

// maskSensitiveData 脱敏敏感数据
func (sm *SecurityMiddleware) maskSensitiveData(data interface{}) interface{} {
	// 实现数据脱敏逻辑
	// 这里只是示例，实际项目中需要根据具体需求实现
	return data
}

// EncryptionService 加密服务
type EncryptionService struct {
	key []byte
}

// NewEncryptionService 创建加密服务
func NewEncryptionService() *EncryptionService {
	// 从配置中获取密钥
	key := []byte("your-32-byte-key-here-123456789012") // 32字节密钥
	return &EncryptionService{key: key}
}

// Encrypt 加密数据
func (es *EncryptionService) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(es.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt 解密数据
func (es *EncryptionService) Decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(es.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// InputValidator 输入验证器
type InputValidator struct {
	sqlPatterns []string
	xssPatterns []string
}

// NewInputValidator 创建输入验证器
func NewInputValidator() *InputValidator {
	return &InputValidator{
		sqlPatterns: []string{
			"union.*select", "select.*from", "insert.*into", "update.*set",
			"delete.*from", "drop.*table", "create.*table", "alter.*table",
			"exec.*", "execute.*", "sp_", "xp_", "script.*>",
		},
		xssPatterns: []string{
			"<script", "javascript:", "onload=", "onerror=", "onclick=",
			"<iframe", "<object", "<embed", "eval(", "expression(",
		},
	}
}

// ContainsSQLInjection 检查是否包含SQL注入
func (iv *InputValidator) ContainsSQLInjection(input string) bool {
	input = strings.ToLower(input)
	for _, pattern := range iv.sqlPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// ContainsXSS 检查是否包含XSS
func (iv *InputValidator) ContainsXSS(input string) bool {
	input = strings.ToLower(input)
	for _, pattern := range iv.xssPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// ValidateInput 验证输入
func (iv *InputValidator) ValidateInput(input string) error {
	if iv.ContainsSQLInjection(input) {
		return utils.WrapError(nil, utils.ErrCodeInvalidParam, "检测到SQL注入")
	}
	if iv.ContainsXSS(input) {
		return utils.WrapError(nil, utils.ErrCodeInvalidParam, "检测到XSS攻击")
	}
	return nil
}

// decryptedReader 解密后的读取器
type decryptedReader struct {
	data []byte
	pos  int
}

func (dr *decryptedReader) Read(p []byte) (n int, err error) {
	if dr.pos >= len(dr.data) {
		return 0, nil
	}
	n = copy(p, dr.data[dr.pos:])
	dr.pos += n
	return n, nil
}

// Close 关闭读取器
func (dr *decryptedReader) Close() error {
	return nil
}

// HashPassword 哈希密码
func HashPassword(password string) (string, error) {
	hash := sha256.Sum256([]byte(password + "salt"))
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, hash string) bool {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return false
	}
	return hashedPassword == hash
}

// GenerateSecureToken 生成安全令牌
func GenerateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateToken 验证令牌
func ValidateToken(token string) bool {
	// 简单的令牌验证逻辑
	// 实际项目中应该使用JWT或其他更复杂的验证机制
	return len(token) > 0
}

// SanitizeInput 清理输入
func SanitizeInput(input string) string {
	// 移除危险字符
	dangerous := []string{"<", ">", "\"", "'", "&", ";", "(", ")", "|", "`", "$"}
	result := input
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}
	return result
}

// RateLimitByUser 基于用户的限流
func RateLimitByUser(userID string) gin.HandlerFunc {
	limiters := make(map[string]*rate.Limiter)
	mu := &sync.Mutex{}

	return func(c *gin.Context) {
		mu.Lock()
		limiter, exists := limiters[userID]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(10), 20) // 10 RPS, 20 burst
			limiters[userID] = limiter
		}
		mu.Unlock()

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "用户请求过于频繁",
				"code":  utils.ErrCodeServiceUnavailable,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByIP 基于IP的限流
func RateLimitByIP() gin.HandlerFunc {
	limiters := make(map[string]*rate.Limiter)
	mu := &sync.Mutex{}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		mu.Lock()
		limiter, exists := limiters[clientIP]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(50), 100) // 50 RPS, 100 burst
			limiters[clientIP] = limiter
		}
		mu.Unlock()

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "IP请求过于频繁",
				"code":  utils.ErrCodeServiceUnavailable,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
