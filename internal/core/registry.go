package core

import (
	"github.com/cloudflare/tf-migrate/internal/core/transform"
)

// Registry holds resource transformers
// Registration happens once at startup, so no thread-safety is needed
type Registry struct {
	transformers []transform.Transformer
}

// NewRegistry creates a new registry
func NewRegistry() *Registry {
	return &Registry{
		transformers: make([]transform.Transformer, 0),
	}
}

// Register adds a transformer to the registry
func (r *Registry) Register(transformer transform.Transformer) {
	r.transformers = append(r.transformers, transformer)
}

// Find looks up a transformer that can handle the given resource type
// Returns nil if no transformer can handle this type
func (r *Registry) Find(resourceType string) transform.Transformer {
	for _, transformer := range r.transformers {
		if transformer.CanHandle(resourceType) {
			return transformer
		}
	}
	return nil
}

// GetAll returns all registered transformers
// Used by PreprocessHandler to apply all preprocessors
func (r *Registry) GetAll() []transform.Transformer {
	return r.transformers
}