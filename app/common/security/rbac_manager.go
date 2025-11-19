package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Role 角色
type Role struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Permissions []string               `json:"permissions"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Permission 权限
type Permission struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Policy 策略
type Policy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rules       []Rule                 `json:"rules"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Rule 规则
type Rule struct {
	ID         string                 `json:"id"`
	Effect     string                 `json:"effect"` // Allow, Deny
	Resources  []string               `json:"resources"`
	Actions    []string               `json:"actions"`
	Roles      []string               `json:"roles"`
	Users      []string               `json:"users"`
	Conditions []Condition            `json:"conditions"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Condition 条件
type Condition struct {
	Type     string                 `json:"type"`
	Field    string                 `json:"field"`
	Operator string                 `json:"operator"`
	Value    interface{}            `json:"value"`
	Metadata map[string]interface{} `json:"metadata"`
}

// User 用户
type User struct {
	ID          string                 `json:"id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RBACManager RBAC管理器
type RBACManager struct {
	roles       map[string]*Role
	permissions map[string]*Permission
	policies    map[string]*Policy
	users       map[string]*User
	enforcer    Enforcer
	mutex       sync.RWMutex
}

// Enforcer 执行器接口
type Enforcer interface {
	Enforce(ctx context.Context, userID, resource, action string, context map[string]interface{}) (bool, error)
	AddPolicy(rule *Rule) error
	RemovePolicy(ruleID string) error
	UpdatePolicy(rule *Rule) error
	GetPolicies() ([]*Rule, error)
}

// NewRBACManager 创建RBAC管理器
func NewRBACManager(enforcer Enforcer) *RBACManager {
	return &RBACManager{
		roles:       make(map[string]*Role),
		permissions: make(map[string]*Permission),
		policies:    make(map[string]*Policy),
		users:       make(map[string]*User),
		enforcer:    enforcer,
	}
}

// CreateRole 创建角色
func (rm *RBACManager) CreateRole(role *Role) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.roles[role.ID]; exists {
		return fmt.Errorf("role already exists: %s", role.ID)
	}

	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	rm.roles[role.ID] = role

	return nil
}

// GetRole 获取角色
func (rm *RBACManager) GetRole(roleID string) (*Role, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	role, exists := rm.roles[roleID]
	if !exists {
		return nil, fmt.Errorf("role not found: %s", roleID)
	}

	return role, nil
}

// UpdateRole 更新角色
func (rm *RBACManager) UpdateRole(role *Role) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.roles[role.ID]; !exists {
		return fmt.Errorf("role not found: %s", role.ID)
	}

	role.UpdatedAt = time.Now()
	rm.roles[role.ID] = role

	return nil
}

// DeleteRole 删除角色
func (rm *RBACManager) DeleteRole(roleID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.roles[roleID]; !exists {
		return fmt.Errorf("role not found: %s", roleID)
	}

	delete(rm.roles, roleID)

	// 从所有用户中移除该角色
	for _, user := range rm.users {
		for i, role := range user.Roles {
			if role == roleID {
				user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
				break
			}
		}
	}

	return nil
}

// CreatePermission 创建权限
func (rm *RBACManager) CreatePermission(permission *Permission) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.permissions[permission.ID]; exists {
		return fmt.Errorf("permission already exists: %s", permission.ID)
	}

	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()
	rm.permissions[permission.ID] = permission

	return nil
}

// GetPermission 获取权限
func (rm *RBACManager) GetPermission(permissionID string) (*Permission, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	permission, exists := rm.permissions[permissionID]
	if !exists {
		return nil, fmt.Errorf("permission not found: %s", permissionID)
	}

	return permission, nil
}

// UpdatePermission 更新权限
func (rm *RBACManager) UpdatePermission(permission *Permission) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.permissions[permission.ID]; !exists {
		return fmt.Errorf("permission not found: %s", permission.ID)
	}

	permission.UpdatedAt = time.Now()
	rm.permissions[permission.ID] = permission

	return nil
}

// DeletePermission 删除权限
func (rm *RBACManager) DeletePermission(permissionID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.permissions[permissionID]; !exists {
		return fmt.Errorf("permission not found: %s", permissionID)
	}

	delete(rm.permissions, permissionID)

	// 从所有角色中移除该权限
	for _, role := range rm.roles {
		for i, perm := range role.Permissions {
			if perm == permissionID {
				role.Permissions = append(role.Permissions[:i], role.Permissions[i+1:]...)
				break
			}
		}
	}

	// 从所有用户中移除该权限
	for _, user := range rm.users {
		for i, perm := range user.Permissions {
			if perm == permissionID {
				user.Permissions = append(user.Permissions[:i], user.Permissions[i+1:]...)
				break
			}
		}
	}

	return nil
}

// CreatePolicy 创建策略
func (rm *RBACManager) CreatePolicy(policy *Policy) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.policies[policy.ID]; exists {
		return fmt.Errorf("policy already exists: %s", policy.ID)
	}

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	rm.policies[policy.ID] = policy

	// 添加规则到执行器
	for _, rule := range policy.Rules {
		if err := rm.enforcer.AddPolicy(&rule); err != nil {
			return fmt.Errorf("failed to add rule to enforcer: %w", err)
		}
	}

	return nil
}

// GetPolicy 获取策略
func (rm *RBACManager) GetPolicy(policyID string) (*Policy, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	policy, exists := rm.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	return policy, nil
}

