package container

import (
	"database/sql"
	"fmt"
	"time"

	"platform/config"
	"platform/utils"

	_ "github.com/go-sql-driver/mysql"
)

// DatabaseFactory 数据库工厂
type DatabaseFactory struct {
	config *config.Config
}

// NewDatabaseFactory 创建数据库工厂
func NewDatabaseFactory() *DatabaseFactory {
	return &DatabaseFactory{
		config: config.Conf,
	}
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	ServiceName     string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// InitDatabaseForService 为服务初始化数据库连接
func (df *DatabaseFactory) InitDatabaseForService(serviceName string) (*sql.DB, error) {
	// 获取MySQL配置
	mysqlConfig, exists := df.config.MySQL[serviceName]
	if !exists {
		return nil, utils.WrapError(fmt.Errorf("mysql config for %s not found", serviceName),
			utils.ErrCodeConfigError, "数据库配置未找到")
	}

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		mysqlConfig.UserName,
		mysqlConfig.Password,
		mysqlConfig.Host,
		mysqlConfig.Port,
		mysqlConfig.Database,
		mysqlConfig.Charset,
	)

	// 打开数据库连接
	db, err := sql.Open(mysqlConfig.DriverName, dsn)
	if err != nil {
		return nil, utils.WrapDatabaseError(err, "打开数据库连接")
	}

	// 设置连接池参数（优化后的配置）
	db.SetMaxOpenConns(200)                 // 增加最大打开连接数
	db.SetMaxIdleConns(100)                 // 增加最大空闲连接数
	db.SetConnMaxLifetime(30 * time.Minute) // 减少连接最大存活时间
	db.SetConnMaxIdleTime(5 * time.Minute)  // 设置连接最大空闲时间

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, utils.WrapDatabaseError(err, "测试数据库连接")
	}

	return db, nil
}

// InitDatabaseWithConfig 使用自定义配置初始化数据库
func (df *DatabaseFactory) InitDatabaseWithConfig(config DatabaseConfig) (*sql.DB, error) {
	// 获取MySQL配置
	mysqlConfig, exists := df.config.MySQL[config.ServiceName]
	if !exists {
		return nil, utils.WrapError(fmt.Errorf("mysql config for %s not found", config.ServiceName),
			utils.ErrCodeConfigError, "数据库配置未找到")
	}

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		mysqlConfig.UserName,
		mysqlConfig.Password,
		mysqlConfig.Host,
		mysqlConfig.Port,
		mysqlConfig.Database,
		mysqlConfig.Charset,
	)

	// 打开数据库连接
	db, err := sql.Open(mysqlConfig.DriverName, dsn)
	if err != nil {
		return nil, utils.WrapDatabaseError(err, "打开数据库连接")
	}

	// 设置连接池参数
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(200)
	}

	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(100)
	}

	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	} else {
		db.SetConnMaxLifetime(30 * time.Minute)
	}

	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	} else {
		db.SetConnMaxIdleTime(5 * time.Minute)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, utils.WrapDatabaseError(err, "测试数据库连接")
	}

	return db, nil
}

// InitAllServiceDatabases 初始化所有服务的数据库连接
func (df *DatabaseFactory) InitAllServiceDatabases() (map[string]*sql.DB, error) {
	databases := make(map[string]*sql.DB)

	for serviceName := range df.config.MySQL {
		db, err := df.InitDatabaseForService(serviceName)
		if err != nil {
			// 关闭已创建的连接
			for _, d := range databases {
				d.Close()
			}
			return nil, utils.WrapError(err, utils.ErrCodeDatabase, "初始化所有服务数据库失败")
		}
		databases[serviceName] = db
	}

	return databases, nil
}

// GetOptimizedConfig 获取优化的数据库配置
func (df *DatabaseFactory) GetOptimizedConfig(serviceName string) DatabaseConfig {
	return DatabaseConfig{
		ServiceName:     serviceName,
		MaxOpenConns:    200,
		MaxIdleConns:    100,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// GetHighPerformanceConfig 获取高性能数据库配置
func (df *DatabaseFactory) GetHighPerformanceConfig(serviceName string) DatabaseConfig {
	return DatabaseConfig{
		ServiceName:     serviceName,
		MaxOpenConns:    500,
		MaxIdleConns:    200,
		ConnMaxLifetime: 15 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}
}

// GetLowLatencyConfig 获取低延迟数据库配置
func (df *DatabaseFactory) GetLowLatencyConfig(serviceName string) DatabaseConfig {
	return DatabaseConfig{
		ServiceName:     serviceName,
		MaxOpenConns:    100,
		MaxIdleConns:    50,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}
