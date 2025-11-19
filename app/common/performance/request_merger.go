package performance

import (
	"context"
	"sync"
	"time"

	"platform/utils"
)

// RequestMerger 请求合并器
type RequestMerger struct {
	requests  chan *MergedRequest
	merger    *Merger
	timeout   time.Duration
	batchSize int
	mu        sync.RWMutex
	closed    bool
}

// MergedRequest 合并请求
type MergedRequest struct {
	Key      string
	Data     interface{}
	Response chan *MergedResponse
	Timeout  time.Duration
}

// MergedResponse 合并响应
type MergedResponse struct {
	Data  interface{}
	Error error
}

// Merger 合并器接口
type Merger interface {
	Merge(requests []*MergedRequest) (map[string]*MergedResponse, error)
}

// NewRequestMerger 创建请求合并器
func NewRequestMerger(merger Merger, batchSize int, timeout time.Duration) *RequestMerger {
	rm := &RequestMerger{
		requests:  make(chan *MergedRequest, batchSize*2),
		merger:    merger,
		timeout:   timeout,
		batchSize: batchSize,
	}

	go rm.processRequests()
	return rm
}

// Submit 提交请求
func (rm *RequestMerger) Submit(ctx context.Context, key string, data interface{}) (*MergedResponse, error) {
	rm.mu.RLock()
	if rm.closed {
		rm.mu.RUnlock()
		return nil, utils.WrapError(nil, utils.ErrCodeServiceUnavailable, "请求合并器已关闭")
	}
	rm.mu.RUnlock()

	responseChan := make(chan *MergedResponse, 1)
	request := &MergedRequest{
		Key:      key,
		Data:     data,
		Response: responseChan,
		Timeout:  rm.timeout,
	}

	select {
	case rm.requests <- request:
		// 等待响应
		select {
		case response := <-responseChan:
			return response, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// processRequests 处理请求
func (rm *RequestMerger) processRequests() {
	var batch []*MergedRequest
	ticker := time.NewTicker(rm.timeout)
	defer ticker.Stop()

	for {
		select {
		case request := <-rm.requests:
			batch = append(batch, request)

			// 达到批次大小，立即处理
			if len(batch) >= rm.batchSize {
				rm.processBatch(batch)
				batch = nil
			}

		case <-ticker.C:
			// 超时处理批次
			if len(batch) > 0 {
				rm.processBatch(batch)
				batch = nil
			}
		}
	}
}

// processBatch 处理批次
func (rm *RequestMerger) processBatch(batch []*MergedRequest) {
	if len(batch) == 0 {
		return
	}

	// 合并请求
	responses, err := rm.merger.Merge(batch)
	if err != nil {
		// 发送错误响应
		for _, req := range batch {
			select {
			case req.Response <- &MergedResponse{Error: err}:
			default:
			}
		}
		return
	}

	// 发送响应
	for _, req := range batch {
		response, exists := responses[req.Key]
		if !exists {
			response = &MergedResponse{Error: utils.WrapError(nil, utils.ErrCodeUnknown, "未找到响应")}
		}

		select {
		case req.Response <- response:
		default:
		}
	}
}

// Close 关闭请求合并器
func (rm *RequestMerger) Close() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.closed = true
	close(rm.requests)
}

// BackpressureController 背压控制器
type BackpressureController struct {
	maxConcurrency int
	semaphore      chan struct{}
	mu             sync.RWMutex
	stats          *BackpressureStats
}

// BackpressureStats 背压统计
type BackpressureStats struct {
	ActiveRequests   int64
	TotalRequests    int64
	RejectedRequests int64
	mu               sync.RWMutex
}

// NewBackpressureController 创建背压控制器
func NewBackpressureController(maxConcurrency int) *BackpressureController {
	return &BackpressureController{
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
		stats:          &BackpressureStats{},
	}
}

// Acquire 获取许可
func (bpc *BackpressureController) Acquire() error {
	select {
	case bpc.semaphore <- struct{}{}:
		bpc.stats.mu.Lock()
		bpc.stats.ActiveRequests++
		bpc.stats.TotalRequests++
		bpc.stats.mu.Unlock()
		return nil
	default:
		bpc.stats.mu.Lock()
		bpc.stats.RejectedRequests++
		bpc.stats.mu.Unlock()
		return utils.WrapError(nil, utils.ErrCodeServiceUnavailable, "系统繁忙，请稍后重试")
	}
}

// Release 释放许可
func (bpc *BackpressureController) Release() {
	select {
	case <-bpc.semaphore:
		bpc.stats.mu.Lock()
		bpc.stats.ActiveRequests--
		bpc.stats.mu.Unlock()
	default:
	}
}

// GetStats 获取统计信息
func (bpc *BackpressureController) GetStats() *BackpressureStats {
	bpc.stats.mu.RLock()
	defer bpc.stats.mu.RUnlock()

	return &BackpressureStats{
		ActiveRequests:   bpc.stats.ActiveRequests,
		TotalRequests:    bpc.stats.TotalRequests,
		RejectedRequests: bpc.stats.RejectedRequests,
	}
}

// RateLimiter 限流器
type RateLimiter struct {
	tokens    chan struct{}
	rate      time.Duration
	burst     int
	mu        sync.RWMutex
	lastToken time.Time
}

// NewRateLimiter 创建限流器
func NewRateLimiter(rate time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		tokens: make(chan struct{}, burst),
		rate:   rate,
		burst:  burst,
	}

	// 填充初始令牌
	for i := 0; i < burst; i++ {
		rl.tokens <- struct{}{}
	}

	go rl.refillTokens()
	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// Wait 等待令牌
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// refillTokens 补充令牌
func (rl *RateLimiter) refillTokens() {
	ticker := time.NewTicker(rl.rate)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
			// 令牌桶已满
		}
	}
}

