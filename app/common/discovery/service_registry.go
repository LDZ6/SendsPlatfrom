package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
)

// ServiceInstance 服务实例
type ServiceInstance struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Version  string            `json:"version"`
	Health   string            `json:"health"`
	Metadata map[string]string `json:"metadata"`
	LastSeen time.Time         `json:"lastSeen"`
	Weight   int               `json:"weight"`
	Tags     []string          `json:"tags"`
}

// ServiceRegistry 服务注册中心
type ServiceRegistry struct {
	client   *clientv3.Client
	services map[string][]*ServiceInstance
	watchers map[string][]ServiceWatcher
	mutex    sync.RWMutex
	leaseID  clientv3.LeaseID
	ttl      int64
	stopCh   chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

// ServiceWatcher 服务观察者
type ServiceWatcher interface {
	OnServiceAdded(service *ServiceInstance)
	OnServiceRemoved(service *ServiceInstance)
	OnServiceUpdated(service *ServiceInstance)
}

// NewServiceRegistry 创建服务注册中心
func NewServiceRegistry(client *clientv3.Client, ttl int64) *ServiceRegistry {
	ctx, cancel := context.WithCancel(context.Background())
	return &ServiceRegistry{
		client:   client,
		services: make(map[string][]*ServiceInstance),
		watchers: make(map[string][]ServiceWatcher),
		ttl:      ttl,
		stopCh:   make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Register 注册服务实例
func (sr *ServiceRegistry) Register(service *ServiceInstance) error {
	// 创建租约
	lease, err := sr.client.Grant(sr.ctx, sr.ttl)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	sr.leaseID = lease.ID

	// 构建服务键
	key := fmt.Sprintf("/services/%s/%s", service.Name, service.ID)

	// 序列化服务实例
	data, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service: %w", err)
	}

	// 注册服务
	_, err = sr.client.Put(sr.ctx, key, string(data), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// 保持租约活跃
	go sr.keepAlive()

	// 更新本地缓存
	sr.mutex.Lock()
	sr.services[service.Name] = append(sr.services[service.Name], service)
	sr.mutex.Unlock()

	// 通知观察者
	sr.notifyWatchers(service.Name, "added", service)

	return nil
}

// Deregister 注销服务实例
func (sr *ServiceRegistry) Deregister(serviceName, serviceID string) error {
	key := fmt.Sprintf("/services/%s/%s", serviceName, serviceID)

	_, err := sr.client.Delete(sr.ctx, key)
	if err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	// 更新本地缓存
	sr.mutex.Lock()
	if services, exists := sr.services[serviceName]; exists {
		for i, service := range services {
			if service.ID == serviceID {
				sr.services[serviceName] = append(services[:i], services[i+1:]...)
				break
			}
		}
	}
	sr.mutex.Unlock()

	// 通知观察者
	service := &ServiceInstance{ID: serviceID, Name: serviceName}
	sr.notifyWatchers(serviceName, "removed", service)

	return nil
}

// Discover 发现服务实例
func (sr *ServiceRegistry) Discover(serviceName string) ([]*ServiceInstance, error) {
	sr.mutex.RLock()
	services, exists := sr.services[serviceName]
	sr.mutex.RUnlock()

	if !exists {
		// 从etcd获取服务
		return sr.fetchFromEtcd(serviceName)
	}

	return services, nil
}

// Watch 监听服务变化
func (sr *ServiceRegistry) Watch(serviceName string, watcher ServiceWatcher) error {
	sr.mutex.Lock()
	sr.watchers[serviceName] = append(sr.watchers[serviceName], watcher)
	sr.mutex.Unlock()

	// 启动监听
	go sr.watchService(serviceName)

	return nil
}

// fetchFromEtcd 从etcd获取服务
func (sr *ServiceRegistry) fetchFromEtcd(serviceName string) ([]*ServiceInstance, error) {
	key := fmt.Sprintf("/services/%s/", serviceName)

	resp, err := sr.client.Get(sr.ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch services: %w", err)
	}

	var services []*ServiceInstance
	for _, kv := range resp.Kvs {
		var service ServiceInstance
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			continue
		}
		services = append(services, &service)
	}

	// 更新本地缓存
	sr.mutex.Lock()
	sr.services[serviceName] = services
	sr.mutex.Unlock()

	return services, nil
}

// watchService 监听服务变化
func (sr *ServiceRegistry) watchService(serviceName string) {
	key := fmt.Sprintf("/services/%s/", serviceName)

	watchCh := sr.client.Watch(sr.ctx, key, clientv3.WithPrefix())

	for watchResp := range watchCh {
		for _, event := range watchResp.Events {
			var service ServiceInstance
			if err := json.Unmarshal(event.Kv.Value, &service); err != nil {
				continue
			}

			switch event.Type {
			case clientv3.EventTypePut:
				sr.updateServiceCache(serviceName, &service)
				sr.notifyWatchers(serviceName, "added", &service)
			case clientv3.EventTypeDelete:
				sr.removeServiceCache(serviceName, &service)
				sr.notifyWatchers(serviceName, "removed", &service)
			}
		}
	}
}

// updateServiceCache 更新服务缓存
func (sr *ServiceRegistry) updateServiceCache(serviceName string, service *ServiceInstance) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	services := sr.services[serviceName]
	for i, s := range services {
		if s.ID == service.ID {
			services[i] = service
			return
		}
	}
	sr.services[serviceName] = append(services, service)
}

// removeServiceCache 移除服务缓存
func (sr *ServiceRegistry) removeServiceCache(serviceName string, service *ServiceInstance) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	services := sr.services[serviceName]
	for i, s := range services {
		if s.ID == service.ID {
			sr.services[serviceName] = append(services[:i], services[i+1:]...)
			return
		}
	}
}

// notifyWatchers 通知观察者
func (sr *ServiceRegistry) notifyWatchers(serviceName, action string, service *ServiceInstance) {
	sr.mutex.RLock()
	watchers := sr.watchers[serviceName]
	sr.mutex.RUnlock()

	for _, watcher := range watchers {
		switch action {
		case "added":
			watcher.OnServiceAdded(service)
		case "removed":
			watcher.OnServiceRemoved(service)
		case "updated":
			watcher.OnServiceUpdated(service)
		}
	}
}

// keepAlive 保持租约活跃
func (sr *ServiceRegistry) keepAlive() {
	ticker := time.NewTicker(time.Duration(sr.ttl/2) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := sr.client.KeepAliveOnce(sr.ctx, sr.leaseID)
			if err != nil {
				// 租约失效，尝试重新注册
				return
			}
		case <-sr.stopCh:
			return
		case <-sr.ctx.Done():
			return
		}
	}
}

// Stop 停止服务注册中心
func (sr *ServiceRegistry) Stop() {
	sr.cancel()
	close(sr.stopCh)

	// 撤销租约
	if sr.leaseID != 0 {
		sr.client.Revoke(sr.ctx, sr.leaseID)
	}
}

// GetServiceStats 获取服务统计信息
func (sr *ServiceRegistry) GetServiceStats() map[string]interface{} {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	stats := make(map[string]interface{})
	for serviceName, services := range sr.services {
		stats[serviceName] = map[string]interface{}{
			"count":     len(services),
			"instances": services,
		}
	}

	return stats
}
