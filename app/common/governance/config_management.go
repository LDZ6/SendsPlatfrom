package governance

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"platform/utils"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	configs    map[string]interface{}
	watchers   map[string][]ConfigWatcher
	mu         sync.RWMutex
	version    int64
	lastUpdate time.Time
}

// ConfigWatcher 配置监听器
type ConfigWatcher interface {
	OnConfigChange(key string, oldValue, newValue interface{})
}

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	Key       string
	OldValue  interface{}
	NewValue  interface{}
	Timestamp time.Time
	Version   int64
}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configs:  make(map[string]interface{}),
		watchers: make(map[string][]ConfigWatcher),
		version:  1,
	}
}

// SetConfig 设置配置
func (cm *ConfigManager) SetConfig(key string, value interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldValue, exists := cm.configs[key]
	cm.configs[key] = value
	cm.version++
	cm.lastUpdate = time.Now()

	// 通知监听器
	if watchers, exists := cm.watchers[key]; exists {
		for _, watcher := range watchers {
			go watcher.OnConfigChange(key, oldValue, value)
		}
	}

	return nil
}

// GetConfig 获取配置
func (cm *ConfigManager) GetConfig(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	value, _ := cm.configs[key]
	return value, true
}

// GetConfigString 获取字符串配置
func (cm *ConfigManager) GetConfigString(key string) (string, error) {
	value, exists := cm.GetConfig(key)
	if !exists {
		return "", utils.WrapError(nil, utils.ErrCodeConfigError, "配置不存在")
	}

	str, ok := value.(string)
	if !ok {
		return "", utils.WrapError(nil, utils.ErrCodeConfigError, "配置类型错误")
	}

	return str, nil
}

// GetConfigInt 获取整数配置
func (cm *ConfigManager) GetConfigInt(key string) (int, error) {
	value, exists := cm.GetConfig(key)
	if !exists {
		return 0, utils.WrapError(nil, utils.ErrCodeConfigError, "配置不存在")
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	default:
		return 0, utils.WrapError(nil, utils.ErrCodeConfigError, "配置类型错误")
	}
}

// GetConfigBool 获取布尔配置
func (cm *ConfigManager) GetConfigBool(key string) (bool, error) {
	value, exists := cm.GetConfig(key)
	if !exists {
		return false, utils.WrapError(nil, utils.ErrCodeConfigError, "配置不存在")
	}

	boolValue, ok := value.(bool)
	if !ok {
		return false, utils.WrapError(nil, utils.ErrCodeConfigError, "配置类型错误")
	}

	return boolValue, nil
}

// WatchConfig 监听配置变更
func (cm *ConfigManager) WatchConfig(key string, watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.watchers[key] = append(cm.watchers[key], watcher)
}

// UnwatchConfig 取消监听配置
func (cm *ConfigManager) UnwatchConfig(key string, watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	watchers, exists := cm.watchers[key]
	if !exists {
		return
	}

	for i, w := range watchers {
		if w == watcher {
			cm.watchers[key] = append(watchers[:i], watchers[i+1:]...)
			break
		}
	}
}

// GetAllConfigs 获取所有配置
func (cm *ConfigManager) GetAllConfigs() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make(map[string]interface{})
	for key, value := range cm.configs {
		configs[key] = value
	}

	return configs
}

// GetVersion 获取配置版本
func (cm *ConfigManager) GetVersion() int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.version
}

// GetLastUpdate 获取最后更新时间
func (cm *ConfigManager) GetLastUpdate() time.Time {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.lastUpdate
}

// ConfigValidator 配置验证器
type ConfigValidator struct {
	rules map[string]ValidationRule
}

// ValidationRule 验证规则
type ValidationRule struct {
	Required bool
	Type     string
	Min      interface{}
	Max      interface{}
	Pattern  string
	Custom   func(interface{}) error
}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		rules: make(map[string]ValidationRule),
	}
}

// AddRule 添加验证规则
func (cv *ConfigValidator) AddRule(key string, rule ValidationRule) {
	cv.rules[key] = rule
}

// Validate 验证配置
func (cv *ConfigValidator) Validate(configs map[string]interface{}) error {
	for key, value := range configs {
		rule, exists := cv.rules[key]
		if !exists {
			continue
		}

		if err := cv.validateValue(key, value, rule); err != nil {
			return err
		}
	}

	return nil
}

