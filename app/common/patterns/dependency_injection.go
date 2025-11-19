package patterns

import (
	"fmt"
	"reflect"
	"sync"
)

// ServiceLifecycle 服务生命周期接口
type ServiceLifecycle interface {
	Initialize() error
	Start() error
	Stop() error
	HealthCheck() error
}

// ServiceScope 服务作用域
type ServiceScope int

const (
	ScopeSingleton ServiceScope = iota
	ScopeTransient
	ScopeScoped
)

// String 作用域字符串
func (s ServiceScope) String() string {
	switch s {
	case ScopeSingleton:
		return "singleton"
	case ScopeTransient:
		return "transient"
	case ScopeScoped:
		return "scoped"
	default:
		return "unknown"
	}
}

// ServiceDescriptor 服务描述符
type ServiceDescriptor struct {
	ServiceType    reflect.Type
	Implementation reflect.Type
	Factory        func() (interface{}, error)
	Instance       interface{}
	Scope          ServiceScope
	Lifecycle      ServiceLifecycle
	Initialized    bool
}

// DIContainer 依赖注入容器
type DIContainer struct {
	services   map[string]*ServiceDescriptor
	singletons map[string]interface{}
	scoped     map[string]interface{}
	mutex      sync.RWMutex
}

// NewDIContainer 创建依赖注入容器
func NewDIContainer() *DIContainer {
	return &DIContainer{
		services:   make(map[string]*ServiceDescriptor),
		singletons: make(map[string]interface{}),
		scoped:     make(map[string]interface{}),
	}
}

// RegisterSingleton 注册单例服务
func (dic *DIContainer) RegisterSingleton(serviceType reflect.Type, implementation reflect.Type) {
	dic.registerService(serviceType, implementation, ScopeSingleton, nil)
}

// RegisterTransient 注册瞬态服务
func (dic *DIContainer) RegisterTransient(serviceType reflect.Type, implementation reflect.Type) {
	dic.registerService(serviceType, implementation, ScopeTransient, nil)
}

// RegisterScoped 注册作用域服务
func (dic *DIContainer) RegisterScoped(serviceType reflect.Type, implementation reflect.Type) {
	dic.registerService(serviceType, implementation, ScopeScoped, nil)
}

// RegisterFactory 注册工厂服务
func (dic *DIContainer) RegisterFactory(serviceType reflect.Type, factory func() (interface{}, error), scope ServiceScope) {
	dic.registerService(serviceType, nil, scope, factory)
}

// RegisterInstance 注册实例
func (dic *DIContainer) RegisterInstance(serviceType reflect.Type, instance interface{}) {
	dic.mutex.Lock()
	defer dic.mutex.Unlock()

	serviceName := dic.getServiceName(serviceType)
	dic.services[serviceName] = &ServiceDescriptor{
		ServiceType:    serviceType,
		Implementation: nil,
		Factory:        nil,
		Instance:       instance,
		Scope:          ScopeSingleton,
		Initialized:    true,
	}
	dic.singletons[serviceName] = instance
}

// registerService 注册服务
func (dic *DIContainer) registerService(serviceType reflect.Type, implementation reflect.Type, scope ServiceScope, factory func() (interface{}, error)) {
	dic.mutex.Lock()
	defer dic.mutex.Unlock()

	serviceName := dic.getServiceName(serviceType)
	dic.services[serviceName] = &ServiceDescriptor{
		ServiceType:    serviceType,
		Implementation: implementation,
		Factory:        factory,
		Instance:       nil,
		Scope:          scope,
		Initialized:    false,
	}
}

// Get 获取服务
func (dic *DIContainer) Get(serviceType reflect.Type) (interface{}, error) {
	serviceName := dic.getServiceName(serviceType)

	dic.mutex.RLock()
	descriptor, exists := dic.services[serviceName]
	dic.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service not registered: %s", serviceName)
	}

	switch descriptor.Scope {
	case ScopeSingleton:
		return dic.getSingleton(serviceName, descriptor)
	case ScopeTransient:
		return dic.createInstance(descriptor)
	case ScopeScoped:
		return dic.getScoped(serviceName, descriptor)
	default:
		return nil, fmt.Errorf("unknown service scope: %s", descriptor.Scope)
	}
}

