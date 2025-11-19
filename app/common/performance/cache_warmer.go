package performance

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WarmupStrategy 预热策略接口
type WarmupStrategy interface {
	Warmup(ctx context.Context) error
	GetName() string
	GetPriority() int
	IsEnabled() bool
}

// CacheWarmer 缓存预热器
type CacheWarmer struct {
	strategies map[string]WarmupStrategy
	scheduler  *Scheduler
	mutex      sync.RWMutex
}

// NewCacheWarmer 创建缓存预热器
func NewCacheWarmer() *CacheWarmer {
	return &CacheWarmer{
		strategies: make(map[string]WarmupStrategy),
		scheduler:  NewScheduler(),
	}
}

// RegisterStrategy 注册预热策略
func (cw *CacheWarmer) RegisterStrategy(strategy WarmupStrategy) {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	cw.strategies[strategy.GetName()] = strategy
}

// Warmup 执行预热
func (cw *CacheWarmer) Warmup(ctx context.Context) error {
	cw.mutex.RLock()
	strategies := make([]WarmupStrategy, 0, len(cw.strategies))
	for _, strategy := range cw.strategies {
		if strategy.IsEnabled() {
			strategies = append(strategies, strategy)
		}
	}
	cw.mutex.RUnlock()

	// 按优先级排序
	for i := 0; i < len(strategies)-1; i++ {
		for j := i + 1; j < len(strategies); j++ {
			if strategies[i].GetPriority() > strategies[j].GetPriority() {
				strategies[i], strategies[j] = strategies[j], strategies[i]
			}
		}
	}

	// 执行预热策略
	for _, strategy := range strategies {
		if err := strategy.Warmup(ctx); err != nil {
			return fmt.Errorf("warmup strategy %s failed: %w", strategy.GetName(), err)
		}
	}

	return nil
}

// ScheduleWarmup 调度预热
func (cw *CacheWarmer) ScheduleWarmup(interval time.Duration) error {
	return cw.scheduler.Schedule("cache_warmup", func(ctx context.Context) error {
		return cw.Warmup(ctx)
	}, interval)
}

// Scheduler 调度器
type Scheduler struct {
	jobs  map[string]*ScheduledJob
	mutex sync.RWMutex
}

// ScheduledJob 调度任务
type ScheduledJob struct {
	ID       string
	Function func(context.Context) error
	Interval time.Duration
	Ticker   *time.Ticker
	Stop     chan bool
	Running  bool
}

// NewScheduler 创建调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*ScheduledJob),
	}
}

// Schedule 调度任务
func (s *Scheduler) Schedule(id string, fn func(context.Context) error, interval time.Duration) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 如果任务已存在，先停止
	if job, exists := s.jobs[id]; exists {
		job.Stop <- true
	}

	job := &ScheduledJob{
		ID:       id,
		Function: fn,
		Interval: interval,
		Ticker:   time.NewTicker(interval),
		Stop:     make(chan bool),
		Running:  true,
	}

	s.jobs[id] = job

	// 启动任务
	go s.runJob(job)

	return nil
}

// runJob 运行任务
func (s *Scheduler) runJob(job *ScheduledJob) {
	for {
		select {
		case <-job.Ticker.C:
			ctx := context.Background()
			if err := job.Function(ctx); err != nil {
				// 记录错误
				continue
			}
		case <-job.Stop:
			job.Ticker.Stop()
			job.Running = false
			return
		}
	}
}

// StopJob 停止任务
func (s *Scheduler) StopJob(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Stop <- true
	delete(s.jobs, id)

	return nil
}

// StopAll 停止所有任务
func (s *Scheduler) StopAll() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, job := range s.jobs {
		job.Stop <- true
	}

	s.jobs = make(map[string]*ScheduledJob)
}

// CacheConsistency 缓存一致性
type CacheConsistency struct {
	invalidationStrategy InvalidationStrategy
	replicationStrategy  ReplicationStrategy
	mutex                sync.RWMutex
}

// InvalidationStrategy 失效策略接口
type InvalidationStrategy interface {
	Invalidate(key string) error
	InvalidatePattern(pattern string) error
	InvalidateAll() error
}

