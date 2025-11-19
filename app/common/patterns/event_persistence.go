package patterns

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// EventStore 事件存储接口
type EventStore interface {
	SaveEvent(event Event) error
	GetEvent(eventID string) (Event, error)
	GetEventsByType(eventType EventType, limit int, offset int) ([]Event, error)
	GetEventsBySource(source string, limit int, offset int) ([]Event, error)
	GetEventsByTimeRange(startTime, endTime time.Time, limit int, offset int) ([]Event, error)
	DeleteEvent(eventID string) error
	DeleteEventsByType(eventType EventType) error
	DeleteEventsBySource(source string) error
	DeleteEventsByTimeRange(startTime, endTime time.Time) error
}

// Event 事件接口
type Event interface {
	GetID() string
	GetType() string
	GetSource() string
	GetTimestamp() time.Time
	GetData() map[string]interface{}
	GetMetadata() map[string]interface{}
}

// BaseEvent 基础事件实现
type BaseEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// GetID 获取事件ID
func (e *BaseEvent) GetID() string {
	return e.ID
}

// GetType 获取事件类型
func (e *BaseEvent) GetType() string {
	return e.Type
}

// GetSource 获取事件来源
func (e *BaseEvent) GetSource() string {
	return e.Source
}

// GetTimestamp 获取时间戳
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetData 获取事件数据
func (e *BaseEvent) GetData() map[string]interface{} {
	return e.Data
}

// GetMetadata 获取元数据
func (e *BaseEvent) GetMetadata() map[string]interface{} {
	return e.Metadata
}

// NewBaseEvent 创建基础事件
func NewBaseEvent(id, eventType, source string, data map[string]interface{}) *BaseEvent {
	return &BaseEvent{
		ID:        id,
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  make(map[string]interface{}),
	}
}

// MemoryEventStore 内存事件存储
type MemoryEventStore struct {
	events map[string]Event
	mutex  sync.RWMutex
}

// NewMemoryEventStore 创建内存事件存储
func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		events: make(map[string]Event),
	}
}

// SaveEvent 保存事件
func (mes *MemoryEventStore) SaveEvent(event Event) error {
	mes.mutex.Lock()
	defer mes.mutex.Unlock()
	mes.events[event.GetID()] = event
	return nil
}

// GetEvent 获取事件
func (mes *MemoryEventStore) GetEvent(eventID string) (Event, error) {
	mes.mutex.RLock()
	defer mes.mutex.RUnlock()
	event, exists := mes.events[eventID]
	if !exists {
		return nil, fmt.Errorf("event not found: %s", eventID)
	}
	return event, nil
}

// GetEventsByType 根据类型获取事件
func (mes *MemoryEventStore) GetEventsByType(eventType EventType, limit int, offset int) ([]Event, error) {
	mes.mutex.RLock()
	defer mes.mutex.RUnlock()

	var events []Event
	count := 0
	skipped := 0

	for _, event := range mes.events {
		if event.GetType() == string(eventType) {
			if skipped < offset {
				skipped++
				continue
			}
			if limit > 0 && count >= limit {
				break
			}
			events = append(events, event)
			count++
		}
	}

	return events, nil
}

// GetEventsBySource 根据来源获取事件
func (mes *MemoryEventStore) GetEventsBySource(source string, limit int, offset int) ([]Event, error) {
	mes.mutex.RLock()
	defer mes.mutex.RUnlock()

	var events []Event
	count := 0
	skipped := 0

	for _, event := range mes.events {
		if event.GetSource() == source {
			if skipped < offset {
				skipped++
				continue
			}
			if limit > 0 && count >= limit {
				break
			}
			events = append(events, event)
			count++
		}
	}

	return events, nil
}

// GetEventsByTimeRange 根据时间范围获取事件
func (mes *MemoryEventStore) GetEventsByTimeRange(startTime, endTime time.Time, limit int, offset int) ([]Event, error) {
	mes.mutex.RLock()
	defer mes.mutex.RUnlock()

	var events []Event
	count := 0
	skipped := 0

	for _, event := range mes.events {
		eventTime := event.GetTimestamp()
		if eventTime.After(startTime) && eventTime.Before(endTime) {
			if skipped < offset {
				skipped++
				continue
			}
			if limit > 0 && count >= limit {
				break
			}
			events = append(events, event)
			count++
		}
	}

	return events, nil
}

