package types

import (
	"time"
)

// BoBingPublishRequest 博饼发布请求
type BoBingPublishRequest struct {
	OpenId   string `json:"openId" validate:"required,openid"`
	StuNum   string `json:"stuNum" validate:"required,stu_num"`
	NickName string `json:"nickName" validate:"required,nickname"`
	Flag     string `json:"flag" validate:"required,oneof=1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16"`
	Check    string `json:"check" validate:"required"`
}

// BoBingInitRequest 博饼初始化请求
type BoBingInitRequest struct {
	OpenId   string `json:"openId" validate:"required,openid"`
	StuNum   string `json:"stuNum" validate:"required,stu_num"`
	NickName string `json:"nickName" validate:"required,nickname"`
}

// BoBingKeyRequest 博饼密钥请求
type BoBingKeyRequest struct {
	OpenId string `json:"openId" validate:"required,openid"`
}

// BoBingBroadcastCheckRequest 博饼广播检查请求
type BoBingBroadcastCheckRequest struct {
	OpenId     string `json:"openId" validate:"required,openid"`
	Ciphertext string `json:"ciphertext" validate:"required"`
}

// UserLoginRequest 用户登录请求
type UserLoginRequest struct {
	Code string `json:"code" validate:"required,min=1,max=100"`
}

// SchoolUserLoginRequest 学校用户登录请求
type SchoolUserLoginRequest struct {
	StuNum   string `json:"stuNum" validate:"required,stu_num"`
	Password string `json:"password" validate:"required,min=6,max=50"`
}

// YearBillLoginRequest 年度账单登录请求
type YearBillLoginRequest struct {
	StuNum   string `json:"stuNum" validate:"required,stu_num"`
	Password string `json:"password" validate:"required,min=6,max=50"`
}

// WxJSSDKRequest 微信JSSDK请求
type WxJSSDKRequest struct {
	URL string `json:"url" validate:"required,url"`
}

// SchoolScheduleRequest 学校课表请求
type SchoolScheduleRequest struct {
	StuNum string `json:"stuNum" validate:"required,stu_num"`
	Term   string `json:"term" validate:"required,min=1,max=20"`
}

// YearBillAppraiseRequest 年度账单评价请求
type YearBillAppraiseRequest struct {
	StuNum  string `json:"stuNum" validate:"required,stu_num"`
	Content string `json:"content" validate:"required,min=1,max=500"`
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
}

// 查询参数结构体
type BoBingTopQuery struct {
	Limit int `form:"limit" validate:"min=1,max=100"`
}

type BoBingDayRankQuery struct {
	Date string `form:"date" validate:"omitempty,datetime=2006-01-02"`
}

type BoBingRecordQuery struct {
	OpenId string `form:"openId" validate:"required,openid"`
}

// 分页请求
type PaginationRequest struct {
	Page     int `form:"page" validate:"min=1"`
	PageSize int `form:"pageSize" validate:"min=1,max=100"`
}

// 时间范围请求
type TimeRangeRequest struct {
	StartTime time.Time `form:"startTime" validate:"required"`
	EndTime   time.Time `form:"endTime" validate:"required"`
}

// 搜索请求
type SearchRequest struct {
	Keyword string `form:"keyword" validate:"required,min=1,max=100"`
	Type    string `form:"type" validate:"omitempty,oneof=user game bill"`
}

// 排序请求
type SortRequest struct {
	Field string `form:"field" validate:"required"`
	Order string `form:"order" validate:"required,oneof=asc desc"`
}

// 过滤请求
type FilterRequest struct {
	Status string `form:"status" validate:"omitempty,oneof=active inactive pending"`
	Type   string `form:"type" validate:"omitempty,oneof=normal premium"`
}
