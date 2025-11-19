package security

import (
	"fmt"
	"sync"
	"time"
)

// Role 角色
type Role struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// Permission 权限
type Permission struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Resource  string            `json:"resource"`
	Action    string            `json:"action"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// User 用户
type User struct {
	ID          string            `json:"id"`
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	Roles       []string          `json:"roles"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// RBAC 基于角色的访问控制
type RBAC struct {
	roles       map[string]*Role
	permissions map[string]*Permission
	users       map[string]*User
	mutex       sync.RWMutex
}

// NewRBAC 创建RBAC实例
func NewRBAC() *RBAC {
	return &RBAC{
		roles:       make(map[string]*Role),
		permissions: make(map[string]*Permission),
		users:       make(map[string]*User),
	}
}

// AddRole 添加角色
func (rbac *RBAC) AddRole(role *Role) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.roles[role.ID]; exists {
		return fmt.Errorf("role %s already exists", role.ID)
	}

	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	rbac.roles[role.ID] = role

	return nil
}

// GetRole 获取角色
func (rbac *RBAC) GetRole(roleID string) (*Role, error) {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	role, exists := rbac.roles[roleID]
	if !exists {
		return nil, fmt.Errorf("role %s not found", roleID)
	}

	return role, nil
}

// UpdateRole 更新角色
func (rbac *RBAC) UpdateRole(role *Role) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.roles[role.ID]; !exists {
		return fmt.Errorf("role %s not found", role.ID)
	}

	role.UpdatedAt = time.Now()
	rbac.roles[role.ID] = role

	return nil
}

// DeleteRole 删除角色
func (rbac *RBAC) DeleteRole(roleID string) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.roles[roleID]; !exists {
		return fmt.Errorf("role %s not found", roleID)
	}

	delete(rbac.roles, roleID)

	return nil
}

// AddPermission 添加权限
func (rbac *RBAC) AddPermission(permission *Permission) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.permissions[permission.ID]; exists {
		return fmt.Errorf("permission %s already exists", permission.ID)
	}

	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()
	rbac.permissions[permission.ID] = permission

	return nil
}

// GetPermission 获取权限
func (rbac *RBAC) GetPermission(permissionID string) (*Permission, error) {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	permission, exists := rbac.permissions[permissionID]
	if !exists {
		return nil, fmt.Errorf("permission %s not found", permissionID)
	}

	return permission, nil
}

// UpdatePermission 更新权限
func (rbac *RBAC) UpdatePermission(permission *Permission) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.permissions[permission.ID]; !exists {
		return fmt.Errorf("permission %s not found", permission.ID)
	}

	permission.UpdatedAt = time.Now()
	rbac.permissions[permission.ID] = permission

	return nil
}

// DeletePermission 删除权限
func (rbac *RBAC) DeletePermission(permissionID string) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.permissions[permissionID]; !exists {
		return fmt.Errorf("permission %s not found", permissionID)
	}

	delete(rbac.permissions, permissionID)

	return nil
}

// AddUser 添加用户
func (rbac *RBAC) AddUser(user *User) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.users[user.ID]; exists {
		return fmt.Errorf("user %s already exists", user.ID)
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	rbac.users[user.ID] = user

	return nil
}

// GetUser 获取用户
func (rbac *RBAC) GetUser(userID string) (*User, error) {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	user, exists := rbac.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	return user, nil
}

// UpdateUser 更新用户
func (rbac *RBAC) UpdateUser(user *User) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.users[user.ID]; !exists {
		return fmt.Errorf("user %s not found", user.ID)
	}

	user.UpdatedAt = time.Now()
	rbac.users[user.ID] = user

	return nil
}

// DeleteUser 删除用户
func (rbac *RBAC) DeleteUser(userID string) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.users[userID]; !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	delete(rbac.users, userID)

	return nil
}

// AssignRoleToUser 为用户分配角色
func (rbac *RBAC) AssignRoleToUser(userID, roleID string) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	user, exists := rbac.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	role, exists := rbac.roles[roleID]
	if !exists {
		return fmt.Errorf("role %s not found", roleID)
	}

	// 检查角色是否已分配
	for _, existingRole := range user.Roles {
		if existingRole == roleID {
			return fmt.Errorf("role %s already assigned to user %s", roleID, userID)
		}
	}

	user.Roles = append(user.Roles, roleID)
	user.UpdatedAt = time.Now()

	// 将角色的权限添加到用户权限中
	for _, permission := range role.Permissions {
		user.Permissions = append(user.Permissions, permission)
	}

	return nil
}

