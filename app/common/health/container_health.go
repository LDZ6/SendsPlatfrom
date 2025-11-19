package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ContainerHealthChecker 容器健康检查器
type ContainerHealthChecker struct {
	checks   map[string]HealthCheck
	results  map[string]HealthResult
	mutex    sync.RWMutex
	interval time.Duration
	timeout  time.Duration
}

// HealthCheck 健康检查接口
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) HealthResult
	Timeout() time.Duration
}

// HealthResult 健康检查结果
type HealthResult struct {
	Name      string                 `json:"name"`
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthStatus 健康状态
type HealthStatus int

const (
	StatusHealthy HealthStatus = iota
	StatusUnhealthy
	StatusDegraded
)

// String 健康状态字符串
func (hs HealthStatus) String() string {
	switch hs {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusDegraded:
		return "degraded"
	default:
		return "unknown"
	}
}

// NewContainerHealthChecker 创建容器健康检查器
func NewContainerHealthChecker(interval, timeout time.Duration) *ContainerHealthChecker {
	return &ContainerHealthChecker{
		checks:   make(map[string]HealthCheck),
		results:  make(map[string]HealthResult),
		interval: interval,
		timeout:  timeout,
	}
}

// AddCheck 添加健康检查
func (chc *ContainerHealthChecker) AddCheck(check HealthCheck) {
	chc.mutex.Lock()
	defer chc.mutex.Unlock()
	chc.checks[check.Name()] = check
}

// RemoveCheck 移除健康检查
func (chc *ContainerHealthChecker) RemoveCheck(name string) {
	chc.mutex.Lock()
	defer chc.mutex.Unlock()
	delete(chc.checks, name)
	delete(chc.results, name)
}

// Start 开始健康检查
func (chc *ContainerHealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(chc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			chc.runChecks(ctx)
		}
	}
}

// runChecks 运行健康检查
func (chc *ContainerHealthChecker) runChecks(ctx context.Context) {
	chc.mutex.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range chc.checks {
		checks[name] = check
	}
	chc.mutex.RUnlock()

	for name, check := range checks {
		go chc.runCheck(ctx, name, check)
	}
}

// runCheck 运行单个健康检查
func (chc *ContainerHealthChecker) runCheck(ctx context.Context, name string, check HealthCheck) {
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout())
	defer cancel()

	start := time.Now()
	result := check.Check(checkCtx)
	result.Duration = time.Since(start)
	result.Timestamp = time.Now()

	chc.mutex.Lock()
	chc.results[name] = result
	chc.mutex.Unlock()
}

// GetResults 获取健康检查结果
func (chc *ContainerHealthChecker) GetResults() map[string]HealthResult {
	chc.mutex.RLock()
	defer chc.mutex.RUnlock()

	results := make(map[string]HealthResult)
	for name, result := range chc.results {
		results[name] = result
	}

	return results
}

// GetOverallStatus 获取整体状态
func (chc *ContainerHealthChecker) GetOverallStatus() HealthStatus {
	chc.mutex.RLock()
	defer chc.mutex.RUnlock()

	if len(chc.results) == 0 {
		return StatusUnhealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range chc.results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}

	return StatusHealthy
}

// HTTPHealthCheck HTTP健康检查
type HTTPHealthCheck struct {
	name    string
	url     string
	timeout time.Duration
	client  *http.Client
}

