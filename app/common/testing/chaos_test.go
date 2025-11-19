package testing

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ChaosTest 混沌工程测试
type ChaosTest struct {
	scenarios []ChaosScenario
	monitor   ChaosMonitor
	mutex     sync.RWMutex
}

// ChaosScenario 混沌场景
type ChaosScenario struct {
	ID          string
	Name        string
	Description string
	Type        ChaosType
	Duration    time.Duration
	Intensity   float64
	Targets     []string
	Parameters  map[string]interface{}
	Enabled     bool
}

// ChaosType 混沌类型
type ChaosType int

const (
	ChaosTypeNetworkDelay ChaosType = iota
	ChaosTypeNetworkLoss
	ChaosTypeNetworkPartition
	ChaosTypeServiceFailure
	ChaosTypeResourceExhaustion
	ChaosTypeDatabaseFailure
	ChaosTypeCacheFailure
	ChaosTypeRandomFailure
	ChaosTypeLoadSpike
	ChaosTypeMemoryLeak
)

// String 混沌类型字符串
func (ct ChaosType) String() string {
	switch ct {
	case ChaosTypeNetworkDelay:
		return "network_delay"
	case ChaosTypeNetworkLoss:
		return "network_loss"
	case ChaosTypeNetworkPartition:
		return "network_partition"
	case ChaosTypeServiceFailure:
		return "service_failure"
	case ChaosTypeResourceExhaustion:
		return "resource_exhaustion"
	case ChaosTypeDatabaseFailure:
		return "database_failure"
	case ChaosTypeCacheFailure:
		return "cache_failure"
	case ChaosTypeRandomFailure:
		return "random_failure"
	case ChaosTypeLoadSpike:
		return "load_spike"
	case ChaosTypeMemoryLeak:
		return "memory_leak"
	default:
		return "unknown"
	}
}

// ChaosMonitor 混沌监控器
type ChaosMonitor struct {
	metrics map[string]*ChaosMetrics
	alerts  []ChaosAlert
	mutex   sync.RWMutex
}

// ChaosMetrics 混沌指标
type ChaosMetrics struct {
	ScenarioID    string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	SuccessCount  int64
	FailureCount  int64
	ErrorRate     float64
	ResponseTime  time.Duration
	Throughput    float64
	ResourceUsage map[string]float64
}

// ChaosAlert 混沌告警
type ChaosAlert struct {
	ID         string
	ScenarioID string
	Type       AlertType
	Message    string
	Timestamp  time.Time
	Severity   AlertSeverity
}

// AlertType 告警类型
type AlertType int

const (
	AlertTypeHighErrorRate AlertType = iota
	AlertTypeHighResponseTime
	AlertTypeLowThroughput
	AlertTypeResourceExhaustion
	AlertTypeServiceUnavailable
	AlertTypeDataLoss
)

// AlertSeverity 告警严重程度
type AlertSeverity int

const (
	SeverityLow AlertSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// NewChaosTest 创建混沌测试
func NewChaosTest() *ChaosTest {
	return &ChaosTest{
		scenarios: make([]ChaosScenario, 0),
		monitor: ChaosMonitor{
			metrics: make(map[string]*ChaosMetrics),
			alerts:  make([]ChaosAlert, 0),
		},
	}
}

// AddScenario 添加混沌场景
func (ct *ChaosTest) AddScenario(scenario ChaosScenario) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	ct.scenarios = append(ct.scenarios, scenario)
}

// RunScenario 运行混沌场景
func (ct *ChaosTest) RunScenario(ctx context.Context, scenarioID string) error {
	ct.mutex.RLock()
	var scenario *ChaosScenario
	for _, s := range ct.scenarios {
		if s.ID == scenarioID {
			scenario = &s
			break
		}
	}
	ct.mutex.RUnlock()

	if scenario == nil {
		return fmt.Errorf("scenario not found: %s", scenarioID)
	}

	if !scenario.Enabled {
		return fmt.Errorf("scenario is disabled: %s", scenarioID)
	}

	// 开始监控
	ct.monitor.StartMonitoring(scenarioID)

	// 运行场景
	switch scenario.Type {
	case ChaosTypeNetworkDelay:
		return ct.runNetworkDelayScenario(ctx, scenario)
	case ChaosTypeNetworkLoss:
		return ct.runNetworkLossScenario(ctx, scenario)
	case ChaosTypeServiceFailure:
		return ct.runServiceFailureScenario(ctx, scenario)
	case ChaosTypeResourceExhaustion:
		return ct.runResourceExhaustionScenario(ctx, scenario)
	case ChaosTypeDatabaseFailure:
		return ct.runDatabaseFailureScenario(ctx, scenario)
	case ChaosTypeCacheFailure:
		return ct.runCacheFailureScenario(ctx, scenario)
	case ChaosTypeRandomFailure:
		return ct.runRandomFailureScenario(ctx, scenario)
	case ChaosTypeLoadSpike:
		return ct.runLoadSpikeScenario(ctx, scenario)
	case ChaosTypeMemoryLeak:
		return ct.runMemoryLeakScenario(ctx, scenario)
	default:
		return fmt.Errorf("unknown chaos type: %s", scenario.Type)
	}
}

