package registry

import (
	"sync"

	"github.com/vaishak/tf-migrate/internal/interfaces"
)

type StrategyRegistry struct {
	strategies map[string]interfaces.ResourceTransformer
	order      []string
	mu         sync.RWMutex
}

func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{
		strategies: make(map[string]interfaces.ResourceTransformer),
		order:      make([]string, 0),
	}
}

func (r *StrategyRegistry) Register(strategy interfaces.ResourceTransformer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	resourceType := strategy.GetResourceType()
	r.strategies[resourceType] = strategy
	r.order = append(r.order, resourceType)
}

func (r *StrategyRegistry) Find(resourceType string) interfaces.ResourceTransformer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if strategy, exists := r.strategies[resourceType]; exists {
		return strategy
	}

	for _, strategy := range r.strategies {
		if strategy.CanHandle(resourceType) {
			return strategy
		}
	}

	return nil
}

func (r *StrategyRegistry) GetAll() []interfaces.ResourceTransformer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]interfaces.ResourceTransformer, 0, len(r.order))
	for _, resourceType := range r.order {
		result = append(result, r.strategies[resourceType])
	}
	return result
}

func (r *StrategyRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.strategies)
}

func (r *StrategyRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.strategies = make(map[string]interfaces.ResourceTransformer)
	r.order = make([]string, 0)
}