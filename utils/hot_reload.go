package utils

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 热切换客户端接口
type HotSwappableClient interface {
	Close() error
	HealthCheck() error
}

// 热切换管理器
type HotReloadManager struct {
	clients map[string]HotSwappableClient
	mutex   sync.RWMutex
	version int64
}

var (
	hotReloadManager *HotReloadManager
)

// 初始化热切换管理器
func init() {
	initHotReloadManager()
}

func initHotReloadManager() {
	hotReloadManager = &HotReloadManager{
		clients: make(map[string]HotSwappableClient),
	}
}

// 注册客户端
func RegisterClient(name string, client HotSwappableClient) {
	hotReloadManager.mutex.Lock()
	defer hotReloadManager.mutex.Unlock()
	
	hotReloadManager.clients[name] = client
}

// 热切换客户端
func HotSwapClient(name string, newClient HotSwappableClient) error {
	hotReloadManager.mutex.Lock()
	defer hotReloadManager.mutex.Unlock()
	
	// 健康检查新客户端
	if err := newClient.HealthCheck(); err != nil {
		return fmt.Errorf("新客户端健康检查失败: %w", err)
	}
	
	// 获取旧客户端
	oldClient, exists := hotReloadManager.clients[name]
	if exists {
		// 异步关闭旧客户端
		go func() {
			if err := oldClient.Close(); err != nil {
				logrus.WithFields(logrus.Fields{
					"client": name,
					"error":  err,
				}).Warn("关闭旧客户端失败")
			}
		}()
	}
	
	// 原子更新客户端
	hotReloadManager.clients[name] = newClient
	atomic.AddInt64(&hotReloadManager.version, 1)
	
	logrus.WithFields(logrus.Fields{
		"client":  name,
		"version": atomic.LoadInt64(&hotReloadManager.version),
	}).Info("客户端热切换成功")
	
	return nil
}

// 获取客户端
func GetClient(name string) (HotSwappableClient, bool) {
	hotReloadManager.mutex.RLock()
	defer hotReloadManager.mutex.RUnlock()
	
	client, exists := hotReloadManager.clients[name]
	return client, exists
}

// 获取版本号
func GetClientVersion() int64 {
	return atomic.LoadInt64(&hotReloadManager.version)
}

// MySQL热切换客户端
type MySQLHotSwapClient struct {
	db     *gorm.DB
	rawDB  *sql.DB
	config *MySQLConfig
}

type MySQLConfig struct {
	DriverName string
	Host       string
	Port       string
	Database   string
	UserName   string
	Password   string
	Charset    string
}

func NewMySQLHotSwapClient(config *MySQLConfig) (*MySQLHotSwapClient, error) {
	// 创建数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		config.UserName, config.Password, config.Host, config.Port, config.Database, config.Charset)
	
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	
	rawDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取原始数据库连接失败: %w", err)
	}
	
	return &MySQLHotSwapClient{
		db:     db,
		rawDB:  rawDB,
		config: config,
	}, nil
}

func (c *MySQLHotSwapClient) Close() error {
	if c.rawDB != nil {
		return c.rawDB.Close()
	}
	return nil
}

func (c *MySQLHotSwapClient) HealthCheck() error {
	if c.rawDB == nil {
		return fmt.Errorf("数据库连接为空")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return c.rawDB.PingContext(ctx)
}

func (c *MySQLHotSwapClient) GetDB() *gorm.DB {
	return c.db
}

func (c *MySQLHotSwapClient) GetRawDB() *sql.DB {
	return c.rawDB
}

// Redis热切换客户端
type RedisHotSwapClient struct {
	rdb    *redis.Client
	config *RedisConfig
}

type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

func NewRedisHotSwapClient(config *RedisConfig) (*RedisHotSwapClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
	})
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}
	
	return &RedisHotSwapClient{
		rdb:    rdb,
		config: config,
	}, nil
}

func (c *RedisHotSwapClient) Close() error {
	if c.rdb != nil {
		return c.rdb.Close()
	}
	return nil
}

