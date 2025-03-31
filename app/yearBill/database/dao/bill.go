package dao

import (
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"platform/app/yearBill/database/model"
)

type BillDao struct {
	*gorm.DB
}

func NewBillDao(ctx context.Context) *BillDao {
	if ctx == nil {
		ctx = context.Background()
	}
	return &BillDao{NewDBClient(ctx)}
}

func (dao *BillDao) WithTableName() *gorm.DB {
	return dao.Table("bills")
}

func (dao *BillDao) ExistBillByStuNum(stuNum string) (cnt int64, err error) {
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

func (dao *BillDao) CreateBill(stuNum string, data model.Bill) (err error) {

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
		Update("bill", 1).
		Error
	if err != nil {
		tx.Rollback()
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}
	tx.Commit()
	return
}

func (dao *BillDao) GetBill(stuNum string) (bill []model.Bill, err error) {
	err = dao.WithTableName().
		Where("stu_num=?", stuNum).
		Find(&bill).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v", err.Error())
		return
	}
	return
}
