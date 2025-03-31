package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	StuNum    string `gorm:"type:varchar(20);not null;unique"` //学号
	Init      int    //数据初始化是否成功
	Appraisal int64  //评价
	Rank      int64  //第几位
}

type DadaInitTask struct {
	JsSessionId  string
	HallTicket   string
	GsSession    string
	Emaphome_WEU string
	StuNum       string
}

type RankCache struct {
	ID        uint
	Appraisal int64
}
