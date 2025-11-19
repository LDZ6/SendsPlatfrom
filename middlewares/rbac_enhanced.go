package middlewares

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"platform/app/common/security"
	"platform/utils"

	"github.com/gin-gonic/gin"
)

// RBACMiddleware RBAC中间件
type RBACMiddleware struct {
	rbac        *security.RBAC
	permissions map[string][]string // 路由到权限的映射
}

// NewRBACMiddleware 创建RBAC中间件
func NewRBACMiddleware(rbac *security.RBAC) *RBACMiddleware {
	return &RBACMiddleware{
		rbac:        rbac,
		permissions: make(map[string][]string),
	}
}

// AddPermission 添加路由权限
func (rm *RBACMiddleware) AddPermission(route, method, permission string) {
	key := fmt.Sprintf("%s:%s", method, route)
	rm.permissions[key] = append(rm.permissions[key], permission)
}

// PermissionCheckMiddleware 权限检查中间件
func (rm *RBACMiddleware) PermissionCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			RespondWithUnauthorized(c, "用户未认证")
			c.Abort()
			return
		}

		// 获取路由权限
		key := fmt.Sprintf("%s:%s", c.Request.Method, c.FullPath())
		requiredPermissions, exists := rm.permissions[key]
		if !exists {
			// 没有权限要求，直接通过
			c.Next()
			return
		}

		// 检查用户权限
		userIDStr := fmt.Sprintf("%v", userID)
		hasPermission := false
		for _, permission := range requiredPermissions {
			parts := strings.Split(permission, ":")
			if len(parts) == 2 {
				resource, action := parts[0], parts[1]
				if rm.rbac.HasPermission(userIDStr, resource, action) {
					hasPermission = true
					break
				}
			}
		}

		if !hasPermission {
			RespondWithForbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RoleCheckMiddleware 角色检查中间件
func (rm *RBACMiddleware) RoleCheckMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			RespondWithUnauthorized(c, "用户未认证")
			c.Abort()
			return
		}

		// 检查用户角色
		userIDStr := fmt.Sprintf("%v", userID)
		hasRole := false
		for _, role := range requiredRoles {
			if rm.rbac.HasRole(userIDStr, role) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			RespondWithForbidden(c, "角色权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuditMiddleware 审计中间件
func (rm *RBACMiddleware) AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 记录请求信息
		auditLog := &AuditLog{
			UserID:    c.GetString("user_id"),
			IP:        c.ClientIP(),
			Method:    c.Request.Method,
			Path:      c.FullPath(),
			UserAgent: c.GetHeader("User-Agent"),
			Timestamp: start,
		}

		// 处理请求
		c.Next()

		// 记录响应信息
		auditLog.StatusCode = c.Writer.Status()
		auditLog.Duration = time.Since(start)
		auditLog.Success = c.Writer.Status() < 400

		// 记录审计日志
		rm.recordAuditLog(auditLog)
	}
}

// AuditLog 审计日志
type AuditLog struct {
	UserID     string
	IP         string
	Method     string
	Path       string
	UserAgent  string
	StatusCode int
	Duration   time.Duration
	Success    bool
	Timestamp  time.Time
}

// recordAuditLog 记录审计日志
func (rm *RBACMiddleware) recordAuditLog(log *AuditLog) {
	// 这里应该将审计日志写入数据库或日志文件
	// 示例实现
	fmt.Printf("Audit Log: %+v\n", log)
}

// ResourceBasedMiddleware 基于资源的中间件
func (rm *RBACMiddleware) ResourceBasedMiddleware(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			RespondWithUnauthorized(c, "用户未认证")
			c.Abort()
			return
		}

		// 检查资源权限
		userIDStr := fmt.Sprintf("%v", userID)
		if !rm.rbac.HasPermission(userIDStr, resource, action) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("没有%s资源的%s权限", resource, action),
				"code":  utils.ErrCodeForbidden,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// DataScopeMiddleware 数据范围中间件
