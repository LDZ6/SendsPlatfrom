package utils

import (
	"context"
	"database/sql"
	"sort"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 慢查询检测器
type SlowQueryDetector struct {
	threshold time.Duration
	queries   map[string]*QueryStats
	mutex     sync.RWMutex
}

// 查询统计
type QueryStats struct {
	Query     string
	Count     int64
	TotalTime time.Duration
	AvgTime   time.Duration
	MaxTime   time.Duration
	MinTime   time.Duration
	LastSeen  time.Time
}

// Redis热点Key检测器
type HotKeyDetector struct {
	keyStats  map[string]*KeyStats
	mutex     sync.RWMutex
	threshold int64 // 访问次数阈值
}

// Key统计
type KeyStats struct {
	Key       string
	Count     int64
	LastSeen  time.Time
	Operation string
}

// 性能监控器
type PerformanceMonitor struct {
	slowQueryDetector *SlowQueryDetector
	hotKeyDetector    *HotKeyDetector
	metrics           *PerformanceMetrics
}

// 性能指标
type PerformanceMetrics struct {
	SlowQueryCount *prometheus.CounterVec
	HotKeyCount    *prometheus.CounterVec
	QueryDuration  *prometheus.HistogramVec
	RedisKeyAccess *prometheus.CounterVec
}

var (
	perfMonitor *PerformanceMonitor
)

// 初始化性能监控
func init() {
	initPerformanceMonitor()
}

func initPerformanceMonitor() {
	perfMonitor = &PerformanceMonitor{
		slowQueryDetector: &SlowQueryDetector{
			threshold: 200 * time.Millisecond, // 200ms阈值
			queries:   make(map[string]*QueryStats),
		},
		hotKeyDetector: &HotKeyDetector{
			keyStats:  make(map[string]*KeyStats),
			threshold: 100, // 100次访问阈值
		},
		metrics: &PerformanceMetrics{
			SlowQueryCount: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "slow_query_count_total",
					Help: "Total number of slow queries",
				},
				[]string{"query", "table", "service"},
			),
			HotKeyCount: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "hot_key_count_total",
					Help: "Total number of hot key accesses",
				},
				[]string{"key", "operation", "service"},
			),
			QueryDuration: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "query_duration_seconds",
					Help:    "Query duration in seconds",
					Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
				},
				[]string{"query", "table", "service"},
			),
			RedisKeyAccess: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "redis_key_access_total",
					Help: "Total number of Redis key accesses",
				},
				[]string{"key", "operation", "service"},
			),
		},
	}

	// 注册指标
	prometheus.MustRegister(perfMonitor.metrics.SlowQueryCount)
	prometheus.MustRegister(perfMonitor.metrics.HotKeyCount)
	prometheus.MustRegister(perfMonitor.metrics.QueryDuration)
	prometheus.MustRegister(perfMonitor.metrics.RedisKeyAccess)

	// 启动定期清理任务
	go perfMonitor.startCleanupTask()
}

// 记录慢查询
func (sqd *SlowQueryDetector) RecordQuery(query string, duration time.Duration, table, service string) {
	if duration < sqd.threshold {
		return
	}

	sqd.mutex.Lock()
	defer sqd.mutex.Unlock()

	stats, exists := sqd.queries[query]
	if !exists {
		stats = &QueryStats{
			Query:    query,
			MinTime:  duration,
			LastSeen: time.Now(),
		}
		sqd.queries[query] = stats
	}

	stats.Count++
	stats.TotalTime += duration
	stats.AvgTime = stats.TotalTime / time.Duration(stats.Count)
	stats.MaxTime = max(stats.MaxTime, duration)
	stats.MinTime = min(stats.MinTime, duration)
	stats.LastSeen = time.Now()

	// 记录到指标
	perfMonitor.metrics.SlowQueryCount.WithLabelValues(query, table, service).Inc()
	perfMonitor.metrics.QueryDuration.WithLabelValues(query, table, service).Observe(duration.Seconds())

	// 记录慢查询日志
	logger := logrus.WithFields(logrus.Fields{
		"query":    query,
		"duration": duration.Milliseconds(),
		"table":    table,
		"service":  service,
		"count":    stats.Count,
		"avg_time": stats.AvgTime.Milliseconds(),
		"max_time": stats.MaxTime.Milliseconds(),
	})
	logger.Warn("Slow query detected")
}

