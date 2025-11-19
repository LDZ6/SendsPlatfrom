package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var Conf *Config
var URL string

// 配置热更新支持
var (
	configVersion int64
	configMutex   sync.RWMutex
	hotReloadChan = make(chan struct{}, 1)
)

type Config struct {
	Server   *Server              `yaml:"server"`
	MySQL    map[string]*MySQL    `yaml:"mysql"`
	Redis    map[string]*Redis    `yaml:"redis"`
	Etcd     *Etcd                `yaml:"etcd"`
	Services map[string]*Service  `yaml:"services"`
	Domain   map[string]*Domain   `yaml:"domain"`
	Mq       map[string]*RabbitMQ `yaml:"mq"`
	// JWT多密钥支持
	JWT *JWTConfig `yaml:"jwt"`
	// 加密配置
	Crypto *CryptoConfig `yaml:"crypto"`
	// 可观测性配置
	Observability *ObservabilityConfig `yaml:"observability"`
}

// JWT配置结构
type JWTConfig struct {
	// 密钥集合，支持多版本
	Keys map[string]string `yaml:"keys"`
	// 当前使用的密钥ID
	CurrentKeyID string `yaml:"currentKeyId"`
	// Token过期时间（小时）
	ExpirationHours int `yaml:"expirationHours"`
}

// 加密配置
type CryptoConfig struct {
	// AES密钥版本
	KeyVersion string `yaml:"keyVersion"`
	// 加密算法
	Algorithm string `yaml:"algorithm"`
}

// 可观测性配置
type ObservabilityConfig struct {
	Logging *LoggingConfig `yaml:"logging"`
	Metrics *MetricsConfig `yaml:"metrics"`
	Tracing *TracingConfig `yaml:"tracing"`
}

type LoggingConfig struct {
	Level    string          `yaml:"level"`
	Format   string          `yaml:"format"` // json, text
	Sampling *SamplingConfig `yaml:"sampling"`
}

type SamplingConfig struct {
	ErrorRate  float64 `yaml:"errorRate"`
	NormalRate float64 `yaml:"normalRate"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	Port    int    `yaml:"port"`
}

type TracingConfig struct {
	Enabled     bool    `yaml:"enabled"`
	SampleRate  float64 `yaml:"sampleRate"`
	ServiceName string  `yaml:"serviceName"`
}

type Server struct {
	Port      string `yaml:"port"`
	Version   string `yaml:"version"`
	JwtSecret string `yaml:"jwtSecret"`
}

type MySQL struct {
	DriverName string `yaml:"driverName"`
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	Database   string `yaml:"database"`
	UserName   string `yaml:"username"`
	Password   string `yaml:"password"`
	Charset    string `yaml:"charset"`
}

type Redis struct {
	UserName string `yaml:"userName"`
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
}

type Etcd struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Service struct {
	Name        string `yaml:"name"`
	LoadBalance bool   `yaml:"loadBalance"`
	Addr        string `yaml:"addr"`
}

type Domain struct {
	Name string `yaml:"name"`
}

type RabbitMQ struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Port     string `yaml:"port"`
}

// InitConfig 初始化配置，支持环境变量覆盖和热更新
func InitConfig() {
	workDir, _ := os.Getwd()
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(workDir + "/config")

	// 设置环境变量前缀和自动绑定
	viper.SetEnvPrefix("PLATFORM")
	viper.AutomaticEnv()

	// 绑定敏感字段到环境变量
	bindSensitiveFields()

	// 读取配置文件
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}

	// 解析配置
	err = viper.Unmarshal(&Conf)
	if err != nil {
		panic(fmt.Errorf("解析配置失败: %w", err))
	}

	// 验证敏感字段
	validateSensitiveFields()

	// 启动配置热更新
	startConfigWatcher()
}

// bindSensitiveFields 绑定敏感字段到环境变量
func bindSensitiveFields() {
	// MySQL密码
	viper.BindEnv("mysql.user.password", "MYSQL_USER_PASSWORD")
	viper.BindEnv("mysql.bobing.password", "MYSQL_BOBING_PASSWORD")
	viper.BindEnv("mysql.year_bill.password", "MYSQL_YEARBILL_PASSWORD")

	// Redis密码
	viper.BindEnv("redis.user.password", "REDIS_USER_PASSWORD")
	viper.BindEnv("redis.bobing.password", "REDIS_BOBING_PASSWORD")
	viper.BindEnv("redis.school.password", "REDIS_SCHOOL_PASSWORD")
	viper.BindEnv("redis.year_bill.password", "REDIS_YEARBILL_PASSWORD")

	// Etcd密码
	viper.BindEnv("etcd.password", "ETCD_PASSWORD")

	// RabbitMQ密码
	viper.BindEnv("mq.year_bill.password", "RABBITMQ_PASSWORD")

	// JWT密钥
	viper.BindEnv("jwt.keys", "JWT_KEYS")
	viper.BindEnv("jwt.currentKeyId", "JWT_CURRENT_KEY_ID")

	// 加密密钥
	viper.BindEnv("crypto.keyVersion", "CRYPTO_KEY_VERSION")
}

// validateSensitiveFields 验证敏感字段是否已设置
func validateSensitiveFields() {
	requiredFields := map[string]string{
		"mysql.user.password":      "MYSQL_USER_PASSWORD",
		"mysql.bobing.password":    "MYSQL_BOBING_PASSWORD",
		"mysql.year_bill.password": "MYSQL_YEARBILL_PASSWORD",
		"redis.user.password":      "REDIS_USER_PASSWORD",
		"redis.bobing.password":    "REDIS_BOBING_PASSWORD",
		"redis.school.password":    "REDIS_SCHOOL_PASSWORD",
		"redis.year_bill.password": "REDIS_YEARBILL_PASSWORD",
		"etcd.password":            "ETCD_PASSWORD",
		"mq.year_bill.password":    "RABBITMQ_PASSWORD",
	}

	var missingFields []string
	for configPath, envVar := range requiredFields {
		if !viper.IsSet(configPath) {
			missingFields = append(missingFields, fmt.Sprintf("%s (环境变量: %s)", configPath, envVar))
		}
	}

	if len(missingFields) > 0 {
		panic(fmt.Errorf("以下敏感字段未设置:\n%s", strings.Join(missingFields, "\n")))
	}
}

// startConfigWatcher 启动配置热更新监听
func startConfigWatcher() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		select {
		case hotReloadChan <- struct{}{}:
		default:
		}
	})
}

// GetConfigVersion 获取当前配置版本
func GetConfigVersion() int64 {
	return atomic.LoadInt64(&configVersion)
}

// HotReloadConfig 热更新配置
func HotReloadConfig() error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// 重新读取配置
	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("重新读取配置失败: %w", err)
	}

	// 创建新配置实例
	newConf := &Config{}
	err = viper.Unmarshal(newConf)
	if err != nil {
		return fmt.Errorf("解析新配置失败: %w", err)
	}

	// 验证新配置
	// TODO: 添加配置验证逻辑

	// 原子更新配置
	Conf = newConf
	atomic.AddInt64(&configVersion, 1)

	return nil
}

// WatchHotReload 监听配置热更新
func WatchHotReload(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-hotReloadChan:
			if err := HotReloadConfig(); err != nil {
				// TODO: 记录错误日志
				fmt.Printf("配置热更新失败: %v\n", err)
			}
		}
	}
}