// UpdatePolicy 更新策略
func (rm *RBACManager) UpdatePolicy(policy *Policy) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.policies[policy.ID]; !exists {
		return fmt.Errorf("policy not found: %s", policy.ID)
	}

	// 移除旧规则
	oldPolicy := rm.policies[policy.ID]
	for _, rule := range oldPolicy.Rules {
		rm.enforcer.RemovePolicy(rule.ID)
	}

	// 添加新规则
	for _, rule := range policy.Rules {
		if err := rm.enforcer.AddPolicy(&rule); err != nil {
			return fmt.Errorf("failed to add rule to enforcer: %w", err)
		}
	}

	policy.UpdatedAt = time.Now()
	rm.policies[policy.ID] = policy

	return nil
}

// DeletePolicy 删除策略
func (rm *RBACManager) DeletePolicy(policyID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	policy, exists := rm.policies[policyID]
	if !exists {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	// 移除规则
	for _, rule := range policy.Rules {
		rm.enforcer.RemovePolicy(rule.ID)
	}

	delete(rm.policies, policyID)

	return nil
}

// CreateUser 创建用户
func (rm *RBACManager) CreateUser(user *User) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.users[user.ID]; exists {
		return fmt.Errorf("user already exists: %s", user.ID)
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	rm.users[user.ID] = user

	return nil
}

// GetUser 获取用户
func (rm *RBACManager) GetUser(userID string) (*User, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	user, exists := rm.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return user, nil
}

// UpdateUser 更新用户
func (rm *RBACManager) UpdateUser(user *User) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.users[user.ID]; !exists {
		return fmt.Errorf("user not found: %s", user.ID)
	}

	user.UpdatedAt = time.Now()
	rm.users[user.ID] = user

	return nil
}

// DeleteUser 删除用户
func (rm *RBACManager) DeleteUser(userID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.users[userID]; !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	delete(rm.users, userID)

	return nil
}

// AssignRole 分配角色
func (rm *RBACManager) AssignRole(userID, roleID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	user, exists := rm.users[userID]
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	role, exists := rm.roles[roleID]
	if !exists {
		return fmt.Errorf("role not found: %s", roleID)
	}

	// 检查是否已经分配
	for _, r := range user.Roles {
		if r == roleID {
			return fmt.Errorf("role already assigned: %s", roleID)
		}
	}

	user.Roles = append(user.Roles, roleID)
	user.UpdatedAt = time.Now()

	// 添加角色权限到用户权限
	for _, permission := range role.Permissions {
		// 检查是否已经存在
		exists := false
		for _, p := range user.Permissions {
			if p == permission {
				exists = true
				break
			}
		}
		if !exists {
			user.Permissions = append(user.Permissions, permission)
		}
	}

	return nil
}

// RevokeRole 撤销角色
func (rm *RBACManager) RevokeRole(userID, roleID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	user, exists := rm.users[userID]
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	// 移除角色
	for i, r := range user.Roles {
		if r == roleID {
			user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
			break
		}
	}

	user.UpdatedAt = time.Now()

	// 重新计算用户权限
	user.Permissions = []string{}
	for _, roleID := range user.Roles {
		if role, exists := rm.roles[roleID]; exists {
			for _, permission := range role.Permissions {
				// 检查是否已经存在
				exists := false
				for _, p := range user.Permissions {
					if p == permission {
						exists = true
						break
					}
				}
				if !exists {
					user.Permissions = append(user.Permissions, permission)
				}
			}
		}
	}

	return nil
}

// AssignPermission 分配权限
func (rm *RBACManager) AssignPermission(userID, permissionID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	user, exists := rm.users[userID]
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	_, exists := rm.permissions[permissionID]
	if !exists {
		return fmt.Errorf("permission not found: %s", permissionID)
	}

	// 检查是否已经分配
	for _, p := range user.Permissions {
		if p == permissionID {
			return fmt.Errorf("permission already assigned: %s", permissionID)
		}
	}

	user.Permissions = append(user.Permissions, permissionID)
	user.UpdatedAt = time.Now()

	return nil
}

// RevokePermission 撤销权限
func (rm *RBACManager) RevokePermission(userID, permissionID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	user, exists := rm.users[userID]
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	// 移除权限
	for i, p := range user.Permissions {
		if p == permissionID {
			user.Permissions = append(user.Permissions[:i], user.Permissions[i+1:]...)
			break
		}
	}

	user.UpdatedAt = time.Now()

	return nil
}

// CheckPermission 检查权限
func (rm *RBACManager) CheckPermission(ctx context.Context, userID, resource, action string, contextData map[string]interface{}) (bool, error) {
	return rm.enforcer.Enforce(ctx, userID, resource, action, contextData)
}

// GetUserPermissions 获取用户权限
func (rm *RBACManager) GetUserPermissions(userID string) ([]string, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	user, exists := rm.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return user.Permissions, nil
}

// GetUserRoles 获取用户角色
func (rm *RBACManager) GetUserRoles(userID string) ([]string, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	user, exists := rm.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return user.Roles, nil
}

// ListRoles 列出所有角色
func (rm *RBACManager) ListRoles() []*Role {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	roles := make([]*Role, 0, len(rm.roles))
	for _, role := range rm.roles {
		roles = append(roles, role)
	}

	return roles
}

// ListPermissions 列出所有权限
func (rm *RBACManager) ListPermissions() []*Permission {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	permissions := make([]*Permission, 0, len(rm.permissions))
	for _, permission := range rm.permissions {
		permissions = append(permissions, permission)
	}

	return permissions
}

// ListPolicies 列出所有策略
func (rm *RBACManager) ListPolicies() []*Policy {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	policies := make([]*Policy, 0, len(rm.policies))
	for _, policy := range rm.policies {
		policies = append(policies, policy)
	}

	return policies
}

// ListUsers 列出所有用户
func (rm *RBACManager) ListUsers() []*User {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	users := make([]*User, 0, len(rm.users))
	for _, user := range rm.users {
		users = append(users, user)
	}

	return users
}
