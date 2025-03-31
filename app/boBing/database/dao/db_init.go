package dao

import (
	"context"
	"fmt"
	"os"
	"platform/app/boBing/database/models"
	"platform/config"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var _db *gorm.DB

func InitDB() {
	mConfig := config.Conf.MySQL["bobing"]
	host := mConfig.Host
	port := mConfig.Port
	database := mConfig.Database
	username := mConfig.UserName
	password := mConfig.Password
	charset := mConfig.Charset
	dsn := strings.Join([]string{username, ":", password, "@tcp(", host, ":", port, ")/", database, "?charset=" + charset + "&parseTime=true&loc=Asia%2FShanghai"}, "")
	err := Database(dsn)
	if err != nil {
		fmt.Println(err)
	}
}

func migration() {
	// 自动迁移模式
	err := _db.Set("gorm:table_options", "charset=utf8mb4").
		AutoMigrate(
			&models.Rank{},
			&models.Record{},
			&models.Submission{})
	//Create(&models.Rank{}).Create(
	//&models.Record{}).Create(
	//&models.Submission{})
	if err != nil {
		logrus.Info("register table fail")
		os.Exit(0)
	}
	logrus.Info("register table success")
}

func Database(connString string) (err error) {
	db, err := gorm.Open(mysql.Open(connString), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(20)  // 设置连接池，空闲
	sqlDB.SetMaxOpenConns(100) // 打开
	sqlDB.SetConnMaxLifetime(time.Second * 30)
	_db = db
	migration()
	//user := models.User{IsAdmin: 0, Organization: 0}
	//db.Where("is_admin=?", 1).Updates(user)
	return err
}

func NewDBClient(ctx context.Context) *gorm.DB {
	db := _db
	return db.WithContext(ctx)
}
