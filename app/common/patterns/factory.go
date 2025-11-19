package patterns

import (
	"fmt"
	"sync"
)

// ObjectPool 对象池
type ObjectPool struct {
	factory   Factory
	pool      chan interface{}
	maxSize   int
	lifecycle LifecycleManager
	mutex     sync.RWMutex
	metrics   *PoolMetrics
}

// PoolMetrics 对象池指标
type PoolMetrics struct {
	Created   int64
	Destroyed int64
	Borrowed  int64
	Returned  int64
	PoolSize  int64
	MaxSize   int64
	mutex     sync.RWMutex
}

// LifecycleManager 生命周期管理器接口
type LifecycleManager interface {
	Create() interface{}
	Destroy(obj interface{})
	Validate(obj interface{}) bool
	Reset(obj interface{}) interface{}
}

// Factory 工厂接口
type Factory interface {
	Create() interface{}
	GetType() string
}

// ObjectPoolFactory 对象池工厂
type ObjectPoolFactory struct {
	pools map[string]*ObjectPool
	mutex sync.RWMutex
}

// NewObjectPoolFactory 创建对象池工厂
func NewObjectPoolFactory() *ObjectPoolFactory {
	return &ObjectPoolFactory{
		pools: make(map[string]*ObjectPool),
	}
}

// CreatePool 创建对象池
func (opf *ObjectPoolFactory) CreatePool(name string, factory Factory, maxSize int, lifecycle LifecycleManager) *ObjectPool {
	opf.mutex.Lock()
	defer opf.mutex.Unlock()

	pool := &ObjectPool{
		factory:   factory,
		pool:      make(chan interface{}, maxSize),
		maxSize:   maxSize,
		lifecycle: lifecycle,
		metrics:   &PoolMetrics{MaxSize: int64(maxSize)},
	}

	opf.pools[name] = pool
	return pool
}

// GetPool 获取对象池
func (opf *ObjectPoolFactory) GetPool(name string) (*ObjectPool, bool) {
	opf.mutex.RLock()
	defer opf.mutex.RUnlock()

	pool, exists := opf.pools[name]
	return pool, exists
}

// NewObjectPool 创建对象池
func NewObjectPool(factory Factory, maxSize int, lifecycle LifecycleManager) *ObjectPool {
	return &ObjectPool{
		factory:   factory,
		pool:      make(chan interface{}, maxSize),
		maxSize:   maxSize,
		lifecycle: lifecycle,
		metrics:   &PoolMetrics{MaxSize: int64(maxSize)},
	}
}

// Borrow 借用对象
func (op *ObjectPool) Borrow() (interface{}, error) {
	select {
	case obj := <-op.pool:
		// 验证对象
		if op.lifecycle != nil && !op.lifecycle.Validate(obj) {
			// 对象无效，销毁并创建新的
			op.lifecycle.Destroy(obj)
			obj = op.lifecycle.Create()
		}

		op.metrics.mutex.Lock()
		op.metrics.Borrowed++
		op.metrics.PoolSize--
		op.metrics.mutex.Unlock()

		return obj, nil
	default:
		// 池中没有可用对象，创建新的
		obj := op.lifecycle.Create()
		if obj == nil {
			return nil, fmt.Errorf("failed to create object")
		}

		op.metrics.mutex.Lock()
		op.metrics.Created++
		op.metrics.Borrowed++
		op.metrics.mutex.Unlock()

		return obj, nil
	}
}

// Return 归还对象
func (op *ObjectPool) Return(obj interface{}) error {
	if obj == nil {
		return fmt.Errorf("cannot return nil object")
	}

	// 重置对象
	if op.lifecycle != nil {
		obj = op.lifecycle.Reset(obj)
	}

	select {
	case op.pool <- obj:
		op.metrics.mutex.Lock()
		op.metrics.Returned++
		op.metrics.PoolSize++
		op.metrics.mutex.Unlock()
		return nil
	default:
		// 池已满，销毁对象
		if op.lifecycle != nil {
			op.lifecycle.Destroy(obj)
		}

		op.metrics.mutex.Lock()
		op.metrics.Destroyed++
		op.metrics.mutex.Unlock()

		return nil
	}
}

// GetMetrics 获取指标
func (op *ObjectPool) GetMetrics() *PoolMetrics {
	op.metrics.mutex.RLock()
	defer op.metrics.mutex.RUnlock()

	return &PoolMetrics{
		Created:   op.metrics.Created,
		Destroyed: op.metrics.Destroyed,
		Borrowed:  op.metrics.Borrowed,
		Returned:  op.metrics.Returned,
		PoolSize:  op.metrics.PoolSize,
		MaxSize:   op.metrics.MaxSize,
	}
}

// Close 关闭对象池
func (op *ObjectPool) Close() {
	close(op.pool)

	// 销毁池中的所有对象
	for obj := range op.pool {
		if op.lifecycle != nil {
			op.lifecycle.Destroy(obj)
		}
	}
}

