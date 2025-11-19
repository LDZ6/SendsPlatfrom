package common

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"platform/app/common/performance"
)

func TestWorkerPool(t *testing.T) {
	// 创建工作池
	wp := performance.NewWorkerPool(3)
	wp.Start()
	defer wp.Stop()

	// 创建测试任务
	task := &TestTask{
		id:   "test-task-1",
		data: "test data",
	}

	// 提交任务
	err := wp.Submit(task)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)

	// 验证任务是否执行
	if !task.executed {
		t.Error("Expected task to be executed")
	}
}

func TestPriorityWorkerPool(t *testing.T) {
	// 创建优先级工作池
	pwp := performance.NewPriorityWorkerPool(2)
	pwp.Start()
	defer pwp.Stop()

	// 创建不同优先级的任务
	highPriorityTask := &TestTask{
		id:       "high-priority",
		priority: 10,
		data:     "high priority data",
	}

	lowPriorityTask := &TestTask{
		id:       "low-priority",
		priority: 1,
		data:     "low priority data",
	}

	// 提交任务
	err := pwp.Submit(highPriorityTask)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = pwp.Submit(lowPriorityTask)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)

	// 验证任务是否执行
	if !highPriorityTask.executed {
		t.Error("Expected high priority task to be executed")
	}
	if !lowPriorityTask.executed {
		t.Error("Expected low priority task to be executed")
	}
}

func TestRequestMerger(t *testing.T) {
	// 创建请求合并器
	merger := &TestMerger{}
	rm := performance.NewRequestMerger(merger, 2, 100*time.Millisecond)
	defer rm.Close()

	// 提交请求
	ctx := context.Background()
	response, err := rm.Submit(ctx, "key1", "data1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response.Data != "processed_data1" {
		t.Errorf("Expected 'processed_data1', got %v", response.Data)
	}
}

func TestBackpressureController(t *testing.T) {
	// 创建背压控制器
	bpc := performance.NewBackpressureController(2)

	// 测试获取许可
	err := bpc.Acquire()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = bpc.Acquire()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试背压
	err = bpc.Acquire()
	if err == nil {
		t.Error("Expected error due to backpressure")
	}

	// 释放许可
	bpc.Release()

	// 再次获取许可
	err = bpc.Acquire()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRateLimiter(t *testing.T) {
	// 创建限流器
	rl := performance.NewRateLimiter(100*time.Millisecond, 2)

	// 测试初始令牌
	if !rl.Allow() {
		t.Error("Expected to allow first request")
	}

	if !rl.Allow() {
		t.Error("Expected to allow second request")
	}

	// 测试限流
	if rl.Allow() {
		t.Error("Expected to reject third request")
	}

	// 等待令牌补充
	time.Sleep(150 * time.Millisecond)

	if !rl.Allow() {
		t.Error("Expected to allow request after refill")
	}
}

func TestBatchProcessor(t *testing.T) {
	var processedBatches [][]interface{}

	// 创建批处理器
	bp := performance.NewBatchProcessor(func(batch []interface{}) error {
		processedBatches = append(processedBatches, batch)
		return nil
	}, 3, 100*time.Millisecond)
	defer bp.Close()

	// 添加项目
	err := bp.Add("item1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = bp.Add("item2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = bp.Add("item3")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 等待处理
	time.Sleep(150 * time.Millisecond)

	// 验证批次处理
	if len(processedBatches) == 0 {
		t.Error("Expected batches to be processed")
	}

	if len(processedBatches[0]) != 3 {
		t.Errorf("Expected batch size 3, got %d", len(processedBatches[0]))
	}
}

func TestConcurrentLimiter(t *testing.T) {
	// 创建并发限制器
	cl := performance.NewConcurrentLimiter(2)

	// 测试并发执行
	var wg sync.WaitGroup
	executed := 0

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cl.Execute(func() error {
				executed++
				time.Sleep(10 * time.Millisecond)
				return nil
			})
			if err != nil {
				// 被限制的请求
			}
		}()
	}

	wg.Wait()

	// 验证执行数量
	if executed > 2 {
		t.Errorf("Expected at most 2 concurrent executions, got %d", executed)
	}
}

