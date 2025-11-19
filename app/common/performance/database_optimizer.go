package performance

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

// DatabasePoolMetrics 数据库连接池指标
type DatabasePoolMetrics struct {
	MaxOpenConns      int           `json:"max_open_conns"`
	OpenConns         int           `json:"open_conns"`
	InUse             int           `json:"in_use"`
	Idle              int           `json:"idle"`
	WaitCount         int64         `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
	MaxIdleClosed     int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed int64         `json:"max_lifetime_closed"`
}

// DatabaseOptimizer 数据库优化器
type DatabaseOptimizer struct {
	db        *sql.DB
	metrics   *DatabasePoolMetrics
	optimizer *ConnectionPoolOptimizer
	monitor   *DatabaseMonitor
	mutex     sync.RWMutex
}

// NewDatabaseOptimizer 创建数据库优化器
func NewDatabaseOptimizer(db *sql.DB) *DatabaseOptimizer {
	return &DatabaseOptimizer{
		db:        db,
		metrics:   &DatabasePoolMetrics{},
		optimizer: NewConnectionPoolOptimizer(),
		monitor:   NewDatabaseMonitor(db),
	}
}

// Optimize 优化数据库连接池
func (do *DatabaseOptimizer) Optimize() error {
	do.mutex.Lock()
	defer do.mutex.Unlock()

	// 获取当前指标
	stats := do.db.Stats()
	do.metrics = &DatabasePoolMetrics{
		MaxOpenConns:      stats.MaxOpenConns,
		OpenConns:         stats.OpenConns,
		InUse:             stats.InUse,
		Idle:              stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxIdleTimeClosed: stats.MaxIdleTimeClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}

	// 优化连接池配置
	config := do.optimizer.Optimize(do.metrics)

	// 应用配置
	do.db.SetMaxOpenConns(config.MaxOpenConns)
	do.db.SetMaxIdleConns(config.MaxIdleConns)
	do.db.SetConnMaxLifetime(config.ConnMaxLifetime)
	do.db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	return nil
}

// GetMetrics 获取指标
func (do *DatabaseOptimizer) GetMetrics() *DatabasePoolMetrics {
	do.mutex.RLock()
	defer do.mutex.RUnlock()

	stats := do.db.Stats()
	return &DatabasePoolMetrics{
		MaxOpenConns:      stats.MaxOpenConns,
		OpenConns:         stats.OpenConns,
		InUse:             stats.InUse,
		Idle:              stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxIdleTimeClosed: stats.MaxIdleTimeClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}
}

// StartMonitoring 开始监控
func (do *DatabaseOptimizer) StartMonitoring(ctx context.Context, interval time.Duration) {
	go do.monitor.Start(ctx, interval)
}

// ConnectionPoolConfig 连接池配置
type ConnectionPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// ConnectionPoolOptimizer 连接池优化器
type ConnectionPoolOptimizer struct {
	configs map[string]*ConnectionPoolConfig
	mutex   sync.RWMutex
}

// NewConnectionPoolOptimizer 创建连接池优化器
func NewConnectionPoolOptimizer() *ConnectionPoolOptimizer {
	return &ConnectionPoolOptimizer{
		configs: make(map[string]*ConnectionPoolConfig),
	}
}

// Optimize 优化连接池配置
func (cpo *ConnectionPoolOptimizer) Optimize(metrics *DatabasePoolMetrics) *ConnectionPoolConfig {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    metrics.MaxOpenConns,
		MaxIdleConns:    metrics.Idle,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	// 根据指标调整配置
	if metrics.WaitCount > 100 {
		// 等待连接过多，增加最大连接数
		config.MaxOpenConns = int(float64(config.MaxOpenConns) * 1.5)
	}

	if metrics.Idle > config.MaxIdleConns {
		// 空闲连接过多，减少最大空闲连接数
		config.MaxIdleConns = int(float64(config.MaxIdleConns) * 0.8)
	}

	if metrics.WaitDuration > 100*time.Millisecond {
		// 等待时间过长，增加最大连接数
		config.MaxOpenConns = int(float64(config.MaxOpenConns) * 1.2)
	}

	// 确保配置在合理范围内
	if config.MaxOpenConns < 10 {
		config.MaxOpenConns = 10
	}
	if config.MaxOpenConns > 100 {
		config.MaxOpenConns = 100
	}

	if config.MaxIdleConns < 5 {
		config.MaxIdleConns = 5
	}
	if config.MaxIdleConns > 50 {
		config.MaxIdleConns = 50
	}

	return config
}

// DatabaseMonitor 数据库监控器
type DatabaseMonitor struct {
	db      *sql.DB
	metrics []*DatabasePoolMetrics
	mutex   sync.RWMutex
}

// NewDatabaseMonitor 创建数据库监控器
func NewDatabaseMonitor(db *sql.DB) *DatabaseMonitor {
	return &DatabaseMonitor{
		db:      db,
		metrics: make([]*DatabasePoolMetrics, 0),
	}
}

// Start 开始监控
func (dm *DatabaseMonitor) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dm.collectMetrics()
		}
	}
}

// collectMetrics 收集指标
func (dm *DatabaseMonitor) collectMetrics() {
	stats := dm.db.Stats()
	metrics := &DatabasePoolMetrics{
		MaxOpenConns:      stats.MaxOpenConns,
		OpenConns:         stats.OpenConns,
		InUse:             stats.InUse,
		Idle:              stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxIdleTimeClosed: stats.MaxIdleTimeClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}

	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.metrics = append(dm.metrics, metrics)

	// 保持最近1000个指标
	if len(dm.metrics) > 1000 {
		dm.metrics = dm.metrics[len(dm.metrics)-1000:]
	}
}

// GetMetricsHistory 获取指标历史
func (dm *DatabaseMonitor) GetMetricsHistory() []*DatabasePoolMetrics {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	metrics := make([]*DatabasePoolMetrics, len(dm.metrics))
	copy(metrics, dm.metrics)

	return metrics
}

// QueryCache 查询缓存
type QueryCache struct {
	cache   map[string]*CachedQuery
	mutex   sync.RWMutex
	ttl     time.Duration
	maxSize int
}

// CachedQuery 缓存的查询
type CachedQuery struct {
	Result    interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// NewQueryCache 创建查询缓存
func NewQueryCache(ttl time.Duration, maxSize int) *QueryCache {
	return &QueryCache{
		cache:   make(map[string]*CachedQuery),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get 获取缓存的查询结果
func (qc *QueryCache) Get(key string) (interface{}, bool) {
	qc.mutex.RLock()
	defer qc.mutex.RUnlock()

	cached, exists := qc.cache[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Since(cached.Timestamp) > cached.TTL {
		return nil, false
	}

	return cached.Result, true
}

// Set 设置缓存的查询结果
func (qc *QueryCache) Set(key string, result interface{}) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	// 检查缓存大小
	if len(qc.cache) >= qc.maxSize {
		// 删除最旧的缓存
		var oldestKey string
		var oldestTime time.Time
		for k, v := range qc.cache {
			if oldestTime.IsZero() || v.Timestamp.Before(oldestTime) {
				oldestTime = v.Timestamp
				oldestKey = k
			}
		}
		delete(qc.cache, oldestKey)
	}

	qc.cache[key] = &CachedQuery{
		Result:    result,
		Timestamp: time.Now(),
		TTL:       qc.ttl,
	}
}

// Clear 清空缓存
func (qc *QueryCache) Clear() {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()
	qc.cache = make(map[string]*CachedQuery)
}

// GetStats 获取缓存统计
func (qc *QueryCache) GetStats() map[string]interface{} {
	qc.mutex.RLock()
	defer qc.mutex.RUnlock()

	return map[string]interface{}{
		"size":     len(qc.cache),
		"max_size": qc.maxSize,
		"ttl":      qc.ttl,
	}
}

// SlowQueryMonitor 慢查询监控器
type SlowQueryMonitor struct {
	queries   []*SlowQuery
	threshold time.Duration
	mutex     sync.RWMutex
}

// SlowQuery 慢查询
type SlowQuery struct {
	Query     string
	Duration  time.Duration
	Timestamp time.Time
	Args      []interface{}
}

// NewSlowQueryMonitor 创建慢查询监控器
func NewSlowQueryMonitor(threshold time.Duration) *SlowQueryMonitor {
	return &SlowQueryMonitor{
		queries:   make([]*SlowQuery, 0),
		threshold: threshold,
	}
}

// RecordQuery 记录查询
func (sqm *SlowQueryMonitor) RecordQuery(query string, duration time.Duration, args []interface{}) {
	if duration < sqm.threshold {
		return
	}

	sqm.mutex.Lock()
	defer sqm.mutex.Unlock()

	slowQuery := &SlowQuery{
		Query:     query,
		Duration:  duration,
		Timestamp: time.Now(),
		Args:      args,
	}

	sqm.queries = append(sqm.queries, slowQuery)

	// 保持最近1000个慢查询
	if len(sqm.queries) > 1000 {
		sqm.queries = sqm.queries[len(sqm.queries)-1000:]
	}
}

// GetSlowQueries 获取慢查询
func (sqm *SlowQueryMonitor) GetSlowQueries() []*SlowQuery {
	sqm.mutex.RLock()
	defer sqm.mutex.RUnlock()

	queries := make([]*SlowQuery, len(sqm.queries))
	copy(queries, sqm.queries)

	return queries
}

// GetSlowQueriesByDuration 根据持续时间获取慢查询
func (sqm *SlowQueryMonitor) GetSlowQueriesByDuration(minDuration time.Duration) []*SlowQuery {
	sqm.mutex.RLock()
	defer sqm.mutex.RUnlock()

	var queries []*SlowQuery
	for _, query := range sqm.queries {
		if query.Duration >= minDuration {
			queries = append(queries, query)
		}
	}

	return queries
}

// DatabasePerformanceAnalyzer 数据库性能分析器
type DatabasePerformanceAnalyzer struct {
	monitor          *DatabaseMonitor
	queryCache       *QueryCache
	slowQueryMonitor *SlowQueryMonitor
	mutex            sync.RWMutex
}

// NewDatabasePerformanceAnalyzer 创建数据库性能分析器
func NewDatabasePerformanceAnalyzer(db *sql.DB) *DatabasePerformanceAnalyzer {
	return &DatabasePerformanceAnalyzer{
		monitor:          NewDatabaseMonitor(db),
		queryCache:       NewQueryCache(5*time.Minute, 1000),
		slowQueryMonitor: NewSlowQueryMonitor(100 * time.Millisecond),
	}
}

// Analyze 分析性能
func (dpa *DatabasePerformanceAnalyzer) Analyze() map[string]interface{} {
	dpa.mutex.RLock()
	defer dpa.mutex.RUnlock()

	metrics := dpa.monitor.GetMetricsHistory()
	slowQueries := dpa.slowQueryMonitor.GetSlowQueries()
	cacheStats := dpa.queryCache.GetStats()

	analysis := map[string]interface{}{
		"connection_pool": dpa.analyzeConnectionPool(metrics),
		"slow_queries":    dpa.analyzeSlowQueries(slowQueries),
		"query_cache":     cacheStats,
		"recommendations": dpa.generateRecommendations(metrics, slowQueries),
	}

	return analysis
}

// analyzeConnectionPool 分析连接池
func (dpa *DatabasePerformanceAnalyzer) analyzeConnectionPool(metrics []*DatabasePoolMetrics) map[string]interface{} {
	if len(metrics) == 0 {
		return map[string]interface{}{}
	}

	latest := metrics[len(metrics)-1]
	avgWaitDuration := time.Duration(0)
	totalWaitCount := int64(0)

	for _, m := range metrics {
		avgWaitDuration += m.WaitDuration
		totalWaitCount += m.WaitCount
	}

	if len(metrics) > 0 {
		avgWaitDuration /= time.Duration(len(metrics))
	}

	return map[string]interface{}{
		"current_connections": latest.OpenConns,
		"max_connections":     latest.MaxOpenConns,
		"in_use":              latest.InUse,
		"idle":                latest.Idle,
		"avg_wait_duration":   avgWaitDuration,
		"total_wait_count":    totalWaitCount,
		"utilization":         float64(latest.InUse) / float64(latest.MaxOpenConns),
	}
}

// analyzeSlowQueries 分析慢查询
func (dpa *DatabasePerformanceAnalyzer) analyzeSlowQueries(queries []*SlowQuery) map[string]interface{} {
	if len(queries) == 0 {
		return map[string]interface{}{
			"count":   0,
			"queries": []*SlowQuery{},
		}
	}

	// 按查询分组
	queryGroups := make(map[string][]*SlowQuery)
	for _, query := range queries {
		queryGroups[query.Query] = append(queryGroups[query.Query], query)
	}

	// 计算每个查询的平均时间
	queryStats := make(map[string]map[string]interface{})
	for query, qs := range queryGroups {
		totalDuration := time.Duration(0)
		for _, q := range qs {
			totalDuration += q.Duration
		}
		avgDuration := totalDuration / time.Duration(len(qs))

		queryStats[query] = map[string]interface{}{
			"count":        len(qs),
			"avg_duration": avgDuration,
			"max_duration": dpa.getMaxDuration(qs),
		}
	}

	return map[string]interface{}{
		"count":       len(queries),
		"query_stats": queryStats,
		"queries":     queries,
	}
}

// getMaxDuration 获取最大持续时间
func (dpa *DatabasePerformanceAnalyzer) getMaxDuration(queries []*SlowQuery) time.Duration {
	max := time.Duration(0)
	for _, q := range queries {
		if q.Duration > max {
			max = q.Duration
		}
	}
	return max
}

// generateRecommendations 生成建议
func (dpa *DatabasePerformanceAnalyzer) generateRecommendations(metrics []*DatabasePoolMetrics, slowQueries []*SlowQuery) []string {
	var recommendations []string

	if len(metrics) > 0 {
		latest := metrics[len(metrics)-1]

		// 连接池建议
		utilization := float64(latest.InUse) / float64(latest.MaxOpenConns)
		if utilization > 0.8 {
			recommendations = append(recommendations, "连接池利用率过高，建议增加最大连接数")
		}

		if latest.WaitCount > 100 {
			recommendations = append(recommendations, "等待连接数过多，建议优化连接池配置")
		}

		if latest.WaitDuration > 100*time.Millisecond {
			recommendations = append(recommendations, "等待时间过长，建议增加连接数或优化查询")
		}
	}

	// 慢查询建议
	if len(slowQueries) > 10 {
		recommendations = append(recommendations, "慢查询数量较多，建议优化查询语句")
	}

	// 查询缓存建议
	cacheStats := dpa.queryCache.GetStats()
	if cacheStats["size"].(int) > 800 {
		recommendations = append(recommendations, "查询缓存使用率较高，建议增加缓存大小")
	}

	return recommendations
}