// BaseLifecycleManager 基础生命周期管理器
type BaseLifecycleManager struct {
	createFunc   func() interface{}
	destroyFunc  func(interface{})
	validateFunc func(interface{}) bool
	resetFunc    func(interface{}) interface{}
}

// NewBaseLifecycleManager 创建基础生命周期管理器
func NewBaseLifecycleManager(
	createFunc func() interface{},
	destroyFunc func(interface{}),
	validateFunc func(interface{}) bool,
	resetFunc func(interface{}) interface{},
) *BaseLifecycleManager {
	return &BaseLifecycleManager{
		createFunc:   createFunc,
		destroyFunc:  destroyFunc,
		validateFunc: validateFunc,
		resetFunc:    resetFunc,
	}
}

// Create 创建对象
func (blm *BaseLifecycleManager) Create() interface{} {
	if blm.createFunc != nil {
		return blm.createFunc()
	}
	return nil
}

// Destroy 销毁对象
func (blm *BaseLifecycleManager) Destroy(obj interface{}) {
	if blm.destroyFunc != nil {
		blm.destroyFunc(obj)
	}
}

// Validate 验证对象
func (blm *BaseLifecycleManager) Validate(obj interface{}) bool {
	if blm.validateFunc != nil {
		return blm.validateFunc(obj)
	}
	return true
}

// Reset 重置对象
func (blm *BaseLifecycleManager) Reset(obj interface{}) interface{} {
	if blm.resetFunc != nil {
		return blm.resetFunc(obj)
	}
	return obj
}

// EnhancedFactory 增强工厂
type EnhancedFactory struct {
	factories   map[string]Factory
	objectPools map[string]*ObjectPool
	lifecycles  map[string]LifecycleManager
	mutex       sync.RWMutex
	metrics     *FactoryMetrics
}

// FactoryMetrics 工厂指标
type FactoryMetrics struct {
	ObjectsCreated   int64
	ObjectsDestroyed int64
	PoolHits         int64
	PoolMisses       int64
	mutex            sync.RWMutex
}

// NewEnhancedFactory 创建增强工厂
func NewEnhancedFactory() *EnhancedFactory {
	return &EnhancedFactory{
		factories:   make(map[string]Factory),
		objectPools: make(map[string]*ObjectPool),
		lifecycles:  make(map[string]LifecycleManager),
		metrics:     &FactoryMetrics{},
	}
}

// RegisterFactory 注册工厂
func (ef *EnhancedFactory) RegisterFactory(name string, factory Factory) {
	ef.mutex.Lock()
	defer ef.mutex.Unlock()
	ef.factories[name] = factory
}

// RegisterLifecycle 注册生命周期管理器
func (ef *EnhancedFactory) RegisterLifecycle(name string, lifecycle LifecycleManager) {
	ef.mutex.Lock()
	defer ef.mutex.Unlock()
	ef.lifecycles[name] = lifecycle
}

// CreateObjectPool 创建对象池
func (ef *EnhancedFactory) CreateObjectPool(name string, maxSize int) error {
	ef.mutex.Lock()
	defer ef.mutex.Unlock()

	factory, exists := ef.factories[name]
	if !exists {
		return fmt.Errorf("factory not found: %s", name)
	}

	lifecycle, exists := ef.lifecycles[name]
	if !exists {
		return fmt.Errorf("lifecycle manager not found: %s", name)
	}

	pool := NewObjectPool(factory, maxSize, lifecycle)
	ef.objectPools[name] = pool

	return nil
}

// Create 创建对象
func (ef *EnhancedFactory) Create(name string) (interface{}, error) {
	ef.mutex.RLock()
	pool, exists := ef.objectPools[name]
	ef.mutex.RUnlock()

	if exists {
		// 使用对象池
		obj, err := pool.Borrow()
		if err != nil {
			ef.metrics.mutex.Lock()
			ef.metrics.PoolMisses++
			ef.metrics.mutex.Unlock()
			return nil, err
		}

		ef.metrics.mutex.Lock()
		ef.metrics.PoolHits++
		ef.metrics.mutex.Unlock()

		return obj, nil
	}

	// 直接创建
	ef.mutex.RLock()
	factory, exists := ef.factories[name]
	ef.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("factory not found: %s", name)
	}

	obj := factory.Create()
	ef.metrics.mutex.Lock()
	ef.metrics.ObjectsCreated++
	ef.metrics.mutex.Unlock()

	return obj, nil
}