func TestPriorityQueue(t *testing.T) {
	pq := performance.NewPriorityQueue()

	// 创建不同优先级的任务
	task1 := &TestTask{id: "task1", priority: 1}
	task2 := &TestTask{id: "task2", priority: 3}
	task3 := &TestTask{id: "task3", priority: 2}

	// 添加任务
	pq.Push(task1)
	pq.Push(task2)
	pq.Push(task3)

	// 验证队列大小
	if pq.Size() != 3 {
		t.Errorf("Expected queue size 3, got %d", pq.Size())
	}

	// 获取任务（应该按优先级排序）
	job1 := pq.Pop()
	if job1.GetPriority() != 3 {
		t.Errorf("Expected priority 3, got %d", job1.GetPriority())
	}

	job2 := pq.Pop()
	if job2.GetPriority() != 2 {
		t.Errorf("Expected priority 2, got %d", job2.GetPriority())
	}

	job3 := pq.Pop()
	if job3.GetPriority() != 1 {
		t.Errorf("Expected priority 1, got %d", job3.GetPriority())
	}

	// 验证队列为空
	if pq.Size() != 0 {
		t.Errorf("Expected empty queue, got size %d", pq.Size())
	}
}

// TestTask 测试任务
type TestTask struct {
	id       string
	priority int
	data     string
	executed bool
}

func (tt *TestTask) Execute(ctx context.Context) error {
	tt.executed = true
	return nil
}

func (tt *TestTask) GetID() string {
	return tt.id
}

func (tt *TestTask) GetPriority() int {
	return tt.priority
}

// TestMerger 测试合并器
type TestMerger struct{}

func (tm *TestMerger) Merge(requests []*performance.MergedRequest) (map[string]*performance.MergedResponse, error) {
	responses := make(map[string]*performance.MergedResponse)

	for _, req := range requests {
		responses[req.Key] = &performance.MergedResponse{
			Data:  "processed_" + req.Data.(string),
			Error: nil,
		}
	}

	return responses, nil
}

func TestWorkerPoolConcurrency(t *testing.T) {
	// 创建工作池
	wp := performance.NewWorkerPool(5)
	wp.Start()
	defer wp.Stop()

	// 创建多个任务
	var wg sync.WaitGroup
	taskCount := 10

	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			task := &TestTask{
				id:   fmt.Sprintf("task-%d", id),
				data: fmt.Sprintf("data-%d", id),
			}

			err := wp.Submit(task)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		}(i)
	}

	wg.Wait()

	// 等待所有任务完成
	time.Sleep(200 * time.Millisecond)
}

func TestWorkerPoolStats(t *testing.T) {
	// 创建工作池
	wp := performance.NewWorkerPool(3)
	wp.Start()
	defer wp.Stop()

	// 获取统计信息
	stats := wp.GetStats()

	if stats["workers"] != 3 {
		t.Errorf("Expected 3 workers, got %v", stats["workers"])
	}

	if stats["queue_cap"] != 1000 {
		t.Errorf("Expected queue capacity 1000, got %v", stats["queue_cap"])
	}
}

func TestRequestMergerTimeout(t *testing.T) {
	// 创建请求合并器
	merger := &TestMerger{}
	rm := performance.NewRequestMerger(merger, 5, 50*time.Millisecond)
	defer rm.Close()

	// 提交单个请求（应该超时处理）
	ctx := context.Background()
	response, err := rm.Submit(ctx, "key1", "data1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response.Data != "processed_data1" {
		t.Errorf("Expected 'processed_data1', got %v", response.Data)
	}
}

func TestRateLimiterWait(t *testing.T) {
	// 创建限流器
	rl := performance.NewRateLimiter(100*time.Millisecond, 1)

	// 消耗初始令牌
	rl.Allow()

	// 测试等待
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRateLimiterWaitTimeout(t *testing.T) {
	// 创建限流器
	rl := performance.NewRateLimiter(200*time.Millisecond, 1)

	// 消耗初始令牌
	rl.Allow()

	// 测试超时
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}
}
