package handlers

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal/interfaces"
	"github.com/cloudflare/tf-migrate/internal/logger"
	"github.com/cloudflare/tf-migrate/internal/registry"
)

type StateTransformHandler struct {
	interfaces.BaseHandler
	registry *registry.StrategyRegistry
}

func NewStateTransformHandler(reg *registry.StrategyRegistry) interfaces.TransformationHandler {
	return &StateTransformHandler{
		registry: reg,
	}
}

func (h *StateTransformHandler) Handle(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
	if len(ctx.Content) == 0 {
		return ctx, fmt.Errorf("state content is empty")
	}

	stateJSON := string(ctx.Content)
	if !gjson.Valid(stateJSON) {
		return ctx, fmt.Errorf("invalid JSON in state file")
	}
	result := gjson.Parse(stateJSON)

	resources := result.Get("resources")
	if !resources.Exists() {
		logger.Warn("No resources found in state file")
		return h.CallNext(ctx)
	}

	modifiedState := stateJSON
	transformedCount := 0

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()
		if resourceType == "" {
			return true // continue
		}

		strategy := h.registry.Find(resourceType)
		if strategy == nil {
			logger.Debug("No strategy found for state resource", "type", resourceType)
			return true
		}

		instances := resource.Get("instances")
		if !instances.Exists() {
			return true // continue
		}

		instances.ForEach(func(instKey, instance gjson.Result) bool {
			resourcePath := fmt.Sprintf("resources.%d.instances.%d", key.Int(), instKey.Int())

			transformedJSON, err := strategy.TransformState(instance, resourcePath)
			if err != nil {
				logger.Error("Error transforming state resource",
					"type", resourceType,
					"path", resourcePath,
					"error", err)
				return true
			}

			if transformedJSON != "" {
				newState, err := sjson.SetRaw(modifiedState, resourcePath, transformedJSON)
				if err != nil {
					logger.Error("Failed to update state JSON",
						"path", resourcePath,
						"error", err)
					return true
				}
				modifiedState = newState
				transformedCount++
			}

			return true
		})

		return true
	})

	if transformedCount > 0 {
		ctx.Content = []byte(modifiedState)
		logger.Info("Transformed state resources", "count", transformedCount)
	}

	ctx.StateJSON = modifiedState
	ctx.Metadata["state_transformations"] = transformedCount

	return h.CallNext(ctx)
}