// BatchProcessor 批处理器
type BatchProcessor struct {
	processor func([]interface{}) error
	batchSize int
	timeout   time.Duration
	items     chan interface{}
	mu        sync.RWMutex
	closed    bool
}

// NewBatchProcessor 创建批处理器
func NewBatchProcessor(processor func([]interface{}) error, batchSize int, timeout time.Duration) *BatchProcessor {
	bp := &BatchProcessor{
		processor: processor,
		batchSize: batchSize,
		timeout:   timeout,
		items:     make(chan interface{}, batchSize*2),
	}

	go bp.process()
	return bp
}

// Add 添加项目
func (bp *BatchProcessor) Add(item interface{}) error {
	bp.mu.RLock()
	if bp.closed {
		bp.mu.RUnlock()
		return utils.WrapError(nil, utils.ErrCodeServiceUnavailable, "批处理器已关闭")
	}
	bp.mu.RUnlock()

	select {
	case bp.items <- item:
		return nil
	default:
		return utils.WrapError(nil, utils.ErrCodeServiceUnavailable, "批处理器队列已满")
	}
}

// process 处理批次
func (bp *BatchProcessor) process() {
	var batch []interface{}
	ticker := time.NewTicker(bp.timeout)
	defer ticker.Stop()

	for {
		select {
		case item := <-bp.items:
			batch = append(batch, item)

			// 达到批次大小，立即处理
			if len(batch) >= bp.batchSize {
				bp.processBatch(batch)
				batch = nil
			}

		case <-ticker.C:
			// 超时处理批次
			if len(batch) > 0 {
				bp.processBatch(batch)
				batch = nil
			}
		}
	}
}

// processBatch 处理批次
func (bp *BatchProcessor) processBatch(batch []interface{}) {
	if len(batch) == 0 {
		return
	}

	// 处理批次
	if err := bp.processor(batch); err != nil {
		// 记录错误
		// TODO: 添加错误日志
	}
}

// Close 关闭批处理器
func (bp *BatchProcessor) Close() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.closed = true
	close(bp.items)
}

// ConcurrentLimiter 并发限制器
type ConcurrentLimiter struct {
	semaphore chan struct{}
	mu        sync.RWMutex
	stats     *ConcurrentStats
}

// ConcurrentStats 并发统计
type ConcurrentStats struct {
	Active   int64
	Total    int64
	Rejected int64
	mu       sync.RWMutex
}

// NewConcurrentLimiter 创建并发限制器
func NewConcurrentLimiter(maxConcurrency int) *ConcurrentLimiter {
	return &ConcurrentLimiter{
		semaphore: make(chan struct{}, maxConcurrency),
		stats:     &ConcurrentStats{},
	}
}

// Execute 执行函数
func (cl *ConcurrentLimiter) Execute(fn func() error) error {
	// 获取许可
	select {
	case cl.semaphore <- struct{}{}:
		cl.stats.mu.Lock()
		cl.stats.Active++
		cl.stats.Total++
		cl.stats.mu.Unlock()

		defer func() {
			<-cl.semaphore
			cl.stats.mu.Lock()
			cl.stats.Active--
			cl.stats.mu.Unlock()
		}()

		return fn()
	default:
		cl.stats.mu.Lock()
		cl.stats.Rejected++
		cl.stats.mu.Unlock()
		return utils.WrapError(nil, utils.ErrCodeServiceUnavailable, "并发限制")
	}
}

// GetStats 获取统计信息
func (cl *ConcurrentLimiter) GetStats() *ConcurrentStats {
	cl.stats.mu.RLock()
	defer cl.stats.mu.RUnlock()

	return &ConcurrentStats{
		Active:   cl.stats.Active,
		Total:    cl.stats.Total,
		Rejected: cl.stats.Rejected,
	}
}