// ReplicationStrategy 复制策略接口
type ReplicationStrategy interface {
	Replicate(key string, value interface{}) error
	ReplicateAll() error
	Sync() error
}

// NewCacheConsistency 创建缓存一致性
func NewCacheConsistency(invalidation InvalidationStrategy, replication ReplicationStrategy) *CacheConsistency {
	return &CacheConsistency{
		invalidationStrategy: invalidation,
		replicationStrategy:  replication,
	}
}

// Invalidate 失效缓存
func (cc *CacheConsistency) Invalidate(key string) error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	return cc.invalidationStrategy.Invalidate(key)
}

// InvalidatePattern 按模式失效缓存
func (cc *CacheConsistency) InvalidatePattern(pattern string) error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	return cc.invalidationStrategy.InvalidatePattern(pattern)
}

// InvalidateAll 失效所有缓存
func (cc *CacheConsistency) InvalidateAll() error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	return cc.invalidationStrategy.InvalidateAll()
}

// Replicate 复制缓存
func (cc *CacheConsistency) Replicate(key string, value interface{}) error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	return cc.replicationStrategy.Replicate(key, value)
}

// ReplicateAll 复制所有缓存
func (cc *CacheConsistency) ReplicateAll() error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	return cc.replicationStrategy.ReplicateAll()
}

// Sync 同步缓存
func (cc *CacheConsistency) Sync() error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	return cc.replicationStrategy.Sync()
}

// AdaptiveWorkerPool 自适应工作池
type AdaptiveWorkerPool struct {
	minWorkers     int
	maxWorkers     int
	currentWorkers int
	loadBalancer   LoadBalancer
	workers        []*Worker
	mutex          sync.RWMutex
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	SelectWorker(workers []*Worker) *Worker
	GetLoad(worker *Worker) float64
}

// Worker 工作者
type Worker struct {
	ID     string
	Tasks  chan Task
	Load   float64
	Active bool
	mutex  sync.RWMutex
}

// Task 任务接口
type Task interface {
	Execute() error
	GetID() string
	GetPriority() int
}

// NewAdaptiveWorkerPool 创建自适应工作池
func NewAdaptiveWorkerPool(minWorkers, maxWorkers int, loadBalancer LoadBalancer) *AdaptiveWorkerPool {
	return &AdaptiveWorkerPool{
		minWorkers:     minWorkers,
		maxWorkers:     maxWorkers,
		currentWorkers: minWorkers,
		loadBalancer:   loadBalancer,
		workers:        make([]*Worker, 0, maxWorkers),
	}
}

// Start 启动工作池
func (awp *AdaptiveWorkerPool) Start() error {
	awp.mutex.Lock()
	defer awp.mutex.Unlock()

	// 创建初始工作者
	for i := 0; i < awp.minWorkers; i++ {
		worker := &Worker{
			ID:     fmt.Sprintf("worker_%d", i),
			Tasks:  make(chan Task, 100),
			Load:   0.0,
			Active: true,
		}
		awp.workers = append(awp.workers, worker)
		go awp.runWorker(worker)
	}

	return nil
}

// runWorker 运行工作者
func (awp *AdaptiveWorkerPool) runWorker(worker *Worker) {
	for {
		select {
		case task := <-worker.Tasks:
			worker.mutex.Lock()
			worker.Load += 0.1
			worker.mutex.Unlock()

			if err := task.Execute(); err != nil {
				// 记录错误
			}

			worker.mutex.Lock()
			worker.Load -= 0.1
			worker.mutex.Unlock()
		}
	}
}

// SubmitTask 提交任务
func (awp *AdaptiveWorkerPool) SubmitTask(task Task) error {
	awp.mutex.RLock()
	workers := make([]*Worker, len(awp.workers))
	copy(workers, awp.workers)
	awp.mutex.RUnlock()

	// 选择工作者
	worker := awp.loadBalancer.SelectWorker(workers)
	if worker == nil {
		return fmt.Errorf("no available worker")
	}

	// 提交任务
	select {
	case worker.Tasks <- task:
		return nil
	default:
		return fmt.Errorf("worker queue is full")
	}
}

