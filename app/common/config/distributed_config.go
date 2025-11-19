package config

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
)

// ConfigManager 分布式配置管理器
type ConfigManager struct {
	client    *clientv3.Client
	configs   map[string]interface{}
	watchers  map[string][]ConfigWatcher
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	keyPrefix string
}

// ConfigWatcher 配置观察者
type ConfigWatcher interface {
	OnConfigChanged(key string, oldValue, newValue interface{})
	OnConfigDeleted(key string, value interface{})
}

// ConfigValidator 配置验证器
type ConfigValidator interface {
	Validate(key string, value interface{}) error
	ValidateChange(key string, oldValue, newValue interface{}) error
}

// ConfigVersion 配置版本
type ConfigVersion struct {
	Version   int64       `json:"version"`
	Timestamp time.Time   `json:"timestamp"`
	Value     interface{} `json:"value"`
	Checksum  string      `json:"checksum"`
}

// ConfigChange 配置变更
type ConfigChange struct {
	Key       string      `json:"key"`
	OldValue  interface{} `json:"oldValue"`
	NewValue  interface{} `json:"newValue"`
	Timestamp time.Time   `json:"timestamp"`
	User      string      `json:"user"`
	Reason    string      `json:"reason"`
}

// NewConfigManager 创建配置管理器
func NewConfigManager(client *clientv3.Client, keyPrefix string) *ConfigManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConfigManager{
		client:    client,
		configs:   make(map[string]interface{}),
		watchers:  make(map[string][]ConfigWatcher),
		ctx:       ctx,
		cancel:    cancel,
		keyPrefix: keyPrefix,
	}
}

// SetConfig 设置配置
func (cm *ConfigManager) SetConfig(key string, value interface{}, user, reason string) error {
	// 验证配置
	if err := cm.validateConfig(key, value); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 序列化配置
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 创建配置版本
	version := &ConfigVersion{
		Version:   time.Now().UnixNano(),
		Timestamp: time.Now(),
		Value:     value,
		Checksum:  cm.calculateChecksum(data),
	}

	versionData, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	// 存储到etcd
	etcdKey := cm.keyPrefix + key
	_, err = cm.client.Put(cm.ctx, etcdKey, string(versionData))
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// 更新本地缓存
	cm.mutex.Lock()
	oldValue := cm.configs[key]
	cm.configs[key] = value
	cm.mutex.Unlock()

	// 记录配置变更
	change := &ConfigChange{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  value,
		Timestamp: time.Now(),
		User:      user,
		Reason:    reason,
	}

	// 存储变更历史
	changeKey := fmt.Sprintf("%s:history:%s:%d", cm.keyPrefix, key, version.Version)
	changeData, _ := json.Marshal(change)
	cm.client.Put(cm.ctx, changeKey, string(changeData))

	// 通知观察者
	cm.notifyWatchers(key, oldValue, value)

	return nil
}

// GetConfig 获取配置
func (cm *ConfigManager) GetConfig(key string) (interface{}, error) {
	cm.mutex.RLock()
	value, exists := cm.configs[key]
	cm.mutex.RUnlock()

	if !exists {
		// 从etcd获取
		return cm.fetchFromEtcd(key)
	}

	return value, nil
}

// GetConfigWithDefault 获取配置，如果不存在则返回默认值
func (cm *ConfigManager) GetConfigWithDefault(key string, defaultValue interface{}) interface{} {
	value, err := cm.GetConfig(key)
	if err != nil {
		return defaultValue
	}
	return value
}

// DeleteConfig 删除配置
func (cm *ConfigManager) DeleteConfig(key string, user, reason string) error {
	etcdKey := cm.keyPrefix + key

	// 获取当前值
	cm.mutex.RLock()
	oldValue := cm.configs[key]
	cm.mutex.RUnlock()

	// 从etcd删除
	_, err := cm.client.Delete(cm.ctx, etcdKey)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	// 更新本地缓存
	cm.mutex.Lock()
	delete(cm.configs, key)
	cm.mutex.Unlock()

	// 记录配置变更
	change := &ConfigChange{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  nil,
		Timestamp: time.Now(),
		User:      user,
		Reason:    reason,
	}

	// 存储变更历史
	changeKey := fmt.Sprintf("%s:history:%s:deleted:%d", cm.keyPrefix, key, time.Now().UnixNano())
	changeData, _ := json.Marshal(change)
	cm.client.Put(cm.ctx, changeKey, string(changeData))

	// 通知观察者
	cm.notifyWatchers(key, oldValue, nil)

	return nil
}

// WatchConfig 监听配置变化
func (cm *ConfigManager) WatchConfig(key string, watcher ConfigWatcher) error {
	cm.mutex.Lock()
	cm.watchers[key] = append(cm.watchers[key], watcher)
	cm.mutex.Unlock()

	// 启动监听
	go cm.watchConfig(key)

	return nil
}

// WatchAllConfigs 监听所有配置变化
func (cm *ConfigManager) WatchAllConfigs(watcher ConfigWatcher) error {
	// 启动监听
	go cm.watchAllConfigs(watcher)

	return nil
}

// fetchFromEtcd 从etcd获取配置
func (cm *ConfigManager) fetchFromEtcd(key string) (interface{}, error) {
	etcdKey := cm.keyPrefix + key

	resp, err := cm.client.Get(cm.ctx, etcdKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("config not found: %s", key)
	}

	var version ConfigVersion
	if err := json.Unmarshal(resp.Kvs[0].Value, &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config version: %w", err)
	}

	// 更新本地缓存
	cm.mutex.Lock()
	cm.configs[key] = version.Value
	cm.mutex.Unlock()

	return version.Value, nil
}

