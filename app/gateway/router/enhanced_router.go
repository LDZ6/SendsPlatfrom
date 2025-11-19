package router

import (
	"time"

	"platform/app/gateway/middlewares"
	"platform/app/gateway/types"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// EnhancedRouter 增强的路由器
func EnhancedRouter() *gin.Engine {
	r := gin.New()

	// 基础中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 安全中间件
	r.Use(middlewares.SecurityMiddleware())
	r.Use(middlewares.InputValidationMiddleware())
	r.Use(middlewares.Cors())

	// 错误处理中间件
	r.Use(middlewares.ErrorHandler())
	r.Use(middlewares.ErrorResponseMiddleware())

	// 指标收集中间件
	r.Use(middlewares.MetricsMiddleware())

	// 请求超时中间件
	r.Use(middlewares.RequestTimeout(30 * time.Second))

	// 请求大小限制
	r.Use(middlewares.RequestSizeLimitMiddleware(10 * 1024 * 1024)) // 10MB

	// 内容类型检查
	r.Use(middlewares.ContentTypeMiddleware())

	// API文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// 健康检查
	r.GET("/health", middlewares.EnhancedHealthCheck())
	r.GET("/ready", middlewares.ReadinessCheck())
	r.GET("/live", middlewares.LivenessCheck())

	// API版本管理
	v1 := r.Group("/v1/api")
	registerV1Routes(v1)

	v2 := r.Group("/v2/api")
	registerV2Routes(v2)

	// 管理接口
	admin := r.Group("/admin")
	registerAdminRoutes(admin)

	return r
}

// registerV1Routes 注册v1路由
func registerV1Routes(r *gin.RouterGroup) {
	// 用户相关路由
	user := r.Group("/user")
	user.Use(middlewares.RateLimitMiddleware())
	registerUserV1(user)

	// 博饼游戏路由
	game := r.Group("/game")
	game.Use(middlewares.AuthMassesCheck())
	game.Use(middlewares.RateLimitMiddleware())
	registerGameV1(game)

	// 学校服务路由
	school := r.Group("/school")
	school.Use(middlewares.AuthUserCheck())
	school.Use(middlewares.RateLimitMiddleware())
	registerSchoolV1(school)

	// 年度账单路由
	bill := r.Group("/bill")
	bill.Use(middlewares.AuthUserCheck())
	bill.Use(middlewares.RateLimitMiddleware())
	registerBillV1(bill)

	// WebSocket路由
	ws := r.Group("/ws")
	ws.Use(middlewares.WsHandler())
	registerWebSocketV1(ws)
}

// registerV2Routes 注册v2路由
func registerV2Routes(r *gin.RouterGroup) {
	// v2版本的新功能
	// 这里可以添加新的API版本
}

// registerAdminRoutes 注册管理路由
func registerAdminRoutes(r *gin.RouterGroup) {
	// 管理接口需要特殊权限
	r.Use(middlewares.AuthUserCheck())
	r.Use(middlewares.IPWhitelistMiddleware([]string{"127.0.0.1", "::1"}))

	// 系统监控
	r.GET("/metrics", middlewares.MetricsMiddleware())

	// 系统配置
	r.GET("/config", getSystemConfig)
	r.PUT("/config", updateSystemConfig)

	// 用户管理
	r.GET("/users", getUsers)
	r.GET("/users/:id", getUser)
	r.PUT("/users/:id", updateUser)
	r.DELETE("/users/:id", deleteUser)

	// 游戏管理
	r.GET("/games", getGames)
	r.GET("/games/:id", getGame)
	r.PUT("/games/:id", updateGame)
	r.DELETE("/games/:id", deleteGame)

	// 系统日志
	r.GET("/logs", getSystemLogs)
	r.GET("/logs/:id", getLog)
}

// registerUserV1 注册用户v1路由
func registerUserV1(r *gin.RouterGroup) {
	// 用户认证
	r.POST("/login", validateAndHandle(types.UserLoginRequest{}, handleUserLogin))
	r.POST("/register", validateAndHandle(types.UserRegisterRequest{}, handleUserRegister))
	r.POST("/logout", handleUserLogout)

	// 用户信息
	r.GET("/profile", handleGetProfile)
	r.PUT("/profile", validateAndHandle(types.UserUpdateRequest{}, handleUpdateProfile))

	// 密码管理
	r.POST("/password/change", validateAndHandle(types.PasswordChangeRequest{}, handleChangePassword))
	r.POST("/password/reset", validateAndHandle(types.PasswordResetRequest{}, handleResetPassword))

	// 微信相关
	r.POST("/wechat/login", validateAndHandle(types.WechatLoginRequest{}, handleWechatLogin))
	r.POST("/wechat/bind", validateAndHandle(types.WechatBindRequest{}, handleWechatBind))
}

// registerGameV1 注册游戏v1路由
func registerGameV1(r *gin.RouterGroup) {
	// 博饼游戏
	r.POST("/bobing/init", validateAndHandle(types.BoBingInitRequest{}, handleBoBingInit))
	r.POST("/bobing/play", validateAndHandle(types.BoBingPublishRequest{}, handleBoBingPlay))
	r.GET("/bobing/rank", handleBoBingRank)
	r.GET("/bobing/history", handleBoBingHistory)

	// 游戏统计
	r.GET("/bobing/stats", handleBoBingStats)
	r.GET("/bobing/leaderboard", handleBoBingLeaderboard)
}

// registerSchoolV1 注册学校v1路由
func registerSchoolV1(r *gin.RouterGroup) {
	// 学校认证
	r.POST("/auth", validateAndHandle(types.SchoolAuthRequest{}, handleSchoolAuth))

	// 课表查询
	r.GET("/schedule", validateAndHandle(types.ScheduleQuery{}, handleGetSchedule))

	// 成绩查询
	r.GET("/grades", validateAndHandle(types.GradesQuery{}, handleGetGrades))

	// 学分查询
	r.GET("/credits", validateAndHandle(types.CreditsQuery{}, handleGetCredits))

	// GPA查询
	r.GET("/gpa", validateAndHandle(types.GPAQuery{}, handleGetGPA))
}

// registerBillV1 注册账单v1路由
func registerBillV1(r *gin.RouterGroup) {
	// 年度账单
	r.GET("/year", validateAndHandle(types.YearBillQuery{}, handleGetYearBill))
	r.POST("/year/analyze", validateAndHandle(types.YearBillAnalyzeRequest{}, handleAnalyzeYearBill))

	// 学习数据
	r.GET("/learn", validateAndHandle(types.LearnDataQuery{}, handleGetLearnData))

	// 消费数据
	r.GET("/expense", validateAndHandle(types.ExpenseDataQuery{}, handleGetExpenseData))

	// 排名数据
	r.GET("/rank", validateAndHandle(types.RankQuery{}, handleGetRank))
}

// registerWebSocketV1 注册WebSocket v1路由
func registerWebSocketV1(r *gin.RouterGroup) {
	// 博饼广播
	r.GET("/bobing/broadcast", handleBoBingBroadcast)

	// 实时通知
	r.GET("/notifications", handleNotifications)

	// 在线状态
	r.GET("/status", handleOnlineStatus)
}

// validateAndHandle 验证请求并处理
func validateAndHandle[T any](reqType T, handler func(*gin.Context, T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T

		// 绑定请求数据
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success":   false,
				"errorCode": 1001,
				"message":   "Invalid request format",
				"details":   err.Error(),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 执行验证
		if err := middlewares.ValidateStruct(req); err != nil {
			c.JSON(400, gin.H{
				"success":   false,
				"errorCode": 1001,
				"message":   "Validation failed",
				"details":   err.Error(),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 调用处理函数
		handler(c, req)
	}
}

// 处理函数示例（需要实际实现）
func handleUserLogin(c *gin.Context, req types.UserLoginRequest) {
	// 实现用户登录逻辑
	c.JSON(200, gin.H{"success": true, "message": "Login successful"})
}

func handleUserRegister(c *gin.Context, req types.UserRegisterRequest) {
	// 实现用户注册逻辑
	c.JSON(200, gin.H{"success": true, "message": "Registration successful"})
}

func handleUserLogout(c *gin.Context) {
	// 实现用户登出逻辑
	c.JSON(200, gin.H{"success": true, "message": "Logout successful"})
}

func handleGetProfile(c *gin.Context) {
	// 实现获取用户信息逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"user": "profile"}})
}

func handleUpdateProfile(c *gin.Context, req types.UserUpdateRequest) {
	// 实现更新用户信息逻辑
	c.JSON(200, gin.H{"success": true, "message": "Profile updated"})
}

func handleChangePassword(c *gin.Context, req types.PasswordChangeRequest) {
	// 实现修改密码逻辑
	c.JSON(200, gin.H{"success": true, "message": "Password changed"})
}

func handleResetPassword(c *gin.Context, req types.PasswordResetRequest) {
	// 实现重置密码逻辑
	c.JSON(200, gin.H{"success": true, "message": "Password reset"})
}

func handleWechatLogin(c *gin.Context, req types.WechatLoginRequest) {
	// 实现微信登录逻辑
	c.JSON(200, gin.H{"success": true, "message": "Wechat login successful"})
}

func handleWechatBind(c *gin.Context, req types.WechatBindRequest) {
	// 实现微信绑定逻辑
	c.JSON(200, gin.H{"success": true, "message": "Wechat bind successful"})
}

// 游戏相关处理函数
func handleBoBingInit(c *gin.Context, req types.BoBingInitRequest) {
	// 实现博饼初始化逻辑
	c.JSON(200, gin.H{"success": true, "message": "BoBing initialized"})
}

func handleBoBingPlay(c *gin.Context, req types.BoBingPublishRequest) {
	// 实现博饼游戏逻辑
	c.JSON(200, gin.H{"success": true, "message": "BoBing played"})
}

func handleBoBingRank(c *gin.Context) {
	// 实现博饼排名逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"rank": "data"}})
}

func handleBoBingHistory(c *gin.Context) {
	// 实现博饼历史逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"history": "data"}})
}

func handleBoBingStats(c *gin.Context) {
	// 实现博饼统计逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"stats": "data"}})
}

func handleBoBingLeaderboard(c *gin.Context) {
	// 实现博饼排行榜逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"leaderboard": "data"}})
}

// 学校相关处理函数
func handleSchoolAuth(c *gin.Context, req types.SchoolAuthRequest) {
	// 实现学校认证逻辑
	c.JSON(200, gin.H{"success": true, "message": "School auth successful"})
}

func handleGetSchedule(c *gin.Context, req types.ScheduleQuery) {
	// 实现课表查询逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"schedule": "data"}})
}

func handleGetGrades(c *gin.Context, req types.GradesQuery) {
	// 实现成绩查询逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"grades": "data"}})
}

func handleGetCredits(c *gin.Context, req types.CreditsQuery) {
	// 实现学分查询逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"credits": "data"}})
}

func handleGetGPA(c *gin.Context, req types.GPAQuery) {
	// 实现GPA查询逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"gpa": "data"}})
}

// 账单相关处理函数
func handleGetYearBill(c *gin.Context, req types.YearBillQuery) {
	// 实现年度账单逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"yearbill": "data"}})
}

func handleAnalyzeYearBill(c *gin.Context, req types.YearBillAnalyzeRequest) {
	// 实现年度账单分析逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"analysis": "data"}})
}

func handleGetLearnData(c *gin.Context, req types.LearnDataQuery) {
	// 实现学习数据逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"learn": "data"}})
}

func handleGetExpenseData(c *gin.Context, req types.ExpenseDataQuery) {
	// 实现消费数据逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"expense": "data"}})
}

func handleGetRank(c *gin.Context, req types.RankQuery) {
	// 实现排名数据逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"rank": "data"}})
}

// WebSocket相关处理函数
func handleBoBingBroadcast(c *gin.Context) {
	// 实现博饼广播逻辑
	c.JSON(200, gin.H{"success": true, "message": "Broadcast started"})
}

func handleNotifications(c *gin.Context) {
	// 实现实时通知逻辑
	c.JSON(200, gin.H{"success": true, "message": "Notifications started"})
}

func handleOnlineStatus(c *gin.Context) {
	// 实现在线状态逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"status": "online"}})
}

// 管理相关处理函数
func getSystemConfig(c *gin.Context) {
	// 实现获取系统配置逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"config": "data"}})
}

func updateSystemConfig(c *gin.Context) {
	// 实现更新系统配置逻辑
	c.JSON(200, gin.H{"success": true, "message": "Config updated"})
}

func getUsers(c *gin.Context) {
	// 实现获取用户列表逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"users": "data"}})
}

func getUser(c *gin.Context) {
	// 实现获取用户详情逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"user": "data"}})
}

func updateUser(c *gin.Context) {
	// 实现更新用户逻辑
	c.JSON(200, gin.H{"success": true, "message": "User updated"})
}

func deleteUser(c *gin.Context) {
	// 实现删除用户逻辑
	c.JSON(200, gin.H{"success": true, "message": "User deleted"})
}

func getGames(c *gin.Context) {
	// 实现获取游戏列表逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"games": "data"}})
}

func getGame(c *gin.Context) {
	// 实现获取游戏详情逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"game": "data"}})
}

func updateGame(c *gin.Context) {
	// 实现更新游戏逻辑
	c.JSON(200, gin.H{"success": true, "message": "Game updated"})
}

func deleteGame(c *gin.Context) {
	// 实现删除游戏逻辑
	c.JSON(200, gin.H{"success": true, "message": "Game deleted"})
}

func getSystemLogs(c *gin.Context) {
	// 实现获取系统日志逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"logs": "data"}})
}

func getLog(c *gin.Context) {
	// 实现获取日志详情逻辑
	c.JSON(200, gin.H{"success": true, "data": gin.H{"log": "data"}})
}