// ScaleUp 扩容
func (awp *AdaptiveWorkerPool) ScaleUp() error {
	awp.mutex.Lock()
	defer awp.mutex.Unlock()

	if awp.currentWorkers >= awp.maxWorkers {
		return fmt.Errorf("already at maximum workers")
	}

	worker := &Worker{
		ID:     fmt.Sprintf("worker_%d", awp.currentWorkers),
		Tasks:  make(chan Task, 100),
		Load:   0.0,
		Active: true,
	}
	awp.workers = append(awp.workers, worker)
	awp.currentWorkers++

	go awp.runWorker(worker)

	return nil
}

// ScaleDown 缩容
func (awp *AdaptiveWorkerPool) ScaleDown() error {
	awp.mutex.Lock()
	defer awp.mutex.Unlock()

	if awp.currentWorkers <= awp.minWorkers {
		return fmt.Errorf("already at minimum workers")
	}

	// 找到负载最低的工作者
	var lowestLoadWorker *Worker
	lowestLoad := float64(1.0)

	for _, worker := range awp.workers {
		if worker.Active {
			worker.mutex.RLock()
			load := worker.Load
			worker.mutex.RUnlock()

			if load < lowestLoad {
				lowestLoad = load
				lowestLoadWorker = worker
			}
		}
	}

	if lowestLoadWorker != nil {
		lowestLoadWorker.Active = false
		close(lowestLoadWorker.Tasks)
		awp.currentWorkers--
	}

	return nil
}

// GetStats 获取统计信息
func (awp *AdaptiveWorkerPool) GetStats() map[string]interface{} {
	awp.mutex.RLock()
	defer awp.mutex.RUnlock()

	stats := map[string]interface{}{
		"min_workers":     awp.minWorkers,
		"max_workers":     awp.maxWorkers,
		"current_workers": awp.currentWorkers,
		"workers":         make([]map[string]interface{}, 0, len(awp.workers)),
	}

	for _, worker := range awp.workers {
		worker.mutex.RLock()
		workerStats := map[string]interface{}{
			"id":     worker.ID,
			"load":   worker.Load,
			"active": worker.Active,
		}
		worker.mutex.RUnlock()
		stats["workers"] = append(stats["workers"].([]map[string]interface{}), workerStats)
	}

	return stats
}

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	lastIndex int
	mutex     sync.Mutex
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		lastIndex: -1,
	}
}

// SelectWorker 选择工作者
func (rrlb *RoundRobinLoadBalancer) SelectWorker(workers []*Worker) *Worker {
	rrlb.mutex.Lock()
	defer rrlb.mutex.Unlock()

	activeWorkers := make([]*Worker, 0)
	for _, worker := range workers {
		if worker.Active {
			activeWorkers = append(activeWorkers, worker)
		}
	}

	if len(activeWorkers) == 0 {
		return nil
	}

	rrlb.lastIndex = (rrlb.lastIndex + 1) % len(activeWorkers)
	return activeWorkers[rrlb.lastIndex]
}

// GetLoad 获取负载
func (rrlb *RoundRobinLoadBalancer) GetLoad(worker *Worker) float64 {
	worker.mutex.RLock()
	defer worker.mutex.RUnlock()
	return worker.Load
}

// LeastConnectionsLoadBalancer 最少连接负载均衡器
type LeastConnectionsLoadBalancer struct{}

// NewLeastConnectionsLoadBalancer 创建最少连接负载均衡器
func NewLeastConnectionsLoadBalancer() *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{}
}

// SelectWorker 选择工作者
func (lclb *LeastConnectionsLoadBalancer) SelectWorker(workers []*Worker) *Worker {
	var selectedWorker *Worker
	minLoad := float64(1.0)

	for _, worker := range workers {
		if worker.Active {
			worker.mutex.RLock()
			load := worker.Load
			worker.mutex.RUnlock()

			if load < minLoad {
				minLoad = load
				selectedWorker = worker
			}
		}
	}

	return selectedWorker
}

// GetLoad 获取负载
func (lclb *LeastConnectionsLoadBalancer) GetLoad(worker *Worker) float64 {
	worker.mutex.RLock()
	defer worker.mutex.RUnlock()
	return worker.Load
}
