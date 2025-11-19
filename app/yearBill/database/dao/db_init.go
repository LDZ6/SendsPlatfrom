package dao

import (
	"context"
	"platform/app/common/database"
	"platform/app/yearBill/database/model"

	"gorm.io/gorm"
)

var _db *gorm.DB
var dbInit *database.UnifiedDBInit

// InitDB 初始化数据库
func InitDB() error {
	// 定义年度账单服务的模型
	models := []interface{}{
		&model.Learn{},
		&model.Book{},
		&model.Bill{},
		&model.User{},
	}

	// 创建统一数据库初始化实例
	dbInit = database.NewUnifiedDBInit("year_bill", models)

	// 初始化数据库
	err := dbInit.InitDB()
	if err != nil {
		return err
	}

	_db = dbInit.GetDB()
	return nil
}

// NewDBClient 创建数据库客户端
func NewDBClient(ctx context.Context) *gorm.DB {
	if dbInit != nil {
		return dbInit.NewDBClient(ctx)
	}
	if _db != nil {
		return _db.WithContext(ctx)
	}
	return nil
}

// SetDB 设置数据库实例
func SetDB(db *gorm.DB) {
	_db = db
	if dbInit != nil {
		dbInit.SetDB(db)
	}
}
