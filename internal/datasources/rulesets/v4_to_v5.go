package rulesets

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_rulesets datasource from v4 to v5.
// Key transformations:
// 1. Remove filter block (client-side filtering removed in v5)
// 2. Remove include_rules field (rules not included in v5 list datasource)
// 3. Keep account_id and zone_id unchanged
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_rulesets datasource v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with "data." prefix to distinguish from resource migration
	internal.RegisterMigrator("data.cloudflare_rulesets", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 datasource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_rulesets"
}

// CanHandle determines if this migrator can handle the given datasource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Only match datasource type (with "data." prefix)
	return resourceType == "data.cloudflare_rulesets"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for rulesets datasource migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
// Transformations:
// 1. Remove filter block (TypeList MaxItems:1)
// 2. Remove include_rules attribute
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Remove filter block (client-side filtering removed in v5)
	tfhcl.RemoveBlocksByType(body, "filter")

	// Remove include_rules field (rules not included in v5 list datasource)
	tfhcl.RemoveAttributes(body, "include_rules")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// This function receives a single datasource instance and returns the transformed instance JSON.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Check if it's a valid datasource instance
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		// Even for invalid instances, set schema_version for v5
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// 1. Remove filter block from state (if present)
	if attrs.Get("filter").Exists() {
		result, _ = sjson.Delete(result, "attributes.filter")
	}

	// 2. Remove include_rules field from state (if present)
	if attrs.Get("include_rules").Exists() {
		result, _ = sjson.Delete(result, "attributes.include_rules")
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