// validateValue 验证单个值
func (cv *ConfigValidator) validateValue(key string, value interface{}, rule ValidationRule) error {
	// 检查必填
	if rule.Required && value == nil {
		return utils.WrapError(nil, utils.ErrCodeConfigError, fmt.Sprintf("配置 %s 是必填的", key))
	}

	// 检查类型
	if rule.Type != "" {
		if err := cv.validateType(key, value, rule.Type); err != nil {
			return err
		}
	}

	// 检查范围
	if rule.Min != nil || rule.Max != nil {
		if err := cv.validateRange(key, value, rule.Min, rule.Max); err != nil {
			return err
		}
	}

	// 检查模式
	if rule.Pattern != "" {
		if err := cv.validatePattern(key, value, rule.Pattern); err != nil {
			return err
		}
	}

	// 自定义验证
	if rule.Custom != nil {
		if err := rule.Custom(value); err != nil {
			return utils.WrapError(err, utils.ErrCodeConfigError, fmt.Sprintf("配置 %s 验证失败", key))
		}
	}

	return nil
}

// validateType 验证类型
func (cv *ConfigValidator) validateType(key string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return utils.WrapError(nil, utils.ErrCodeConfigError, fmt.Sprintf("配置 %s 必须是字符串", key))
		}
	case "int":
		if _, ok := value.(int); !ok {
			return utils.WrapError(nil, utils.ErrCodeConfigError, fmt.Sprintf("配置 %s 必须是整数", key))
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return utils.WrapError(nil, utils.ErrCodeConfigError, fmt.Sprintf("配置 %s 必须是布尔值", key))
		}
	case "float":
		if _, ok := value.(float64); !ok {
			return utils.WrapError(nil, utils.ErrCodeConfigError, fmt.Sprintf("配置 %s 必须是浮点数", key))
		}
	}
	return nil
}

// validateRange 验证范围
func (cv *ConfigValidator) validateRange(key string, value interface{}, min, max interface{}) error {
	// 实现范围验证逻辑
	return nil
}

// validatePattern 验证模式
func (cv *ConfigValidator) validatePattern(key string, value interface{}, pattern string) error {
	// 实现模式验证逻辑
	return nil
}

// ConfigBackup 配置备份
type ConfigBackup struct {
	configs   map[string]interface{}
	version   int64
	timestamp time.Time
}

// NewConfigBackup 创建配置备份
func NewConfigBackup(configs map[string]interface{}, version int64) *ConfigBackup {
	return &ConfigBackup{
		configs:   configs,
		version:   version,
		timestamp: time.Now(),
	}
}

// ToJSON 转换为JSON
func (cb *ConfigBackup) ToJSON() ([]byte, error) {
	return json.Marshal(cb)
}

// FromJSON 从JSON恢复
func (cb *ConfigBackup) FromJSON(data []byte) error {
	return json.Unmarshal(data, cb)
}

// ConfigManagerWithBackup 带备份的配置管理器
type ConfigManagerWithBackup struct {
	*ConfigManager
	backups    []*ConfigBackup
	maxBackups int
	mu         sync.RWMutex
}

// NewConfigManagerWithBackup 创建带备份的配置管理器
func NewConfigManagerWithBackup(maxBackups int) *ConfigManagerWithBackup {
	return &ConfigManagerWithBackup{
		ConfigManager: NewConfigManager(),
		backups:       make([]*ConfigBackup, 0),
		maxBackups:    maxBackups,
	}
}

// SetConfig 设置配置（带备份）
func (cmb *ConfigManagerWithBackup) SetConfig(key string, value interface{}) error {
	// 创建备份
	cmb.createBackup()

	// 设置配置
	return cmb.ConfigManager.SetConfig(key, value)
}

// createBackup 创建备份
func (cmb *ConfigManagerWithBackup) createBackup() {
	cmb.mu.Lock()
	defer cmb.mu.Unlock()

	// 获取当前配置
	configs := cmb.ConfigManager.GetAllConfigs()
	version := cmb.ConfigManager.GetVersion()

	// 创建备份
	backup := NewConfigBackup(configs, version)
	cmb.backups = append(cmb.backups, backup)

	// 限制备份数量
	if len(cmb.backups) > cmb.maxBackups {
		cmb.backups = cmb.backups[1:]
	}
}

