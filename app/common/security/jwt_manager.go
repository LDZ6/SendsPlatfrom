package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager JWT管理器
type JWTManager struct {
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	keys            map[string]string
	currentKeyID    string
	refreshStore    RefreshTokenStore
	mutex           sync.RWMutex
}

// RefreshTokenStore 刷新令牌存储接口
type RefreshTokenStore interface {
	Save(token *RefreshToken) error
	Get(tokenID string) (*RefreshToken, error)
	Delete(tokenID string) error
	DeleteByUserID(userID string) error
	Cleanup() error
}

// RefreshToken 刷新令牌
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IsUsed    bool      `json:"is_used"`
	ClientIP  string    `json:"client_ip"`
	UserAgent string    `json:"user_agent"`
}

// TokenPair 令牌对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Claims JWT声明
type Claims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	ClientID    string   `json:"client_id"`
	SessionID   string   `json:"session_id"`
	jwt.RegisteredClaims
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(accessTokenTTL, refreshTokenTTL time.Duration, refreshStore RefreshTokenStore) *JWTManager {
	return &JWTManager{
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		keys:            make(map[string]string),
		refreshStore:    refreshStore,
	}
}

// GenerateKeyPair 生成密钥对
func (jm *JWTManager) GenerateKeyPair() (string, string, error) {
	// 生成随机密钥
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random key: %w", err)
	}

	keyID := generateKeyID()
	key := hex.EncodeToString(keyBytes)

	jm.mutex.Lock()
	jm.keys[keyID] = key
	jm.currentKeyID = keyID
	jm.mutex.Unlock()

	return keyID, key, nil
}

// SetKey 设置密钥
func (jm *JWTManager) SetKey(keyID, key string) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()
	jm.keys[keyID] = key
}

// SetCurrentKey 设置当前密钥
func (jm *JWTManager) SetCurrentKey(keyID string) error {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	if _, exists := jm.keys[keyID]; !exists {
		return fmt.Errorf("key not found: %s", keyID)
	}

	jm.currentKeyID = keyID
	return nil
}

// GenerateAccessToken 生成访问令牌
func (jm *JWTManager) GenerateAccessToken(userID, username string, roles, permissions []string, clientID, sessionID string) (string, error) {
	jm.mutex.RLock()
	key, exists := jm.keys[jm.currentKeyID]
	jm.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("current key not found")
	}

	now := time.Now()
	claims := &Claims{
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		ClientID:    clientID,
		SessionID:   sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "platform",
			Subject:   userID,
			Audience:  []string{"platform-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(jm.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = jm.currentKeyID

	return token.SignedString([]byte(key))
}

// GenerateRefreshToken 生成刷新令牌
func (jm *JWTManager) GenerateRefreshToken(userID, clientIP, userAgent string) (*RefreshToken, error) {
	tokenID := generateTokenID()
	token := generateRandomToken()

	refreshToken := &RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(jm.refreshTokenTTL),
		CreatedAt: time.Now(),
		ClientIP:  clientIP,
		UserAgent: userAgent,
	}

	if err := jm.refreshStore.Save(refreshToken); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return refreshToken, nil
}

