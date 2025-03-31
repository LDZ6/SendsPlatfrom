package dao

import (
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"platform/app/boBing/database/models"
	BoBingPb "platform/idl/pb/boBing"
)

type RecordDao struct {
	*gorm.DB
}

func NewRecordDao(ctx context.Context) *RecordDao {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RecordDao{NewDBClient(ctx)}
}

func (dao *RecordDao) WithTableName() *gorm.DB {
	return dao.Table("records")
}

func (dao *RecordDao) ExistRecord(check string) (cnt int64, err error) {
	err = dao.WithTableName().
		Where("checked=?", check).
		Count(&cnt).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}
func (dao *RecordDao) SaveRecord(req *BoBingPb.BoBingPublishRequest, score int, types string) (err error) {
	record := models.Record{
		Model:   gorm.Model{},
		StuNum:  req.StuNum,
		Score:   score,
		Checked: req.Check,
		Types:   types,
		OpenId:  req.OpenId,
	}
	err = dao.WithTableName().
		Create(&record).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *RecordDao) GetRecordByOpenId(openid string) (records []models.Record, err error) {
	err = dao.WithTableName().
		Where("open_id=?", openid).
		Find(&records).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}
