package dao

import (
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"platform/app/yearBill/database/model"
)

type UserDao struct {
	*gorm.DB
}

func NewUserDao(ctx context.Context) *UserDao {
	if ctx == nil {
		ctx = context.Background()
	}
	return &UserDao{NewDBClient(ctx)}
}

func (dao *UserDao) WithTableName() *gorm.DB {
	return dao.Table("users")
}

func (dao *UserDao) CreateUser(stuNum string) (err error) {
	user := model.User{
		StuNum: stuNum,
	}
	err = dao.WithTableName().
		Create(&user).
		Error
	if err != nil {
		logrus.Info("[DBERROR1]:%v\n", err.Error())
		return
	}
	return
}

func (dao *UserDao) FindUser(stuNum string) (users []model.User, err error) {
	err = dao.WithTableName().
		Where("stu_num=?", stuNum).
		Find(&users).
		Error
	if err != nil {
		logrus.Info("[DBERROR2]:%v\n", err.Error())
		return
	}
	return
}

func (dao *UserDao) UpdateAppraisal(stuNum string, appraisal int64) (err error) {
	err = dao.WithTableName().
		Where("stu_num=?", stuNum).
		Update("appraisal", appraisal).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *UserDao) StorageData(stuNum string, index int64, bill model.Bill, learn model.Learn) (err error) {

	tx := dao.Begin()
	err = tx.Table("bills").Create(&bill).Error
	if err != nil {
		tx.Rollback()
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}

	err = tx.Table("learns").Create(&learn).Error
	if err != nil {
		tx.Rollback()
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}

	err = tx.Model(&model.User{}).
		Where("stu_num = ?", stuNum).
		Updates(map[string]any{"init": 1, "rank": index}).
		Error
	if err != nil {
		tx.Rollback()
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}
	tx.Commit()
	return
}
