package models

import "gorm.io/gorm"

type Rank struct {
	gorm.Model
	OpenId      string `gorm:"type:varchar(200);unique"`
	StuNum      string
	NickName    string
	Score       int
	Submissions []Submission `gorm:"foreignKey:RankOpenId;references:OpenId"`
}