// DeleteEvent 删除事件
func (mes *MemoryEventStore) DeleteEvent(eventID string) error {
	mes.mutex.Lock()
	defer mes.mutex.Unlock()
	delete(mes.events, eventID)
	return nil
}

// DeleteEventsByType 根据类型删除事件
func (mes *MemoryEventStore) DeleteEventsByType(eventType EventType) error {
	mes.mutex.Lock()
	defer mes.mutex.Unlock()

	for id, event := range mes.events {
		if event.GetType() == string(eventType) {
			delete(mes.events, id)
		}
	}

	return nil
}

// DeleteEventsBySource 根据来源删除事件
func (mes *MemoryEventStore) DeleteEventsBySource(source string) error {
	mes.mutex.Lock()
	defer mes.mutex.Unlock()

	for id, event := range mes.events {
		if event.GetSource() == source {
			delete(mes.events, id)
		}
	}

	return nil
}

// DeleteEventsByTimeRange 根据时间范围删除事件
func (mes *MemoryEventStore) DeleteEventsByTimeRange(startTime, endTime time.Time) error {
	mes.mutex.Lock()
	defer mes.mutex.Unlock()

	for id, event := range mes.events {
		eventTime := event.GetTimestamp()
		if eventTime.After(startTime) && eventTime.Before(endTime) {
			delete(mes.events, id)
		}
	}

	return nil
}

// PersistentEventBus 持久化事件总线
type PersistentEventBus struct {
	*EnhancedEventBus
	eventStore EventStore
	persist    bool
	mutex      sync.RWMutex
}

// NewPersistentEventBus 创建持久化事件总线
func NewPersistentEventBus(eventStore EventStore) *PersistentEventBus {
	return &PersistentEventBus{
		EnhancedEventBus: NewEnhancedEventBus(),
		eventStore:       eventStore,
		persist:          true,
	}
}

// SetPersistence 设置是否持久化
func (peb *PersistentEventBus) SetPersistence(persist bool) {
	peb.mutex.Lock()
	defer peb.mutex.Unlock()
	peb.persist = persist
}

// Publish 发布事件
func (peb *PersistentEventBus) Publish(ctx context.Context, event Event) error {
	// 持久化事件
	if peb.persist {
		if err := peb.eventStore.SaveEvent(event); err != nil {
			return fmt.Errorf("failed to persist event: %w", err)
		}
	}

	// 发布事件
	return peb.EnhancedEventBus.Publish(ctx, event)
}

// ReplayEvents 重放事件
func (peb *PersistentEventBus) ReplayEvents(ctx context.Context, eventType EventType, limit int) error {
	events, err := peb.eventStore.GetEventsByType(eventType, limit, 0)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	for _, event := range events {
		if err := peb.EnhancedEventBus.Publish(ctx, event); err != nil {
			return fmt.Errorf("failed to replay event %s: %w", event.GetID(), err)
		}
	}

	return nil
}

// GetEventHistory 获取事件历史
func (peb *PersistentEventBus) GetEventHistory(eventType EventType, limit int, offset int) ([]Event, error) {
	return peb.eventStore.GetEventsByType(eventType, limit, offset)
}

// EventReplayManager 事件重放管理器
type EventReplayManager struct {
	eventStore EventStore
	eventBus   *EnhancedEventBus
	mutex      sync.RWMutex
}

// NewEventReplayManager 创建事件重放管理器
func NewEventReplayManager(eventStore EventStore, eventBus *EnhancedEventBus) *EventReplayManager {
	return &EventReplayManager{
		eventStore: eventStore,
		eventBus:   eventBus,
	}
}

// ReplayEventsByType 根据类型重放事件
func (erm *EventReplayManager) ReplayEventsByType(ctx context.Context, eventType EventType, limit int) error {
	events, err := erm.eventStore.GetEventsByType(eventType, limit, 0)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	return erm.replayEvents(ctx, events)
}

// ReplayEventsBySource 根据来源重放事件
func (erm *EventReplayManager) ReplayEventsBySource(ctx context.Context, source string, limit int) error {
	events, err := erm.eventStore.GetEventsBySource(source, limit, 0)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	return erm.replayEvents(ctx, events)
}

// ReplayEventsByTimeRange 根据时间范围重放事件
func (erm *EventReplayManager) ReplayEventsByTimeRange(ctx context.Context, startTime, endTime time.Time, limit int) error {
	events, err := erm.eventStore.GetEventsByTimeRange(startTime, endTime, limit, 0)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	return erm.replayEvents(ctx, events)
}