func (rm *RBACMiddleware) DataScopeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			RespondWithUnauthorized(c, "用户未认证")
			c.Abort()
			return
		}

		// 获取用户数据范围
		userIDStr := fmt.Sprintf("%v", userID)
		dataScope := rm.getUserDataScope(userIDStr)

		// 设置数据范围到上下文
		c.Set("data_scope", dataScope)
		c.Next()
	}
}

// getUserDataScope 获取用户数据范围
func (rm *RBACMiddleware) getUserDataScope(userID string) string {
	// 根据用户角色确定数据范围
	// 1: 全部数据权限
	// 2: 自定义数据权限
	// 3: 本部门数据权限
	// 4: 本部门及以下数据权限
	// 5: 仅本人数据权限

	roles, err := rm.rbac.GetUserRoles(userID)
	if err != nil {
		return "5" // 默认仅本人数据权限
	}

	// 检查是否有全部数据权限
	for _, role := range roles {
		if role.Name == "admin" || role.Name == "super_admin" {
			return "1"
		}
	}

	// 检查是否有部门数据权限
	for _, role := range roles {
		if role.Name == "dept_manager" {
			return "3"
		}
	}

	return "5" // 默认仅本人数据权限
}

// PermissionCache 权限缓存
type PermissionCache struct {
	cache map[string]map[string]bool
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewPermissionCache 创建权限缓存
func NewPermissionCache(ttl time.Duration) *PermissionCache {
	pc := &PermissionCache{
		cache: make(map[string]map[string]bool),
		ttl:   ttl,
	}

	// 启动清理协程
	go pc.cleanup()

	return pc
}

// HasPermission 检查权限
func (pc *PermissionCache) HasPermission(userID, resource, action string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	key := fmt.Sprintf("%s:%s:%s", userID, resource, action)
	permissions, exists := pc.cache[key]
	if !exists {
		return false
	}

	permission, exists := permissions[action]
	return exists && permission
}

// SetPermission 设置权限
func (pc *PermissionCache) SetPermission(userID, resource, action string, hasPermission bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", userID, resource, action)
	if pc.cache[key] == nil {
		pc.cache[key] = make(map[string]bool)
	}
	pc.cache[key][action] = hasPermission
}

// cleanup 清理过期缓存
func (pc *PermissionCache) cleanup() {
	ticker := time.NewTicker(pc.ttl)
	defer ticker.Stop()

	for range ticker.C {
		pc.mu.Lock()
		pc.cache = make(map[string]map[string]bool)
		pc.mu.Unlock()
	}
}

// ContextKey 上下文键
type ContextKey string

const (
	UserIDKey     ContextKey = "user_id"
	RoleKey       ContextKey = "role"
	PermissionKey ContextKey = "permission"
	DataScopeKey  ContextKey = "data_scope"
)

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(string(UserIDKey))
	if !exists {
		return "", false
	}
	return fmt.Sprintf("%v", userID), true
}

// GetRole 从上下文获取角色
func GetRole(c *gin.Context) (string, bool) {
	role, exists := c.Get(string(RoleKey))
	if !exists {
		return "", false
	}
	return fmt.Sprintf("%v", role), true
}

// GetPermission 从上下文获取权限
func GetPermission(c *gin.Context) (string, bool) {
	permission, exists := c.Get(string(PermissionKey))
	if !exists {
		return "", false
	}
	return fmt.Sprintf("%v", permission), true
}

// GetDataScope 从上下文获取数据范围
func GetDataScope(c *gin.Context) (string, bool) {
	dataScope, exists := c.Get(string(DataScopeKey))
	if !exists {
		return "", false
	}
	return fmt.Sprintf("%v", dataScope), true
}

// SetUserContext 设置用户上下文
func SetUserContext(c *gin.Context, userID, role string) {
	c.Set(string(UserIDKey), userID)
	c.Set(string(RoleKey), role)
}

// SetPermissionContext 设置权限上下文
func SetPermissionContext(c *gin.Context, permission string) {
	c.Set(string(PermissionKey), permission)
}

// SetDataScopeContext 设置数据范围上下文
func SetDataScopeContext(c *gin.Context, dataScope string) {
	c.Set(string(DataScopeKey), dataScope)
}
