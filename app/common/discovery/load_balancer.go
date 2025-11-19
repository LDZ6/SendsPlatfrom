package discovery

import (
	"math/rand"
	"sync"
	"time"
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Select(instances []*ServiceInstance) (*ServiceInstance, error)
	GetStrategy() string
}

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	current int
	mutex   sync.Mutex
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{}
}

// Select 选择服务实例
func (rr *RoundRobinLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	// 过滤健康的实例
	healthyInstances := make([]*ServiceInstance, 0)
	for _, instance := range instances {
		if instance.Health == "healthy" {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil, ErrNoHealthyInstances
	}

	rr.mutex.Lock()
	defer rr.mutex.Unlock()

	instance := healthyInstances[rr.current%len(healthyInstances)]
	rr.current++
	return instance, nil
}

// GetStrategy 获取策略名称
func (rr *RoundRobinLoadBalancer) GetStrategy() string {
	return "round_robin"
}

// RandomLoadBalancer 随机负载均衡器
type RandomLoadBalancer struct {
	rand *rand.Rand
}

// NewRandomLoadBalancer 创建随机负载均衡器
func NewRandomLoadBalancer() *RandomLoadBalancer {
	return &RandomLoadBalancer{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select 选择服务实例
func (r *RandomLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	// 过滤健康的实例
	healthyInstances := make([]*ServiceInstance, 0)
	for _, instance := range instances {
		if instance.Health == "healthy" {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil, ErrNoHealthyInstances
	}

	index := r.rand.Intn(len(healthyInstances))
	return healthyInstances[index], nil
}

// GetStrategy 获取策略名称
func (r *RandomLoadBalancer) GetStrategy() string {
	return "random"
}

// WeightedRoundRobinLoadBalancer 加权轮询负载均衡器
type WeightedRoundRobinLoadBalancer struct {
	current int
	weights []int
	mutex   sync.Mutex
}

// NewWeightedRoundRobinLoadBalancer 创建加权轮询负载均衡器
func NewWeightedRoundRobinLoadBalancer() *WeightedRoundRobinLoadBalancer {
	return &WeightedRoundRobinLoadBalancer{}
}

// Select 选择服务实例
func (wrr *WeightedRoundRobinLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	// 过滤健康的实例
	healthyInstances := make([]*ServiceInstance, 0)
	for _, instance := range instances {
		if instance.Health == "healthy" {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil, ErrNoHealthyInstances
	}

	wrr.mutex.Lock()
	defer wrr.mutex.Unlock()

	// 计算总权重
	totalWeight := 0
	for _, instance := range healthyInstances {
		totalWeight += instance.Weight
	}

	if totalWeight == 0 {
		// 如果所有权重为0，使用轮询
		instance := healthyInstances[wrr.current%len(healthyInstances)]
		wrr.current++
		return instance, nil
	}

	// 加权轮询算法
	currentWeight := wrr.current % totalWeight
	weightSum := 0

	for _, instance := range healthyInstances {
		weightSum += instance.Weight
		if currentWeight < weightSum {
			wrr.current++
			return instance, nil
		}
	}

	// 兜底：返回第一个实例
	return healthyInstances[0], nil
}

// GetStrategy 获取策略名称
func (wrr *WeightedRoundRobinLoadBalancer) GetStrategy() string {
	return "weighted_round_robin"
}

// LeastConnectionsLoadBalancer 最少连接负载均衡器
type LeastConnectionsLoadBalancer struct {
	connections map[string]int
	mutex       sync.RWMutex
}

// NewLeastConnectionsLoadBalancer 创建最少连接负载均衡器
func NewLeastConnectionsLoadBalancer() *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{
		connections: make(map[string]int),
	}
}

// Select 选择服务实例
func (lc *LeastConnectionsLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	// 过滤健康的实例
	healthyInstances := make([]*ServiceInstance, 0)
	for _, instance := range instances {
		if instance.Health == "healthy" {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil, ErrNoHealthyInstances
	}

	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	// 找到连接数最少的实例
	minConnections := int(^uint(0) >> 1) // 最大int值
	var selected *ServiceInstance

	for _, instance := range healthyInstances {
		connections := lc.connections[instance.ID]
		if connections < minConnections {
			minConnections = connections
			selected = instance
		}
	}

	return selected, nil
}

// IncrementConnections 增加连接数
func (lc *LeastConnectionsLoadBalancer) IncrementConnections(instanceID string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	lc.connections[instanceID]++
}

// DecrementConnections 减少连接数
func (lc *LeastConnectionsLoadBalancer) DecrementConnections(instanceID string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	if lc.connections[instanceID] > 0 {
		lc.connections[instanceID]--
	}
}

// GetStrategy 获取策略名称
func (lc *LeastConnectionsLoadBalancer) GetStrategy() string {
	return "least_connections"
}

// ConsistentHashLoadBalancer 一致性哈希负载均衡器
type ConsistentHashLoadBalancer struct {
	ring  map[uint32]*ServiceInstance
	keys  []uint32
	mutex sync.RWMutex
}

// NewConsistentHashLoadBalancer 创建一致性哈希负载均衡器
func NewConsistentHashLoadBalancer() *ConsistentHashLoadBalancer {
	return &ConsistentHashLoadBalancer{
		ring: make(map[uint32]*ServiceInstance),
	}
}

// Select 选择服务实例
func (ch *ConsistentHashLoadBalancer) Select(instances []*ServiceInstance, key string) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	// 过滤健康的实例
	healthyInstances := make([]*ServiceInstance, 0)
	for _, instance := range instances {
		if instance.Health == "healthy" {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil, ErrNoHealthyInstances
	}

	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	// 重新构建哈希环
	ch.buildRing(healthyInstances)

	// 计算key的哈希值
	hash := ch.hash(key)

	// 找到第一个大于等于hash的节点
	for _, ringKey := range ch.keys {
		if ringKey >= hash {
			return ch.ring[ringKey], nil
		}
	}

	// 如果没有找到，返回第一个节点（环形结构）
	return ch.ring[ch.keys[0]], nil
}

// buildRing 构建哈希环
func (ch *ConsistentHashLoadBalancer) buildRing(instances []*ServiceInstance) {
	ch.ring = make(map[uint32]*ServiceInstance)
	ch.keys = make([]uint32, 0)

	for _, instance := range instances {
		// 为每个实例创建多个虚拟节点
		for i := 0; i < 150; i++ {
			virtualKey := instance.ID + ":" + string(rune(i))
			hash := ch.hash(virtualKey)
			ch.ring[hash] = instance
			ch.keys = append(ch.keys, hash)
		}
	}

	// 排序
	for i := 0; i < len(ch.keys)-1; i++ {
		for j := i + 1; j < len(ch.keys); j++ {
			if ch.keys[i] > ch.keys[j] {
				ch.keys[i], ch.keys[j] = ch.keys[j], ch.keys[i]
			}
		}
	}
}

// hash 计算哈希值
func (ch *ConsistentHashLoadBalancer) hash(key string) uint32 {
	hash := uint32(0)
	for _, c := range key {
		hash = hash*31 + uint32(c)
	}
	return hash
}

// GetStrategy 获取策略名称
func (ch *ConsistentHashLoadBalancer) GetStrategy() string {
	return "consistent_hash"
}

// LoadBalancerFactory 负载均衡器工厂
type LoadBalancerFactory struct {
	strategies map[string]func() LoadBalancer
}

// NewLoadBalancerFactory 创建负载均衡器工厂
func NewLoadBalancerFactory() *LoadBalancerFactory {
	factory := &LoadBalancerFactory{
		strategies: make(map[string]func() LoadBalancer),
	}

	// 注册默认策略
	factory.RegisterStrategy("round_robin", func() LoadBalancer {
		return NewRoundRobinLoadBalancer()
	})
	factory.RegisterStrategy("random", func() LoadBalancer {
		return NewRandomLoadBalancer()
	})
	factory.RegisterStrategy("weighted_round_robin", func() LoadBalancer {
		return NewWeightedRoundRobinLoadBalancer()
	})
	factory.RegisterStrategy("least_connections", func() LoadBalancer {
		return NewLeastConnectionsLoadBalancer()
	})
	factory.RegisterStrategy("consistent_hash", func() LoadBalancer {
		return NewConsistentHashLoadBalancer()
	})

	return factory
}

// RegisterStrategy 注册策略
func (f *LoadBalancerFactory) RegisterStrategy(name string, creator func() LoadBalancer) {
	f.strategies[name] = creator
}

// CreateLoadBalancer 创建负载均衡器
func (f *LoadBalancerFactory) CreateLoadBalancer(strategy string) (LoadBalancer, error) {
	creator, exists := f.strategies[strategy]
	if !exists {
		return nil, ErrUnknownStrategy
	}
	return creator(), nil
}

// GetSupportedStrategies 获取支持的策略
func (f *LoadBalancerFactory) GetSupportedStrategies() []string {
	strategies := make([]string, 0, len(f.strategies))
	for strategy := range f.strategies {
		strategies = append(strategies, strategy)
	}
	return strategies
}

// 错误定义
var (
	ErrNoInstances        = &LoadBalancerError{message: "no service instances available"}
	ErrNoHealthyInstances = &LoadBalancerError{message: "no healthy service instances available"}
	ErrUnknownStrategy    = &LoadBalancerError{message: "unknown load balancing strategy"}
)

// LoadBalancerError 负载均衡器错误
type LoadBalancerError struct {
	message string
}

func (e *LoadBalancerError) Error() string {
	return e.message
}