// runNetworkDelayScenario 运行网络延迟场景
func (ct *ChaosTest) runNetworkDelayScenario(ctx context.Context, scenario *ChaosScenario) error {
	delay := time.Duration(scenario.Intensity * float64(time.Second))

	// 模拟网络延迟
	time.Sleep(delay)

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID:   scenario.ID,
		StartTime:    time.Now(),
		Duration:     delay,
		ResponseTime: delay,
	})

	return nil
}

// runNetworkLossScenario 运行网络丢包场景
func (ct *ChaosTest) runNetworkLossScenario(ctx context.Context, scenario *ChaosScenario) error {
	lossRate := scenario.Intensity

	// 模拟网络丢包
	if rand.Float64() < lossRate {
		return fmt.Errorf("network packet lost")
	}

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID: scenario.ID,
		StartTime:  time.Now(),
	})

	return nil
}

// runServiceFailureScenario 运行服务故障场景
func (ct *ChaosTest) runServiceFailureScenario(ctx context.Context, scenario *ChaosScenario) error {
	failureRate := scenario.Intensity

	// 模拟服务故障
	if rand.Float64() < failureRate {
		ct.monitor.RecordAlert(ChaosAlert{
			ID:         fmt.Sprintf("alert_%d", time.Now().UnixNano()),
			ScenarioID: scenario.ID,
			Type:       AlertTypeServiceUnavailable,
			Message:    "Service failure detected",
			Timestamp:  time.Now(),
			Severity:   SeverityHigh,
		})
		return fmt.Errorf("service failure")
	}

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID: scenario.ID,
		StartTime:  time.Now(),
	})

	return nil
}

// runResourceExhaustionScenario 运行资源耗尽场景
func (ct *ChaosTest) runResourceExhaustionScenario(ctx context.Context, scenario *ChaosScenario) error {
	// 模拟资源耗尽
	ct.monitor.RecordAlert(ChaosAlert{
		ID:         fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		ScenarioID: scenario.ID,
		Type:       AlertTypeResourceExhaustion,
		Message:    "Resource exhaustion detected",
		Timestamp:  time.Now(),
		Severity:   SeverityCritical,
	})

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID: scenario.ID,
		StartTime:  time.Now(),
		ResourceUsage: map[string]float64{
			"cpu":    100.0,
			"memory": 100.0,
			"disk":   100.0,
		},
	})

	return nil
}

// runDatabaseFailureScenario 运行数据库故障场景
func (ct *ChaosTest) runDatabaseFailureScenario(ctx context.Context, scenario *ChaosScenario) error {
	// 模拟数据库故障
	ct.monitor.RecordAlert(ChaosAlert{
		ID:         fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		ScenarioID: scenario.ID,
		Type:       AlertTypeServiceUnavailable,
		Message:    "Database failure detected",
		Timestamp:  time.Now(),
		Severity:   SeverityCritical,
	})

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID: scenario.ID,
		StartTime:  time.Now(),
	})

	return nil
}

// runCacheFailureScenario 运行缓存故障场景
func (ct *ChaosTest) runCacheFailureScenario(ctx context.Context, scenario *ChaosScenario) error {
	// 模拟缓存故障
	ct.monitor.RecordAlert(ChaosAlert{
		ID:         fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		ScenarioID: scenario.ID,
		Type:       AlertTypeServiceUnavailable,
		Message:    "Cache failure detected",
		Timestamp:  time.Now(),
		Severity:   SeverityMedium,
	})

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID: scenario.ID,
		StartTime:  time.Now(),
	})

	return nil
}

// runRandomFailureScenario 运行随机故障场景
func (ct *ChaosTest) runRandomFailureScenario(ctx context.Context, scenario *ChaosScenario) error {
	failureRate := scenario.Intensity

	// 模拟随机故障
	if rand.Float64() < failureRate {
		ct.monitor.RecordAlert(ChaosAlert{
			ID:         fmt.Sprintf("alert_%d", time.Now().UnixNano()),
			ScenarioID: scenario.ID,
			Type:       AlertTypeRandomFailure,
			Message:    "Random failure detected",
			Timestamp:  time.Now(),
			Severity:   SeverityMedium,
		})
		return fmt.Errorf("random failure")
	}

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID: scenario.ID,
		StartTime:  time.Now(),
	})

	return nil
}

// runLoadSpikeScenario 运行负载峰值场景
func (ct *ChaosTest) runLoadSpikeScenario(ctx context.Context, scenario *ChaosScenario) error {
	// 模拟负载峰值
	ct.monitor.RecordAlert(ChaosAlert{
		ID:         fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		ScenarioID: scenario.ID,
		Type:       AlertTypeHighResponseTime,
		Message:    "Load spike detected",
		Timestamp:  time.Now(),
		Severity:   SeverityHigh,
	})

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID:   scenario.ID,
		StartTime:    time.Now(),
		ResponseTime: time.Duration(scenario.Intensity * float64(time.Second)),
		Throughput:   scenario.Intensity * 1000,
	})

	return nil
}