// RestoreFromBackup 从备份恢复
func (cmb *ConfigManagerWithBackup) RestoreFromBackup(version int64) error {
	cmb.mu.RLock()
	defer cmb.mu.RUnlock()

	// 查找指定版本的备份
	var backup *ConfigBackup
	for _, b := range cmb.backups {
		if b.version == version {
			backup = b
			break
		}
	}

	if backup == nil {
		return utils.WrapError(nil, utils.ErrCodeConfigError, "备份不存在")
	}

	// 恢复配置
	for key, value := range backup.configs {
		cmb.ConfigManager.SetConfig(key, value)
	}

	return nil
}

// ListBackups 列出所有备份
func (cmb *ConfigManagerWithBackup) ListBackups() []*ConfigBackup {
	cmb.mu.RLock()
	defer cmb.mu.RUnlock()

	backups := make([]*ConfigBackup, len(cmb.backups))
	copy(backups, cmb.backups)
	return backups
}

// ConfigHotReload 配置热重载
type ConfigHotReload struct {
	manager  *ConfigManager
	watchers map[string]ConfigWatcher
	interval time.Duration
	stopChan chan struct{}
	mu       sync.RWMutex
}

// NewConfigHotReload 创建配置热重载
func NewConfigHotReload(manager *ConfigManager, interval time.Duration) *ConfigHotReload {
	return &ConfigHotReload{
		manager:  manager,
		watchers: make(map[string]ConfigWatcher),
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start 启动热重载
func (chr *ConfigHotReload) Start() {
	go chr.reloadLoop()
}

// Stop 停止热重载
func (chr *ConfigHotReload) Stop() {
	close(chr.stopChan)
}

// reloadLoop 重载循环
func (chr *ConfigHotReload) reloadLoop() {
	ticker := time.NewTicker(chr.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			chr.reloadConfigs()
		case <-chr.stopChan:
			return
		}
	}
}

// reloadConfigs 重载配置
func (chr *ConfigHotReload) reloadConfigs() {
	// 从配置源重新加载配置
	// 这里应该实现具体的重载逻辑
	// 例如从文件、数据库或配置中心加载
}

// AddWatcher 添加监听器
func (chr *ConfigHotReload) AddWatcher(name string, watcher ConfigWatcher) {
	chr.mu.Lock()
	defer chr.mu.Unlock()
	chr.watchers[name] = watcher
}

// RemoveWatcher 移除监听器
func (chr *ConfigHotReload) RemoveWatcher(name string) {
	chr.mu.Lock()
	defer chr.mu.Unlock()
	delete(chr.watchers, name)
}

// ConfigAudit 配置审计
type ConfigAudit struct {
	changes []ConfigChangeEvent
	mu      sync.RWMutex
}

// NewConfigAudit 创建配置审计
func NewConfigAudit() *ConfigAudit {
	return &ConfigAudit{
		changes: make([]ConfigChangeEvent, 0),
	}
}

// RecordChange 记录变更
func (ca *ConfigAudit) RecordChange(key string, oldValue, newValue interface{}, version int64) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	event := ConfigChangeEvent{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  newValue,
		Timestamp: time.Now(),
		Version:   version,
	}

	ca.changes = append(ca.changes, event)
}

// GetChanges 获取变更记录
func (ca *ConfigAudit) GetChanges() []ConfigChangeEvent {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	changes := make([]ConfigChangeEvent, len(ca.changes))
	copy(changes, ca.changes)
	return changes
}

// GetChangesByKey 根据键获取变更记录
func (ca *ConfigAudit) GetChangesByKey(key string) []ConfigChangeEvent {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var changes []ConfigChangeEvent
	for _, change := range ca.changes {
		if change.Key == key {
			changes = append(changes, change)
		}
	}

	return changes
}

// GetChangesByTimeRange 根据时间范围获取变更记录
func (ca *ConfigAudit) GetChangesByTimeRange(start, end time.Time) []ConfigChangeEvent {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var changes []ConfigChangeEvent
	for _, change := range ca.changes {
		if change.Timestamp.After(start) && change.Timestamp.Before(end) {
			changes = append(changes, change)
		}
	}

	return changes
}