// 获取慢查询统计
func (sqd *SlowQueryDetector) GetSlowQueries() []*QueryStats {
	sqd.mutex.RLock()
	defer sqd.mutex.RUnlock()

	var queries []*QueryStats
	for _, stats := range sqd.queries {
		queries = append(queries, stats)
	}

	// 按平均时间排序
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].AvgTime > queries[j].AvgTime
	})

	return queries
}

// 记录Redis Key访问
func (hkd *HotKeyDetector) RecordKeyAccess(key, operation, service string) {
	hkd.mutex.Lock()
	defer hkd.mutex.Unlock()

	stats, exists := hkd.keyStats[key]
	if !exists {
		stats = &KeyStats{
			Key:       key,
			Operation: operation,
			LastSeen:  time.Now(),
		}
		hkd.keyStats[key] = stats
	}

	stats.Count++
	stats.LastSeen = time.Now()

	// 记录到指标
	perfMonitor.metrics.RedisKeyAccess.WithLabelValues(key, operation, service).Inc()

	// 检查是否是热点Key
	if stats.Count >= hkd.threshold {
		perfMonitor.metrics.HotKeyCount.WithLabelValues(key, operation, service).Inc()

		// 记录热点Key日志
		logger := logrus.WithFields(logrus.Fields{
			"key":       key,
			"operation": operation,
			"count":     stats.Count,
			"service":   service,
		})
		logger.Warn("Hot key detected")
	}
}

// 获取热点Key统计
func (hkd *HotKeyDetector) GetHotKeys() []*KeyStats {
	hkd.mutex.RLock()
	defer hkd.mutex.RUnlock()

	var keys []*KeyStats
	for _, stats := range hkd.keyStats {
		if stats.Count >= hkd.threshold {
			keys = append(keys, stats)
		}
	}

	// 按访问次数排序
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Count > keys[j].Count
	})

	return keys
}

// 定期清理任务
func (pm *PerformanceMonitor) startCleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pm.cleanup()
	}
}

// 清理过期数据
func (pm *PerformanceMonitor) cleanup() {
	now := time.Now()

	// 清理慢查询统计（保留1小时）
	pm.slowQueryDetector.mutex.Lock()
	for query, stats := range pm.slowQueryDetector.queries {
		if now.Sub(stats.LastSeen) > time.Hour {
			delete(pm.slowQueryDetector.queries, query)
		}
	}
	pm.slowQueryDetector.mutex.Unlock()

	// 清理热点Key统计（保留30分钟）
	pm.hotKeyDetector.mutex.Lock()
	for key, stats := range pm.hotKeyDetector.keyStats {
		if now.Sub(stats.LastSeen) > 30*time.Minute {
			delete(pm.hotKeyDetector.keyStats, key)
		}
	}
	pm.hotKeyDetector.mutex.Unlock()
}

// GORM慢查询中间件
func GORMSlowQueryMiddleware() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		start := time.Now()

		// 执行查询
		err := db.Error
		duration := time.Since(start)

		// 记录慢查询
		if err == nil && duration > 200*time.Millisecond {
			query := db.Statement.SQL.String()
			table := db.Statement.Table
			service := "unknown"

			perfMonitor.slowQueryDetector.RecordQuery(query, duration, table, service)
		}
	}
}

// Redis慢命令中间件
func RedisSlowCommandMiddleware(rdb *redis.Client) redis.Hook {
	return &redisSlowCommandHook{rdb: rdb}
}