// runMemoryLeakScenario 运行内存泄漏场景
func (ct *ChaosTest) runMemoryLeakScenario(ctx context.Context, scenario *ChaosScenario) error {
	// 模拟内存泄漏
	ct.monitor.RecordAlert(ChaosAlert{
		ID:         fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		ScenarioID: scenario.ID,
		Type:       AlertTypeResourceExhaustion,
		Message:    "Memory leak detected",
		Timestamp:  time.Now(),
		Severity:   SeverityHigh,
	})

	// 记录指标
	ct.monitor.RecordMetrics(scenario.ID, &ChaosMetrics{
		ScenarioID: scenario.ID,
		StartTime:  time.Now(),
		ResourceUsage: map[string]float64{
			"memory": scenario.Intensity * 100,
		},
	})

	return nil
}

// StartMonitoring 开始监控
func (cm *ChaosMonitor) StartMonitoring(scenarioID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.metrics[scenarioID] = &ChaosMetrics{
		ScenarioID: scenarioID,
		StartTime:  time.Now(),
	}
}

// RecordMetrics 记录指标
func (cm *ChaosMonitor) RecordMetrics(scenarioID string, metrics *ChaosMetrics) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if existing, exists := cm.metrics[scenarioID]; exists {
		existing.EndTime = time.Now()
		existing.Duration = existing.EndTime.Sub(existing.StartTime)
		existing.SuccessCount += 1
		existing.ErrorRate = float64(existing.FailureCount) / float64(existing.SuccessCount+existing.FailureCount)
	} else {
		cm.metrics[scenarioID] = metrics
	}
}

// RecordAlert 记录告警
func (cm *ChaosMonitor) RecordAlert(alert ChaosAlert) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.alerts = append(cm.alerts, alert)

	// 保持最近1000条告警
	if len(cm.alerts) > 1000 {
		cm.alerts = cm.alerts[len(cm.alerts)-1000:]
	}
}

// GetMetrics 获取指标
func (cm *ChaosMonitor) GetMetrics(scenarioID string) (*ChaosMetrics, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	metrics, exists := cm.metrics[scenarioID]
	if !exists {
		return nil, fmt.Errorf("metrics not found for scenario: %s", scenarioID)
	}

	return metrics, nil
}

// GetAllMetrics 获取所有指标
func (cm *ChaosMonitor) GetAllMetrics() map[string]*ChaosMetrics {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	metrics := make(map[string]*ChaosMetrics)
	for k, v := range cm.metrics {
		metrics[k] = v
	}

	return metrics
}

// GetAlerts 获取告警
func (cm *ChaosMonitor) GetAlerts() []ChaosAlert {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	alerts := make([]ChaosAlert, len(cm.alerts))
	copy(alerts, cm.alerts)

	return alerts
}

// GetAlertsBySeverity 根据严重程度获取告警
func (cm *ChaosMonitor) GetAlertsBySeverity(severity AlertSeverity) []ChaosAlert {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var alerts []ChaosAlert
	for _, alert := range cm.alerts {
		if alert.Severity == severity {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// GetAlertsByType 根据类型获取告警
func (cm *ChaosMonitor) GetAlertsByType(alertType AlertType) []ChaosAlert {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var alerts []ChaosAlert
	for _, alert := range cm.alerts {
		if alert.Type == alertType {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// ChaosTestSuite 混沌测试套件
type ChaosTestSuite struct {
	tests []*ChaosTest
	mutex sync.RWMutex
}

// NewChaosTestSuite 创建混沌测试套件
func NewChaosTestSuite() *ChaosTestSuite {
	return &ChaosTestSuite{
		tests: make([]*ChaosTest, 0),
	}
}

// AddTest 添加测试
func (cts *ChaosTestSuite) AddTest(test *ChaosTest) {
	cts.mutex.Lock()
	defer cts.mutex.Unlock()
	cts.tests = append(cts.tests, test)
}

// RunAllScenarios 运行所有场景
func (cts *ChaosTestSuite) RunAllScenarios(ctx context.Context) error {
	cts.mutex.RLock()
	tests := make([]*ChaosTest, len(cts.tests))
	copy(tests, cts.tests)
	cts.mutex.RUnlock()

	for _, test := range tests {
		for _, scenario := range test.scenarios {
			if scenario.Enabled {
				if err := test.RunScenario(ctx, scenario.ID); err != nil {
					// 记录错误但继续运行其他场景
					continue
				}
			}
		}
	}

	return nil
}

// GetTestCount 获取测试数量
func (cts *ChaosTestSuite) GetTestCount() int {
	cts.mutex.RLock()
	defer cts.mutex.RUnlock()
	return len(cts.tests)
}
