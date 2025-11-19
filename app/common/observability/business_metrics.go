package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics 业务指标
type BusinessMetrics struct {
	userRegistrations prometheus.Counter
	userLogins        prometheus.Counter
	userLogouts       prometheus.Counter
	gameSessions      prometheus.Histogram
	gameWins          prometheus.Counter
	gameLosses        prometheus.Counter
	revenue           prometheus.Counter
	conversionRate    prometheus.Gauge
	activeUsers       prometheus.Gauge
	responseTime      prometheus.Histogram
	errorRate         prometheus.Gauge
	throughput        prometheus.Counter
	customMetrics     map[string]prometheus.Collector
	mutex             sync.RWMutex
	registry          prometheus.Registerer
}

// NewBusinessMetrics 创建业务指标
func NewBusinessMetrics(registry prometheus.Registerer) *BusinessMetrics {
	bm := &BusinessMetrics{
		registry:      registry,
		customMetrics: make(map[string]prometheus.Collector),
	}

	// 用户注册指标
	bm.userRegistrations = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Name: "business_user_registrations_total",
		Help: "Total number of user registrations",
	})

	// 用户登录指标
	bm.userLogins = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Name: "business_user_logins_total",
		Help: "Total number of user logins",
	})

	// 用户登出指标
	bm.userLogouts = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Name: "business_user_logouts_total",
		Help: "Total number of user logouts",
	})

	// 游戏会话指标
	bm.gameSessions = promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
		Name:    "business_game_sessions_duration_seconds",
		Help:    "Duration of game sessions in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// 游戏胜利指标
	bm.gameWins = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Name: "business_game_wins_total",
		Help: "Total number of game wins",
	})

	// 游戏失败指标
	bm.gameLosses = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Name: "business_game_losses_total",
		Help: "Total number of game losses",
	})

	// 收入指标
	bm.revenue = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Name: "business_revenue_total",
		Help: "Total revenue in currency units",
	})

	// 转化率指标
	bm.conversionRate = promauto.With(registry).NewGauge(prometheus.GaugeOpts{
		Name: "business_conversion_rate",
		Help: "Current conversion rate",
	})

	// 活跃用户指标
	bm.activeUsers = promauto.With(registry).NewGauge(prometheus.GaugeOpts{
		Name: "business_active_users",
		Help: "Current number of active users",
	})

	// 响应时间指标
	bm.responseTime = promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
		Name:    "business_response_time_seconds",
		Help:    "Response time in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// 错误率指标
	bm.errorRate = promauto.With(registry).NewGauge(prometheus.GaugeOpts{
		Name: "business_error_rate",
		Help: "Current error rate",
	})

	// 吞吐量指标
	bm.throughput = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Name: "business_throughput_total",
		Help: "Total number of requests processed",
	})

	return bm
}

// RecordUserRegistration 记录用户注册
func (bm *BusinessMetrics) RecordUserRegistration() {
	bm.userRegistrations.Inc()
}

// RecordUserLogin 记录用户登录
func (bm *BusinessMetrics) RecordUserLogin() {
	bm.userLogins.Inc()
}

// RecordUserLogout 记录用户登出
func (bm *BusinessMetrics) RecordUserLogout() {
	bm.userLogouts.Inc()
}

// RecordGameSession 记录游戏会话
func (bm *BusinessMetrics) RecordGameSession(duration time.Duration) {
	bm.gameSessions.Observe(duration.Seconds())
}

// RecordGameWin 记录游戏胜利
func (bm *BusinessMetrics) RecordGameWin() {
	bm.gameWins.Inc()
}

// RecordGameLoss 记录游戏失败
func (bm *BusinessMetrics) RecordGameLoss() {
	bm.gameLosses.Inc()
}

// RecordRevenue 记录收入
func (bm *BusinessMetrics) RecordRevenue(amount float64) {
	bm.revenue.Add(amount)
}

// SetConversionRate 设置转化率
func (bm *BusinessMetrics) SetConversionRate(rate float64) {
	bm.conversionRate.Set(rate)
}

// SetActiveUsers 设置活跃用户数
func (bm *BusinessMetrics) SetActiveUsers(count float64) {
	bm.activeUsers.Set(count)
}

// RecordResponseTime 记录响应时间
func (bm *BusinessMetrics) RecordResponseTime(duration time.Duration) {
	bm.responseTime.Observe(duration.Seconds())
}

// SetErrorRate 设置错误率
func (bm *BusinessMetrics) SetErrorRate(rate float64) {
	bm.errorRate.Set(rate)
}

// RecordThroughput 记录吞吐量
func (bm *BusinessMetrics) RecordThroughput() {
	bm.throughput.Inc()
}

// AddCustomMetric 添加自定义指标
func (bm *BusinessMetrics) AddCustomMetric(name string, metric prometheus.Collector) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if _, exists := bm.customMetrics[name]; exists {
		return fmt.Errorf("custom metric already exists: %s", name)
	}

	bm.customMetrics[name] = metric
	bm.registry.MustRegister(metric)

	return nil
}

// RemoveCustomMetric 移除自定义指标
func (bm *BusinessMetrics) RemoveCustomMetric(name string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	metric, exists := bm.customMetrics[name]
	if !exists {
		return fmt.Errorf("custom metric not found: %s", name)
	}

	bm.registry.Unregister(metric)
	delete(bm.customMetrics, name)

	return nil
}

