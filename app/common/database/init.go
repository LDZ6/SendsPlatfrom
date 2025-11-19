package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"platform/config"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	ServiceName string
	ServiceKey  string
	Models      []interface{}
}

// UnifiedDBInit 统一数据库初始化结构
type UnifiedDBInit struct {
	serviceKey string
	models     []interface{}
	db         *gorm.DB
}

// NewUnifiedDBInit 创建统一数据库初始化实例
func NewUnifiedDBInit(serviceKey string, models []interface{}) *UnifiedDBInit {
	return &UnifiedDBInit{
		serviceKey: serviceKey,
		models:     models,
	}
}

// InitDatabase 初始化数据库（兼容旧接口）
func InitDatabase(config *DatabaseConfig) (*gorm.DB, error) {
	unified := NewUnifiedDBInit(config.ServiceKey, config.Models)
	if err := unified.InitDB(); err != nil {
		return nil, err
	}
	return unified.GetDB(), nil
}

// InitDB 初始化数据库连接
func (u *UnifiedDBInit) InitDB() error {
	mConfig := config.Conf.MySQL[u.serviceKey]
	if mConfig == nil {
		return fmt.Errorf("MySQL配置未找到: %s", u.serviceKey)
	}

	host := mConfig.Host
	port := mConfig.Port
	database := mConfig.Database
	username := mConfig.UserName
	password := mConfig.Password
	charset := mConfig.Charset

	dsn := strings.Join([]string{
		username, ":", password, "@tcp(", host, ":", port, ")/",
		database, "?charset=" + charset + "&parseTime=true&loc=Asia%2FShanghai",
	}, "")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(20)  // 设置连接池，空闲
	sqlDB.SetMaxOpenConns(100) // 打开
	sqlDB.SetConnMaxLifetime(time.Second * 30)

	u.db = db

	// 执行数据库迁移
	if err := u.migration(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	logrus.Infof("%s数据库初始化成功", u.serviceKey)
	return nil
}

// migration 数据库迁移
func (u *UnifiedDBInit) migration() error {
	if len(u.models) == 0 {
		return nil
	}

	// 自动迁移模式
	err := u.db.Set("gorm:table_options", "charset=utf8mb4").
		AutoMigrate(u.models...)

	if err != nil {
		logrus.Error("数据库迁移失败")
		os.Exit(1)
	}

	logrus.Info("数据库迁移成功")
	return nil
}

// GetDB 获取数据库实例
func (u *UnifiedDBInit) GetDB() *gorm.DB {
	return u.db
}

// NewDBClient 创建数据库客户端（兼容旧接口）
func NewDBClient(ctx context.Context, db *gorm.DB) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return db.WithContext(ctx)
}

// NewDBClient 创建数据库客户端
func (u *UnifiedDBInit) NewDBClient(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return u.db.WithContext(ctx)
}

// SetDB 设置数据库实例
func (u *UnifiedDBInit) SetDB(db *gorm.DB) {
	u.db = db
}