// watchConfig 监听配置变化
func (cm *ConfigManager) watchConfig(key string) {
	etcdKey := cm.keyPrefix + key

	watchCh := cm.client.Watch(cm.ctx, etcdKey)

	for watchResp := range watchCh {
		for _, event := range watchResp.Events {
			switch event.Type {
			case clientv3.EventTypePut:
				var version ConfigVersion
				if err := json.Unmarshal(event.Kv.Value, &version); err != nil {
					continue
				}

				cm.mutex.Lock()
				oldValue := cm.configs[key]
				cm.configs[key] = version.Value
				cm.mutex.Unlock()

				cm.notifyWatchers(key, oldValue, version.Value)

			case clientv3.EventTypeDelete:
				cm.mutex.Lock()
				oldValue := cm.configs[key]
				delete(cm.configs, key)
				cm.mutex.Unlock()

				cm.notifyWatchers(key, oldValue, nil)
			}
		}
	}
}

// watchAllConfigs 监听所有配置变化
func (cm *ConfigManager) watchAllConfigs(watcher ConfigWatcher) {
	watchCh := cm.client.Watch(cm.ctx, cm.keyPrefix, clientv3.WithPrefix())

	for watchResp := range watchCh {
		for _, event := range watchResp.Events {
			key := string(event.Kv.Key)[len(cm.keyPrefix):]

			switch event.Type {
			case clientv3.EventTypePut:
				var version ConfigVersion
				if err := json.Unmarshal(event.Kv.Value, &version); err != nil {
					continue
				}

				cm.mutex.Lock()
				oldValue := cm.configs[key]
				cm.configs[key] = version.Value
				cm.mutex.Unlock()

				watcher.OnConfigChanged(key, oldValue, version.Value)

			case clientv3.EventTypeDelete:
				cm.mutex.Lock()
				oldValue := cm.configs[key]
				delete(cm.configs, key)
				cm.mutex.Unlock()

				watcher.OnConfigDeleted(key, oldValue)
			}
		}
	}
}

// notifyWatchers 通知观察者
func (cm *ConfigManager) notifyWatchers(key string, oldValue, newValue interface{}) {
	cm.mutex.RLock()
	watchers := cm.watchers[key]
	cm.mutex.RUnlock()

	for _, watcher := range watchers {
		if newValue == nil {
			watcher.OnConfigDeleted(key, oldValue)
		} else {
			watcher.OnConfigChanged(key, oldValue, newValue)
		}
	}
}

// validateConfig 验证配置
func (cm *ConfigManager) validateConfig(key string, value interface{}) error {
	// 这里可以实现配置验证逻辑
	// 例如：检查配置格式、值范围等
	return nil
}

// calculateChecksum 计算校验和
func (cm *ConfigManager) calculateChecksum(data []byte) string {
	// 这里可以实现校验和计算逻辑
	// 简化实现，返回数据长度
	return fmt.Sprintf("%d", len(data))
}

// GetConfigHistory 获取配置历史
func (cm *ConfigManager) GetConfigHistory(key string, limit int) ([]*ConfigChange, error) {
	historyKey := fmt.Sprintf("%s:history:%s:", cm.keyPrefix, key)

	resp, err := cm.client.Get(cm.ctx, historyKey, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get config history: %w", err)
	}

	var changes []*ConfigChange
	for _, kv := range resp.Kvs {
		var change ConfigChange
		if err := json.Unmarshal(kv.Value, &change); err != nil {
			continue
		}
		changes = append(changes, &change)
	}

	// 按时间戳排序
	for i := 0; i < len(changes)-1; i++ {
		for j := i + 1; j < len(changes); j++ {
			if changes[i].Timestamp.After(changes[j].Timestamp) {
				changes[i], changes[j] = changes[j], changes[i]
			}
		}
	}

	// 限制返回数量
	if limit > 0 && len(changes) > limit {
		changes = changes[:limit]
	}

	return changes, nil
}

// RollbackConfig 回滚配置
func (cm *ConfigManager) RollbackConfig(key string, version int64, user, reason string) error {
	// 获取指定版本的配置
	versionKey := fmt.Sprintf("%s:history:%s:%d", cm.keyPrefix, key, version)

	resp, err := cm.client.Get(cm.ctx, versionKey)
	if err != nil {
		return fmt.Errorf("failed to get config version: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("config version not found: %d", version)
	}

	var change ConfigChange
	if err := json.Unmarshal(resp.Kvs[0].Value, &change); err != nil {
		return fmt.Errorf("failed to unmarshal config change: %w", err)
	}

	// 回滚到指定版本
	return cm.SetConfig(key, change.OldValue, user, reason)
}

// GetConfigStats 获取配置统计信息
func (cm *ConfigManager) GetConfigStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_configs"] = len(cm.configs)
	stats["configs"] = make(map[string]interface{})

	for key, value := range cm.configs {
		stats["configs"].(map[string]interface{})[key] = map[string]interface{}{
			"value": value,
			"type":  fmt.Sprintf("%T", value),
		}
	}

	return stats
}

// Stop 停止配置管理器
func (cm *ConfigManager) Stop() {
	cm.cancel()
}
