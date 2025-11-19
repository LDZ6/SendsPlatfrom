package utils

import (
	"crypto/md5"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

// JWT密钥管理器
type JWTKeyManager struct {
	keys        map[string][]byte
	currentKeyID string
	mutex       sync.RWMutex
}

var jwtKeyManager *JWTKeyManager

// 初始化JWT密钥管理器
func initJWTKeyManager() {
	jwtKeyManager = &JWTKeyManager{
		keys: make(map[string][]byte),
	}
	
	// 从配置加载密钥
	loadJWTKeys()
}

// 加载JWT密钥
func loadJWTKeys() {
	jwtKeyManager.mutex.Lock()
	defer jwtKeyManager.mutex.Unlock()
	
	// 从环境变量或配置文件获取密钥
	keysConfig := viper.GetStringMapString("jwt.keys")
	for keyID, keyValue := range keysConfig {
		jwtKeyManager.keys[keyID] = []byte(keyValue)
	}
	
	// 设置当前密钥ID
	jwtKeyManager.currentKeyID = viper.GetString("jwt.currentKeyId")
	if jwtKeyManager.currentKeyID == "" {
		jwtKeyManager.currentKeyID = "v1" // 默认密钥ID
	}
}

// 获取当前密钥
func (j *JWTKeyManager) GetCurrentKey() ([]byte, string, error) {
	j.mutex.RLock()
	defer j.mutex.RUnlock()
	
	key, exists := j.keys[j.currentKeyID]
	if !exists {
		return nil, "", fmt.Errorf("当前密钥ID %s 不存在", j.currentKeyID)
	}
	
	return key, j.currentKeyID, nil
}

// 根据密钥ID获取密钥
func (j *JWTKeyManager) GetKeyByID(keyID string) ([]byte, error) {
	j.mutex.RLock()
	defer j.mutex.RUnlock()
	
	key, exists := j.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("密钥ID %s 不存在", keyID)
	}
	
	return key, nil
}

// 更新密钥集合
func (j *JWTKeyManager) UpdateKeys(keys map[string][]byte, currentKeyID string) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	
	j.keys = keys
	j.currentKeyID = currentKeyID
}

// 自定义Claims结构，支持kid
type CustomClaims struct {
	OpenId string `json:"openId"`
	StuNum string `json:"stuNum"`
	Exp    int64  `json:"exp"`
	jwt.RegisteredClaims
}

type MassesClaims struct {
	OpenId   string `json:"openId"`
	NickName string `json:"nick_name"`
	Exp      int64  `json:"exp"`
	jwt.RegisteredClaims
}

type AdminClaims struct {
	OpenId       string `json:"openId"`
	Organization uint32 `json:"organization"`
	Exp          int64  `json:"exp"`
	jwt.RegisteredClaims
}

// GetMd5
// 生成 md5
func GetMd5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// 初始化JWT密钥管理器
func init() {
	initJWTKeyManager()
}

// 获取过期时间
func getExpirationTime() int64 {
	expirationHours := viper.GetInt("jwt.expirationHours")
	if expirationHours <= 0 {
		expirationHours = 24 // 默认24小时
	}
	return time.Now().Add(time.Duration(expirationHours) * time.Hour).Unix()
}