// GetCustomMetric 获取自定义指标
func (bm *BusinessMetrics) GetCustomMetric(name string) (prometheus.Collector, bool) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	metric, exists := bm.customMetrics[name]
	return metric, exists
}

// CustomMetrics 自定义指标管理器
type CustomMetrics struct {
	registry prometheus.Registerer
	metrics  map[string]prometheus.Collector
	mutex    sync.RWMutex
}

// NewCustomMetrics 创建自定义指标管理器
func NewCustomMetrics(registry prometheus.Registerer) *CustomMetrics {
	return &CustomMetrics{
		registry: registry,
		metrics:  make(map[string]prometheus.Collector),
	}
}

// CreateCounter 创建计数器
func (cm *CustomMetrics) CreateCounter(name, help string, labels []string) (prometheus.Counter, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.metrics[name]; exists {
		return nil, fmt.Errorf("metric already exists: %s", name)
	}

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})

	cm.metrics[name] = counter
	cm.registry.MustRegister(counter)

	return counter, nil
}

// CreateGauge 创建仪表
func (cm *CustomMetrics) CreateGauge(name, help string, labels []string) (prometheus.Gauge, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.metrics[name]; exists {
		return nil, fmt.Errorf("metric already exists: %s", name)
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})

	cm.metrics[name] = gauge
	cm.registry.MustRegister(gauge)

	return gauge, nil
}

// CreateHistogram 创建直方图
func (cm *CustomMetrics) CreateHistogram(name, help string, buckets []float64) (prometheus.Histogram, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.metrics[name]; exists {
		return nil, fmt.Errorf("metric already exists: %s", name)
	}

	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	})

	cm.metrics[name] = histogram
	cm.registry.MustRegister(histogram)

	return histogram, nil
}

// CreateSummary 创建摘要
func (cm *CustomMetrics) CreateSummary(name, help string, objectives map[float64]float64) (prometheus.Summary, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.metrics[name]; exists {
		return nil, fmt.Errorf("metric already exists: %s", name)
	}

	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       name,
		Help:       help,
		Objectives: objectives,
	})

	cm.metrics[name] = summary
	cm.registry.MustRegister(summary)

	return summary, nil
}

// GetMetric 获取指标
func (cm *CustomMetrics) GetMetric(name string) (prometheus.Collector, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	metric, exists := cm.metrics[name]
	return metric, exists
}

// RemoveMetric 移除指标
func (cm *CustomMetrics) RemoveMetric(name string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	metric, exists := cm.metrics[name]
	if !exists {
		return fmt.Errorf("metric not found: %s", name)
	}

	cm.registry.Unregister(metric)
	delete(cm.metrics, name)

	return nil
}

// ListMetrics 列出所有指标
func (cm *CustomMetrics) ListMetrics() []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	names := make([]string, 0, len(cm.metrics))
	for name := range cm.metrics {
		names = append(names, name)
	}

	return names
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	businessMetrics *BusinessMetrics
	customMetrics   *CustomMetrics
	registry        prometheus.Registerer
	mutex           sync.RWMutex
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(registry prometheus.Registerer) *MetricsCollector {
	return &MetricsCollector{
		businessMetrics: NewBusinessMetrics(registry),
		customMetrics:   NewCustomMetrics(registry),
		registry:        registry,
	}
}

// GetBusinessMetrics 获取业务指标
func (mc *MetricsCollector) GetBusinessMetrics() *BusinessMetrics {
	return mc.businessMetrics
}

// GetCustomMetrics 获取自定义指标
func (mc *MetricsCollector) GetCustomMetrics() *CustomMetrics {
	return mc.customMetrics
}

// Collect 收集指标
func (mc *MetricsCollector) Collect(ctx context.Context) error {
	// 这里可以实现指标收集逻辑
	// 例如：从数据库收集数据、计算指标等
	return nil
}

// Export 导出指标
func (mc *MetricsCollector) Export() ([]byte, error) {
	// 这里可以实现指标导出逻辑
	// 例如：导出为Prometheus格式
	return nil, nil
}

// MetricsExporter 指标导出器
type MetricsExporter struct {
	collectors map[string]*MetricsCollector
	mutex      sync.RWMutex
}

// NewMetricsExporter 创建指标导出器
func NewMetricsExporter() *MetricsExporter {
	return &MetricsExporter{
		collectors: make(map[string]*MetricsCollector),
	}
}

// AddCollector 添加收集器
func (me *MetricsExporter) AddCollector(name string, collector *MetricsCollector) {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	me.collectors[name] = collector
}

// GetCollector 获取收集器
func (me *MetricsExporter) GetCollector(name string) (*MetricsCollector, bool) {
	me.mutex.RLock()
	defer me.mutex.RUnlock()
	collector, exists := me.collectors[name]
	return collector, exists
}

// RemoveCollector 移除收集器
func (me *MetricsExporter) RemoveCollector(name string) {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	delete(me.collectors, name)
}

// ExportAll 导出所有指标
func (me *MetricsExporter) ExportAll() (map[string][]byte, error) {
	me.mutex.RLock()
	defer me.mutex.RUnlock()

	results := make(map[string][]byte)
	for name, collector := range me.collectors {
		data, err := collector.Export()
		if err != nil {
			return nil, fmt.Errorf("failed to export metrics from %s: %w", name, err)
		}
		results[name] = data
	}

	return results, nil
}
