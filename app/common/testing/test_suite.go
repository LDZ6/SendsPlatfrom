package testing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TestSuite 测试套件
type TestSuite struct {
	unitTests        []UnitTest
	integrationTests []IntegrationTest
	loadTests        []LoadTest
	performanceTests []PerformanceTest
	securityTests    []SecurityTest
	mutex            sync.RWMutex
	config           *TestConfig
}

// TestConfig 测试配置
type TestConfig struct {
	Timeout        time.Duration
	Parallel       bool
	MaxConcurrency int
	RetryCount     int
	RetryDelay     time.Duration
	Cleanup        bool
	Verbose        bool
	Coverage       bool
	Profile        bool
}

// UnitTest 单元测试
type UnitTest struct {
	Name        string
	Description string
	Function    func() error
	Timeout     time.Duration
	RetryCount  int
	Cleanup     func() error
}

// IntegrationTest 集成测试
type IntegrationTest struct {
	Name         string
	Description  string
	Setup        func() error
	Function     func() error
	Teardown     func() error
	Timeout      time.Duration
	RetryCount   int
	Dependencies []string
}

// LoadTest 负载测试
type LoadTest struct {
	Name        string
	Description string
	Function    func(concurrency int) error
	Concurrency int
	Duration    time.Duration
	RampUp      time.Duration
	RampDown    time.Duration
	Timeout     time.Duration
}

// PerformanceTest 性能测试
type PerformanceTest struct {
	Name        string
	Description string
	Function    func() (time.Duration, error)
	Threshold   time.Duration
	Timeout     time.Duration
	RetryCount  int
}

// SecurityTest 安全测试
type SecurityTest struct {
	Name        string
	Description string
	Function    func() error
	Timeout     time.Duration
	RetryCount  int
	Severity    string
}

// TestResult 测试结果
type TestResult struct {
	Name       string
	Type       string
	Status     string
	Duration   time.Duration
	Error      error
	RetryCount int
	Timestamp  time.Time
	Metadata   map[string]interface{}
}

// TestSuiteManager 测试套件管理器
type TestSuiteManager struct {
	suites map[string]*TestSuite
	mutex  sync.RWMutex
}

// NewTestSuite 创建测试套件
func NewTestSuite(config *TestConfig) *TestSuite {
	return &TestSuite{
		config: config,
	}
}

// AddUnitTest 添加单元测试
func (ts *TestSuite) AddUnitTest(test UnitTest) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.unitTests = append(ts.unitTests, test)
}

// AddIntegrationTest 添加集成测试
func (ts *TestSuite) AddIntegrationTest(test IntegrationTest) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.integrationTests = append(ts.integrationTests, test)
}

// AddLoadTest 添加负载测试
func (ts *TestSuite) AddLoadTest(test LoadTest) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.loadTests = append(ts.loadTests, test)
}

// AddPerformanceTest 添加性能测试
func (ts *TestSuite) AddPerformanceTest(test PerformanceTest) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.performanceTests = append(ts.performanceTests, test)
}

// AddSecurityTest 添加安全测试
func (ts *TestSuite) AddSecurityTest(test SecurityTest) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.securityTests = append(ts.securityTests, test)
}