// NewHTTPHealthCheck 创建HTTP健康检查
func NewHTTPHealthCheck(name, url string, timeout time.Duration) *HTTPHealthCheck {
	return &HTTPHealthCheck{
		name:    name,
		url:     url,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

// Name 获取名称
func (hhc *HTTPHealthCheck) Name() string {
	return hhc.name
}

// Check 执行检查
func (hhc *HTTPHealthCheck) Check(ctx context.Context) HealthResult {
	req, err := http.NewRequestWithContext(ctx, "GET", hhc.url, nil)
	if err != nil {
		return HealthResult{
			Name:    hhc.name,
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	resp, err := hhc.client.Do(req)
	if err != nil {
		return HealthResult{
			Name:    hhc.name,
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return HealthResult{
			Name:    hhc.name,
			Status:  StatusHealthy,
			Message: "HTTP check passed",
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
				"url":         hhc.url,
			},
		}
	}

	return HealthResult{
		Name:    hhc.name,
		Status:  StatusUnhealthy,
		Message: fmt.Sprintf("HTTP check failed with status: %d", resp.StatusCode),
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
			"url":         hhc.url,
		},
	}
}

// Timeout 获取超时时间
func (hhc *HTTPHealthCheck) Timeout() time.Duration {
	return hhc.timeout
}

// DatabaseHealthCheck 数据库健康检查
type DatabaseHealthCheck struct {
	name      string
	checkFunc func(ctx context.Context) error
	timeout   time.Duration
}

// NewDatabaseHealthCheck 创建数据库健康检查
func NewDatabaseHealthCheck(name string, checkFunc func(ctx context.Context) error, timeout time.Duration) *DatabaseHealthCheck {
	return &DatabaseHealthCheck{
		name:      name,
		checkFunc: checkFunc,
		timeout:   timeout,
	}
}

// Name 获取名称
func (dhc *DatabaseHealthCheck) Name() string {
	return dhc.name
}

// Check 执行检查
func (dhc *DatabaseHealthCheck) Check(ctx context.Context) HealthResult {
	err := dhc.checkFunc(ctx)
	if err != nil {
		return HealthResult{
			Name:    dhc.name,
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Database check failed: %v", err),
		}
	}

	return HealthResult{
		Name:    dhc.name,
		Status:  StatusHealthy,
		Message: "Database check passed",
	}
}

// Timeout 获取超时时间
func (dhc *DatabaseHealthCheck) Timeout() time.Duration {
	return dhc.timeout
}

// CacheHealthCheck 缓存健康检查
type CacheHealthCheck struct {
	name      string
	checkFunc func(ctx context.Context) error
	timeout   time.Duration
}

// NewCacheHealthCheck 创建缓存健康检查
func NewCacheHealthCheck(name string, checkFunc func(ctx context.Context) error, timeout time.Duration) *CacheHealthCheck {
	return &CacheHealthCheck{
		name:      name,
		checkFunc: checkFunc,
		timeout:   timeout,
	}
}

// Name 获取名称
func (chc *CacheHealthCheck) Name() string {
	return chc.name
}

// Check 执行检查
func (chc *CacheHealthCheck) Check(ctx context.Context) HealthResult {
	err := chc.checkFunc(ctx)
	if err != nil {
		return HealthResult{
			Name:    chc.name,
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Cache check failed: %v", err),
		}
	}

	return HealthResult{
		Name:    chc.name,
		Status:  StatusHealthy,
		Message: "Cache check passed",
	}
}

// Timeout 获取超时时间
func (chc *CacheHealthCheck) Timeout() time.Duration {
	return chc.timeout
}

// ResourceHealthCheck 资源健康检查
type ResourceHealthCheck struct {
	name      string
	checkFunc func(ctx context.Context) (map[string]interface{}, error)
	timeout   time.Duration
}

// NewResourceHealthCheck 创建资源健康检查
func NewResourceHealthCheck(name string, checkFunc func(ctx context.Context) (map[string]interface{}, error), timeout time.Duration) *ResourceHealthCheck {
	return &ResourceHealthCheck{
		name:      name,
		checkFunc: checkFunc,
		timeout:   timeout,
	}
}

// Name 获取名称
func (rhc *ResourceHealthCheck) Name() string {
	return rhc.name
}

// Check 执行检查
func (rhc *ResourceHealthCheck) Check(ctx context.Context) HealthResult {
	details, err := rhc.checkFunc(ctx)
	if err != nil {
		return HealthResult{
			Name:    rhc.name,
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Resource check failed: %v", err),
		}
	}

	// 检查资源使用率
	cpuUsage, ok := details["cpu_usage"].(float64)
	if ok && cpuUsage > 90 {
		return HealthResult{
			Name:    rhc.name,
			Status:  StatusDegraded,
			Message: "High CPU usage detected",
			Details: details,
		}
	}

	memoryUsage, ok := details["memory_usage"].(float64)
	if ok && memoryUsage > 90 {
		return HealthResult{
			Name:    rhc.name,
			Status:  StatusDegraded,
			Message: "High memory usage detected",
			Details: details,
		}
	}

	return HealthResult{
		Name:    rhc.name,
		Status:  StatusHealthy,
		Message: "Resource check passed",
		Details: details,
	}
}

// Timeout 获取超时时间
func (rhc *ResourceHealthCheck) Timeout() time.Duration {
	return rhc.timeout
}

// HealthCheckHandler 健康检查处理器
type HealthCheckHandler struct {
	checker *ContainerHealthChecker
}

// NewHealthCheckHandler 创建健康检查处理器
func NewHealthCheckHandler(checker *ContainerHealthChecker) *HealthCheckHandler {
	return &HealthCheckHandler{
		checker: checker,
	}
}

// LivenessHandler 存活检查处理器
func (hch *HealthCheckHandler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	status := hch.checker.GetOverallStatus()

	if status == StatusHealthy || status == StatusDegraded {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Write([]byte(status.String()))
}

// ReadinessHandler 就绪检查处理器
func (hch *HealthCheckHandler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	status := hch.checker.GetOverallStatus()

	if status == StatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Write([]byte(status.String()))
}

// HealthHandler 健康检查处理器
func (hch *HealthCheckHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	results := hch.checker.GetResults()
	overallStatus := hch.checker.GetOverallStatus()

	response := map[string]interface{}{
		"status":    overallStatus.String(),
		"checks":    results,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")

	if overallStatus == StatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}
