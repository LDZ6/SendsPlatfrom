package models

import "gorm.io/gorm"

type Record struct {
	gorm.Model
	OpenId  string
	StuNum  string
	Score   int
	Types   string
	Checked string
}
