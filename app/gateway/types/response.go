package types

import (
	"time"
)

// 统一响应结构
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	ErrorCode int         `json:"errorCode,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"requestId,omitempty"`
}

// 分页响应
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// 博饼相关响应
type BoBingPublishResponse struct {
	Score      int    `json:"score"`
	Types      string `json:"types"`
	Count      int    `json:"count"`
	Ciphertext string `json:"ciphertext,omitempty"`
}

type BoBingKeyResponse struct {
	Key string `json:"key"`
}

type BoBingTopResponse struct {
	Rank []BoBingRank `json:"rank"`
}

type BoBingRank struct {
	Rank     int    `json:"rank"`
	NickName string `json:"nickName"`
	Score    int64  `json:"score"`
	OpenId   string `json:"openId"`
}

type BoBingMyRank struct {
	Rank     int    `json:"rank"`
	NickName string `json:"nickName"`
	Score    int64  `json:"score"`
}

type BoBingDayRankResponse struct {
	BoBingRank []BoBingRank `json:"boBingRank"`
	BingMyRank BoBingMyRank `json:"bingMyRank"`
}

type BoBingToTalTenResponse struct {
	BoBingRank []BoBingRank `json:"boBingRank"`
	BingMyRank BoBingMyRank `json:"bingMyRank"`
}

type BoBingTianXuanResponse struct {
	BoBingTianXuan []BoBingTianXuan `json:"boBingTianXuan"`
}

type BoBingTianXuan struct {
	NickName string `json:"nickName"`
	Types    string `json:"types"`
	Time     string `json:"time"`
}

type BoBingGetCountResponse struct {
	Count int64 `json:"count"`
}

type BoBingRecordResponse struct {
	Maps map[string]int64 `json:"maps"`
}

type BoBingGetNumberResponse struct {
	Number int64 `json:"number"`
}

// 用户相关响应
type UserLoginResponse struct {
	Token    string    `json:"token"`
	UserInfo UserInfo  `json:"userInfo"`
	Expires  time.Time `json:"expires"`
}

type UserInfo struct {
	OpenId   string `json:"openId"`
	NickName string `json:"nickName"`
	Avatar   string `json:"avatar"`
	StuNum   string `json:"stuNum"`
}

type SchoolUserLoginResponse struct {
	Token    string     `json:"token"`
	UserInfo SchoolUser `json:"userInfo"`
	Expires  time.Time  `json:"expires"`
}

type SchoolUser struct {
	StuNum   string `json:"stuNum"`
	RealName string `json:"realName"`
	Class    string `json:"class"`
	Major    string `json:"major"`
}

type YearBillLoginResponse struct {
	Token    string       `json:"token"`
	UserInfo YearBillUser `json:"userInfo"`
	Expires  time.Time    `json:"expires"`
}

type YearBillUser struct {
	StuNum   string `json:"stuNum"`
	RealName string `json:"realName"`
	Class    string `json:"class"`
	Major    string `json:"major"`
}

// 微信JSSDK响应
type WxJSSDKResponse struct {
	AppId     string `json:"appId"`
	Timestamp string `json:"timestamp"`
	NonceStr  string `json:"nonceStr"`
	Signature string `json:"signature"`
}

// 学校相关响应
type SchoolScheduleResponse struct {
	Schedule []ScheduleItem `json:"schedule"`
	Term     string         `json:"term"`
}

type ScheduleItem struct {
	CourseName string `json:"courseName"`
	Teacher    string `json:"teacher"`
	Location   string `json:"location"`
	Time       string `json:"time"`
	Week       string `json:"week"`
}

type SchoolXueFenResponse struct {
	XueFen []XueFenItem `json:"xueFen"`
	Total  float64      `json:"total"`
}

type XueFenItem struct {
	CourseName string  `json:"courseName"`
	Credit     float64 `json:"credit"`
	Grade      string  `json:"grade"`
	Score      float64 `json:"score"`
}

type SchoolGpaResponse struct {
	GPA   float64 `json:"gpa"`
	Rank  int     `json:"rank"`
	Total int     `json:"total"`
}

type SchoolGradeResponse struct {
	Grades []GradeItem `json:"grades"`
}

type GradeItem struct {
	CourseName string  `json:"courseName"`
	Grade      string  `json:"grade"`
	Score      float64 `json:"score"`
	Credit     float64 `json:"credit"`
	Term       string  `json:"term"`
}

// 年度账单相关响应
type YearBillLearnResponse struct {
	LearnData []LearnItem `json:"learnData"`
	Total     int         `json:"total"`
}

type LearnItem struct {
	CourseName string  `json:"courseName"`
	Credit     float64 `json:"credit"`
	Grade      string  `json:"grade"`
	Score      float64 `json:"score"`
	Term       string  `json:"term"`
}

type YearBillPayResponse struct {
	PayData []PayItem `json:"payData"`
	Total   float64   `json:"total"`
}

type PayItem struct {
	Amount   float64 `json:"amount"`
	Type     string  `json:"type"`
	Date     string  `json:"date"`
	Location string  `json:"location"`
	Balance  float64 `json:"balance"`
}

type YearBillRankResponse struct {
	Rank []RankItem `json:"rank"`
}

type RankItem struct {
	Rank     int     `json:"rank"`
	StuNum   string  `json:"stuNum"`
	RealName string  `json:"realName"`
	Score    float64 `json:"score"`
}

type YearBillAppraiseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// 健康检查响应
type HealthCheckResponse struct {
	Status    string                 `json:"status"`
	Timestamp int64                  `json:"timestamp"`
	Version   string                 `json:"version"`
	Uptime    float64                `json:"uptime"`
	Services  map[string]string      `json:"services"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// 统计响应
type StatisticsResponse struct {
	TotalUsers   int64   `json:"totalUsers"`
	ActiveUsers  int64   `json:"activeUsers"`
	TotalGames   int64   `json:"totalGames"`
	TotalBills   int64   `json:"totalBills"`
	AverageScore float64 `json:"averageScore"`
	SuccessRate  float64 `json:"successRate"`
}

// 创建成功响应
func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// 创建错误响应
func NewErrorResponse(message string, errorCode int) *Response {
	return &Response{
		Success:   false,
		Message:   message,
		ErrorCode: errorCode,
		Timestamp: time.Now().Unix(),
	}
}

// 创建分页响应
func NewPaginatedResponse(data interface{}, page, pageSize int, total int64) *PaginatedResponse {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &PaginatedResponse{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}
