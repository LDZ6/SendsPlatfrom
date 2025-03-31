package model

import "gorm.io/gorm"

type Learn struct {
	gorm.Model
	StuNum     string `gorm:"type:varchar(20);not null;unique"` //学号
	MostCourse string //上得最多的课程名
	Most       int64  //上得最多的课的节数
	Eight      int64  //早八次数
	Ten        int64  //晚十次数
	SumLesson  int64  //上课总节数
}

type LearnCache struct {
	MostCourse string
	Eight      int64
	Ten        int64
	Most       int64
	SumLesson  int64
}