// replayEvents 重放事件
func (erm *EventReplayManager) replayEvents(ctx context.Context, events []Event) error {
	for _, event := range events {
		if err := erm.eventBus.Publish(ctx, event); err != nil {
			return fmt.Errorf("failed to replay event %s: %w", event.GetID(), err)
		}
	}

	return nil
}

// EventSnapshot 事件快照
type EventSnapshot struct {
	ID        string    `json:"id"`
	EventType string    `json:"event_type"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Data      string    `json:"data"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
}

// EventSnapshotStore 事件快照存储
type EventSnapshotStore struct {
	snapshots map[string]*EventSnapshot
	mutex     sync.RWMutex
}

// NewEventSnapshotStore 创建事件快照存储
func NewEventSnapshotStore() *EventSnapshotStore {
	return &EventSnapshotStore{
		snapshots: make(map[string]*EventSnapshot),
	}
}

// CreateSnapshot 创建快照
func (ess *EventSnapshotStore) CreateSnapshot(event Event) (*EventSnapshot, error) {
	data, err := json.Marshal(event.GetData())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	metadata, err := json.Marshal(event.GetMetadata())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event metadata: %w", err)
	}

	snapshot := &EventSnapshot{
		ID:        event.GetID(),
		EventType: event.GetType(),
		Source:    event.GetSource(),
		Timestamp: event.GetTimestamp(),
		Data:      string(data),
		Metadata:  string(metadata),
		CreatedAt: time.Now(),
	}

	ess.mutex.Lock()
	defer ess.mutex.Unlock()
	ess.snapshots[event.GetID()] = snapshot

	return snapshot, nil
}

// GetSnapshot 获取快照
func (ess *EventSnapshotStore) GetSnapshot(eventID string) (*EventSnapshot, error) {
	ess.mutex.RLock()
	defer ess.mutex.RUnlock()

	snapshot, exists := ess.snapshots[eventID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", eventID)
	}

	return snapshot, nil
}

// RestoreEvent 从快照恢复事件
func (ess *EventSnapshotStore) RestoreEvent(snapshot *EventSnapshot) (Event, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(snapshot.Data), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(snapshot.Metadata), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event metadata: %w", err)
	}

	event := &BaseEvent{
		ID:        snapshot.ID,
		Type:      snapshot.EventType,
		Source:    snapshot.Source,
		Timestamp: snapshot.Timestamp,
		Data:      data,
		Metadata:  metadata,
	}

	return event, nil
}

// EventArchiveManager 事件归档管理器
type EventArchiveManager struct {
	eventStore    EventStore
	archiveStore  EventStore
	retentionDays int
	mutex         sync.RWMutex
}

// NewEventArchiveManager 创建事件归档管理器
func NewEventArchiveManager(eventStore, archiveStore EventStore, retentionDays int) *EventArchiveManager {
	return &EventArchiveManager{
		eventStore:    eventStore,
		archiveStore:  archiveStore,
		retentionDays: retentionDays,
	}
}

// ArchiveOldEvents 归档旧事件
func (eam *EventArchiveManager) ArchiveOldEvents() error {
	cutoffTime := time.Now().AddDate(0, 0, -eam.retentionDays)

	// 获取需要归档的事件
	events, err := eam.eventStore.GetEventsByTimeRange(time.Time{}, cutoffTime, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to get old events: %w", err)
	}

	// 归档事件
	for _, event := range events {
		if err := eam.archiveStore.SaveEvent(event); err != nil {
			return fmt.Errorf("failed to archive event %s: %w", event.GetID(), err)
		}
	}

	// 删除已归档的事件
	if err := eam.eventStore.DeleteEventsByTimeRange(time.Time{}, cutoffTime); err != nil {
		return fmt.Errorf("failed to delete archived events: %w", err)
	}

	return nil
}

// RestoreFromArchive 从归档恢复事件
func (eam *EventArchiveManager) RestoreFromArchive(eventType EventType, limit int) error {
	events, err := eam.archiveStore.GetEventsByType(eventType, limit, 0)
	if err != nil {
		return fmt.Errorf("failed to get archived events: %w", err)
	}

	// 恢复事件
	for _, event := range events {
		if err := eam.eventStore.SaveEvent(event); err != nil {
			return fmt.Errorf("failed to restore event %s: %w", event.GetID(), err)
		}
	}

	return nil
}
