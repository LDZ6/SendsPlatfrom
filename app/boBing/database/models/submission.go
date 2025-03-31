package models

import (
	"gorm.io/gorm"
	"time"
)

type Submission struct {
	gorm.Model
	RankOpenId string    // 用户，外键
	Date       time.Time // 提交日期
	Count      int       // 提交次数
	Condition  int       // 增加的次数
	TuiWen     int       //推文查看次数
}
