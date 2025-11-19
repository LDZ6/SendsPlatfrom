package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"
)

// EncryptionKey 加密密钥
type EncryptionKey struct {
	Key    []byte
	Salt   []byte
	Rounds int
}

// NewEncryptionKey 创建加密密钥
func NewEncryptionKey(password string, salt []byte) *EncryptionKey {
	key := sha256.Sum256(append([]byte(password), salt...))
	return &EncryptionKey{
		Key:    key[:],
		Salt:   salt,
		Rounds: 10000,
	}
}

// FieldEncryption 字段级加密
type FieldEncryption struct {
	key *EncryptionKey
}

// NewFieldEncryption 创建字段级加密
func NewFieldEncryption(key *EncryptionKey) *FieldEncryption {
	return &FieldEncryption{
		key: key,
	}
}

// EncryptField 加密字段
func (fe *FieldEncryption) EncryptField(data interface{}) (string, error) {
	// 将数据序列化为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// 加密数据
	encrypted, err := fe.encrypt(jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}

	// 返回Base64编码的加密数据
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptField 解密字段
func (fe *FieldEncryption) DecryptField(encryptedData string, target interface{}) error {
	// 解码Base64
	encrypted, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	// 解密数据
	decrypted, err := fe.decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	// 反序列化数据
	if err := json.Unmarshal(decrypted, target); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// encrypt 加密数据
func (fe *FieldEncryption) encrypt(data []byte) ([]byte, error) {
	// 创建AES加密器
	block, err := aes.NewCipher(fe.key.Key)
	if err != nil {
		return nil, err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt 解密数据
func (fe *FieldEncryption) decrypt(data []byte) ([]byte, error) {
	// 创建AES解密器
	block, err := aes.NewCipher(fe.key.Key)
	if err != nil {
		return nil, err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 提取nonce
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// DataMasking 数据脱敏
type DataMasking struct {
	rules map[string]MaskingRule
}

// MaskingRule 脱敏规则
type MaskingRule struct {
	Type        MaskingType
	Replacement string
	KeepLength  bool
	KeepPrefix  int
	KeepSuffix  int
}

// MaskingType 脱敏类型
type MaskingType int

const (
	MaskingTypeEmail MaskingType = iota
	MaskingTypePhone
	MaskingTypeIDCard
	MaskingTypeCreditCard
	MaskingTypeName
	MaskingTypeAddress
	MaskingTypeCustom
)

// NewDataMasking 创建数据脱敏
func NewDataMasking() *DataMasking {
	dm := &DataMasking{
		rules: make(map[string]MaskingRule),
	}
	dm.registerDefaultRules()
	return dm
}

// registerDefaultRules 注册默认脱敏规则
func (dm *DataMasking) registerDefaultRules() {
	dm.rules["email"] = MaskingRule{
		Type:        MaskingTypeEmail,
		Replacement: "***",
		KeepLength:  false,
		KeepPrefix:  2,
		KeepSuffix:  2,
	}

	dm.rules["phone"] = MaskingRule{
		Type:        MaskingTypePhone,
		Replacement: "*",
		KeepLength:  true,
		KeepPrefix:  3,
		KeepSuffix:  4,
	}

	dm.rules["idcard"] = MaskingRule{
		Type:        MaskingTypeIDCard,
		Replacement: "*",
		KeepLength:  true,
		KeepPrefix:  4,
		KeepSuffix:  4,
	}

	dm.rules["creditcard"] = MaskingRule{
		Type:        MaskingTypeCreditCard,
		Replacement: "*",
		KeepLength:  true,
		KeepPrefix:  4,
		KeepSuffix:  4,
	}

	dm.rules["name"] = MaskingRule{
		Type:        MaskingTypeName,
		Replacement: "*",
		KeepLength:  false,
		KeepPrefix:  1,
		KeepSuffix:  0,
	}

	dm.rules["address"] = MaskingRule{
		Type:        MaskingTypeAddress,
		Replacement: "***",
		KeepLength:  false,
		KeepPrefix:  2,
		KeepSuffix:  2,
	}
}

// AddRule 添加脱敏规则
func (dm *DataMasking) AddRule(fieldName string, rule MaskingRule) {
	dm.rules[fieldName] = rule
}

// MaskData 脱敏数据
func (dm *DataMasking) MaskData(data interface{}) (interface{}, error) {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

	if dataValue.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data must be a struct")
	}

	// 创建新的结构体副本
	newData := reflect.New(dataValue.Type()).Elem()

	// 遍历所有字段
	for i := 0; i < dataValue.NumField(); i++ {
		field := dataValue.Field(i)
		fieldType := dataValue.Type().Field(i)
		newField := newData.Field(i)

		// 检查是否有脱敏标签
		if tag, ok := fieldType.Tag.Lookup("mask"); ok {
			if field.CanSet() && newField.CanSet() {
				maskedValue, err := dm.maskField(field, tag)
				if err != nil {
					return nil, err
				}
				newField.Set(maskedValue)
			} else {
				newField.Set(field)
			}
		} else {
			newField.Set(field)
		}
	}

	return newData.Interface(), nil
}

// maskField 脱敏字段
func (dm *DataMasking) maskField(field reflect.Value, tag string) (reflect.Value, error) {
	if field.Kind() != reflect.String {
		return field, nil
	}

	value := field.String()
	if value == "" {
		return field, nil
	}

	rule, exists := dm.rules[tag]
	if !exists {
		return field, nil
	}

	maskedValue := dm.applyMaskingRule(value, rule)
	return reflect.ValueOf(maskedValue), nil
}

// applyMaskingRule 应用脱敏规则
func (dm *DataMasking) applyMaskingRule(value string, rule MaskingRule) string {
	switch rule.Type {
	case MaskingTypeEmail:
		return dm.maskEmail(value, rule)
	case MaskingTypePhone:
		return dm.maskPhone(value, rule)
	case MaskingTypeIDCard:
		return dm.maskIDCard(value, rule)
	case MaskingTypeCreditCard:
		return dm.maskCreditCard(value, rule)
	case MaskingTypeName:
		return dm.maskName(value, rule)
	case MaskingTypeAddress:
		return dm.maskAddress(value, rule)
	case MaskingTypeCustom:
		return dm.maskCustom(value, rule)
	default:
		return value
	}
}

// maskEmail 脱敏邮箱
func (dm *DataMasking) maskEmail(value string, rule MaskingRule) string {
	parts := strings.Split(value, "@")
	if len(parts) != 2 {
		return value
	}

	username := parts[0]
	domain := parts[1]

	if len(username) <= rule.KeepPrefix+rule.KeepSuffix {
		return value
	}

	maskedUsername := username[:rule.KeepPrefix] + rule.Replacement + username[len(username)-rule.KeepSuffix:]
	return maskedUsername + "@" + domain
}

// maskPhone 脱敏手机号
func (dm *DataMasking) maskPhone(value string, rule MaskingRule) string {
	if len(value) <= rule.KeepPrefix+rule.KeepSuffix {
		return value
	}

	prefix := value[:rule.KeepPrefix]
	suffix := value[len(value)-rule.KeepSuffix:]

	if rule.KeepLength {
		middle := strings.Repeat(rule.Replacement, len(value)-rule.KeepPrefix-rule.KeepSuffix)
		return prefix + middle + suffix
	}

	return prefix + rule.Replacement + suffix
}

// maskIDCard 脱敏身份证号
func (dm *DataMasking) maskIDCard(value string, rule MaskingRule) string {
	if len(value) <= rule.KeepPrefix+rule.KeepSuffix {
		return value
	}

	prefix := value[:rule.KeepPrefix]
	suffix := value[len(value)-rule.KeepSuffix:]

	if rule.KeepLength {
		middle := strings.Repeat(rule.Replacement, len(value)-rule.KeepPrefix-rule.KeepSuffix)
		return prefix + middle + suffix
	}

	return prefix + rule.Replacement + suffix
}

// maskCreditCard 脱敏信用卡号
func (dm *DataMasking) maskCreditCard(value string, rule MaskingRule) string {
	if len(value) <= rule.KeepPrefix+rule.KeepSuffix {
		return value
	}

	prefix := value[:rule.KeepPrefix]
	suffix := value[len(value)-rule.KeepSuffix:]

	if rule.KeepLength {
		middle := strings.Repeat(rule.Replacement, len(value)-rule.KeepPrefix-rule.KeepSuffix)
		return prefix + middle + suffix
	}

	return prefix + rule.Replacement + suffix
}

// maskName 脱敏姓名
func (dm *DataMasking) maskName(value string, rule MaskingRule) string {
	if len(value) <= rule.KeepPrefix {
		return value
	}

	prefix := value[:rule.KeepPrefix]
	suffix := value[len(value)-rule.KeepSuffix:]

	if rule.KeepLength {
		middle := strings.Repeat(rule.Replacement, len(value)-rule.KeepPrefix-rule.KeepSuffix)
		return prefix + middle + suffix
	}

	return prefix + rule.Replacement + suffix
}

// maskAddress 脱敏地址
func (dm *DataMasking) maskAddress(value string, rule MaskingRule) string {
	if len(value) <= rule.KeepPrefix+rule.KeepSuffix {
		return value
	}

	prefix := value[:rule.KeepPrefix]
	suffix := value[len(value)-rule.KeepSuffix:]

	if rule.KeepLength {
		middle := strings.Repeat(rule.Replacement, len(value)-rule.KeepPrefix-rule.KeepSuffix)
		return prefix + middle + suffix
	}

	return prefix + rule.Replacement + suffix
}

// maskCustom 自定义脱敏
func (dm *DataMasking) maskCustom(value string, rule MaskingRule) string {
	if len(value) <= rule.KeepPrefix+rule.KeepSuffix {
		return value
	}

	prefix := value[:rule.KeepPrefix]
	suffix := value[len(value)-rule.KeepSuffix:]

	if rule.KeepLength {
		middle := strings.Repeat(rule.Replacement, len(value)-rule.KeepPrefix-rule.KeepSuffix)
		return prefix + middle + suffix
	}

	return prefix + rule.Replacement + suffix
}

// SecurityAuditLogger 安全审计日志
type SecurityAuditLogger struct {
	logs  []SecurityAuditLog
	mutex sync.RWMutex
}

// SecurityAuditLog 安全审计日志
type SecurityAuditLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Result    string    `json:"result"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

// NewSecurityAuditLogger 创建安全审计日志
func NewSecurityAuditLogger() *SecurityAuditLogger {
	return &SecurityAuditLogger{
		logs: make([]SecurityAuditLog, 0),
	}
}

// Log 记录安全审计日志
func (sal *SecurityAuditLogger) Log(userID, action, resource, ip, userAgent, result, details string) {
	sal.mutex.Lock()
	defer sal.mutex.Unlock()

	log := SecurityAuditLog{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		IP:        ip,
		UserAgent: userAgent,
		Result:    result,
		Details:   details,
		Timestamp: time.Now(),
	}

	sal.logs = append(sal.logs, log)

	// 保持最近10000条日志
	if len(sal.logs) > 10000 {
		sal.logs = sal.logs[len(sal.logs)-10000:]
	}
}

// GetLogs 获取审计日志
func (sal *SecurityAuditLogger) GetLogs() []SecurityAuditLog {
	sal.mutex.RLock()
	defer sal.mutex.RUnlock()

	logs := make([]SecurityAuditLog, len(sal.logs))
	copy(logs, sal.logs)

	return logs
}

// GetLogsByUser 根据用户获取审计日志
func (sal *SecurityAuditLogger) GetLogsByUser(userID string) []SecurityAuditLog {
	sal.mutex.RLock()
	defer sal.mutex.RUnlock()

	var userLogs []SecurityAuditLog
	for _, log := range sal.logs {
		if log.UserID == userID {
			userLogs = append(userLogs, log)
		}
	}

	return userLogs
}

// GetLogsByAction 根据操作获取审计日志
func (sal *SecurityAuditLogger) GetLogsByAction(action string) []SecurityAuditLog {
	sal.mutex.RLock()
	defer sal.mutex.RUnlock()

	var actionLogs []SecurityAuditLog
	for _, log := range sal.logs {
		if log.Action == action {
			actionLogs = append(actionLogs, log)
		}
	}

	return actionLogs
}

// GetLogsByResult 根据结果获取审计日志
func (sal *SecurityAuditLogger) GetLogsByResult(result string) []SecurityAuditLog {
	sal.mutex.RLock()
	defer sal.mutex.RUnlock()

	var resultLogs []SecurityAuditLog
	for _, log := range sal.logs {
		if log.Result == result {
			resultLogs = append(resultLogs, log)
		}
	}

	return resultLogs
}
