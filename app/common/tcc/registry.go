package tcc

import (
	"fmt"
	"sync"
)

// registryCenter 注册中心
type registryCenter struct {
	components map[string]TCCComponent
	mu         sync.RWMutex
}

// newRegistryCenter 创建新的注册中心
func newRegistryCenter() *registryCenter {
	return &registryCenter{
		components: make(map[string]TCCComponent),
	}
}

// register 注册组件
func (r *registryCenter) register(component TCCComponent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if component == nil {
		return fmt.Errorf("component cannot be nil")
	}

	componentID := component.ID()
	if componentID == "" {
		return fmt.Errorf("component ID cannot be empty")
	}

	if _, exists := r.components[componentID]; exists {
		return fmt.Errorf("component with ID %s already registered", componentID)
	}

	r.components[componentID] = component
	return nil
}

// getComponents 获取组件
func (r *registryCenter) getComponents(componentIDs ...string) ([]TCCComponent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	components := make([]TCCComponent, 0, len(componentIDs))
	for _, componentID := range componentIDs {
		component, exists := r.components[componentID]
		if !exists {
			return nil, fmt.Errorf("component with ID %s not found", componentID)
		}
		components = append(components, component)
	}

	return components, nil
}

// getAllComponents 获取所有组件
func (r *registryCenter) getAllComponents() []TCCComponent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	components := make([]TCCComponent, 0, len(r.components))
	for _, component := range r.components {
		components = append(components, component)
	}

	return components
}

// unregister 注销组件
func (r *registryCenter) unregister(componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[componentID]; !exists {
		return fmt.Errorf("component with ID %s not found", componentID)
	}

	delete(r.components, componentID)
	return nil
}

// isRegistered 检查组件是否已注册
func (r *registryCenter) isRegistered(componentID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.components[componentID]
	return exists
}

// getComponentCount 获取组件数量
func (r *registryCenter) getComponentCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.components)
}
