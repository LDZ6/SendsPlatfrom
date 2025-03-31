package router

import (
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "platform/app/gateway/cmd/docs"
	"platform/app/gateway/internal"
	"platform/app/gateway/middlewares"
)

func Router() *gin.Engine {
	r := gin.Default()
	r.Use(middlewares.Cors())
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	ping := r.Group("/ping")
	registerPing(ping)

	user := r.Group("/user")
	registerUser(user)

	boBing := r.Group("/boBing", middlewares.AuthMassesCheck())
	registerBoBing(boBing)

	ws := r.Group("/ws", middlewares.WsHandler())
	registerWs(ws)

	school := r.Group("/school")
	school.Use(middlewares.LimiterHandler(20)).
		Use(middlewares.AuthUserCheck())
	registerSchool(school)

	yearBill := r.Group("/yearBill", middlewares.AuthUserCheck())
	registerYearBill(yearBill)
	r.GET("/yearBill/init", middlewares.WebsocketMiddleware(), internal.YearBillDataInit)

	return r
}

func registerWs(group *gin.RouterGroup) {
	group.GET("/broadcast", internal.BoBingBroadcastMessage)
}
func registerPing(group *gin.RouterGroup) {
	group.GET("/user", internal.UserPing)
	group.GET("/boBing", internal.BoBingPing)
	group.GET("/yearBill", internal.YearBillPing)
}

func registerUser(group *gin.RouterGroup) {
	group.POST("/login", internal.UserLogin)
	group.POST("/jssdk", internal.WxJSSDK)
	group.POST("/school_login", internal.SchoolUserLogin)
	group.POST("/bill_login", internal.YearBillLogin)
}

func registerBoBing(group *gin.RouterGroup) {
	group.GET("/top", internal.BoBingToTalTen)
	group.GET("/dayRank", internal.BoBingDayRank)
	group.POST("/publish", internal.BoBingPublish)
	group.GET("/key", internal.BoBingKey)
	group.GET("/init", internal.BoBingDayInit)
	group.GET("/tianXuan", internal.BoBingTianXuan)
	group.GET("/getCount", internal.BoBingGetCount)
	group.GET("/addCount", internal.BoBingRetransmission)
	group.GET("/record", internal.BoBingRecord)
}

func registerSchool(group *gin.RouterGroup) {
	group.POST("/schedule", internal.SchoolSchedule)
	group.GET("/xuefen", internal.SchoolXueFen)
	group.GET("/gpa", internal.SchoolGpa)
	group.GET("/grade", internal.SchoolGrade)
}

func registerYearBill(group *gin.RouterGroup) {
	group.GET("/learn", internal.GetLearnData)
	group.GET("/pay", internal.GetPayData)
	group.GET("/rank", internal.GetRank)
	group.POST("/appraise", internal.Appraise)
}
