package model

import (
	"gorm.io/gorm"
	"time"
)

type Bill struct {
	gorm.Model
	StuNum            string    `gorm:"type:varchar(20);not null;unique"` //学号
	BestRestaurant    string    `gorm:"type:varchar(20);"`                //最喜欢餐厅
	BestRestaurantPay float64   //最喜欢餐厅的花销
	EarlyTime         time.Time //最早用餐
	LastTime          time.Time //最晚用餐
	OtherPay          float64   //除餐厅外其他地点的花销
	LibraryPay        float64   //图书馆借书产生的费用
	RestaurantPay     float64   //餐厅消费
}

type BillCache struct {
	BestRestaurant    string    //最喜欢餐厅
	BestRestaurantPay float64   //最喜欢餐厅的花销
	EarlyTime         time.Time //最早用餐
	LastTime          time.Time //最晚用餐
	OtherPay          float64   //其他消费
	RestaurantPay     float64   //餐厅消费
	LibraryPay        float64   //图书馆借书的花销
}