type redisSlowCommandHook struct {
	rdb *redis.Client
}

func (h *redisSlowCommandHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	return context.WithValue(ctx, "start_time", time.Now()), nil
}

func (h *redisSlowCommandHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	startTime, ok := ctx.Value("start_time").(time.Time)
	if !ok {
		return nil
	}

	duration := time.Since(startTime)

	// 记录Redis Key访问
	key := cmd.Name()
	operation := cmd.Name()
	service := "unknown"

	perfMonitor.hotKeyDetector.RecordKeyAccess(key, operation, service)

	// 记录慢命令
	if duration > 10*time.Millisecond {
		logger := logrus.WithFields(logrus.Fields{
			"command":  cmd.Name(),
			"duration": duration.Milliseconds(),
			"args":     cmd.Args(),
		})
		logger.Warn("Slow Redis command detected")
	}

	return nil
}

func (h *redisSlowCommandHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return context.WithValue(ctx, "start_time", time.Now()), nil
}

func (h *redisSlowCommandHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	startTime, ok := ctx.Value("start_time").(time.Time)
	if !ok {
		return nil
	}

	duration := time.Since(startTime)

	// 记录管道操作
	for _, cmd := range cmds {
		key := cmd.Name()
		operation := "pipeline"
		service := "unknown"

		perfMonitor.hotKeyDetector.RecordKeyAccess(key, operation, service)
	}

	// 记录慢管道操作
	if duration > 50*time.Millisecond {
		logger := logrus.WithFields(logrus.Fields{
			"command_count": len(cmds),
			"duration":      duration.Milliseconds(),
		})
		logger.Warn("Slow Redis pipeline detected")
	}

	return nil
}

// 获取性能监控器
func GetPerformanceMonitor() *PerformanceMonitor {
	return perfMonitor
}

// 获取慢查询统计
func GetSlowQueries() []*QueryStats {
	return perfMonitor.slowQueryDetector.GetSlowQueries()
}

// 获取热点Key统计
func GetHotKeys() []*KeyStats {
	return perfMonitor.hotKeyDetector.GetHotKeys()
}

// 生成性能报告
func GeneratePerformanceReport() map[string]interface{} {
	slowQueries := GetSlowQueries()
	hotKeys := GetHotKeys()

	report := map[string]interface{}{
		"timestamp":        time.Now().Format(time.RFC3339),
		"slow_queries":     slowQueries,
		"hot_keys":         hotKeys,
		"slow_query_count": len(slowQueries),
		"hot_key_count":    len(hotKeys),
	}

	return report
}

// 工具函数
func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// 数据库连接池监控
func MonitorDBConnections(db *sql.DB, service string) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := db.Stats()

			// 记录连接池指标
			SetMySQLConnInUse("unknown", service, float64(stats.InUse))

			// 检查连接池健康状态
			if stats.InUse > stats.MaxOpenConnections*0.8 {
				logger := logrus.WithFields(logrus.Fields{
					"in_use":              stats.InUse,
					"max_open":            stats.MaxOpenConnections,
					"idle":                stats.Idle,
					"wait_count":          stats.WaitCount,
					"wait_duration":       stats.WaitDuration,
					"max_idle_closed":     stats.MaxIdleClosed,
					"max_lifetime_closed": stats.MaxLifetimeClosed,
					"service":             service,
				})
				logger.Warn("Database connection pool high utilization")
			}
		}
	}()
}

// Redis连接监控
func MonitorRedisConnections(rdb *redis.Client, service string) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ctx := context.Background()
			info, err := rdb.Info(ctx, "clients").Result()
			if err != nil {
				continue
			}

			// 解析Redis信息并记录指标
			// 这里需要根据实际的Redis信息格式解析
			logger := logrus.WithFields(logrus.Fields{
				"info":    info,
				"service": service,
			})
			logger.Debug("Redis connection info")
		}
	}()
}
