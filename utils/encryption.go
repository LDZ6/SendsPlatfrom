package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var AcquiesceKey = ""
var Encrypt *Encryption

// 加密密钥管理器
type CryptoKeyManager struct {
	keys        map[string][]byte
	currentKeyID string
	mutex       sync.RWMutex
}

var cryptoKeyManager *CryptoKeyManager

// AES-GCM 加密算法
type Encryption struct {
	keyManager *CryptoKeyManager
}

func init() {
	initCryptoKeyManager()
	Encrypt = NewEncryption()
}

func initCryptoKeyManager() {
	cryptoKeyManager = &CryptoKeyManager{
		keys: make(map[string][]byte),
	}
	
	// 从配置加载密钥
	loadCryptoKeys()
}

// 加载加密密钥
func loadCryptoKeys() {
	cryptoKeyManager.mutex.Lock()
	defer cryptoKeyManager.mutex.Unlock()
	
	// 从环境变量或配置文件获取密钥
	keyVersion := viper.GetString("crypto.keyVersion")
	if keyVersion == "" {
		keyVersion = "v1"
	}
	
	// 生成或获取密钥
	key := generateOrGetKey(keyVersion)
	cryptoKeyManager.keys[keyVersion] = key
	cryptoKeyManager.currentKeyID = keyVersion
}

// 生成或获取密钥
func generateOrGetKey(keyID string) []byte {
	// 尝试从环境变量获取
	envKey := viper.GetString(fmt.Sprintf("CRYPTO_KEY_%s", strings.ToUpper(keyID)))
	if envKey != "" {
		return []byte(envKey)
	}
	
	// 生成新密钥
	key := make([]byte, 32) // AES-256需要32字节密钥
	if _, err := rand.Read(key); err != nil {
		panic(fmt.Errorf("生成加密密钥失败: %w", err))
	}
	
	return key
}

func NewEncryption() *Encryption {
	return &Encryption{
		keyManager: cryptoKeyManager,
	}
}

// 去掉填充的部分
func UnPadPwd(dst []byte) ([]byte, error) {
	//if len(dst) <= 0 {
	//	return dst, errors.New("长度有误")
	//}
	// 去掉的长度
	unpadNum := int(dst[len(dst)-1])
	strErr := "error"
	op := []byte(strErr)
	if len(dst) < unpadNum {
		return op, nil
	}
	str := dst[:(len(dst) - unpadNum)]
	return str, nil
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func encrypt(message, key string) string {
	keyBytes := []byte(key)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		panic(err)
	}
	messageBytes := []byte(message)
	messageBytes = pkcs7Padding(messageBytes, block.BlockSize())
	ciphertext := make([]byte, len(messageBytes))
	iv := make([]byte, block.BlockSize())
	encrypter := cipher.NewCBCEncrypter(block, iv)
	encrypter.CryptBlocks(ciphertext, messageBytes)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// 填充密码长度
func PadPwd(srcByte []byte, blockSize int) []byte {
	padNum := blockSize - len(srcByte)%blockSize
	ret := bytes.Repeat([]byte{byte(padNum)}, padNum)
	srcByte = append(srcByte, ret...)
	return srcByte
}

// 加密
func (k *Encryption) AesEncoding(src string) string {
	keyBytes := []byte(k.key)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		panic(err)
	}
	messageBytes := []byte(src)
	messageBytes = pkcs7Padding(messageBytes, block.BlockSize())
	ciphertext := make([]byte, len(messageBytes))
	iv := make([]byte, block.BlockSize())
	encrypter := cipher.NewCBCEncrypter(block, iv)
	encrypter.CryptBlocks(ciphertext, messageBytes)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// 解密
func (k *Encryption) AesDecoding(pwd string) (error, string) {
	pwdByte := []byte(pwd)
	pwdByte, err := base64.StdEncoding.DecodeString(pwd)
	if err != nil {
		return err, pwd
	}
	key := k.key
	block, errBlock := aes.NewCipher([]byte(key))
	if errBlock != nil {
		return errBlock, pwd
	}
	// 从密文中提取初始向量
	if len(pwdByte) < block.BlockSize() {
		return errors.New("密文长度有误"), pwd
	}
	iv := make([]byte, block.BlockSize())

	// 创建一个 CBC 模式的解密器
	mode := cipher.NewCBCDecrypter(block, iv)
	dst := make([]byte, len(pwdByte))
	mode.CryptBlocks(dst, pwdByte)
	dst, err = UnPadPwd(dst)
	if err != nil {
		logrus.Info("[AESRROR]:%v\n", err.Error())
		return err, ""
	}
	return nil, string(dst)
}

// AES-GCM 加密
func (k *Encryption) AesGCMEncrypt(plaintext string) (string, error) {
	// 获取当前密钥
	key, keyID, err := k.getCurrentKey()
	if err != nil {
		return "", fmt.Errorf("获取加密密钥失败: %w", err)
	}
	
	// 创建AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}
	
	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}
	
	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}
	
	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// 格式: v{keyId}:{base64(nonce)}:{base64(ciphertext)}
	nonceB64 := base64.StdEncoding.EncodeToString(nonce)
	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)
	
	return fmt.Sprintf("v%s:%s:%s", keyID, nonceB64, ciphertextB64), nil
}

