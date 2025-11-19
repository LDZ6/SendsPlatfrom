package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"platform/utils"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// GracefulShutdown 优雅关闭中间件
type GracefulShutdown struct {
	server   *http.Server
	shutdown chan os.Signal
	logger   *logrus.Logger
}

// NewGracefulShutdown 创建优雅关闭实例
func NewGracefulShutdown(server *http.Server, logger *logrus.Logger) *GracefulShutdown {
	return &GracefulShutdown{
		server:   server,
		shutdown: make(chan os.Signal, 1),
		logger:   logger,
	}
}

// Start 启动优雅关闭监听
func (gs *GracefulShutdown) Start() {
	// 监听系统信号
	signal.Notify(gs.shutdown, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器
	go func() {
		gs.logger.Info("服务器启动中...")
		if err := gs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			gs.logger.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待关闭信号
	<-gs.shutdown
	gs.logger.Info("收到关闭信号，开始优雅关闭...")

	// 创建关闭上下文，30秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭服务器
	if err := gs.server.Shutdown(ctx); err != nil {
		gs.logger.Errorf("服务器关闭失败: %v", err)
	} else {
		gs.logger.Info("服务器已优雅关闭")
	}
}

// HealthCheck 健康检查中间件
func HealthCheck() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 带超时检查关键依赖
		type depStatus struct {
			name string
			err  error
		}
		results := make(chan depStatus, 4)

		// MySQL ping (example: user db)
		go func() {
			raw := "mysql_user"
			if db, err := utils.GetMySQLClient("user"); err != nil {
				results <- depStatus{raw, err}
			} else if sqlDB, _ := db.DB(); sqlDB == nil {
				results <- depStatus{raw, fmt.Errorf("nil sqlDB")}
			} else {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
				defer cancel()
				results <- depStatus{raw, sqlDB.PingContext(ctx)}
			}
		}()

		// Redis ping (example: user cache via hot reload accessor if present)
		go func() {
			raw := "redis_user"
			rdb, err := utils.GetRedisClient("user")
			if err != nil {
				results <- depStatus{raw, err}
				return
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			results <- depStatus{raw, rdb.Ping(ctx).Err()}
		}()

		// RabbitMQ simple check if configured
		go func() {
			raw := "rabbitmq_default"
			conn, err := utils.GetRabbitMQClient("default")
			if err != nil {
				results <- depStatus{raw, err}
				return
			}
			select {
			case <-conn.NotifyClose(make(chan *amqp.Error)):
				results <- depStatus{raw, fmt.Errorf("mq closed")}
			default:
				results <- depStatus{raw, nil}
			}
		}()

		// Collect with overall timeout
		deps := map[string]string{"mysql_user": "ok", "redis_user": "ok", "rabbitmq_default": "ok"}
		deadline := time.After(4 * time.Second)
		pending := 3
		var firstErr error
		for pending > 0 {
			select {
			case r := <-results:
				if r.err != nil {
					deps[r.name] = r.err.Error()
					if firstErr == nil {
						firstErr = r.err
					}
				}
				pending--
			case <-deadline:
				firstErr = fmt.Errorf("health timeout")
				break
			}
		}

		status := http.StatusOK
		if firstErr != nil {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, gin.H{
			"status":    map[bool]string{true: "healthy", false: "degraded"}[firstErr == nil],
			"timestamp": time.Now().Unix(),
			"deps":      deps,
		})
	})
}

// ReadinessCheck 就绪检查中间件
func ReadinessCheck() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 仅就绪关键依赖（更严格）
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		ready := true
		details := map[string]string{}

		if db, err := utils.GetMySQLClient("user"); err != nil {
			ready = false
			details["mysql_user"] = err.Error()
		} else if sqlDB, _ := db.DB(); sqlDB == nil {
			ready = false
			details["mysql_user"] = "nil sqlDB"
		} else if err := sqlDB.PingContext(ctx); err != nil {
			ready = false
			details["mysql_user"] = err.Error()
		}

		if rdb, err := utils.GetRedisClient("user"); err != nil {
			ready = false
			details["redis_user"] = err.Error()
		} else if err := rdb.Ping(ctx).Err(); err != nil {
			ready = false
			details["redis_user"] = err.Error()
		}

		code := http.StatusOK
		if !ready {
			code = http.StatusServiceUnavailable
		}
		c.JSON(code, gin.H{
			"status":    map[bool]string{true: "ready", false: "not_ready"}[ready],
			"timestamp": time.Now().Unix(),
			"deps":      details,
		})
	})
}

// LivenessCheck 存活检查中间件
func LivenessCheck() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 简单的存活检查
		c.JSON(http.StatusOK, gin.H{
			"status":    "alive",
			"timestamp": time.Now().Unix(),
		})
	})
}

// MetricsMiddleware 指标收集中间件
func MetricsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start)
		status := c.Writer.Status()

		// 这里可以集成Prometheus等监控系统
		fmt.Printf("请求耗时: %v, 状态码: %d, 路径: %s\n",
			duration, status, c.Request.URL.Path)
	})
}
