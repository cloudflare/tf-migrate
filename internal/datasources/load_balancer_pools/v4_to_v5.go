package load_balancer_pools

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_load_balancer_pools datasource from v4 to v5.
// Key transformations:
// 1. Remove filter block (no v5 equivalent - v4's regex name filtering not available in v5)
// 2. Minimal state transformation (datasources are mostly computed)
// 3. Output field renamed: pools → result (handled by provider, not migration)
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_load_balancer_pools datasource v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with "data." prefix to distinguish from resource migration
	internal.RegisterMigrator("data.cloudflare_load_balancer_pools", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 datasource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_load_balancer_pools"
}

// GetAttributeRenames returns attribute renames for cross-file reference updates.
// The load_balancer_pools datasource changed its output attribute from "pools" to "result".
func (m *V4ToV5Migrator) GetAttributeRenames() []transform.AttributeRename {
	return []transform.AttributeRename{
		{
			ResourceType: "data.cloudflare_load_balancer_pools",
			OldAttribute: "pools",
			NewAttribute: "result",
		},
	}
}

// CanHandle determines if this migrator can handle the given datasource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Only match datasource type (with "data." prefix)
	return resourceType == "data.cloudflare_load_balancer_pools"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for load_balancer_pools datasource migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
// Main transformation:
// - Remove filter block (v4's regex name filtering has no v5 equivalent)
// - account_id remains unchanged
// - New v5 fields (monitor, max_items) are not added during migration
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Remove filter block (no v5 equivalent for regex name filtering)
	tfhcl.RemoveBlocksByType(body, "filter")

	// account_id stays as-is (required in both v4 and v5)
	// monitor and max_items are new in v5 - not added during migration

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// Main transformations:
// 1. Set schema_version = 0
// 2. Remove filter field (if present)
// 3. Rename pools → result (preserving data)
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		// No attributes to transform, but still set schema_version
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Remove filter field (if present in state)
	result, _ = sjson.Delete(result, "attributes.filter")

	// Rename pools → result (preserving data)
	poolsField := attrs.Get("pools")
	if poolsField.Exists() {
		if poolsField.IsArray() {
			// Copy pools to result
			result, _ = sjson.Set(result, "attributes.result", poolsField.Value())
			// Delete old pools field
			result, _ = sjson.Delete(result, "attributes.pools")
		} else {
			// pools is null or missing - just delete it
			result, _ = sjson.Delete(result, "attributes.pools")
		}
	}

	return result, nil
}

func init() {
	// Register the migrator on package initialization
	NewV4ToV5Migrator()
}