func GenerateUserToken(openid string, stunum string) string {
	// 获取当前密钥
	key, keyID, err := jwtKeyManager.GetCurrentKey()
	if err != nil {
		return ""
	}
	
	UserClaim := &CustomClaims{
		OpenId: openid,
		StuNum: stunum,
		Exp:    getExpirationTime(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(getExpirationTime(), 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaim)
	
	// 设置kid头
	token.Header["kid"] = keyID
	
	tokenString, err := token.SignedString(key)
	if err != nil {
		return ""
	}
	return tokenString
}

func GenerateMassesToken(openid string, nickName string) string {
	// 获取当前密钥
	key, keyID, err := jwtKeyManager.GetCurrentKey()
	if err != nil {
		return ""
	}
	
	MassesClaim := &MassesClaims{
		OpenId:   openid,
		Exp:      getExpirationTime(),
		NickName: nickName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(getExpirationTime(), 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, MassesClaim)
	
	// 设置kid头
	token.Header["kid"] = keyID
	
	tokenString, err := token.SignedString(key)
	if err != nil {
		return ""
	}
	return tokenString
}

func GenerateAdminToken(openid string, organization uint32) string {
	// 获取当前密钥
	key, keyID, err := jwtKeyManager.GetCurrentKey()
	if err != nil {
		return ""
	}
	
	AdminClaim := &AdminClaims{
		OpenId:       openid,
		Organization: organization,
		Exp:          getExpirationTime(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(getExpirationTime(), 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, AdminClaim)
	
	// 设置kid头
	token.Header["kid"] = keyID
	
	tokenString, err := token.SignedString(key)
	if err != nil {
		return ""
	}
	return tokenString
}

// AnalyseUserToken 解析用户Token，支持多密钥
func AnalyseUserToken(tokenString string) (*CustomClaims, error) {
	userClaim := new(CustomClaims)
	claims, err := jwt.ParseWithClaims(tokenString, userClaim, func(token *jwt.Token) (interface{}, error) {
		// 获取kid头
		kid, ok := token.Header["kid"].(string)
		if !ok {
			// 如果没有kid，尝试使用当前密钥
			key, _, err := jwtKeyManager.GetCurrentKey()
			return key, err
		}
		
		// 根据kid获取对应密钥
		key, err := jwtKeyManager.GetKeyByID(kid)
		if err != nil {
			// 如果找不到对应密钥，尝试使用当前密钥
			key, _, err = jwtKeyManager.GetCurrentKey()
			return key, err
		}
		
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if !claims.Valid {
		return nil, fmt.Errorf("analyse Token Error:%v", err)
	}
	return userClaim, nil
}

func AnalyseMassesToken(tokenString string) (*MassesClaims, error) {
	massesClaim := new(MassesClaims)
	claims, err := jwt.ParseWithClaims(tokenString, massesClaim, func(token *jwt.Token) (interface{}, error) {
		// 获取kid头
		kid, ok := token.Header["kid"].(string)
		if !ok {
			// 如果没有kid，尝试使用当前密钥
			key, _, err := jwtKeyManager.GetCurrentKey()
			return key, err
		}
		
		// 根据kid获取对应密钥
		key, err := jwtKeyManager.GetKeyByID(kid)
		if err != nil {
			// 如果找不到对应密钥，尝试使用当前密钥
			key, _, err = jwtKeyManager.GetCurrentKey()
			return key, err
		}
		
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if !claims.Valid {
		return nil, fmt.Errorf("analyse Token Error:%v", err)
	}
	return massesClaim, nil
}

func AnalyseAdminToken(tokenString string) (*AdminClaims, error) {
	adminClaim := new(AdminClaims)
	claims, err := jwt.ParseWithClaims(tokenString, adminClaim, func(token *jwt.Token) (interface{}, error) {
		// 获取kid头
		kid, ok := token.Header["kid"].(string)
		if !ok {
			// 如果没有kid，尝试使用当前密钥
			key, _, err := jwtKeyManager.GetCurrentKey()
			return key, err
		}
		
		// 根据kid获取对应密钥
		key, err := jwtKeyManager.GetKeyByID(kid)
		if err != nil {
			// 如果找不到对应密钥，尝试使用当前密钥
			key, _, err = jwtKeyManager.GetCurrentKey()
			return key, err
		}
		
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if !claims.Valid {
		return nil, fmt.Errorf("analyse Token Error:%v", err)
	}
	return adminClaim, nil
}

// 检查token是否过期
func ParseToken(token string) (err error) {
	claims, err := AnalyseUserToken(token)
	if err != nil {
		return
	}
	if claims.Exp < time.Now().Unix() {
		return errors.New("Token已过期")
	}
	return
}

func parseToken(exp int64) (err error) {
	if exp < time.Now().Unix() {
		return errors.New("Token已过期")
	}
	return
}

// 更新JWT密钥（用于密钥轮转）
func UpdateJWTKeys(keys map[string][]byte, currentKeyID string) {
	jwtKeyManager.UpdateKeys(keys, currentKeyID)
}

// 获取当前密钥ID
func GetCurrentKeyID() string {
	jwtKeyManager.mutex.RLock()
	defer jwtKeyManager.mutex.RUnlock()
	return jwtKeyManager.currentKeyID
}
