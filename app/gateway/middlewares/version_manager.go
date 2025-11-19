package middlewares

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// VersionInfo 版本信息
type VersionInfo struct {
	Version     string
	Deprecated  bool
	SunsetDate  *time.Time
	Description string
	Changes     []string
}

// VersionManager 版本管理器
type VersionManager struct {
	versions     map[string]VersionInfo
	deprecations map[string]time.Time
	mutex        sync.RWMutex
}

// NewVersionManager 创建版本管理器
func NewVersionManager() *VersionManager {
	vm := &VersionManager{
		versions:     make(map[string]VersionInfo),
		deprecations: make(map[string]time.Time),
	}

	// 注册默认版本
	vm.registerDefaultVersions()
	return vm
}

// registerDefaultVersions 注册默认版本
func (vm *VersionManager) registerDefaultVersions() {
	vm.RegisterVersion("v1", VersionInfo{
		Version:     "v1",
		Deprecated:  false,
		Description: "Initial API version",
		Changes:     []string{"Initial release"},
	})

	vm.RegisterVersion("v2", VersionInfo{
		Version:     "v2",
		Deprecated:  false,
		Description: "Enhanced API version with improved features",
		Changes: []string{
			"Added pagination support",
			"Improved error handling",
			"Enhanced authentication",
		},
	})
}

// RegisterVersion 注册版本
func (vm *VersionManager) RegisterVersion(version string, info VersionInfo) {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	vm.versions[version] = info
}

// GetVersion 获取版本信息
func (vm *VersionManager) GetVersion(version string) (VersionInfo, bool) {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()
	info, exists := vm.versions[version]
	return info, exists
}

// ListVersions 列出所有版本
func (vm *VersionManager) ListVersions() map[string]VersionInfo {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	versions := make(map[string]VersionInfo)
	for k, v := range vm.versions {
		versions[k] = v
	}
	return versions
}

// DeprecateVersion 废弃版本
func (vm *VersionManager) DeprecateVersion(version string, sunsetDate time.Time) {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	if info, exists := vm.versions[version]; exists {
		info.Deprecated = true
		info.SunsetDate = &sunsetDate
		vm.versions[version] = info
		vm.deprecations[version] = sunsetDate
	}
}

// IsVersionDeprecated 检查版本是否已废弃
func (vm *VersionManager) IsVersionDeprecated(version string) bool {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	info, exists := vm.versions[version]
	return exists && info.Deprecated
}

// GetSunsetDate 获取废弃日期
func (vm *VersionManager) GetSunsetDate(version string) (time.Time, bool) {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	sunset, exists := vm.deprecations[version]
	return sunset, exists
}

// VersionMiddleware 版本中间件
func VersionMiddleware(vm *VersionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从URL路径中提取版本
		version := extractVersionFromPath(c.Request.URL.Path)

		if version == "" {
			// 默认使用最新版本
			version = "v2"
		}

		// 检查版本是否存在
		info, exists := vm.GetVersion(version)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"success":   false,
				"errorCode": 4040,
				"message":   "API version not found",
				"details":   fmt.Sprintf("Version %s is not supported", version),
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		// 检查版本是否已废弃
		if info.Deprecated {
			headers := c.Writer.Header()
			headers.Set("X-API-Version", version)
			headers.Set("X-API-Deprecated", "true")

			if info.SunsetDate != nil {
				headers.Set("X-API-Sunset", info.SunsetDate.Format(time.RFC3339))
			}

			// 如果是废弃版本，添加警告头
			c.Header("Warning", fmt.Sprintf("299 - \"Version %s is deprecated\"", version))
		}

		// 将版本信息存储到上下文中
		c.Set("api_version", version)
		c.Set("version_info", info)

		c.Next()
	}
}

// extractVersionFromPath 从路径中提取版本
func extractVersionFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && strings.HasPrefix(parts[0], "v") {
		return parts[0]
	}
	return ""
}

// VersionHeaderMiddleware 版本头中间件
func VersionHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 添加版本相关的响应头
		c.Header("X-API-Version", "v2")
		c.Header("X-API-Supported-Versions", "v1,v2")

		c.Next()
	}
}

// VersionCheckHandler 版本检查处理器
func VersionCheckHandler(vm *VersionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		versions := vm.ListVersions()

		response := gin.H{
			"success": true,
			"data": gin.H{
				"versions": versions,
				"current":  "v2",
				"latest":   "v2",
			},
			"timestamp": time.Now().Unix(),
		}

		c.JSON(http.StatusOK, response)
	}
}

// RequestResponseTransformMiddleware 请求响应转换中间件
func RequestResponseTransformMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 请求转换
		transformRequest(c)

		// 处理请求
		c.Next()

		// 响应转换
		transformResponse(c)
	}
}

// transformRequest 转换请求
func transformRequest(c *gin.Context) {
	// 根据版本进行请求转换
	version, exists := c.Get("api_version")
	if !exists {
		return
	}

	switch version {
	case "v1":
		// v1版本的请求转换逻辑
		transformV1Request(c)
	case "v2":
		// v2版本的请求转换逻辑
		transformV2Request(c)
	}
}

// transformResponse 转换响应
func transformResponse(c *gin.Context) {
	// 根据版本进行响应转换
	version, exists := c.Get("api_version")
	if !exists {
		return
	}

	switch version {
	case "v1":
		// v1版本的响应转换逻辑
		transformV1Response(c)
	case "v2":
		// v2版本的响应转换逻辑
		transformV2Response(c)
	}
}

// transformV1Request v1请求转换
func transformV1Request(c *gin.Context) {
	// v1特定的请求转换逻辑
	// 例如：字段名转换、数据格式转换等
}

// transformV2Request v2请求转换
func transformV2Request(c *gin.Context) {
	// v2特定的请求转换逻辑
	// 例如：字段名转换、数据格式转换等
}

// transformV1Response v1响应转换
func transformV1Response(c *gin.Context) {
	// v1特定的响应转换逻辑
	// 例如：字段名转换、数据格式转换等
}

// transformV2Response v2响应转换
func transformV2Response(c *gin.Context) {
	// v2特定的响应转换逻辑
	// 例如：字段名转换、数据格式转换等
}

// VersionContext 版本上下文
type VersionContext struct {
	Version    string
	Info       VersionInfo
	Transform  bool
	Deprecated bool
	SunsetDate *time.Time
}

// GetVersionContext 获取版本上下文
func GetVersionContext(c *gin.Context) *VersionContext {
	version, _ := c.Get("api_version")
	info, _ := c.Get("version_info")

	versionStr, _ := version.(string)
	versionInfo, _ := info.(VersionInfo)

	return &VersionContext{
		Version:    versionStr,
		Info:       versionInfo,
		Transform:  true,
		Deprecated: versionInfo.Deprecated,
		SunsetDate: versionInfo.SunsetDate,
	}
}
