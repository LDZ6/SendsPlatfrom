package config

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ConfigVersion 配置版本
type ConfigVersion struct {
	Version     string                 `json:"version"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	Description string                 `json:"description"`
	Checksum    string                 `json:"checksum"`
	IsActive    bool                   `json:"is_active"`
}

// ConfigChange 配置变更
type ConfigChange struct {
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	Changes     map[string]interface{} `json:"changes"`
	ChangeType  ChangeType             `json:"change_type"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	Description string                 `json:"description"`
	ApprovedBy  string                 `json:"approved_by"`
	ApprovedAt  *time.Time             `json:"approved_at"`
	Status      ChangeStatus           `json:"status"`
}

// ChangeType 变更类型
type ChangeType int

const (
	ChangeTypeCreate ChangeType = iota
	ChangeTypeUpdate
	ChangeTypeDelete
	ChangeTypeRollback
)

// String 变更类型字符串
func (ct ChangeType) String() string {
	switch ct {
	case ChangeTypeCreate:
		return "create"
	case ChangeTypeUpdate:
		return "update"
	case ChangeTypeDelete:
		return "delete"
	case ChangeTypeRollback:
		return "rollback"
	default:
		return "unknown"
	}
}

// ChangeStatus 变更状态
type ChangeStatus int

const (
	ChangeStatusPending ChangeStatus = iota
	ChangeStatusApproved
	ChangeStatusRejected
	ChangeStatusApplied
	ChangeStatusFailed
)

