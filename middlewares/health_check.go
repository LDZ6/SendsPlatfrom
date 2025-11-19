package middlewares

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"platform/app/common/container"
	"platform/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var startTime = time.Now()

// HealthCheck 健康检查中间件
func HealthCheck() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		health := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
			"uptime":    time.Since(startTime).Seconds(),
		}

		// 检查依赖服务
		deps := make(map[string]string)

		// 检查数据库
		if err := checkDatabase(); err != nil {
			deps["database"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			deps["database"] = "healthy"
		}

		// 检查Redis
		if err := checkRedis(); err != nil {
			deps["redis"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			deps["redis"] = "healthy"
		}

		// 检查系统资源
		systemInfo := getSystemInfo()
		health["system"] = systemInfo

		// 检查服务容器
		container := container.GetServiceContainer()
		containerHealth := container.HealthCheck()
		health["container"] = containerHealth

		health["dependencies"] = deps

		status := http.StatusOK
		if health["status"] == "degraded" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, health)
	})
}

// ReadinessCheck 就绪检查中间件
func ReadinessCheck() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		ready := true
		checks := make(map[string]bool)

		// 检查数据库连接
		if err := checkDatabase(); err != nil {
			checks["database"] = false
			ready = false
		} else {
			checks["database"] = true
		}

		// 检查Redis连接
		if err := checkRedis(); err != nil {
			checks["redis"] = false
			ready = false
		} else {
			checks["redis"] = true
		}

		// 检查服务容器
		container := container.GetServiceContainer()
		containerHealth := container.HealthCheck()
		checks["container"] = containerHealth["status"] == "healthy"

		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, gin.H{
			"ready":     ready,
			"checks":    checks,
			"timestamp": time.Now().Unix(),
		})
	})
}

// LivenessCheck 存活检查中间件
func LivenessCheck() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 简单的存活检查，只检查应用是否在运行
		c.JSON(http.StatusOK, gin.H{
			"alive":     true,
			"timestamp": time.Now().Unix(),
			"uptime":    time.Since(startTime).Seconds(),
		})
	})
}

// checkDatabase 检查数据库连接
func checkDatabase() error {
	// 这里应该检查实际的数据库连接
	// 为了示例，我们使用配置中的数据库信息
	if config.Conf == nil {
		return fmt.Errorf("configuration not initialized")
	}

	// 检查是否有数据库配置
	if len(config.Conf.MySQL) == 0 {
		return fmt.Errorf("no database configuration found")
	}

	// TODO: 实际检查数据库连接
	// 这里应该尝试连接数据库并执行简单查询

	return nil
}

// checkRedis 检查Redis连接
func checkRedis() error {
	// 这里应该检查实际的Redis连接
	// 为了示例，我们使用配置中的Redis信息
	if config.Conf == nil {
		return fmt.Errorf("configuration not initialized")
	}

	// 检查是否有Redis配置
	if len(config.Conf.Redis) == 0 {
		return fmt.Errorf("no redis configuration found")
	}

	// TODO: 实际检查Redis连接
	// 这里应该尝试连接Redis并执行简单命令

	return nil
}

// getSystemInfo 获取系统信息
func getSystemInfo() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"memory": map[string]interface{}{
			"alloc":       m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":         m.Sys,
			"num_gc":      m.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
		"cpu_count":  runtime.NumCPU(),
	}
}

// EnhancedHealthCheck 增强的健康检查
func EnhancedHealthCheck() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		health := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
			"uptime":    time.Since(startTime).Seconds(),
		}

		// 检查依赖服务
		deps := make(map[string]string)

		// 检查数据库
		if err := checkDatabase(); err != nil {
			deps["database"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			deps["database"] = "healthy"
		}

		// 检查Redis
		if err := checkRedis(); err != nil {
			deps["redis"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			deps["redis"] = "healthy"
		}

		// 检查系统资源
		systemInfo := getSystemInfo()
		health["system"] = systemInfo

		// 检查服务容器
		container := container.GetServiceContainer()
		containerHealth := container.HealthCheck()
		health["container"] = containerHealth

		// 检查业务指标
		businessMetrics := getBusinessMetrics()
		health["business"] = businessMetrics

		health["dependencies"] = deps

		status := http.StatusOK
		if health["status"] == "degraded" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, health)
	})
}

// getBusinessMetrics 获取业务指标
func getBusinessMetrics() map[string]interface{} {
	// 这里应该从实际的业务逻辑中获取指标
	// 为了示例，我们返回一些模拟数据
	return map[string]interface{}{
		"active_users":   1000,
		"total_requests": 50000,
		"error_rate":     0.01,
		"response_time":  100,  // ms
		"throughput":     1000, // requests per second
	}
}

// DatabaseHealthCheck 数据库健康检查
func DatabaseHealthCheck(db *sql.DB) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"error":     err.Error(),
				"timestamp": time.Now().Unix(),
			})
			return
		}

		// 获取数据库统计信息
		stats := db.Stats()

		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"stats": map[string]interface{}{
				"open_conns":          stats.OpenConnections,
				"in_use":              stats.InUse,
				"idle":                stats.Idle,
				"wait_count":          stats.WaitCount,
				"wait_duration":       stats.WaitDuration.String(),
				"max_idle_closed":     stats.MaxIdleClosed,
				"max_lifetime_closed": stats.MaxLifetimeClosed,
			},
			"timestamp": time.Now().Unix(),
		})
	})
}

// RedisHealthCheck Redis健康检查
func RedisHealthCheck(client *redis.Client) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"error":     err.Error(),
				"timestamp": time.Now().Unix(),
			})
			return
		}

		// 获取Redis信息
		info, err := client.Info(ctx).Result()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"error":     "failed to get info: " + err.Error(),
				"timestamp": time.Now().Unix(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"info":      info,
			"timestamp": time.Now().Unix(),
		})
	})
}
