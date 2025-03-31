package dao

import (
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"platform/app/user/database/models"
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

func (dao *UserDao) ExistUserByOpenid(openid string) (user []*models.User, cnt int64, err error) {
	err = dao.WithTableName().
		Where("open_id = ?", openid).
		Find(&user).
		Count(&cnt).
		Error
	return
}

func (dao *UserDao) FindUserByStuNum(stuNum string) (user []*models.User, cnt int64, err error) {
	err = dao.WithTableName().
		Where("stu_num=?", stuNum).
		Find(&user).
		Count(&cnt).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *UserDao) CreateUser(user *models.User) (err error) {
	err = dao.DB.Create(&user).Error
	return
}
