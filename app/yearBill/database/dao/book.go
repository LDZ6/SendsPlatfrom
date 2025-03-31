package dao

import (
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"platform/app/yearBill/database/model"
	"platform/app/yearBill/types"
)

type BookDao struct {
	*gorm.DB
}

func NewBookDao(ctx context.Context) *BookDao {
	if ctx == nil {
		ctx = context.Background()
	}
	return &BookDao{NewDBClient(ctx)}
}

func (dao *BookDao) WithTableName() *gorm.DB {
	return dao.Table("books")
}

func (dao *BookDao) ExistBookByStuNum(stuNum string) (cnt int64, err error) {
	err = dao.WithTableName().
		Where("stu_num=?", stuNum).
		Count(&cnt).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}
	return
}

func (dao *BookDao) CreateBook(stuNum string, data types.BookData) (err error) {

	tx := dao.Begin()
	Book := model.Book{
		StuNum:   stuNum,
		Read:     data.Read,
		Reading:  data.Reading,
		BookName: data.BookName,
		Longest:  data.Longest,
	}
	err = dao.WithTableName().
		Create(&Book).
		Error
	if err != nil {
		tx.Rollback()
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	err = tx.Model(&model.User{}).
		Where("stu_num = ?", stuNum).
		Update("book", 1).
		Error
	if err != nil {
		tx.Rollback()
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}
	tx.Commit()
	return
}