// RunUnitTests 运行单元测试
func (ts *TestSuite) RunUnitTests() ([]TestResult, error) {
	ts.mutex.RLock()
	tests := ts.unitTests
	ts.mutex.RUnlock()

	var results []TestResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, test := range tests {
		wg.Add(1)
		go func(test UnitTest) {
			defer wg.Done()

			result := ts.runUnitTest(test)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(test)
	}

	wg.Wait()
	return results, nil
}

// runUnitTest 运行单个单元测试
func (ts *TestSuite) runUnitTest(test UnitTest) TestResult {
	start := time.Now()
	result := TestResult{
		Name:      test.Name,
		Type:      "unit",
		Status:    "running",
		Timestamp: start,
	}

	// 设置超时
	timeout := test.Timeout
	if timeout == 0 {
		timeout = ts.config.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 重试逻辑
	var err error
	for i := 0; i <= test.RetryCount; i++ {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break
		default:
			err = test.Function()
			if err == nil {
				break
			}
			if i < test.RetryCount {
				time.Sleep(ts.config.RetryDelay)
			}
		}
	}

	// 清理
	if test.Cleanup != nil {
		cleanupErr := test.Cleanup()
		if cleanupErr != nil && err == nil {
			err = cleanupErr
		}
	}

	result.Duration = time.Since(start)
	result.RetryCount = test.RetryCount

	if err != nil {
		result.Status = "failed"
		result.Error = err
	} else {
		result.Status = "passed"
	}

	return result
}

// RunIntegrationTests 运行集成测试
func (ts *TestSuite) RunIntegrationTests() ([]TestResult, error) {
	ts.mutex.RLock()
	tests := ts.integrationTests
	ts.mutex.RUnlock()

	var results []TestResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, test := range tests {
		wg.Add(1)
		go func(test IntegrationTest) {
			defer wg.Done()

			result := ts.runIntegrationTest(test)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(test)
	}

	wg.Wait()
	return results, nil
}

// runIntegrationTest 运行单个集成测试
func (ts *TestSuite) runIntegrationTest(test IntegrationTest) TestResult {
	start := time.Now()
	result := TestResult{
		Name:      test.Name,
		Type:      "integration",
		Status:    "running",
		Timestamp: start,
	}

	// 设置超时
	timeout := test.Timeout
	if timeout == 0 {
		timeout = ts.config.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 设置
	if test.Setup != nil {
		if err := test.Setup(); err != nil {
			result.Status = "failed"
			result.Error = err
			result.Duration = time.Since(start)
			return result
		}
	}

	// 重试逻辑
	var err error
	for i := 0; i <= test.RetryCount; i++ {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break
		default:
			err = test.Function()
			if err == nil {
				break
			}
			if i < test.RetryCount {
				time.Sleep(ts.config.RetryDelay)
			}
		}
	}

	// 清理
	if test.Teardown != nil {
		teardownErr := test.Teardown()
		if teardownErr != nil && err == nil {
			err = teardownErr
		}
	}

	result.Duration = time.Since(start)
	result.RetryCount = test.RetryCount

	if err != nil {
		result.Status = "failed"
		result.Error = err
	} else {
		result.Status = "passed"
	}

	return result
}

// RunLoadTests 运行负载测试
func (ts *TestSuite) RunLoadTests() ([]TestResult, error) {
	ts.mutex.RLock()
	tests := ts.loadTests
	ts.mutex.RUnlock()

	var results []TestResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, test := range tests {
		wg.Add(1)
		go func(test LoadTest) {
			defer wg.Done()

			result := ts.runLoadTest(test)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(test)
	}

	wg.Wait()
	return results, nil
}

// runLoadTest 运行单个负载测试
func (ts *TestSuite) runLoadTest(test LoadTest) TestResult {
	start := time.Now()
	result := TestResult{
		Name:      test.Name,
		Type:      "load",
		Status:    "running",
		Timestamp: start,
	}

	// 设置超时
	timeout := test.Timeout
	if timeout == 0 {
		timeout = ts.config.Timeout
	}

	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 运行负载测试
	err := test.Function(test.Concurrency)

	result.Duration = time.Since(start)

	if err != nil {
		result.Status = "failed"
		result.Error = err
	} else {
		result.Status = "passed"
	}

	return result
}

// RunPerformanceTests 运行性能测试
func (ts *TestSuite) RunPerformanceTests() ([]TestResult, error) {
	ts.mutex.RLock()
	tests := ts.performanceTests
	ts.mutex.RUnlock()

	var results []TestResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, test := range tests {
		wg.Add(1)
		go func(test PerformanceTest) {
			defer wg.Done()

			result := ts.runPerformanceTest(test)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(test)
	}

	wg.Wait()
	return results, nil
}

// runPerformanceTest 运行单个性能测试
func (ts *TestSuite) runPerformanceTest(test PerformanceTest) TestResult {
	start := time.Now()
	result := TestResult{
		Name:      test.Name,
		Type:      "performance",
		Status:    "running",
		Timestamp: start,
	}

	// 设置超时
	timeout := test.Timeout
	if timeout == 0 {
		timeout = ts.config.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 重试逻辑
	var err error
	var duration time.Duration
	for i := 0; i <= test.RetryCount; i++ {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break
		default:
			duration, err = test.Function()
			if err == nil {
				break
			}
			if i < test.RetryCount {
				time.Sleep(ts.config.RetryDelay)
			}
		}
	}

	result.Duration = time.Since(start)
	result.RetryCount = test.RetryCount

	if err != nil {
		result.Status = "failed"
		result.Error = err
	} else if duration > test.Threshold {
		result.Status = "failed"
		result.Error = fmt.Errorf("performance threshold exceeded: %v > %v", duration, test.Threshold)
	} else {
		result.Status = "passed"
		result.Metadata = map[string]interface{}{
			"duration":  duration,
			"threshold": test.Threshold,
		}
	}

	return result
}

// RunSecurityTests 运行安全测试
func (ts *TestSuite) RunSecurityTests() ([]TestResult, error) {
	ts.mutex.RLock()
	tests := ts.securityTests
	ts.mutex.RUnlock()

	var results []TestResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, test := range tests {
		wg.Add(1)
		go func(test SecurityTest) {
			defer wg.Done()

			result := ts.runSecurityTest(test)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(test)
	}

	wg.Wait()
	return results, nil
}

// runSecurityTest 运行单个安全测试
func (ts *TestSuite) runSecurityTest(test SecurityTest) TestResult {
	start := time.Now()
	result := TestResult{
		Name:      test.Name,
		Type:      "security",
		Status:    "running",
		Timestamp: start,
	}

	// 设置超时
	timeout := test.Timeout
	if timeout == 0 {
		timeout = ts.config.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 重试逻辑
	var err error
	for i := 0; i <= test.RetryCount; i++ {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break
		default:
			err = test.Function()
			if err == nil {
				break
			}
			if i < test.RetryCount {
				time.Sleep(ts.config.RetryDelay)
			}
		}
	}

	result.Duration = time.Since(start)
	result.RetryCount = test.RetryCount

	if err != nil {
		result.Status = "failed"
		result.Error = err
	} else {
		result.Status = "passed"
	}

	return result
}

// RunAllTests 运行所有测试
func (ts *TestSuite) RunAllTests() ([]TestResult, error) {
	var allResults []TestResult

	// 运行单元测试
	unitResults, err := ts.RunUnitTests()
	if err != nil {
		return nil, fmt.Errorf("unit tests failed: %w", err)
	}
	allResults = append(allResults, unitResults...)

	// 运行集成测试
	integrationResults, err := ts.RunIntegrationTests()
	if err != nil {
		return nil, fmt.Errorf("integration tests failed: %w", err)
	}
	allResults = append(allResults, integrationResults...)

	// 运行性能测试
	performanceResults, err := ts.RunPerformanceTests()
	if err != nil {
		return nil, fmt.Errorf("performance tests failed: %w", err)
	}
	allResults = append(allResults, performanceResults...)

	// 运行安全测试
	securityResults, err := ts.RunSecurityTests()
	if err != nil {
		return nil, fmt.Errorf("security tests failed: %w", err)
	}
	allResults = append(allResults, securityResults...)

	// 运行负载测试
	loadResults, err := ts.RunLoadTests()
	if err != nil {
		return nil, fmt.Errorf("load tests failed: %w", err)
	}
	allResults = append(allResults, loadResults...)

	return allResults, nil
}

// GetTestStats 获取测试统计
func (ts *TestSuite) GetTestStats(results []TestResult) map[string]interface{} {
	stats := map[string]interface{}{
		"total":     len(results),
		"passed":    0,
		"failed":    0,
		"running":   0,
		"duration":  time.Duration(0),
		"by_type":   make(map[string]int),
		"by_status": make(map[string]int),
	}

	for _, result := range results {
		// 统计状态
		status := result.Status
		stats["by_status"].(map[string]int)[status]++

		// 统计类型
		testType := result.Type
		stats["by_type"].(map[string]int)[testType]++

		// 统计通过/失败
		if status == "passed" {
			stats["passed"] = stats["passed"].(int) + 1
		} else if status == "failed" {
			stats["failed"] = stats["failed"].(int) + 1
		} else if status == "running" {
			stats["running"] = stats["running"].(int) + 1
		}

		// 累计持续时间
		stats["duration"] = stats["duration"].(time.Duration) + result.Duration
	}

	return stats
}

// NewTestSuiteManager 创建测试套件管理器
func NewTestSuiteManager() *TestSuiteManager {
	return &TestSuiteManager{
		suites: make(map[string]*TestSuite),
	}
}

// AddSuite 添加测试套件
func (tsm *TestSuiteManager) AddSuite(name string, suite *TestSuite) {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()
	tsm.suites[name] = suite
}

// GetSuite 获取测试套件
func (tsm *TestSuiteManager) GetSuite(name string) (*TestSuite, bool) {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	suite, exists := tsm.suites[name]
	return suite, exists
}

// RunSuite 运行测试套件
func (tsm *TestSuiteManager) RunSuite(name string) ([]TestResult, error) {
	suite, exists := tsm.GetSuite(name)
	if !exists {
		return nil, fmt.Errorf("test suite not found: %s", name)
	}

	return suite.RunAllTests()
}

// RunAllSuites 运行所有测试套件
func (tsm *TestSuiteManager) RunAllSuites() (map[string][]TestResult, error) {
	tsm.mutex.RLock()
	suites := tsm.suites
	tsm.mutex.RUnlock()

	results := make(map[string][]TestResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, suite := range suites {
		wg.Add(1)
		go func(name string, suite *TestSuite) {
			defer wg.Done()

			suiteResults, err := suite.RunAllTests()
			if err != nil {
				suiteResults = []TestResult{{
					Name:   "suite_error",
					Type:   "error",
					Status: "failed",
					Error:  err,
				}}
			}

			mu.Lock()
			results[name] = suiteResults
			mu.Unlock()
		}(name, suite)
	}

	wg.Wait()
	return results, nil
}
