package patterns

import (
	"context"
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventTypeUser     EventType = "user"
	EventTypeGame     EventType = "game"
	EventTypeSystem   EventType = "system"
	EventTypeBusiness EventType = "business"
)

// EventFilter 事件过滤器接口
type EventFilter interface {
	Filter(event Event) bool
	GetName() string
}

// PriorityManager 优先级管理器
type PriorityManager interface {
	GetPriority(subscriber Subscriber) int
	SetPriority(subscriber Subscriber, priority int)
}

// EnhancedEventBus 增强的事件总线
type EnhancedEventBus struct {
	subscribers map[EventType][]Subscriber
	filters     map[EventType][]EventFilter
	priority    PriorityManager
	mutex       sync.RWMutex
	metrics     *EventBusMetrics
}

// EventBusMetrics 事件总线指标
type EventBusMetrics struct {
	EventsPublished  int64
	EventsFiltered   int64
	EventsDelivered  int64
	EventsFailed     int64
	SubscribersCount int64
	FilterCount      int64
	mutex            sync.RWMutex
}

// NewEnhancedEventBus 创建增强的事件总线
func NewEnhancedEventBus() *EnhancedEventBus {
	return &EnhancedEventBus{
		subscribers: make(map[EventType][]Subscriber),
		filters:     make(map[EventType][]EventFilter),
		priority:    NewDefaultPriorityManager(),
		metrics:     &EventBusMetrics{},
	}
}

// Subscribe 订阅事件
func (eeb *EnhancedEventBus) Subscribe(eventType EventType, subscriber Subscriber) error {
	eeb.mutex.Lock()
	defer eeb.mutex.Unlock()

	// 检查是否已经订阅
	for _, sub := range eeb.subscribers[eventType] {
		if sub.GetID() == subscriber.GetID() {
			return &ObserverError{message: "subscriber already exists: " + subscriber.GetID()}
		}
	}

	eeb.subscribers[eventType] = append(eeb.subscribers[eventType], subscriber)

	// 更新指标
	eeb.metrics.mutex.Lock()
	eeb.metrics.SubscribersCount++
	eeb.metrics.mutex.Unlock()

	return nil
}

// Unsubscribe 取消订阅
func (eeb *EnhancedEventBus) Unsubscribe(eventType EventType, subscriberID string) error {
	eeb.mutex.Lock()
	defer eeb.mutex.Unlock()

	subscribers := eeb.subscribers[eventType]
	for i, sub := range subscribers {
		if sub.GetID() == subscriberID {
			eeb.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)

			// 更新指标
			eeb.metrics.mutex.Lock()
			eeb.metrics.SubscribersCount--
			eeb.metrics.mutex.Unlock()

			return nil
		}
	}

	return &ObserverError{message: "subscriber not found: " + subscriberID}
}

// Publish 发布事件
func (eeb *EnhancedEventBus) Publish(ctx context.Context, event Event) error {
	eeb.mutex.RLock()
	subscribers := eeb.subscribers[EventType(event.GetType())]
	filters := eeb.filters[EventType(event.GetType())]
	eeb.mutex.RUnlock()

	// 更新指标
	eeb.metrics.mutex.Lock()
	eeb.metrics.EventsPublished++
	eeb.metrics.mutex.Unlock()

	// 应用过滤器
	if !eeb.applyFilters(event, filters) {
		eeb.metrics.mutex.Lock()
		eeb.metrics.EventsFiltered++
		eeb.metrics.mutex.Unlock()
		return nil
	}

	// 按优先级排序订阅者
	sortedSubscribers := eeb.sortSubscribersByPriority(subscribers)

	// 通知所有订阅者
	var errors []error
	for _, subscriber := range sortedSubscribers {
		if err := subscriber.Update(ctx, event); err != nil {
			errors = append(errors, err)

			// 更新指标
			eeb.metrics.mutex.Lock()
			eeb.metrics.EventsFailed++
			eeb.metrics.mutex.Unlock()
		} else {
			// 更新指标
			eeb.metrics.mutex.Lock()
			eeb.metrics.EventsDelivered++
			eeb.metrics.mutex.Unlock()
		}
	}

	if len(errors) > 0 {
		return &ObserverError{message: "some subscribers failed", errors: errors}
	}

	return nil
}

// AddFilter 添加过滤器
func (eeb *EnhancedEventBus) AddFilter(eventType EventType, filter EventFilter) {
	eeb.mutex.Lock()
	defer eeb.mutex.Unlock()

	eeb.filters[eventType] = append(eeb.filters[eventType], filter)

	// 更新指标
	eeb.metrics.mutex.Lock()
	eeb.metrics.FilterCount++
	eeb.metrics.mutex.Unlock()
}