// Destroy 销毁对象
func (ef *EnhancedFactory) Destroy(name string, obj interface{}) error {
	ef.mutex.RLock()
	pool, exists := ef.objectPools[name]
	ef.mutex.RUnlock()

	if exists {
		// 归还到对象池
		return pool.Return(obj)
	}

	// 直接销毁
	ef.mutex.RLock()
	lifecycle, exists := ef.lifecycles[name]
	ef.mutex.RUnlock()

	if exists {
		lifecycle.Destroy(obj)
	}

	ef.metrics.mutex.Lock()
	ef.metrics.ObjectsDestroyed++
	ef.metrics.mutex.Unlock()

	return nil
}

// GetMetrics 获取指标
func (ef *EnhancedFactory) GetMetrics() *FactoryMetrics {
	ef.metrics.mutex.RLock()
	defer ef.metrics.mutex.RUnlock()

	return &FactoryMetrics{
		ObjectsCreated:   ef.metrics.ObjectsCreated,
		ObjectsDestroyed: ef.metrics.ObjectsDestroyed,
		PoolHits:         ef.metrics.PoolHits,
		PoolMisses:       ef.metrics.PoolMisses,
	}
}

// GetPoolMetrics 获取对象池指标
func (ef *EnhancedFactory) GetPoolMetrics(name string) (*PoolMetrics, error) {
	ef.mutex.RLock()
	defer ef.mutex.RUnlock()

	pool, exists := ef.objectPools[name]
	if !exists {
		return nil, fmt.Errorf("object pool not found: %s", name)
	}

	return pool.GetMetrics(), nil
}

// Close 关闭工厂
func (ef *EnhancedFactory) Close() {
	ef.mutex.Lock()
	defer ef.mutex.Unlock()

	for _, pool := range ef.objectPools {
		pool.Close()
	}
}

// SingletonFactory 单例工厂
type SingletonFactory struct {
	instance interface{}
	once     sync.Once
	factory  Factory
	mutex    sync.RWMutex
}

// NewSingletonFactory 创建单例工厂
func NewSingletonFactory(factory Factory) *SingletonFactory {
	return &SingletonFactory{
		factory: factory,
	}
}

// Create 创建单例
func (sf *SingletonFactory) Create() interface{} {
	sf.once.Do(func() {
		sf.instance = sf.factory.Create()
	})
	return sf.instance
}

// GetType 获取类型
func (sf *SingletonFactory) GetType() string {
	return sf.factory.GetType()
}

// GetInstance 获取实例
func (sf *SingletonFactory) GetInstance() interface{} {
	sf.mutex.RLock()
	defer sf.mutex.RUnlock()
	return sf.instance
}

// Reset 重置单例
func (sf *SingletonFactory) Reset() {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.once = sync.Once{}
	sf.instance = nil
}

// PrototypeFactory 原型工厂
type PrototypeFactory struct {
	prototype interface{}
	factory   Factory
	mutex     sync.RWMutex
}

// NewPrototypeFactory 创建原型工厂
func NewPrototypeFactory(factory Factory) *PrototypeFactory {
	return &PrototypeFactory{
		factory: factory,
	}
}

// Create 创建原型
func (pf *PrototypeFactory) Create() interface{} {
	pf.mutex.RLock()
	defer pf.mutex.RUnlock()

	if pf.prototype == nil {
		pf.prototype = pf.factory.Create()
	}

	// 这里应该实现深拷贝逻辑
	// 简化实现，直接返回原型
	return pf.prototype
}

// GetType 获取类型
func (pf *PrototypeFactory) GetType() string {
	return pf.factory.GetType()
}

// SetPrototype 设置原型
func (pf *PrototypeFactory) SetPrototype(prototype interface{}) {
	pf.mutex.Lock()
	defer pf.mutex.Unlock()
	pf.prototype = prototype
}

// FactoryRegistry 工厂注册表
type FactoryRegistry struct {
	factories map[string]Factory
	mutex     sync.RWMutex
}

// NewFactoryRegistry 创建工厂注册表
func NewFactoryRegistry() *FactoryRegistry {
	return &FactoryRegistry{
		factories: make(map[string]Factory),
	}
}

// Register 注册工厂
func (fr *FactoryRegistry) Register(name string, factory Factory) {
	fr.mutex.Lock()
	defer fr.mutex.Unlock()
	fr.factories[name] = factory
}

// Get 获取工厂
func (fr *FactoryRegistry) Get(name string) (Factory, error) {
	fr.mutex.RLock()
	defer fr.mutex.RUnlock()

	factory, exists := fr.factories[name]
	if !exists {
		return nil, fmt.Errorf("factory not found: %s", name)
	}

	return factory, nil
}

// List 列出所有工厂
func (fr *FactoryRegistry) List() []string {
	fr.mutex.RLock()
	defer fr.mutex.RUnlock()

	names := make([]string, 0, len(fr.factories))
	for name := range fr.factories {
		names = append(names, name)
	}

	return names
}

// Unregister 取消注册
func (fr *FactoryRegistry) Unregister(name string) {
	fr.mutex.Lock()
	defer fr.mutex.Unlock()
	delete(fr.factories, name)
}