// String 变更状态字符串
func (cs ChangeStatus) String() string {
	switch cs {
	case ChangeStatusPending:
		return "pending"
	case ChangeStatusApproved:
		return "approved"
	case ChangeStatusRejected:
		return "rejected"
	case ChangeStatusApplied:
		return "applied"
	case ChangeStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ConfigVersionManager 配置版本管理器
type ConfigVersionManager struct {
	versions      map[string]*ConfigVersion
	changes       map[string]*ConfigChange
	activeVersion string
	validator     *ConfigValidator
	mutex         sync.RWMutex
}

// NewConfigVersionManager 创建配置版本管理器
func NewConfigVersionManager() *ConfigVersionManager {
	return &ConfigVersionManager{
		versions:  make(map[string]*ConfigVersion),
		changes:   make(map[string]*ConfigChange),
		validator: NewConfigValidator(),
	}
}

// CreateVersion 创建配置版本
func (cvm *ConfigVersionManager) CreateVersion(version string, config map[string]interface{}, createdBy, description string) (*ConfigVersion, error) {
	cvm.mutex.Lock()
	defer cvm.mutex.Unlock()

	// 检查版本是否已存在
	if _, exists := cvm.versions[version]; exists {
		return nil, fmt.Errorf("version %s already exists", version)
	}

	// 计算配置校验和
	checksum, err := cvm.calculateChecksum(config)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// 创建版本
	configVersion := &ConfigVersion{
		Version:     version,
		Config:      config,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
		Description: description,
		Checksum:    checksum,
		IsActive:    false,
	}

	cvm.versions[version] = configVersion

	// 记录变更
	change := &ConfigChange{
		ID:          cvm.generateChangeID(),
		Version:     version,
		Changes:     config,
		ChangeType:  ChangeTypeCreate,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
		Description: description,
		Status:      ChangeStatusPending,
	}
	cvm.changes[change.ID] = change

	return configVersion, nil
}

// UpdateVersion 更新配置版本
func (cvm *ConfigVersionManager) UpdateVersion(version string, config map[string]interface{}, updatedBy, description string) (*ConfigVersion, error) {
	cvm.mutex.Lock()
	defer cvm.mutex.Unlock()

	// 检查版本是否存在
	existingVersion, exists := cvm.versions[version]
	if !exists {
		return nil, fmt.Errorf("version %s not found", version)
	}

	// 计算配置校验和
	checksum, err := cvm.calculateChecksum(config)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// 更新版本
	existingVersion.Config = config
	existingVersion.Checksum = checksum
	existingVersion.CreatedBy = updatedBy
	existingVersion.Description = description
	existingVersion.CreatedAt = time.Now()

	// 记录变更
	change := &ConfigChange{
		ID:          cvm.generateChangeID(),
		Version:     version,
		Changes:     config,
		ChangeType:  ChangeTypeUpdate,
		CreatedAt:   time.Now(),
		CreatedBy:   updatedBy,
		Description: description,
		Status:      ChangeStatusPending,
	}
	cvm.changes[change.ID] = change

	return existingVersion, nil
}

// ActivateVersion 激活配置版本
func (cvm *ConfigVersionManager) ActivateVersion(version string, activatedBy string) error {
	cvm.mutex.Lock()
	defer cvm.mutex.Unlock()

	// 检查版本是否存在
	configVersion, exists := cvm.versions[version]
	if !exists {
		return fmt.Errorf("version %s not found", version)
	}

	// 取消激活当前版本
	if cvm.activeVersion != "" {
		if activeVersion, exists := cvm.versions[cvm.activeVersion]; exists {
			activeVersion.IsActive = false
		}
	}

	// 激活新版本
	configVersion.IsActive = true
	cvm.activeVersion = version

	// 记录变更
	change := &ConfigChange{
		ID:          cvm.generateChangeID(),
		Version:     version,
		Changes:     map[string]interface{}{"is_active": true},
		ChangeType:  ChangeTypeUpdate,
		CreatedAt:   time.Now(),
		CreatedBy:   activatedBy,
		Description: fmt.Sprintf("Activated version %s", version),
		Status:      ChangeStatusApplied,
	}
	cvm.changes[change.ID] = change

	return nil
}

// RollbackVersion 回滚到指定版本
func (cvm *ConfigVersionManager) RollbackVersion(version string, rolledBackBy string) error {
	cvm.mutex.Lock()
	defer cvm.mutex.Unlock()

	// 检查版本是否存在
	configVersion, exists := cvm.versions[version]
	if !exists {
		return fmt.Errorf("version %s not found", version)
	}

	// 取消激活当前版本
	if cvm.activeVersion != "" {
		if activeVersion, exists := cvm.versions[cvm.activeVersion]; exists {
			activeVersion.IsActive = false
		}
	}

	// 激活回滚版本
	configVersion.IsActive = true
	cvm.activeVersion = version

	// 记录变更
	change := &ConfigChange{
		ID:          cvm.generateChangeID(),
		Version:     version,
		Changes:     map[string]interface{}{"is_active": true},
		ChangeType:  ChangeTypeRollback,
		CreatedAt:   time.Now(),
		CreatedBy:   rolledBackBy,
		Description: fmt.Sprintf("Rolled back to version %s", version),
		Status:      ChangeStatusApplied,
	}
	cvm.changes[change.ID] = change

	return nil
}

// GetVersion 获取配置版本
func (cvm *ConfigVersionManager) GetVersion(version string) (*ConfigVersion, error) {
	cvm.mutex.RLock()
	defer cvm.mutex.RUnlock()

	configVersion, exists := cvm.versions[version]
	if !exists {
		return nil, fmt.Errorf("version %s not found", version)
	}

	return configVersion, nil
}

// GetActiveVersion 获取当前激活的版本
func (cvm *ConfigVersionManager) GetActiveVersion() (*ConfigVersion, error) {
	cvm.mutex.RLock()
	defer cvm.mutex.RUnlock()

	if cvm.activeVersion == "" {
		return nil, fmt.Errorf("no active version")
	}

	configVersion, exists := cvm.versions[cvm.activeVersion]
	if !exists {
		return nil, fmt.Errorf("active version %s not found", cvm.activeVersion)
	}

	return configVersion, nil
}

// ListVersions 列出所有版本
func (cvm *ConfigVersionManager) ListVersions() []*ConfigVersion {
	cvm.mutex.RLock()
	defer cvm.mutex.RUnlock()

	var versions []*ConfigVersion
	for _, version := range cvm.versions {
		versions = append(versions, version)
	}

	return versions
}

// ListChanges 列出所有变更
func (cvm *ConfigVersionManager) ListChanges() []*ConfigChange {
	cvm.mutex.RLock()
	defer cvm.mutex.RUnlock()

	var changes []*ConfigChange
	for _, change := range cvm.changes {
		changes = append(changes, change)
	}

	return changes
}

// ApproveChange 批准变更
func (cvm *ConfigVersionManager) ApproveChange(changeID string, approvedBy string) error {
	cvm.mutex.Lock()
	defer cvm.mutex.Unlock()

	change, exists := cvm.changes[changeID]
	if !exists {
		return fmt.Errorf("change %s not found", changeID)
	}

	if change.Status != ChangeStatusPending {
		return fmt.Errorf("change %s is not pending", changeID)
	}

	change.Status = ChangeStatusApproved
	change.ApprovedBy = approvedBy
	now := time.Now()
	change.ApprovedAt = &now

	return nil
}

// RejectChange 拒绝变更
func (cvm *ConfigVersionManager) RejectChange(changeID string, rejectedBy string) error {
	cvm.mutex.Lock()
	defer cvm.mutex.Unlock()

	change, exists := cvm.changes[changeID]
	if !exists {
		return fmt.Errorf("change %s not found", changeID)
	}

	if change.Status != ChangeStatusPending {
		return fmt.Errorf("change %s is not pending", changeID)
	}

	change.Status = ChangeStatusRejected
	change.ApprovedBy = rejectedBy
	now := time.Now()
	change.ApprovedAt = &now

	return nil
}

// calculateChecksum 计算配置校验和
func (cvm *ConfigVersionManager) calculateChecksum(config map[string]interface{}) (string, error) {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	// 这里可以使用更复杂的校验和算法，如SHA256
	// 简化实现，直接返回字符串长度
	return fmt.Sprintf("%d", len(configBytes)), nil
}

// generateChangeID 生成变更ID
func (cvm *ConfigVersionManager) generateChangeID() string {
	return fmt.Sprintf("change_%d", time.Now().UnixNano())
}

// ConfigAuditLogger 配置审计日志
type ConfigAuditLogger struct {
	logs  []ConfigAuditLog
	mutex sync.RWMutex
}

// ConfigAuditLog 配置审计日志
type ConfigAuditLog struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Version   string    `json:"version"`
	User      string    `json:"user"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details"`
}

// NewConfigAuditLogger 创建配置审计日志
func NewConfigAuditLogger() *ConfigAuditLogger {
	return &ConfigAuditLogger{
		logs: make([]ConfigAuditLog, 0),
	}
}

// Log 记录审计日志
func (cal *ConfigAuditLogger) Log(action, version, user, details string) {
	cal.mutex.Lock()
	defer cal.mutex.Unlock()

	log := ConfigAuditLog{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Action:    action,
		Version:   version,
		User:      user,
		Timestamp: time.Now(),
		Details:   details,
	}

	cal.logs = append(cal.logs, log)
}

// GetLogs 获取审计日志
func (cal *ConfigAuditLogger) GetLogs() []ConfigAuditLog {
	cal.mutex.RLock()
	defer cal.mutex.RUnlock()

	logs := make([]ConfigAuditLog, len(cal.logs))
	copy(logs, cal.logs)

	return logs
}

// GetLogsByUser 根据用户获取审计日志
func (cal *ConfigAuditLogger) GetLogsByUser(user string) []ConfigAuditLog {
	cal.mutex.RLock()
	defer cal.mutex.RUnlock()

	var userLogs []ConfigAuditLog
	for _, log := range cal.logs {
		if log.User == user {
			userLogs = append(userLogs, log)
		}
	}

	return userLogs
}

// GetLogsByVersion 根据版本获取审计日志
func (cal *ConfigAuditLogger) GetLogsByVersion(version string) []ConfigAuditLog {
	cal.mutex.RLock()
	defer cal.mutex.RUnlock()

	var versionLogs []ConfigAuditLog
	for _, log := range cal.logs {
		if log.Version == version {
			versionLogs = append(versionLogs, log)
		}
	}

	return versionLogs
}

// ConfigHotReloader 配置热重载器
type ConfigHotReloader struct {
	versionManager *ConfigVersionManager
	auditLogger    *ConfigAuditLogger
	watchers       map[string]ConfigWatcher
	mutex          sync.RWMutex
}

// ConfigWatcher 配置观察者
type ConfigWatcher interface {
	OnConfigChange(version string, config map[string]interface{}) error
	GetWatcherID() string
}

// NewConfigHotReloader 创建配置热重载器
func NewConfigHotReloader(versionManager *ConfigVersionManager, auditLogger *ConfigAuditLogger) *ConfigHotReloader {
	return &ConfigHotReloader{
		versionManager: versionManager,
		auditLogger:    auditLogger,
		watchers:       make(map[string]ConfigWatcher),
	}
}

// RegisterWatcher 注册观察者
func (chr *ConfigHotReloader) RegisterWatcher(watcher ConfigWatcher) {
	chr.mutex.Lock()
	defer chr.mutex.Unlock()
	chr.watchers[watcher.GetWatcherID()] = watcher
}

// UnregisterWatcher 取消注册观察者
func (chr *ConfigHotReloader) UnregisterWatcher(watcherID string) {
	chr.mutex.Lock()
	defer chr.mutex.Unlock()
	delete(chr.watchers, watcherID)
}

// ReloadConfig 重载配置
func (chr *ConfigHotReloader) ReloadConfig(ctx context.Context, version string) error {
	// 获取配置版本
	configVersion, err := chr.versionManager.GetVersion(version)
	if err != nil {
		return err
	}

	// 激活版本
	if err := chr.versionManager.ActivateVersion(version, "system"); err != nil {
		return err
	}

	// 通知所有观察者
	chr.mutex.RLock()
	watchers := make([]ConfigWatcher, 0, len(chr.watchers))
	for _, watcher := range chr.watchers {
		watchers = append(watchers, watcher)
	}
	chr.mutex.RUnlock()

	for _, watcher := range watchers {
		if err := watcher.OnConfigChange(version, configVersion.Config); err != nil {
			// 记录错误但继续通知其他观察者
			chr.auditLogger.Log("config_reload_error", version, "system",
				fmt.Sprintf("Failed to notify watcher %s: %v", watcher.GetWatcherID(), err))
		}
	}

	// 记录审计日志
	chr.auditLogger.Log("config_reloaded", version, "system",
		fmt.Sprintf("Configuration reloaded to version %s", version))

	return nil
}

// WatchConfigChanges 监听配置变更
func (chr *ConfigHotReloader) WatchConfigChanges(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 检查是否有新的配置版本
			// 这里可以实现具体的检查逻辑
		}
	}
}
