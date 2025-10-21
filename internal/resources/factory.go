package resources

import (
	"github.com/cloudflare/tf-migrate/internal/interfaces"
	"github.com/cloudflare/tf-migrate/internal/logger"
	"github.com/cloudflare/tf-migrate/internal/registry"
)

// ResourceFactory is a function that creates a new resource transformer
type ResourceFactory func() interfaces.ResourceTransformer

// ResourceFactories maps resource names to their factory functions
// This will be populated by init() functions in resource-specific packages
var ResourceFactories = map[string]ResourceFactory{}

// RegisterFromFactories registers resource transformers with the registry
func RegisterFromFactories(reg *registry.StrategyRegistry, names ...string) {
	if len(names) == 0 {
		logger.Debug("Registering all available resources")
		for _, factory := range ResourceFactories {
			reg.Register(factory())
		}
		return
	}

	// Register only specified resources
	logger.Debug("Registering resources", "resources", names)
	for _, name := range names {
		if factory, ok := ResourceFactories[name]; ok {
			reg.Register(factory())
		} else {
			logger.Warn("Resource factory not found", "name", name)
		}
	}
}

// GetAvailableResources returns a list of all available resource names
func GetAvailableResources() []string {
	names := make([]string, 0, len(ResourceFactories))
	for name := range ResourceFactories {
		names = append(names, name)
	}
	return names
}

// RegisterResourceFactory registers a resource factory
// This is typically called from init() functions in resource packages
func RegisterResourceFactory(name string, factory ResourceFactory) {
	ResourceFactories[name] = factory
}
