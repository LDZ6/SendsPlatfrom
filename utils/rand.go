package utils

import (
	"crypto/rand"
	"github.com/sirupsen/logrus"
)

func RandString(length int) (str string, err error) {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	randomBytes := make([]byte, length)

	_, err = rand.Read(randomBytes)
	if err != nil {
		logrus.Info("随机数生成失败:", err)
		return
	}
	for i := 0; i < length; i++ {
		randomBytes[i] = charset[randomBytes[i]%byte(len(charset))]
	}
	str = string(randomBytes)
	return
}
