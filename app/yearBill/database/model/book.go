package model

import (
	"gorm.io/gorm"
	"time"
)

type Book struct {
	gorm.Model
	StuNum   string    `gorm:"type:varchar(20);unique"` //学号
	Read     int       //借书总数目
	Reading  int       //在阅数目
	BookName string    //曾经借过最长时间的书
	Longest  time.Time //曾经借过最长时间的书的时间
}
