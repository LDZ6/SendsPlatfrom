package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"platform/app/gateway/internal"
	"platform/app/gateway/middlewares"
	"platform/app/gateway/router"
	"platform/app/gateway/rpc"
	"platform/config"
	"platform/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化配置
	config.InitConfig()

	// 初始化日志
	logger := utils.GetLogger()
	logger.Info("启动SendsPlatform网关服务...")

	// 初始化RPC连接
	if err := rpc.Init(); err != nil {
		logger.Fatalf("RPC初始化失败: %v", err)
	}

	// 设置Gin模式
	if config.Conf.Server.Version == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	r := router.Router()

	// 添加健康检查路由
	r.GET("/health", middlewares.HealthCheck())
	r.GET("/ready", middlewares.ReadinessCheck())
	r.GET("/live", middlewares.LivenessCheck())

	// 添加指标收集中间件
	r.Use(middlewares.MetricsMiddleware())

	// 启动后台任务
	go internal.YearBillDataInitSync()

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         config.Conf.Server.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 在单独的信号监听中关闭RPC资源（与HTTP优雅关闭并行）
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		rpc.Close()
	}()

	// 启动优雅关闭
	gracefulShutdown := middlewares.NewGracefulShutdown(server, logger)
	gracefulShutdown.Start()
}