// GetByType 根据类型获取服务
func (dic *DIContainer) GetByType(serviceType interface{}) (interface{}, error) {
	serviceTypeValue := reflect.TypeOf(serviceType)
	if serviceTypeValue.Kind() == reflect.Ptr {
		serviceTypeValue = serviceTypeValue.Elem()
	}
	return dic.Get(serviceTypeValue)
}

// getSingleton 获取单例服务
func (dic *DIContainer) getSingleton(serviceName string, descriptor *ServiceDescriptor) (interface{}, error) {
	dic.mutex.RLock()
	instance, exists := dic.singletons[serviceName]
	dic.mutex.RUnlock()

	if exists {
		return instance, nil
	}

	dic.mutex.Lock()
	defer dic.mutex.Unlock()

	// 双重检查
	if instance, exists := dic.singletons[serviceName]; exists {
		return instance, nil
	}

	// 创建实例
	instance, err := dic.createInstance(descriptor)
	if err != nil {
		return nil, err
	}

	dic.singletons[serviceName] = instance
	return instance, nil
}

// getScoped 获取作用域服务
func (dic *DIContainer) getScoped(serviceName string, descriptor *ServiceDescriptor) (interface{}, error) {
	dic.mutex.RLock()
	instance, exists := dic.scoped[serviceName]
	dic.mutex.RUnlock()

	if exists {
		return instance, nil
	}

	dic.mutex.Lock()
	defer dic.mutex.Unlock()

	// 双重检查
	if instance, exists := dic.scoped[serviceName]; exists {
		return instance, nil
	}

	// 创建实例
	instance, err := dic.createInstance(descriptor)
	if err != nil {
		return nil, err
	}

	dic.scoped[serviceName] = instance
	return instance, nil
}

// createInstance 创建实例
func (dic *DIContainer) createInstance(descriptor *ServiceDescriptor) (interface{}, error) {
	if descriptor.Factory != nil {
		return descriptor.Factory()
	}

	if descriptor.Implementation == nil {
		return nil, fmt.Errorf("no implementation or factory provided")
	}

	// 创建实例
	instance := reflect.New(descriptor.Implementation).Interface()

	// 注入依赖
	if err := dic.injectDependencies(instance); err != nil {
		return nil, err
	}

	// 初始化生命周期
	if lifecycle, ok := instance.(ServiceLifecycle); ok {
		if err := lifecycle.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize service: %w", err)
		}
		descriptor.Lifecycle = lifecycle
		descriptor.Initialized = true
	}

	return instance, nil
}

// injectDependencies 注入依赖
func (dic *DIContainer) injectDependencies(instance interface{}) error {
	instanceValue := reflect.ValueOf(instance)
	if instanceValue.Kind() == reflect.Ptr {
		instanceValue = instanceValue.Elem()
	}

	instanceType := instanceValue.Type()

	// 遍历所有字段
	for i := 0; i < instanceType.NumField(); i++ {
		field := instanceValue.Field(i)
		fieldType := instanceType.Field(i)

		// 检查是否有inject标签
		if tag, ok := fieldType.Tag.Lookup("inject"); ok {
			if !field.CanSet() {
				continue
			}

			// 根据标签获取服务类型
			serviceType, err := dic.getServiceTypeFromTag(tag)
			if err != nil {
				return err
			}

			// 获取服务实例
			serviceInstance, err := dic.Get(serviceType)
			if err != nil {
				return err
			}

			// 设置字段值
			field.Set(reflect.ValueOf(serviceInstance))
		}
	}

	return nil
}

// getServiceTypeFromTag 从标签获取服务类型
func (dic *DIContainer) getServiceTypeFromTag(tag string) (reflect.Type, error) {
	// 这里可以根据标签解析服务类型
	// 简化实现，假设标签就是服务类型名称
	return reflect.TypeOf((*interface{})(nil)).Elem(), nil
}

// getServiceName 获取服务名称
func (dic *DIContainer) getServiceName(serviceType reflect.Type) string {
	return serviceType.String()
}

