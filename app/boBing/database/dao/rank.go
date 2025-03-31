package dao

import (
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"platform/app/boBing/database/models"
	BoBingPb "platform/idl/pb/boBing"
)

type RankDao struct {
	*gorm.DB
}

func NewRankDao(ctx context.Context) *RankDao {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RankDao{NewDBClient(ctx)}
}

func (dao *RankDao) WithTableName() *gorm.DB {
	return dao.Table("ranks")
}

func (dao *RankDao) ExistRankByOpenId(openid string) (cnt int64, err error) {
	err = dao.WithTableName().
		Where("open_id=?", openid).
		Count(&cnt).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *RankDao) CreatRank(openid string, stuNum string, nickName string, score int) (err error) {
	rank := models.Rank{
		OpenId:   openid,
		StuNum:   stuNum,
		NickName: nickName,
		Score:    score,
	}
	err = dao.WithTableName().
		Create(&rank).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *RankDao) UpdateRank(req *BoBingPb.BoBingPublishRequest, score int, count int, submission models.Submission) (submissionResp models.Submission, err error) {

	//开启事务
	tx := dao.Begin()

	err = tx.Model(&models.Rank{}).
		Where("open_id=?", req.OpenId).
		Updates(map[string]interface{}{
			"score": gorm.Expr("score+?", score),
		}).
		Error
	if err != nil {
		//回滚
		tx.Rollback()
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	if req.Flag == "1" {
		submission.Count += 2
		if submission.Condition+10-submission.Count < 0 {
			submission.Count--
		}
	} else {
		submission.Count += 1
		submission.Condition += count
	}
	err = tx.Model(&models.Submission{}).
		Where("id = ?", submission.ID).
		Updates(map[string]interface{}{
			"count":     submission.Count,
			"condition": submission.Condition,
		}).
		First(&submissionResp).
		Error
	if err != nil {
		//回滚
		tx.Rollback()
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	//提交事务
	tx.Commit()
	return
}

func (dao *RankDao) GetRankByOpenId(openid string) (rank models.Rank, err error) {
	err = dao.WithTableName().
		Where("open_id=?", openid).
		First(&rank).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *RankDao) CountRank() (cnt int64, err error) {
	err = dao.WithTableName().
		Count(&cnt).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}
