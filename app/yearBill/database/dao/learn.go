package dao

import (
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"platform/app/yearBill/database/model"
)

type LearnDao struct {
	*gorm.DB
}

func NewLearnDao(ctx context.Context) *LearnDao {
	if ctx == nil {
		ctx = context.Background()
	}
	return &LearnDao{NewDBClient(ctx)}
}

func (dao *LearnDao) WithTableName() *gorm.DB {
	return dao.Table("learns")
}

func (dao *LearnDao) ExistLearnByStuNum(stuNum string) (cnt int64, err error) {
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

func (dao *LearnDao) CreateLearn(stuNum string, data model.Learn) (err error) {

	tx := dao.Begin()
	err = dao.WithTableName().
		Create(&data).
		Error
	if err != nil {
		tx.Rollback()
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}
	err = tx.Model(&model.User{}).
		Where("stu_num = ?", stuNum).
		Update("learn", 1).
		Error
	if err != nil {
		tx.Rollback()
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}
	tx.Commit()
	return
}

func (dao *LearnDao) GetLearn(stuNum string) (learns []model.Learn, err error) {
	err = dao.WithTableName().
		Where("stu_num=?", stuNum).
		Find(&learns).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}
	return
}