// RemoveFilter 移除过滤器
func (eeb *EnhancedEventBus) RemoveFilter(eventType EventType, filterName string) {
	eeb.mutex.Lock()
	defer eeb.mutex.Unlock()

	filters := eeb.filters[eventType]
	for i, filter := range filters {
		if filter.GetName() == filterName {
			eeb.filters[eventType] = append(filters[:i], filters[i+1:]...)

			// 更新指标
			eeb.metrics.mutex.Lock()
			eeb.metrics.FilterCount--
			eeb.metrics.mutex.Unlock()

			break
		}
	}
}

// applyFilters 应用过滤器
func (eeb *EnhancedEventBus) applyFilters(event Event, filters []EventFilter) bool {
	for _, filter := range filters {
		if !filter.Filter(event) {
			return false
		}
	}
	return true
}

// sortSubscribersByPriority 按优先级排序订阅者
func (eeb *EnhancedEventBus) sortSubscribersByPriority(subscribers []Subscriber) []Subscriber {
	sorted := make([]Subscriber, len(subscribers))
	copy(sorted, subscribers)

	// 简单的冒泡排序
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if eeb.priority.GetPriority(sorted[i]) < eeb.priority.GetPriority(sorted[j]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// GetMetrics 获取指标
func (eeb *EnhancedEventBus) GetMetrics() *EventBusMetrics {
	eeb.metrics.mutex.RLock()
	defer eeb.metrics.mutex.RUnlock()

	// 返回副本
	return &EventBusMetrics{
		EventsPublished:  eeb.metrics.EventsPublished,
		EventsFiltered:   eeb.metrics.EventsFiltered,
		EventsDelivered:  eeb.metrics.EventsDelivered,
		EventsFailed:     eeb.metrics.EventsFailed,
		SubscribersCount: eeb.metrics.SubscribersCount,
		FilterCount:      eeb.metrics.FilterCount,
	}
}

// DefaultPriorityManager 默认优先级管理器
type DefaultPriorityManager struct {
	priorities map[string]int
	mutex      sync.RWMutex
}

// NewDefaultPriorityManager 创建默认优先级管理器
func NewDefaultPriorityManager() *DefaultPriorityManager {
	return &DefaultPriorityManager{
		priorities: make(map[string]int),
	}
}

// GetPriority 获取优先级
func (dpm *DefaultPriorityManager) GetPriority(subscriber Subscriber) int {
	dpm.mutex.RLock()
	defer dpm.mutex.RUnlock()

	priority, exists := dpm.priorities[subscriber.GetID()]
	if !exists {
		return subscriber.GetPriority()
	}
	return priority
}

// SetPriority 设置优先级
func (dpm *DefaultPriorityManager) SetPriority(subscriber Subscriber, priority int) {
	dpm.mutex.Lock()
	defer dpm.mutex.Unlock()

	dpm.priorities[subscriber.GetID()] = priority
}

// BaseEventFilter 基础事件过滤器
type BaseEventFilter struct {
	name       string
	filterFunc func(Event) bool
}

// NewBaseEventFilter 创建基础事件过滤器
func NewBaseEventFilter(name string, filterFunc func(Event) bool) *BaseEventFilter {
	return &BaseEventFilter{
		name:       name,
		filterFunc: filterFunc,
	}
}

// Filter 过滤事件
func (bef *BaseEventFilter) Filter(event Event) bool {
	return bef.filterFunc(event)
}

// GetName 获取过滤器名称
func (bef *BaseEventFilter) GetName() string {
	return bef.name
}

// TimeBasedFilter 基于时间的过滤器
type TimeBasedFilter struct {
	*BaseEventFilter
	startTime time.Time
	endTime   time.Time
}

// NewTimeBasedFilter 创建基于时间的过滤器
func NewTimeBasedFilter(name string, startTime, endTime time.Time) *TimeBasedFilter {
	return &TimeBasedFilter{
		BaseEventFilter: NewBaseEventFilter(name, func(event Event) bool {
			eventTime := event.GetTimestamp()
			return eventTime.After(startTime) && eventTime.Before(endTime)
		}),
		startTime: startTime,
		endTime:   endTime,
	}
}

// SourceBasedFilter 基于来源的过滤器
type SourceBasedFilter struct {
	*BaseEventFilter
	allowedSources []string
}

// NewSourceBasedFilter 创建基于来源的过滤器
func NewSourceBasedFilter(name string, allowedSources []string) *SourceBasedFilter {
	return &SourceBasedFilter{
		BaseEventFilter: NewBaseEventFilter(name, func(event Event) bool {
			eventSource := event.GetSource()
			for _, source := range allowedSources {
				if eventSource == source {
					return true
				}
			}
			return false
		}),
		allowedSources: allowedSources,
	}
}

// PrioritySubscriber 优先级订阅者
type PrioritySubscriber struct {
	subscriber Subscriber
	priority   int
}

// NewPrioritySubscriber 创建优先级订阅者
func NewPrioritySubscriber(subscriber Subscriber, priority int) *PrioritySubscriber {
	return &PrioritySubscriber{
		subscriber: subscriber,
		priority:   priority,
	}
}

// Update 更新订阅者
func (ps *PrioritySubscriber) Update(ctx context.Context, event Event) error {
	return ps.subscriber.Update(ctx, event)
}

// GetID 获取订阅者ID
func (ps *PrioritySubscriber) GetID() string {
	return ps.subscriber.GetID()
}

// GetPriority 获取优先级
func (ps *PrioritySubscriber) GetPriority() int {
	return ps.priority
}

// AsyncSubscriber 异步订阅者
type AsyncSubscriber struct {
	subscriber Subscriber
	queue      chan Event
	workers    int
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAsyncSubscriber 创建异步订阅者
func NewAsyncSubscriber(subscriber Subscriber, queueSize, workers int) *AsyncSubscriber {
	ctx, cancel := context.WithCancel(context.Background())

	as := &AsyncSubscriber{
		subscriber: subscriber,
		queue:      make(chan Event, queueSize),
		workers:    workers,
		ctx:        ctx,
		cancel:     cancel,
	}

	// 启动工作协程
	for i := 0; i < workers; i++ {
		go as.worker()
	}

	return as
}

// Update 更新订阅者
func (as *AsyncSubscriber) Update(ctx context.Context, event Event) error {
	select {
	case as.queue <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return &ObserverError{message: "subscriber queue is full"}
	}
}

// GetID 获取订阅者ID
func (as *AsyncSubscriber) GetID() string {
	return as.subscriber.GetID()
}

// GetPriority 获取优先级
func (as *AsyncSubscriber) GetPriority() int {
	return as.subscriber.GetPriority()
}

// worker 工作协程
func (as *AsyncSubscriber) worker() {
	for {
		select {
		case event := <-as.queue:
			as.subscriber.Update(as.ctx, event)
		case <-as.ctx.Done():
			return
		}
	}
}

// Stop 停止异步订阅者
func (as *AsyncSubscriber) Stop() {
	as.cancel()
	close(as.queue)
}

// EventBusManager 事件总线管理器
type EventBusManager struct {
	eventBuses map[string]*EnhancedEventBus
	mutex      sync.RWMutex
}

// NewEventBusManager 创建事件总线管理器
func NewEventBusManager() *EventBusManager {
	return &EventBusManager{
		eventBuses: make(map[string]*EnhancedEventBus),
	}
}

// GetEventBus 获取事件总线
func (ebm *EventBusManager) GetEventBus(name string) *EnhancedEventBus {
	ebm.mutex.Lock()
	defer ebm.mutex.Unlock()

	if eventBus, exists := ebm.eventBuses[name]; exists {
		return eventBus
	}

	eventBus := NewEnhancedEventBus()
	ebm.eventBuses[name] = eventBus
	return eventBus
}

// RemoveEventBus 移除事件总线
func (ebm *EventBusManager) RemoveEventBus(name string) {
	ebm.mutex.Lock()
	defer ebm.mutex.Unlock()

	delete(ebm.eventBuses, name)
}

// ListEventBuses 列出所有事件总线
func (ebm *EventBusManager) ListEventBuses() []string {
	ebm.mutex.RLock()
	defer ebm.mutex.RUnlock()

	names := make([]string, 0, len(ebm.eventBuses))
	for name := range ebm.eventBuses {
		names = append(names, name)
	}

	return names
}

// GetGlobalMetrics 获取全局指标
func (ebm *EventBusManager) GetGlobalMetrics() map[string]*EventBusMetrics {
	ebm.mutex.RLock()
	defer ebm.mutex.RUnlock()

	metrics := make(map[string]*EventBusMetrics)
	for name, eventBus := range ebm.eventBuses {
		metrics[name] = eventBus.GetMetrics()
	}

	return metrics
}
