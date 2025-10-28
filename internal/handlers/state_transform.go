package handlers

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

type StateTransformHandler struct {
	transform.BaseHandler
	log      hclog.Logger
	provider transform.Provider
}

func NewStateTransformHandler(log hclog.Logger, provider transform.Provider) transform.TransformationHandler {
	return &StateTransformHandler{
		log:      log,
		provider: provider,
	}
}

func (h *StateTransformHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
	if len(ctx.StateJSON) == 0 {
		return ctx, fmt.Errorf("state content is empty")
	}

	stateJSON := ctx.StateJSON
	if !gjson.Valid(stateJSON) {
		return ctx, fmt.Errorf("invalid JSON in state file")
	}
	result := gjson.Parse(stateJSON)

	resources := result.Get("resources")
	if !resources.Exists() {
		h.log.Warn("No resources found in state file")
		return h.Next(ctx)
	}

	modifiedState := stateJSON
	transformedCount := 0

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()
		if resourceType == "" {
			return true
		}

		migrator := h.provider.GetMigrator(resourceType, ctx.SourceVersion, ctx.TargetVersion)
		if migrator == nil {
			ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  fmt.Sprintf("Failed to transform resource: %s", resourceType),
				Detail:   fmt.Sprintf("No migrator found for state resource: %s (v%d -> v%d)", resourceType, ctx.SourceVersion, ctx.TargetVersion),
			})
			h.log.Debug("No migrator found for state resource", "type", resourceType, "source", ctx.SourceVersion, "target", ctx.TargetVersion)
			return true
		}

		instances := resource.Get("instances")
		if !instances.Exists() {
			return true
		}

		instances.ForEach(func(instKey, instance gjson.Result) bool {
			resourcePath := fmt.Sprintf("resources.%d.instances.%d", key.Int(), instKey.Int())

			transformedJSON, err := migrator.TransformState(ctx, instance, resourcePath)
			if err != nil {
				h.log.Error("Error transforming state resource",
					"type", resourceType,
					"path", resourcePath,
					"error", err)
				ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Failed to transform resource: %s", resourceType),
					Detail:   err.Error(),
				})
				return true
			}

			if transformedJSON != "" {
				newState, err := sjson.SetRaw(modifiedState, resourcePath, transformedJSON)
				if err != nil {
					h.log.Error("Failed to update state JSON",
						"path", resourcePath,
						"error", err)
					ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Failed to update state JSON for resource: %s", resourceType),
						Detail:   err.Error(),
					})
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
		h.log.Debug("Transformed state resources", "count", transformedCount)
	}

	ctx.StateJSON = modifiedState
	ctx.Metadata["state_transformations"] = transformedCount

	return h.Next(ctx)
}
