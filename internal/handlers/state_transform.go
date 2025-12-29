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
	provider transform.MigrationProvider
}

func NewStateTransformHandler(log hclog.Logger, provider transform.MigrationProvider) transform.TransformationHandler {
	return &StateTransformHandler{
		log:      log,
		provider: provider,
	}
}

func (h *StateTransformHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
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
		h.log.Warn("No resources found in state file")
		return h.Next(ctx)
	}

	modifiedState := stateJSON
	transformedCount := 0
	datasourceIndices := []int{} // Track datasource indices to remove them later
	resourcesToDelete := []int{} // Track resources with no instances left to remove them later

	resources.ForEach(func(key, resource gjson.Result) bool {
		mode := resource.Get("mode").String()
		resourceType := resource.Get("type").String()
		if resourceType == "" {
			return true
		}

		// For datasources (mode="data"), check if there's a registered migrator
		// If no migrator exists, mark for removal (datasources are ephemeral)
		// If migrator exists, process the transformation
		lookupType := resourceType
		if mode == "data" {
			lookupType = "data." + resourceType
			migrator := h.provider.GetMigrator(lookupType, ctx.SourceVersion, ctx.TargetVersion)
			if migrator == nil {
				// No migrator - mark for removal (datasources are ephemeral and will be refreshed)
				datasourceIndices = append(datasourceIndices, int(key.Int()))
				h.log.Debug("Missing datasource migration - marking for removal", "type", resourceType)
				return true
			}
			// Migrator exists - process transformation below
			h.log.Debug("Processing datasource with registered migrator", "type", lookupType)
		}

		migrator := h.provider.GetMigrator(lookupType, ctx.SourceVersion, ctx.TargetVersion)
		if migrator == nil {
			resourceKind := "resource"
			if mode == "data" {
				resourceKind = "datasource"
			}
			ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  fmt.Sprintf("Missing %s migration for %s", resourceKind, resourceType),
				Detail:   fmt.Sprintf("No migrator found for %s: %s (v%s -> v%s)", resourceKind, resourceType, ctx.SourceVersion, ctx.TargetVersion),
			})
			h.log.Debug("No migrator found", "kind", resourceKind, "type", resourceType, "source", ctx.SourceVersion, "target", ctx.TargetVersion)
			return true
		}

		instances := resource.Get("instances")
		if !instances.Exists() {
			return true
		}

		// Check if this migrator can handle the resource and transform the type
		if migrator.CanHandle(resourceType) {
			// Update the resource type if it changed (e.g., teams_list -> zero_trust_list)
			newResourceType := migrator.GetResourceType()
			if newResourceType != "" && newResourceType != resourceType {
				resourcePath := fmt.Sprintf("resources.%d.type", key.Int())
				modifiedState, _ = sjson.Set(modifiedState, resourcePath, newResourceType)
				h.log.Debug("Updated resource type", "from", resourceType, "to", newResourceType)
			}
		}

		resourceName := resource.Get("name").String()
		instancesToDelete := []int{}

		instances.ForEach(func(instKey, instance gjson.Result) bool {
			resourcePath := fmt.Sprintf("resources.%d.instances.%d", key.Int(), instKey.Int())

			transformedJSON, err := migrator.TransformState(ctx, instance, resourcePath, resourceName)
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
			} else {
				// Empty string means delete this instance
				instancesToDelete = append(instancesToDelete, int(instKey.Int()))
			}

			return true
		})

		// Delete instances in reverse order to avoid index shifting
		for i := len(instancesToDelete) - 1; i >= 0; i-- {
			instIdx := instancesToDelete[i]
			instancePath := fmt.Sprintf("resources.%d.instances.%d", key.Int(), instIdx)
			modifiedState, _ = sjson.Delete(modifiedState, instancePath)
			h.log.Debug("Deleted instance from state", "resource", resourceType, "instance_index", instIdx)
		}

		// If all instances were deleted, mark this resource for removal
		if len(instancesToDelete) > 0 {
			// Check if there are any instances left
			remainingInstances := gjson.Get(modifiedState, fmt.Sprintf("resources.%d.instances", key.Int()))
			if !remainingInstances.Exists() || len(remainingInstances.Array()) == 0 {
				resourcesToDelete = append(resourcesToDelete, int(key.Int()))
				h.log.Debug("Marking resource for deletion (no instances left)", "resource", resourceType, "name", resourceName)
			}
		}

		// Check if the migrator stored a dynamic resource type in StateTypeRenames
		// This handles cases like cloudflare_argo which splits into different types based on instance attributes
		if ctx.StateTypeRenames != nil {
			stateTypeRenameKey := fmt.Sprintf("%s.%s", resourceType, resourceName)
			if dynamicType, ok := ctx.StateTypeRenames[stateTypeRenameKey]; ok {
				if dynamicTypeStr, ok := dynamicType.(string); ok && dynamicTypeStr != "" && dynamicTypeStr != resourceType {
					resourceTypePath := fmt.Sprintf("resources.%d.type", key.Int())
					modifiedState, _ = sjson.Set(modifiedState, resourceTypePath, dynamicTypeStr)
					h.log.Debug("Updated resource type from metadata", "from", resourceType, "to", dynamicTypeStr)
				}
			}
		}

		return true
	})

	// Remove datasources from state (in reverse order to avoid index shifting)
	for i := len(datasourceIndices) - 1; i >= 0; i-- {
		idx := datasourceIndices[i]
		resourcePath := fmt.Sprintf("resources.%d", idx)
		modifiedState, _ = sjson.Delete(modifiedState, resourcePath)
		h.log.Debug("Removed datasource from state", "index", idx)
	}

	if len(datasourceIndices) > 0 {
		h.log.Info("Removed datasources from state (will be refreshed by Terraform)", "count", len(datasourceIndices))
	}

	// Remove resources with no instances (in reverse order to avoid index shifting)
	// This handles one-to-many transformations where all instances are deleted
	for i := len(resourcesToDelete) - 1; i >= 0; i-- {
		idx := resourcesToDelete[i]
		resourcePath := fmt.Sprintf("resources.%d", idx)
		modifiedState, _ = sjson.Delete(modifiedState, resourcePath)
		h.log.Debug("Removed resource with no instances from state", "index", idx)
	}

	if len(resourcesToDelete) > 0 {
		h.log.Info("Removed resources with no instances from state", "count", len(resourcesToDelete))
	}

	if transformedCount > 0 || len(datasourceIndices) > 0 || len(resourcesToDelete) > 0 {
		ctx.Content = []byte(modifiedState)
		h.log.Debug("Transformed state resources", "count", transformedCount)
	}

	ctx.StateJSON = modifiedState
	ctx.Metadata["state_transformations"] = transformedCount
	ctx.Metadata["datasources_removed"] = len(datasourceIndices)
	ctx.Metadata["resources_removed"] = len(resourcesToDelete)

	return h.Next(ctx)
}