// AES-GCM 解密
func (k *Encryption) AesGCMDecrypt(encrypted string) (string, error) {
	// 解析格式: v{keyId}:{base64(nonce)}:{base64(ciphertext)}
	parts := strings.Split(encrypted, ":")
	if len(parts) != 3 {
		return "", fmt.Errorf("加密格式错误")
	}
	
	keyID := strings.TrimPrefix(parts[0], "v")
	nonceB64 := parts[1]
	ciphertextB64 := parts[2]
	
	// 获取对应密钥
	key, err := k.getKeyByID(keyID)
	if err != nil {
		// 如果找不到对应密钥，尝试使用当前密钥
		key, _, err = k.getCurrentKey()
		if err != nil {
			return "", fmt.Errorf("获取解密密钥失败: %w", err)
		}
	}
	
	// 解码nonce和密文
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("解码nonce失败: %w", err)
	}
	
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("解码密文失败: %w", err)
	}
	
	// 创建AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}
	
	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}
	
	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}
	
	return string(plaintext), nil
}

// 获取当前密钥
func (k *Encryption) getCurrentKey() ([]byte, string, error) {
	k.keyManager.mutex.RLock()
	defer k.keyManager.mutex.RUnlock()
	
	key, exists := k.keyManager.keys[k.keyManager.currentKeyID]
	if !exists {
		return nil, "", fmt.Errorf("当前密钥ID %s 不存在", k.keyManager.currentKeyID)
	}
	
	return key, k.keyManager.currentKeyID, nil
}

// 根据密钥ID获取密钥
func (k *Encryption) getKeyByID(keyID string) ([]byte, error) {
	k.keyManager.mutex.RLock()
	defer k.keyManager.mutex.RUnlock()
	
	key, exists := k.keyManager.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("密钥ID %s 不存在", keyID)
	}
	
	return key, nil
}

// 更新加密密钥（用于密钥轮转）
func (k *Encryption) UpdateKeys(keys map[string][]byte, currentKeyID string) {
	k.keyManager.mutex.Lock()
	defer k.keyManager.mutex.Unlock()
	
	k.keyManager.keys = keys
	k.keyManager.currentKeyID = currentKeyID
}

// 获取当前密钥ID
func (k *Encryption) GetCurrentKeyID() string {
	k.keyManager.mutex.RLock()
	defer k.keyManager.mutex.RUnlock()
	return k.keyManager.currentKeyID
}

// 兼容旧版本的加密方法（使用AES-GCM）
func (k *Encryption) AesEncoding(src string) string {
	encrypted, err := k.AesGCMEncrypt(src)
	if err != nil {
		logrus.Errorf("AES-GCM加密失败: %v", err)
		return ""
	}
	return encrypted
}

// 兼容旧版本的解密方法（使用AES-GCM）
func (k *Encryption) AesDecoding(pwd string) (error, string) {
	// 检查是否是新的加密格式
	if strings.Contains(pwd, ":") && strings.HasPrefix(pwd, "v") {
		decrypted, err := k.AesGCMDecrypt(pwd)
		if err != nil {
			return err, ""
		}
		return nil, decrypted
	}
	
	// 兼容旧格式的CBC解密
	return k.legacyAesDecoding(pwd)
}

// 旧版本CBC解密（兼容性）
func (k *Encryption) legacyAesDecoding(pwd string) (error, string) {
	pwdByte := []byte(pwd)
	pwdByte, err := base64.StdEncoding.DecodeString(pwd)
	if err != nil {
		return err, pwd
	}
	
	// 获取当前密钥
	key, _, err := k.getCurrentKey()
	if err != nil {
		return err, pwd
	}
	
	block, errBlock := aes.NewCipher(key)
	if errBlock != nil {
		return errBlock, pwd
	}
	
	// 从密文中提取初始向量
	if len(pwdByte) < block.BlockSize() {
		return errors.New("密文长度有误"), pwd
	}
	iv := make([]byte, block.BlockSize())

	// 创建一个 CBC 模式的解密器
	mode := cipher.NewCBCDecrypter(block, iv)
	dst := make([]byte, len(pwdByte))
	mode.CryptBlocks(dst, pwdByte)
	dst, err = UnPadPwd(dst)
	if err != nil {
		logrus.Info("[AESRROR]:%v\n", err.Error())
		return err, ""
	}
	return nil, string(dst)
}

// set方法（兼容性）
func (k *Encryption) SetKey(key string) {
	// 为了兼容性，将密钥设置为当前密钥
	k.keyManager.mutex.Lock()
	defer k.keyManager.mutex.Unlock()
	
	k.keyManager.keys[k.keyManager.currentKeyID] = []byte(key)
}
