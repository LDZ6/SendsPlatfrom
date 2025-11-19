package container

import (
	"database/sql"
	"fmt"
	"sync"

	"platform/config"
	"platform/utils"
)

// DatabaseManager 数据库管理器
type DatabaseManager struct {
	clients map[string]*sql.DB
	mutex   sync.RWMutex
	config  *config.Config
}

// NewDatabaseManager 创建数据库管理器
func NewDatabaseManager() *DatabaseManager {
	return &DatabaseManager{
		clients: make(map[string]*sql.DB),
		config:  config.Conf,
	}
}

// GetClient 获取数据库客户端
func (dm *DatabaseManager) GetClient(name string) (*sql.DB, error) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	if client, exists := dm.clients[name]; exists {
		return client, nil
	}
	return nil, fmt.Errorf("database client %s not found", name)
}

// AddClient 添加数据库客户端
func (dm *DatabaseManager) AddClient(name string, client *sql.DB) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.clients[name] = client
}

// InitClient 初始化数据库客户端
func (dm *DatabaseManager) InitClient(name string) (*sql.DB, error) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// 检查是否已存在
	if client, exists := dm.clients[name]; exists {
		return client, nil
	}

	// 使用数据库工厂创建客户端
	factory := NewDatabaseFactory()
	db, err := factory.InitDatabaseForService(name)
	if err != nil {
		return nil, err
	}

	dm.clients[name] = db
	return db, nil
}

// GetOrCreateClient 获取或创建数据库客户端
func (dm *DatabaseManager) GetOrCreateClient(name string) (*sql.DB, error) {
	// 先尝试获取现有客户端
	if client, err := dm.GetClient(name); err == nil {
		return client, nil
	}

	// 如果不存在，创建新客户端
	return dm.InitClient(name)
}

// Close 关闭所有数据库连接
func (dm *DatabaseManager) Close() error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	var errors []error
	for name, client := range dm.clients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close database %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing databases: %v", errors)
	}

	return nil
}

// HealthCheck 健康检查
func (dm *DatabaseManager) HealthCheck() map[string]interface{} {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	health := make(map[string]interface{})

	for name, client := range dm.clients {
		// 测试连接
		if err := client.Ping(); err != nil {
			health[name] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			// 获取数据库统计信息
			stats := client.Stats()
			health[name] = map[string]interface{}{
				"status":           "healthy",
				"open_connections": stats.OpenConnections,
				"in_use":           stats.InUse,
				"idle":             stats.Idle,
				"wait_count":       stats.WaitCount,
				"wait_duration":    stats.WaitDuration.String(),
			}
		}
	}

	return health
}

// GetStats 获取数据库统计信息
func (dm *DatabaseManager) GetStats() map[string]interface{} {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	stats := make(map[string]interface{})

	for name, client := range dm.clients {
		dbStats := client.Stats()
		stats[name] = map[string]interface{}{
			"max_open_connections": dbStats.MaxOpenConnections,
			"open_connections":     dbStats.OpenConnections,
			"in_use":               dbStats.InUse,
			"idle":                 dbStats.Idle,
			"wait_count":           dbStats.WaitCount,
			"wait_duration":        dbStats.WaitDuration.String(),
			"max_idle_closed":      dbStats.MaxIdleClosed,
			"max_idle_time_closed": dbStats.MaxIdleTimeClosed,
			"max_lifetime_closed":  dbStats.MaxLifetimeClosed,
		}
	}

	return stats
}

// OptimizeConnections 优化连接池
func (dm *DatabaseManager) OptimizeConnections(name string, config DatabaseConfig) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	client, exists := dm.clients[name]
	if !exists {
		return fmt.Errorf("database client %s not found", name)
	}

	// 应用优化配置
	if config.MaxOpenConns > 0 {
		client.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		client.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		client.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		client.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	return nil
}

// WarmupConnections 预热连接池
func (dm *DatabaseManager) WarmupConnections(name string, count int) error {
	dm.mutex.RLock()
	client, exists := dm.clients[name]
	dm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("database client %s not found", name)
	}

	// 预热连接
	for i := 0; i < count; i++ {
		if err := client.Ping(); err != nil {
			return utils.WrapDatabaseError(err, "预热连接")
		}
	}

	return nil
}
