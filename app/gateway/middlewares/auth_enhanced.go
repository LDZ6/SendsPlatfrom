package middlewares

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthProvider 认证提供者接口
type AuthProvider interface {
	ValidateToken(token string) (*Claims, error)
	GenerateToken(claims *Claims) (string, error)
	RefreshToken(refreshToken string) (string, error)
	RevokeToken(token string) error
}

// Claims JWT声明
type Claims struct {
	UserID      string                 `json:"user_id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	ExpiresAt   time.Time              `json:"exp"`
	IssuedAt    time.Time              `json:"iat"`
	NotBefore   time.Time              `json:"nbf"`
	Issuer      string                 `json:"iss"`
	Audience    string                 `json:"aud"`
	Subject     string                 `json:"sub"`
	Extra       map[string]interface{} `json:"extra"`
}

// Valid 验证声明
func (c *Claims) Valid() error {
	now := time.Now()

	if now.Before(c.NotBefore) {
		return fmt.Errorf("token not yet valid")
	}

	if now.After(c.ExpiresAt) {
		return fmt.Errorf("token expired")
	}

	return nil
}

// JWTProvider JWT认证提供者
type JWTProvider struct {
	secretKey     []byte
	publicKey     *rsa.PublicKey
	privateKey    *rsa.PrivateKey
	signingMethod jwt.SigningMethod
	issuer        string
	audience      string
}

// NewJWTProvider 创建JWT提供者
func NewJWTProvider(secretKey []byte, issuer, audience string) *JWTProvider {
	return &JWTProvider{
		secretKey:     secretKey,
		signingMethod: jwt.SigningMethodHS256,
		issuer:        issuer,
		audience:      audience,
	}
}

// NewRSAJWTProvider 创建RSA JWT提供者
func NewRSAJWTProvider(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, issuer, audience string) *JWTProvider {
	return &JWTProvider{
		privateKey:    privateKey,
		publicKey:     publicKey,
		signingMethod: jwt.SigningMethodRS256,
		issuer:        issuer,
		audience:      audience,
	}
}

// ValidateToken 验证令牌
func (p *JWTProvider) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
			return p.secretKey, nil
		}
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
			return p.publicKey, nil
		}
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GenerateToken 生成令牌
func (p *JWTProvider) GenerateToken(claims *Claims) (string, error) {
	now := time.Now()
	claims.IssuedAt = now
	claims.NotBefore = now
	claims.Issuer = p.issuer
	claims.Audience = p.audience

	token := jwt.NewWithClaims(p.signingMethod, claims)

	var key interface{}
	if p.privateKey != nil {
		key = p.privateKey
	} else {
		key = p.secretKey
	}

	return token.SignedString(key)
}

// RefreshToken 刷新令牌
func (p *JWTProvider) RefreshToken(refreshToken string) (string, error) {
	// 实现刷新令牌逻辑
	claims, err := p.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	// 生成新的访问令牌
	claims.ExpiresAt = time.Now().Add(time.Hour)
	return p.GenerateToken(claims)
}

// RevokeToken 撤销令牌
func (p *JWTProvider) RevokeToken(token string) error {
	// 实现令牌撤销逻辑
	// 这里可以将令牌添加到黑名单
	return nil
}

// OAuth2Provider OAuth2认证提供者
type OAuth2Provider struct {
	clients        map[string]OAuth2Client
	tokenStore     TokenStore
	scopeValidator ScopeValidator
}

// OAuth2Client OAuth2客户端
type OAuth2Client struct {
	ID           string
	Secret       string
	RedirectURIs []string
	Scopes       []string
	GrantTypes   []string
}

// TokenStore 令牌存储接口
type TokenStore interface {
	Store(token string, claims *Claims, expiresAt time.Time) error
	Get(token string) (*Claims, error)
	Delete(token string) error
	Cleanup() error
}

// ScopeValidator 作用域验证器接口
type ScopeValidator interface {
	ValidateScope(scope string, claims *Claims) bool
	ValidateScopes(scopes []string, claims *Claims) bool
}

// NewOAuth2Provider 创建OAuth2提供者
func NewOAuth2Provider(tokenStore TokenStore, scopeValidator ScopeValidator) *OAuth2Provider {
	return &OAuth2Provider{
		clients:        make(map[string]OAuth2Client),
		tokenStore:     tokenStore,
		scopeValidator: scopeValidator,
	}
}

// ValidateToken 验证令牌
func (p *OAuth2Provider) ValidateToken(token string) (*Claims, error) {
	return p.tokenStore.Get(token)
}

// GenerateToken 生成令牌
func (p *OAuth2Provider) GenerateToken(claims *Claims) (string, error) {
	// 生成随机令牌
	token := generateRandomToken()

	// 存储令牌
	expiresAt := time.Now().Add(time.Hour)
	err := p.tokenStore.Store(token, claims, expiresAt)
	if err != nil {
		return "", err
	}

	return token, nil
}

// RefreshToken 刷新令牌
func (p *OAuth2Provider) RefreshToken(refreshToken string) (string, error) {
	claims, err := p.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	// 生成新的访问令牌
	claims.ExpiresAt = time.Now().Add(time.Hour)
	return p.GenerateToken(claims)
}

// RevokeToken 撤销令牌
func (p *OAuth2Provider) RevokeToken(token string) error {
	return p.tokenStore.Delete(token)
}

// generateRandomToken 生成随机令牌
func generateRandomToken() string {
	// 实现随机令牌生成逻辑
	return "random_token_" + fmt.Sprintf("%d", time.Now().UnixNano())
}

// MFAProvider 多因素认证提供者
type MFAProvider struct {
	methods []MFAMethod
	policy  MFAPolicy
}

// MFAMethod 多因素认证方法
type MFAMethod interface {
	GenerateCode(userID string) (string, error)
	ValidateCode(userID string, code string) bool
	GetType() string
}

// MFAPolicy 多因素认证策略
type MFAPolicy struct {
	RequiredMethods []string
	OptionalMethods []string
	MaxAttempts     int
	LockoutDuration time.Duration
}

// NewMFAProvider 创建多因素认证提供者
func NewMFAProvider(methods []MFAMethod, policy MFAPolicy) *MFAProvider {
	return &MFAProvider{
		methods: methods,
		policy:  policy,
	}
}

// EnhancedAuthMiddleware 增强的认证中间件
func EnhancedAuthMiddleware(provider AuthProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中获取令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":   false,
				"errorCode": 4010,
				"message":   "Authorization header required",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 解析Bearer令牌
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":   false,
				"errorCode": 4011,
				"message":   "Invalid authorization header format",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// 验证令牌
		claims, err := provider.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":   false,
				"errorCode": 4012,
				"message":   "Invalid token",
				"details":   err.Error(),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("claims", claims)

		c.Next()
	}
}

// RoleBasedAuthMiddleware 基于角色的认证中间件
func RoleBasedAuthMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4030,
				"message":   "User roles not found",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4031,
				"message":   "Invalid user roles format",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 检查用户是否具有所需角色
		hasRequiredRole := false
		for _, requiredRole := range requiredRoles {
			for _, userRole := range userRoles {
				if userRole == requiredRole {
					hasRequiredRole = true
					break
				}
			}
			if hasRequiredRole {
				break
			}
		}

		if !hasRequiredRole {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4032,
				"message":   "Insufficient permissions",
				"details":   fmt.Sprintf("Required roles: %v, User roles: %v", requiredRoles, userRoles),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PermissionBasedAuthMiddleware 基于权限的认证中间件
func PermissionBasedAuthMiddleware(requiredPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4033,
				"message":   "User permissions not found",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		userPermissions, ok := permissions.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4034,
				"message":   "Invalid user permissions format",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 检查用户是否具有所需权限
		hasRequiredPermission := false
		for _, requiredPermission := range requiredPermissions {
			for _, userPermission := range userPermissions {
				if userPermission == requiredPermission {
					hasRequiredPermission = true
					break
				}
			}
			if hasRequiredPermission {
				break
			}
		}

		if !hasRequiredPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4035,
				"message":   "Insufficient permissions",
				"details":   fmt.Sprintf("Required permissions: %v, User permissions: %v", requiredPermissions, userPermissions),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ScopeAuthMiddleware 作用域认证中间件
func ScopeAuthMiddleware(requiredScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4036,
				"message":   "User claims not found",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		userClaims, ok := claims.(*Claims)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4037,
				"message":   "Invalid user claims format",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 检查用户是否具有所需作用域
		hasRequiredScope := false
		for _, requiredScope := range requiredScopes {
			if userClaims.Extra != nil {
				if scopes, exists := userClaims.Extra["scopes"]; exists {
					if scopeList, ok := scopes.([]string); ok {
						for _, scope := range scopeList {
							if scope == requiredScope {
								hasRequiredScope = true
								break
							}
						}
					}
				}
			}
			if hasRequiredScope {
				break
			}
		}

		if !hasRequiredScope {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"errorCode": 4038,
				"message":   "Insufficient scope",
				"details":   fmt.Sprintf("Required scopes: %v", requiredScopes),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUser 获取当前用户信息
func GetCurrentUser(c *gin.Context) *Claims {
	claims, exists := c.Get("claims")
	if !exists {
		return nil
	}

	userClaims, ok := claims.(*Claims)
	if !ok {
		return nil
	}

	return userClaims
}

// GetUserID 获取用户ID
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return ""
	}

	return userIDStr
}

// GetUserRoles 获取用户角色
func GetUserRoles(c *gin.Context) []string {
	roles, exists := c.Get("roles")
	if !exists {
		return nil
	}

	userRoles, ok := roles.([]string)
	if !ok {
		return nil
	}

	return userRoles
}

// GetUserPermissions 获取用户权限
func GetUserPermissions(c *gin.Context) []string {
	permissions, exists := c.Get("permissions")
	if !exists {
		return nil
	}

	userPermissions, ok := permissions.([]string)
	if !ok {
		return nil
	}

	return userPermissions
}
