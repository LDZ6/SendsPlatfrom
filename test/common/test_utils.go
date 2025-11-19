package common

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"platform/app/common/container"
	"platform/config"
	"platform/utils"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestEnvironment 测试环境
type TestEnvironment struct {
	DB        *gorm.DB
	Redis     *redis.Client
	Container *container.ServiceContainer
	Cleanup   func()
}

// SetupTestEnvironment 设置测试环境
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	// 设置测试配置
	config.Conf = &config.Config{
		Server: &config.Server{
			Port:    "8080",
			Version: "test",
		},
		MySQL: map[string]*config.MySQL{
			"test": {
				DriverName: "sqlite",
				Database:   ":memory:",
			},
		},
		Redis: map[string]*config.Redis{
			"test": {
				Address: "localhost:6379",
			},
		},
	}

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 创建Redis客户端（使用测试数据库）
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration tests")
	}

	// 创建服务容器
	container := container.GetServiceContainer()

	// 清理函数
	cleanup := func() {
		// 清理数据库
		if db != nil {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}

		// 清理Redis
		if redisClient != nil {
			redisClient.FlushDB(ctx)
			redisClient.Close()
		}

		// 清理容器
		if container != nil {
			container.Shutdown()
		}
	}

	return &TestEnvironment{
		DB:        db,
		Redis:     redisClient,
		Container: container,
		Cleanup:   cleanup,
	}
}

// MockDatabase 模拟数据库
type MockDatabase struct {
	users map[string]interface{}
	mutex chan struct{}
}

// NewMockDatabase 创建模拟数据库
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		users: make(map[string]interface{}),
		mutex: make(chan struct{}, 1),
	}
}

// Get 获取数据
func (m *MockDatabase) Get(key string) (interface{}, bool) {
	m.mutex <- struct{}{}
	defer func() { <-m.mutex }()

	value, exists := m.users[key]
	return value, exists
}

// Set 设置数据
func (m *MockDatabase) Set(key string, value interface{}) {
	m.mutex <- struct{}{}
	defer func() { <-m.mutex }()

	m.users[key] = value
}

// Delete 删除数据
func (m *MockDatabase) Delete(key string) {
	m.mutex <- struct{}{}
	defer func() { <-m.mutex }()

	delete(m.users, key)
}

// MockCache 模拟缓存
type MockCache struct {
	data  map[string]interface{}
	mutex chan struct{}
}

// NewMockCache 创建模拟缓存
func NewMockCache() *MockCache {
	return &MockCache{
		data:  make(map[string]interface{}),
		mutex: make(chan struct{}, 1),
	}
}

// Get 获取缓存
func (m *MockCache) Get(key string) (interface{}, bool) {
	m.mutex <- struct{}{}
	defer func() { <-m.mutex }()

	value, exists := m.data[key]
	return value, exists
}

// Set 设置缓存
func (m *MockCache) Set(key string, value interface{}) {
	m.mutex <- struct{}{}
	defer func() { <-m.mutex }()

	m.data[key] = value
}

// Delete 删除缓存
func (m *MockCache) Delete(key string) {
	m.mutex <- struct{}{}
	defer func() { <-m.mutex }()

	delete(m.data, key)
}

// TestCase 测试用例
type TestCase struct {
	Name        string
	Input       interface{}
	Expected    interface{}
	ExpectedErr bool
	Setup       func()
	Cleanup     func()
}

// RunTestCases 运行测试用例
func RunTestCases(t *testing.T, testCases []TestCase, testFunc func(t *testing.T, tc TestCase)) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Setup != nil {
				tc.Setup()
			}

			if tc.Cleanup != nil {
				defer tc.Cleanup()
			}

			testFunc(t, tc)
		})
	}
}

// AssertError 断言错误
func AssertError(t *testing.T, err error, expected bool) {
	if expected {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}

// AssertAppError 断言应用错误
func AssertAppError(t *testing.T, err error, expectedCode utils.ErrorCode) {
	require.Error(t, err)

	appErr, ok := err.(*utils.AppError)
	require.True(t, ok, "Expected AppError, got %T", err)
	assert.Equal(t, expectedCode, appErr.Code)
}

// BenchmarkTestCase 基准测试用例
type BenchmarkTestCase struct {
	Name     string
	Input    interface{}
	Setup    func()
	Cleanup  func()
	TestFunc func(b *testing.B, input interface{})
}

// RunBenchmarkTestCases 运行基准测试用例
func RunBenchmarkTestCases(b *testing.B, testCases []BenchmarkTestCase) {
	for _, tc := range testCases {
		b.Run(tc.Name, func(b *testing.B) {
			if tc.Setup != nil {
				tc.Setup()
			}

			if tc.Cleanup != nil {
				defer tc.Cleanup()
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tc.TestFunc(b, tc.Input)
			}
		})
	}
}

// LoadTestData 加载测试数据
func LoadTestData(t *testing.T, db *gorm.DB, data interface{}) {
	err := db.Create(data).Error
	require.NoError(t, err)
}

// CleanTestData 清理测试数据
func CleanTestData(t *testing.T, db *gorm.DB, table string) {
	err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error
	require.NoError(t, err)
}

// CreateTestUser 创建测试用户
func CreateTestUser(t *testing.T, db *gorm.DB) map[string]interface{} {
	user := map[string]interface{}{
		"id":         "test_user_1",
		"username":   "testuser",
		"email":      "test@example.com",
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	err := db.Table("users").Create(user).Error
	require.NoError(t, err)

	return user
}

// CreateTestGame 创建测试游戏
func CreateTestGame(t *testing.T, db *gorm.DB) map[string]interface{} {
	game := map[string]interface{}{
		"id":         "test_game_1",
		"type":       "bobing",
		"user_id":    "test_user_1",
		"score":      100,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	err := db.Table("games").Create(game).Error
	require.NoError(t, err)

	return game
}

// TestHTTPClient 测试HTTP客户端
type TestHTTPClient struct {
	BaseURL string
	Headers map[string]string
}

// NewTestHTTPClient 创建测试HTTP客户端
func NewTestHTTPClient(baseURL string) *TestHTTPClient {
	return &TestHTTPClient{
		BaseURL: baseURL,
		Headers: make(map[string]string),
	}
}

// SetHeader 设置请求头
func (c *TestHTTPClient) SetHeader(key, value string) {
	c.Headers[key] = value
}

// Get 发送GET请求
func (c *TestHTTPClient) Get(path string) (int, string, error) {
	// 这里应该实现实际的HTTP请求
	// 为了示例，我们返回模拟数据
	return 200, `{"success": true}`, nil
}

// Post 发送POST请求
func (c *TestHTTPClient) Post(path string, data interface{}) (int, string, error) {
	// 这里应该实现实际的HTTP请求
	// 为了示例，我们返回模拟数据
	return 200, `{"success": true}`, nil
}

// TestConfig 测试配置
type TestConfig struct {
	DatabaseURL string
	RedisURL    string
	LogLevel    string
}

// GetTestConfig 获取测试配置
func GetTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL: os.Getenv("TEST_DATABASE_URL"),
		RedisURL:    os.Getenv("TEST_REDIS_URL"),
		LogLevel:    "debug",
	}
}

// SkipIfNotIntegration 如果不是集成测试则跳过
func SkipIfNotIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}
}

// SkipIfNoRedis 如果没有Redis则跳过
func SkipIfNoRedis(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	client.Close()
}

// SkipIfNoDatabase 如果没有数据库则跳过
func SkipIfNoDatabase(t *testing.T) {
	// 这里应该检查数据库连接
	// 为了示例，我们检查环境变量
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("Database not available")
	}
}
