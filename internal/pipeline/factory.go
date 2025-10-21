package pipeline

import (
	"github.com/cloudflare/tf-migrate/internal/handlers"
	"github.com/cloudflare/tf-migrate/internal/interfaces"
	"github.com/cloudflare/tf-migrate/internal/registry"
)

// HandlerFactory is a function that creates a handler
type HandlerFactory func(*registry.StrategyRegistry) interfaces.TransformationHandler

// Predefined handler factories for common handlers
var (
	Preprocess = func(reg *registry.StrategyRegistry) interfaces.TransformationHandler {
		return handlers.NewPreprocessHandler(reg)
	}

	Parse = func(_ *registry.StrategyRegistry) interfaces.TransformationHandler {
		return handlers.NewParseHandler()
	}

	TransformResources = func(reg *registry.StrategyRegistry) interfaces.TransformationHandler {
		return handlers.NewResourceTransformHandler(reg)
	}

	Format = func(_ *registry.StrategyRegistry) interfaces.TransformationHandler {
		return handlers.NewFormatterHandler()
	}

	TransformState = func(reg *registry.StrategyRegistry) interfaces.TransformationHandler {
		return handlers.NewStateTransformHandler(reg)
	}

	FormatState = func(_ *registry.StrategyRegistry) interfaces.TransformationHandler {
		return handlers.NewStateFormatterHandler()
	}
)
