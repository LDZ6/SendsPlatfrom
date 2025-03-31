package models

import "gorm.io/gorm"

// 用户列表
type User struct {
	gorm.Model
	OpenId       string //用户对应的openid
	StuNum       string //学号
	IsAdmin      int    //管理员标识
	IsSuperAdmin int    `gorm:"column:is_super_admin;type:varchar(255);" json:"IsSuperAdmin"` //超级管理员标识
	Avatar       string //头像
	Nickname     string //昵称
	Organization uint   //组织的id
}
