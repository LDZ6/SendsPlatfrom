package dao

import (
	"context"
	"platform/app/boBing/database/models"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SubmissionDao struct {
	*gorm.DB
}

func NewSubmissionDao(ctx context.Context) *SubmissionDao {
	if ctx == nil {
		ctx = context.Background()
	}
	return &SubmissionDao{NewDBClient(ctx)}
}

func (dao *SubmissionDao) WithTableName() *gorm.DB {
	return dao.Table("submissions")
}

func (dao *SubmissionDao) ExistDaySubmissionByOpenId(openid string, today time.Time) (cnt int64, err error) {
	err = dao.WithTableName().
		Where("rank_open_id=? AND date=?", openid, today).
		Count(&cnt).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *SubmissionDao) GetDaySubmissionByOpenId(openid string, today time.Time) (submission models.Submission, err error) {
	logrus.Info(today)
	err = dao.WithTableName().
		Where("rank_open_id=? AND date=?", openid, today).
		First(&submission).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *SubmissionDao) CreateDaySubmission(openid string, rank models.Rank, today time.Time) (err error) {
	submission := models.Submission{
		RankOpenId: openid,
		Date:       today,
		Count:      0,
		Condition:  0,
		TuiWen:     3,
	}
	err = dao.Model(&rank).
		Association("Submissions").
		Append(&submission)
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}

func (dao *SubmissionDao) UpdateSubmission(submission models.Submission) (err error) {
	logrus.Info(submission.TuiWen)
	err = dao.WithTableName().
		Where("rank_open_id=? AND date=?", submission.RankOpenId, submission.Date).
		Updates(map[string]interface{}{
			"condition": submission.Condition,
			"tui_wen":   submission.TuiWen,
		}).
		Error
	if err != nil {
		logrus.Info("[DBERROR]:%v\n", err.Error())
		return
	}
	return
}