// GenerateTokenPair 生成令牌对
func (jm *JWTManager) GenerateTokenPair(userID, username string, roles, permissions []string, clientID, sessionID, clientIP, userAgent string) (*TokenPair, error) {
	// 生成访问令牌
	accessToken, err := jm.GenerateAccessToken(userID, username, roles, permissions, clientID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 生成刷新令牌
	refreshToken, err := jm.GenerateRefreshToken(userID, clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    int64(jm.accessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken 验证访问令牌
func (jm *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// 获取密钥ID
		keyID, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing key ID")
		}

		// 获取密钥
		jm.mutex.RLock()
		key, exists := jm.keys[keyID]
		jm.mutex.RUnlock()

		if !exists {
			return nil, fmt.Errorf("key not found: %s", keyID)
		}

		return []byte(key), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

// RefreshAccessToken 刷新访问令牌
func (jm *JWTManager) RefreshAccessToken(refreshTokenString string) (*TokenPair, error) {
	// 获取刷新令牌
	refreshToken, err := jm.refreshStore.Get(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// 检查令牌是否过期
	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	// 检查令牌是否已使用
	if refreshToken.IsUsed {
		return nil, fmt.Errorf("refresh token already used")
	}

	// 标记令牌为已使用
	refreshToken.IsUsed = true
	jm.refreshStore.Save(refreshToken)

	// 生成新的令牌对
	// 这里需要从数据库获取用户信息
	// 简化实现，使用默认值
	userID := refreshToken.UserID
	username := "user" // 应该从数据库获取
	roles := []string{"user"}
	permissions := []string{"read", "write"}
	clientID := "default"
	sessionID := generateTokenID()

	return jm.GenerateTokenPair(userID, username, roles, permissions, clientID, sessionID, refreshToken.ClientIP, refreshToken.UserAgent)
}

// RevokeRefreshToken 撤销刷新令牌
func (jm *JWTManager) RevokeRefreshToken(tokenID string) error {
	return jm.refreshStore.Delete(tokenID)
}

// RevokeUserTokens 撤销用户所有令牌
func (jm *JWTManager) RevokeUserTokens(userID string) error {
	return jm.refreshStore.DeleteByUserID(userID)
}

// CleanupExpiredTokens 清理过期令牌
func (jm *JWTManager) CleanupExpiredTokens() error {
	return jm.refreshStore.Cleanup()
}

// generateKeyID 生成密钥ID
func generateKeyID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateTokenID 生成令牌ID
func generateTokenID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateRandomToken 生成随机令牌
func generateRandomToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// MemoryRefreshTokenStore 内存刷新令牌存储
type MemoryRefreshTokenStore struct {
	tokens map[string]*RefreshToken
	mutex  sync.RWMutex
}

// NewMemoryRefreshTokenStore 创建内存刷新令牌存储
func NewMemoryRefreshTokenStore() *MemoryRefreshTokenStore {
	return &MemoryRefreshTokenStore{
		tokens: make(map[string]*RefreshToken),
	}
}

// Save 保存刷新令牌
func (mrts *MemoryRefreshTokenStore) Save(token *RefreshToken) error {
	mrts.mutex.Lock()
	defer mrts.mutex.Unlock()
	mrts.tokens[token.Token] = token
	return nil
}

// Get 获取刷新令牌
func (mrts *MemoryRefreshTokenStore) Get(tokenID string) (*RefreshToken, error) {
	mrts.mutex.RLock()
	defer mrts.mutex.RUnlock()

	token, exists := mrts.tokens[tokenID]
	if !exists {
		return nil, fmt.Errorf("refresh token not found")
	}

	return token, nil
}

// Delete 删除刷新令牌
func (mrts *MemoryRefreshTokenStore) Delete(tokenID string) error {
	mrts.mutex.Lock()
	defer mrts.mutex.Unlock()
	delete(mrts.tokens, tokenID)
	return nil
}

// DeleteByUserID 删除用户所有令牌
func (mrts *MemoryRefreshTokenStore) DeleteByUserID(userID string) error {
	mrts.mutex.Lock()
	defer mrts.mutex.Unlock()

	for tokenID, token := range mrts.tokens {
		if token.UserID == userID {
			delete(mrts.tokens, tokenID)
		}
	}

	return nil
}

// Cleanup 清理过期令牌
func (mrts *MemoryRefreshTokenStore) Cleanup() error {
	mrts.mutex.Lock()
	defer mrts.mutex.Unlock()

	now := time.Now()
	for tokenID, token := range mrts.tokens {
		if now.After(token.ExpiresAt) {
			delete(mrts.tokens, tokenID)
		}
	}

	return nil
}