// StartAll 启动所有服务
func (dic *DIContainer) StartAll() error {
	dic.mutex.RLock()
	defer dic.mutex.RUnlock()

	for _, descriptor := range dic.services {
		if descriptor.Lifecycle != nil && descriptor.Initialized {
			if err := descriptor.Lifecycle.Start(); err != nil {
				return fmt.Errorf("failed to start service %s: %w", dic.getServiceName(descriptor.ServiceType), err)
			}
		}
	}

	return nil
}

// StopAll 停止所有服务
func (dic *DIContainer) StopAll() error {
	dic.mutex.RLock()
	defer dic.mutex.RUnlock()

	for _, descriptor := range dic.services {
		if descriptor.Lifecycle != nil && descriptor.Initialized {
			if err := descriptor.Lifecycle.Stop(); err != nil {
				return fmt.Errorf("failed to stop service %s: %w", dic.getServiceName(descriptor.ServiceType), err)
			}
		}
	}

	return nil
}

// HealthCheckAll 健康检查所有服务
func (dic *DIContainer) HealthCheckAll() map[string]error {
	dic.mutex.RLock()
	defer dic.mutex.RUnlock()

	results := make(map[string]error)

	for _, descriptor := range dic.services {
		if descriptor.Lifecycle != nil && descriptor.Initialized {
			serviceName := dic.getServiceName(descriptor.ServiceType)
			if err := descriptor.Lifecycle.HealthCheck(); err != nil {
				results[serviceName] = err
			} else {
				results[serviceName] = nil
			}
		}
	}

	return results
}

// ClearScoped 清除作用域服务
func (dic *DIContainer) ClearScoped() {
	dic.mutex.Lock()
	defer dic.mutex.Unlock()

	dic.scoped = make(map[string]interface{})
}

// ListServices 列出所有服务
func (dic *DIContainer) ListServices() []string {
	dic.mutex.RLock()
	defer dic.mutex.RUnlock()

	var services []string
	for serviceName := range dic.services {
		services = append(services, serviceName)
	}

	return services
}

// GetServiceInfo 获取服务信息
func (dic *DIContainer) GetServiceInfo(serviceType reflect.Type) (*ServiceDescriptor, error) {
	serviceName := dic.getServiceName(serviceType)

	dic.mutex.RLock()
	defer dic.mutex.RUnlock()

	descriptor, exists := dic.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service not registered: %s", serviceName)
	}

	return descriptor, nil
}

// ServiceBuilder 服务构建器
type ServiceBuilder struct {
	container   *DIContainer
	serviceType reflect.Type
}

// NewServiceBuilder 创建服务构建器
func NewServiceBuilder(container *DIContainer, serviceType reflect.Type) *ServiceBuilder {
	return &ServiceBuilder{
		container:   container,
		serviceType: serviceType,
	}
}

// AsSingleton 设置为单例
func (sb *ServiceBuilder) AsSingleton() *ServiceBuilder {
	sb.container.RegisterSingleton(sb.serviceType, sb.serviceType)
	return sb
}

// AsTransient 设置为瞬态
func (sb *ServiceBuilder) AsTransient() *ServiceBuilder {
	sb.container.RegisterTransient(sb.serviceType, sb.serviceType)
	return sb
}

// AsScoped 设置为作用域
func (sb *ServiceBuilder) AsScoped() *ServiceBuilder {
	sb.container.RegisterScoped(sb.serviceType, sb.serviceType)
	return sb
}

// WithFactory 设置工厂
func (sb *ServiceBuilder) WithFactory(factory func() (interface{}, error), scope ServiceScope) *ServiceBuilder {
	sb.container.RegisterFactory(sb.serviceType, factory, scope)
	return sb
}

// WithInstance 设置实例
func (sb *ServiceBuilder) WithInstance(instance interface{}) *ServiceBuilder {
	sb.container.RegisterInstance(sb.serviceType, instance)
	return sb
}

// Build 构建服务
func (sb *ServiceBuilder) Build() error {
	// 服务已经在注册时构建
	return nil
}