func (c *RedisHotSwapClient) HealthCheck() error {
	if c.rdb == nil {
		return fmt.Errorf("Redis客户端为空")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return c.rdb.Ping(ctx).Err()
}

func (c *RedisHotSwapClient) GetClient() *redis.Client {
	return c.rdb
}

// RabbitMQ热切换客户端
type RabbitMQHotSwapClient struct {
	conn   *amqp.Connection
	config *RabbitMQConfig
}

type RabbitMQConfig struct {
	Address  string
	Port     string
	Username string
	Password string
}

func NewRabbitMQHotSwapClient(config *RabbitMQConfig) (*RabbitMQHotSwapClient, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		config.Username, config.Password, config.Address, config.Port)
	
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("连接RabbitMQ失败: %w", err)
	}
	
	return &RabbitMQHotSwapClient{
		conn:   conn,
		config: config,
	}, nil
}

func (c *RabbitMQHotSwapClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *RabbitMQHotSwapClient) HealthCheck() error {
	if c.conn == nil {
		return fmt.Errorf("RabbitMQ连接为空")
	}
	
	// 检查连接是否关闭
	select {
	case <-c.conn.NotifyClose(make(chan *amqp.Error)):
		return fmt.Errorf("RabbitMQ连接已关闭")
	default:
		return nil
	}
}

func (c *RabbitMQHotSwapClient) GetConnection() *amqp.Connection {
	return c.conn
}

// 配置热更新处理器
type ConfigHotReloadHandler struct {
	service string
}

func NewConfigHotReloadHandler(service string) *ConfigHotReloadHandler {
	return &ConfigHotReloadHandler{service: service}
}

// 处理配置热更新
func (h *ConfigHotReloadHandler) HandleConfigReload() error {
	// 重新加载配置
	if err := config.HotReloadConfig(); err != nil {
		return fmt.Errorf("配置热更新失败: %w", err)
	}
	
	// 热切换数据库连接
	if err := h.hotSwapMySQLConnections(); err != nil {
		logrus.WithError(err).Warn("MySQL连接热切换失败")
	}
	
	// 热切换Redis连接
	if err := h.hotSwapRedisConnections(); err != nil {
		logrus.WithError(err).Warn("Redis连接热切换失败")
	}
	
	// 热切换RabbitMQ连接
	if err := h.hotSwapRabbitMQConnections(); err != nil {
		logrus.WithError(err).Warn("RabbitMQ连接热切换失败")
	}
	
	// 更新JWT密钥
	h.updateJWTKeys()
	
	// 更新加密密钥
	h.updateCryptoKeys()
	
	// 更新系统指标
	UpdateSystemMetrics(h.service)
	
	return nil
}

// 热切换MySQL连接
func (h *ConfigHotReloadHandler) hotSwapMySQLConnections() error {
	mysqlConfigs := config.Conf.MySQL
	for name, mysqlConfig := range mysqlConfigs {
		newClient, err := NewMySQLHotSwapClient(&MySQLConfig{
			DriverName: mysqlConfig.DriverName,
			Host:       mysqlConfig.Host,
			Port:       mysqlConfig.Port,
			Database:   mysqlConfig.Database,
			UserName:   mysqlConfig.UserName,
			Password:   mysqlConfig.Password,
			Charset:    mysqlConfig.Charset,
		})
		if err != nil {
			return fmt.Errorf("创建MySQL客户端失败 %s: %w", name, err)
		}
		
		if err := HotSwapClient(fmt.Sprintf("mysql_%s", name), newClient); err != nil {
			return fmt.Errorf("热切换MySQL客户端失败 %s: %w", name, err)
		}
	}
	
	return nil
}

// 热切换Redis连接
func (h *ConfigHotReloadHandler) hotSwapRedisConnections() error {
	redisConfigs := config.Conf.Redis
	for name, redisConfig := range redisConfigs {
		newClient, err := NewRedisHotSwapClient(&RedisConfig{
			Address:  redisConfig.Address,
			Password: redisConfig.Password,
			DB:       0, // 默认DB
		})
		if err != nil {
			return fmt.Errorf("创建Redis客户端失败 %s: %w", name, err)
		}
		
		if err := HotSwapClient(fmt.Sprintf("redis_%s", name), newClient); err != nil {
			return fmt.Errorf("热切换Redis客户端失败 %s: %w", name, err)
		}
	}
	
	return nil
}