// RemoveRoleFromUser 移除用户角色
func (rbac *RBAC) RemoveRoleFromUser(userID, roleID string) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	user, exists := rbac.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	// 查找并移除角色
	for i, existingRole := range user.Roles {
		if existingRole == roleID {
			user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
			user.UpdatedAt = time.Now()

			// 移除角色的权限
			role, exists := rbac.roles[roleID]
			if exists {
				for _, permission := range role.Permissions {
					for j, userPermission := range user.Permissions {
						if userPermission == permission {
							user.Permissions = append(user.Permissions[:j], user.Permissions[j+1:]...)
							break
						}
					}
				}
			}

			return nil
		}
	}

	return fmt.Errorf("role %s not assigned to user %s", roleID, userID)
}

// AssignPermissionToRole 为角色分配权限
func (rbac *RBAC) AssignPermissionToRole(roleID, permissionID string) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	role, exists := rbac.roles[roleID]
	if !exists {
		return fmt.Errorf("role %s not found", roleID)
	}

	_, exists := rbac.permissions[permissionID]
	if !exists {
		return fmt.Errorf("permission %s not found", permissionID)
	}

	// 检查权限是否已分配
	for _, existingPermission := range role.Permissions {
		if existingPermission == permissionID {
			return fmt.Errorf("permission %s already assigned to role %s", permissionID, roleID)
		}
	}

	role.Permissions = append(role.Permissions, permissionID)
	role.UpdatedAt = time.Now()

	// 为所有拥有该角色的用户添加权限
	for _, user := range rbac.users {
		for _, userRole := range user.Roles {
			if userRole == roleID {
				user.Permissions = append(user.Permissions, permissionID)
				user.UpdatedAt = time.Now()
			}
		}
	}

	return nil
}

// RemovePermissionFromRole 移除角色权限
func (rbac *RBAC) RemovePermissionFromRole(roleID, permissionID string) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	role, exists := rbac.roles[roleID]
	if !exists {
		return fmt.Errorf("role %s not found", roleID)
	}

	// 查找并移除权限
	for i, existingPermission := range role.Permissions {
		if existingPermission == permissionID {
			role.Permissions = append(role.Permissions[:i], role.Permissions[i+1:]...)
			role.UpdatedAt = time.Now()

			// 从所有拥有该角色的用户中移除权限
			for _, user := range rbac.users {
				for _, userRole := range user.Roles {
					if userRole == roleID {
						for j, userPermission := range user.Permissions {
							if userPermission == permissionID {
								user.Permissions = append(user.Permissions[:j], user.Permissions[j+1:]...)
								user.UpdatedAt = time.Now()
								break
							}
						}
					}
				}
			}

			return nil
		}
	}

	return fmt.Errorf("permission %s not assigned to role %s", permissionID, roleID)
}

// HasPermission 检查用户是否有权限
func (rbac *RBAC) HasPermission(userID, resource, action string) bool {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	user, exists := rbac.users[userID]
	if !exists {
		return false
	}

	// 检查用户直接权限
	for _, permissionID := range user.Permissions {
		permission, exists := rbac.permissions[permissionID]
		if exists && permission.Resource == resource && permission.Action == action {
			return true
		}
	}

	// 检查角色权限
	for _, roleID := range user.Roles {
		role, exists := rbac.roles[roleID]
		if exists {
			for _, permissionID := range role.Permissions {
				permission, exists := rbac.permissions[permissionID]
				if exists && permission.Resource == resource && permission.Action == action {
					return true
				}
			}
		}
	}

	return false
}

// HasRole 检查用户是否有角色
func (rbac *RBAC) HasRole(userID, roleID string) bool {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	user, exists := rbac.users[userID]
	if !exists {
		return false
	}

	for _, existingRole := range user.Roles {
		if existingRole == roleID {
			return true
		}
	}

	return false
}

// GetUserPermissions 获取用户权限
func (rbac *RBAC) GetUserPermissions(userID string) ([]*Permission, error) {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	user, exists := rbac.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	var permissions []*Permission
	for _, permissionID := range user.Permissions {
		permission, exists := rbac.permissions[permissionID]
		if exists {
			permissions = append(permissions, permission)
		}
	}

	return permissions, nil
}

// GetUserRoles 获取用户角色
func (rbac *RBAC) GetUserRoles(userID string) ([]*Role, error) {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	user, exists := rbac.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	var roles []*Role
	for _, roleID := range user.Roles {
		role, exists := rbac.roles[roleID]
		if exists {
			roles = append(roles, role)
		}
	}

	return roles, nil
}

// ListRoles 列出所有角色
func (rbac *RBAC) ListRoles() []*Role {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	var roles []*Role
	for _, role := range rbac.roles {
		roles = append(roles, role)
	}

	return roles
}

// ListPermissions 列出所有权限
func (rbac *RBAC) ListPermissions() []*Permission {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	var permissions []*Permission
	for _, permission := range rbac.permissions {
		permissions = append(permissions, permission)
	}

	return permissions
}

// ListUsers 列出所有用户
func (rbac *RBAC) ListUsers() []*User {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	var users []*User
	for _, user := range rbac.users {
		users = append(users, user)
	}

	return users
}
