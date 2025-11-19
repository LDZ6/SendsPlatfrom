package performance

import (
	"context"
	"sync"
	"time"
)

// Job 任务接口
type Job interface {
	Execute(ctx context.Context) error
	GetID() string
	GetPriority() int
}

// Worker 工作者
type Worker struct {
	id         int
	jobQueue   chan Job
	workerPool chan chan Job
	quit       chan bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWorker 创建工作者
func NewWorker(id int, workerPool chan chan Job) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		id:         id,
		jobQueue:   make(chan Job),
		workerPool: workerPool,
		quit:       make(chan bool),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动工作者
func (w *Worker) Start() {
	go func() {
		for {
			// 将工作者的任务队列注册到工作池
			w.workerPool <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				// 执行任务
				if err := job.Execute(w.ctx); err != nil {
					// 记录错误，但不停止工作者
					// TODO: 添加错误日志
				}
			case <-w.quit:
				return
			case <-w.ctx.Done():
				return
			}
		}
	}()
}

// Stop 停止工作者
func (w *Worker) Stop() {
	w.cancel()
	w.quit <- true
}

// WorkerPool 工作池
type WorkerPool struct {
	workers    int
	jobQueue   chan Job
	workerPool chan chan Job
	quit       chan bool
	ctx        context.Context
	cancel     context.CancelFunc
	workers    []*Worker
	mutex      sync.RWMutex
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:    workers,
		jobQueue:   make(chan Job, 1000), // 任务队列缓冲区
		workerPool: make(chan chan Job, workers),
		quit:       make(chan bool),
		ctx:        ctx,
		cancel:     cancel,
		workers:    make([]*Worker, workers),
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	// 启动所有工作者
	for i := 0; i < wp.workers; i++ {
		worker := NewWorker(i, wp.workerPool)
		wp.workers[i] = worker
		worker.Start()
	}

	// 启动调度器
	go wp.dispatch()
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.quit <- true

	// 停止所有工作者
	wp.mutex.RLock()
	for _, worker := range wp.workers {
		worker.Stop()
	}
	wp.mutex.RUnlock()
}

// dispatch 调度任务
func (wp *WorkerPool) dispatch() {
	for {
		select {
		case job := <-wp.jobQueue:
			// 获取可用的工作者
			worker := <-wp.workerPool
			// 将任务分配给工作者
			worker <- job
		case <-wp.quit:
			return
		case <-wp.ctx.Done():
			return
		}
	}
}

// Submit 提交任务
func (wp *WorkerPool) Submit(job Job) error {
	select {
	case wp.jobQueue <- job:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		return ErrWorkerPoolFull
	}
}

// SubmitWithTimeout 带超时的任务提交
func (wp *WorkerPool) SubmitWithTimeout(job Job, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(wp.ctx, timeout)
	defer cancel()

	select {
	case wp.jobQueue <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetStats 获取工作池统计信息
func (wp *WorkerPool) GetStats() map[string]interface{} {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()

	return map[string]interface{}{
		"workers":     wp.workers,
		"queue_size":  len(wp.jobQueue),
		"queue_cap":   cap(wp.jobQueue),
		"active_jobs": len(wp.workerPool),
	}
}

// 错误定义
var (
	ErrWorkerPoolFull = &WorkerPoolError{message: "worker pool is full"}
)

// WorkerPoolError 工作池错误
type WorkerPoolError struct {
	message string
}

func (e *WorkerPoolError) Error() string {
	return e.message
}

// PriorityJob 优先级任务
type PriorityJob struct {
	id       string
	priority int
	task     func(ctx context.Context) error
}

// NewPriorityJob 创建优先级任务
func NewPriorityJob(id string, priority int, task func(ctx context.Context) error) *PriorityJob {
	return &PriorityJob{
		id:       id,
		priority: priority,
		task:     task,
	}
}

// Execute 执行任务
func (pj *PriorityJob) Execute(ctx context.Context) error {
	return pj.task(ctx)
}

// GetID 获取任务ID
func (pj *PriorityJob) GetID() string {
	return pj.id
}

// GetPriority 获取优先级
func (pj *PriorityJob) GetPriority() int {
	return pj.priority
}

// PriorityWorkerPool 优先级工作池
type PriorityWorkerPool struct {
	*WorkerPool
	priorityQueue *PriorityQueue
}

// NewPriorityWorkerPool 创建优先级工作池
func NewPriorityWorkerPool(workers int) *PriorityWorkerPool {
	return &PriorityWorkerPool{
		WorkerPool:    NewWorkerPool(workers),
		priorityQueue: NewPriorityQueue(),
	}
}

// Submit 提交优先级任务
func (pwp *PriorityWorkerPool) Submit(job Job) error {
	// 将任务添加到优先级队列
	pwp.priorityQueue.Push(job)

	// 尝试从优先级队列获取任务并提交到工作池
	if topJob := pwp.priorityQueue.Pop(); topJob != nil {
		return pwp.WorkerPool.Submit(topJob)
	}

	return nil
}

// PriorityQueue 优先级队列
type PriorityQueue struct {
	jobs  []Job
	mutex sync.RWMutex
}

// NewPriorityQueue 创建优先级队列
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		jobs: make([]Job, 0),
	}
}

// Push 添加任务
func (pq *PriorityQueue) Push(job Job) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.jobs = append(pq.jobs, job)
	// 按优先级排序（优先级高的在前）
	pq.sort()
}

// Pop 获取最高优先级任务
func (pq *PriorityQueue) Pop() Job {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if len(pq.jobs) == 0 {
		return nil
	}

	job := pq.jobs[0]
	pq.jobs = pq.jobs[1:]
	return job
}

// sort 按优先级排序
func (pq *PriorityQueue) sort() {
	// 简单的冒泡排序，实际项目中可以使用堆排序
	for i := 0; i < len(pq.jobs)-1; i++ {
		for j := 0; j < len(pq.jobs)-1-i; j++ {
			if pq.jobs[j].GetPriority() < pq.jobs[j+1].GetPriority() {
				pq.jobs[j], pq.jobs[j+1] = pq.jobs[j+1], pq.jobs[j]
			}
		}
	}
}

// Size 获取队列大小
func (pq *PriorityQueue) Size() int {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	return len(pq.jobs)
}