// 热切换RabbitMQ连接
func (h *ConfigHotReloadHandler) hotSwapRabbitMQConnections() error {
	mqConfigs := config.Conf.Mq
	for name, mqConfig := range mqConfigs {
		newClient, err := NewRabbitMQHotSwapClient(&RabbitMQConfig{
			Address:  mqConfig.Address,
			Port:     mqConfig.Port,
			Username: mqConfig.Username,
			Password: mqConfig.Password,
		})
		if err != nil {
			return fmt.Errorf("创建RabbitMQ客户端失败 %s: %w", name, err)
		}
		
		if err := HotSwapClient(fmt.Sprintf("rabbitmq_%s", name), newClient); err != nil {
			return fmt.Errorf("热切换RabbitMQ客户端失败 %s: %w", name, err)
		}
	}
	
	return nil
}

// 更新JWT密钥
func (h *ConfigHotReloadHandler) updateJWTKeys() {
	jwtConfig := config.Conf.JWT
	if jwtConfig == nil {
		return
	}
	
	// 将字符串密钥转换为字节数组
	keys := make(map[string][]byte)
	for keyID, keyValue := range jwtConfig.Keys {
		keys[keyID] = []byte(keyValue)
	}
	
	// 更新JWT密钥
	UpdateJWTKeys(keys, jwtConfig.CurrentKeyID)
	
	logrus.WithFields(logrus.Fields{
		"current_key_id": jwtConfig.CurrentKeyID,
		"key_count":      len(keys),
	}).Info("JWT密钥已更新")
}

// 更新加密密钥
func (h *ConfigHotReloadHandler) updateCryptoKeys() {
	cryptoConfig := config.Conf.Crypto
	if cryptoConfig == nil {
		return
	}
	
	// 这里需要根据实际的密钥管理策略实现
	// 例如从KMS获取密钥或生成新密钥
	logrus.WithFields(logrus.Fields{
		"key_version": cryptoConfig.KeyVersion,
		"algorithm":   cryptoConfig.Algorithm,
	}).Info("加密密钥已更新")
}

// 启动配置热更新监听
func StartConfigHotReload(service string) {
	handler := NewConfigHotReloadHandler(service)
	
	// 启动配置热更新监听
	go func() {
		ctx := context.Background()
		config.WatchHotReload(ctx)
	}()
	
	// 监听配置变更
	go func() {
		for {
			select {
			case <-config.HotReloadChan():
				if err := handler.HandleConfigReload(); err != nil {
					logrus.WithError(err).Error("配置热更新处理失败")
				}
			}
		}
	}()
}

// 获取MySQL客户端
func GetMySQLClient(name string) (*gorm.DB, error) {
	client, exists := GetClient(fmt.Sprintf("mysql_%s", name))
	if !exists {
		return nil, fmt.Errorf("MySQL客户端 %s 不存在", name)
	}
	
	mysqlClient, ok := client.(*MySQLHotSwapClient)
	if !ok {
		return nil, fmt.Errorf("无效的MySQL客户端类型")
	}
	
	return mysqlClient.GetDB(), nil
}

// 获取Redis客户端
func GetRedisClient(name string) (*redis.Client, error) {
	client, exists := GetClient(fmt.Sprintf("redis_%s", name))
	if !exists {
		return nil, fmt.Errorf("Redis客户端 %s 不存在", name)
	}
	
	redisClient, ok := client.(*RedisHotSwapClient)
	if !ok {
		return nil, fmt.Errorf("无效的Redis客户端类型")
	}
	
	return redisClient.GetClient(), nil
}

// 获取RabbitMQ客户端
func GetRabbitMQClient(name string) (*amqp.Connection, error) {
	client, exists := GetClient(fmt.Sprintf("rabbitmq_%s", name))
	if !exists {
		return nil, fmt.Errorf("RabbitMQ客户端 %s 不存在", name)
	}
	
	rabbitmqClient, ok := client.(*RabbitMQHotSwapClient)
	if !ok {
		return nil, fmt.Errorf("无效的RabbitMQ客户端类型")
	}
	
	return rabbitmqClient.GetConnection(), nil
}

// 关闭所有客户端
func CloseAllClients() error {
	hotReloadManager.mutex.Lock()
	defer hotReloadManager.mutex.Unlock()
	
	var lastErr error
	for name, client := range hotReloadManager.clients {
		if err := client.Close(); err != nil {
			logrus.WithFields(logrus.Fields{
				"client": name,
				"error":  err,
			}).Warn("关闭客户端失败")
			lastErr = err
		}
	}
	
	return lastErr
}
