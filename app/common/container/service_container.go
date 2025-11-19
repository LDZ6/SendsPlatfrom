package container

import (
	"sync"
	"time"

	"platform/app/boBing/service"
	"platform/app/gateway/rpc"
	"platform/config"
	"platform/utils"
)

// ServiceContainer 服务容器
type ServiceContainer struct {
	// 数据库管理器
	dbManager *DatabaseManager

	// 缓存管理器
	cacheManager *CacheManager

	// 服务实例
	boBingService *service.BoBingSrv
	userService   interface{} // 用户服务接口
	gameService   interface{} // 游戏服务接口

	// RPC连接
	rpcConnections map[string]interface{}

	// 配置
	config *config.Config

	// 互斥锁
	mutex sync.RWMutex

	// 启动时间
	startTime time.Time
}

var (
	container *ServiceContainer
	once      sync.Once
)

// GetServiceContainer 获取服务容器单例
func GetServiceContainer() *ServiceContainer {
	once.Do(func() {
		container = &ServiceContainer{
			rpcConnections: make(map[string]interface{}),
			startTime:      time.Now(),
		}
		container.init()
	})
	return container
}

// init 初始化容器
func (sc *ServiceContainer) init() {
	sc.config = config.Conf
	sc.dbManager = NewDatabaseManager()
	sc.cacheManager = NewCacheManager()
}

// GetDatabaseManager 获取数据库管理器
func (sc *ServiceContainer) GetDatabaseManager() *DatabaseManager {
	return sc.dbManager
}

// GetCacheManager 获取缓存管理器
func (sc *ServiceContainer) GetCacheManager() *CacheManager {
	return sc.cacheManager
}

// GetBoBingService 获取博饼服务
func (sc *ServiceContainer) GetBoBingService() *service.BoBingSrv {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if sc.boBingService == nil {
		sc.boBingService = service.GetBoBingSrv()
	}
	return sc.boBingService
}

// GetUserService 获取用户服务
func (sc *ServiceContainer) GetUserService() interface{} {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if sc.userService == nil {
		// TODO: 实现用户服务
		sc.userService = nil
	}
	return sc.userService
}

// GetGameService 获取游戏服务
func (sc *ServiceContainer) GetGameService() interface{} {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if sc.gameService == nil {
		// TODO: 实现游戏服务
		sc.gameService = nil
	}
	return sc.gameService
}

// GetRPCConnection 获取RPC连接
func (sc *ServiceContainer) GetRPCConnection(serviceName string) (interface{}, error) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	if conn, exists := sc.rpcConnections[serviceName]; exists {
		return conn, nil
	}

	// 如果连接不存在，创建新连接
	sc.mutex.RUnlock()
	sc.mutex.Lock()
	defer sc.mutex.RUnlock()

	// 双重检查
	if conn, exists := sc.rpcConnections[serviceName]; exists {
		return conn, nil
	}

	// 创建新连接
	conn, err := rpc.GetConnection(serviceName)
	if err != nil {
		return nil, err
	}

	sc.rpcConnections[serviceName] = conn
	return conn, nil
}

// GetConfig 获取配置
func (sc *ServiceContainer) GetConfig() *config.Config {
	return sc.config
}

// GetUptime 获取运行时间
func (sc *ServiceContainer) GetUptime() time.Duration {
	return time.Since(sc.startTime)
}

// Shutdown 关闭容器
func (sc *ServiceContainer) Shutdown() error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// 关闭RPC连接
	for name, conn := range sc.rpcConnections {
		if closer, ok := conn.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				utils.GetLogger().Errorf("关闭RPC连接 %s 失败: %v", name, err)
			}
		}
	}

	// 关闭数据库连接
	if sc.dbManager != nil {
		sc.dbManager.Close()
	}

	// 关闭缓存连接
	if sc.cacheManager != nil {
		sc.cacheManager.Close()
	}

	return nil
}

// HealthCheck 健康检查
func (sc *ServiceContainer) HealthCheck() map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"uptime":    sc.GetUptime().Seconds(),
		"timestamp": time.Now().Unix(),
	}

	// 检查数据库连接
	if sc.dbManager != nil {
		dbHealth := sc.dbManager.HealthCheck()
		health["database"] = dbHealth
	}

	// 检查缓存连接
	if sc.cacheManager != nil {
		cacheHealth := sc.cacheManager.HealthCheck()
		health["cache"] = cacheHealth
	}

	// 检查RPC连接
	rpcHealth := make(map[string]string)
	for name, conn := range sc.rpcConnections {
		if checker, ok := conn.(interface{ HealthCheck() error }); ok {
			if err := checker.HealthCheck(); err != nil {
				rpcHealth[name] = "unhealthy"
				health["status"] = "degraded"
			} else {
				rpcHealth[name] = "healthy"
			}
		} else {
			rpcHealth[name] = "unknown"
		}
	}
	health["rpc"] = rpcHealth

	return health
}
