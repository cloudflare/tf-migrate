package rulesets

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

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

// TransformState is a no-op for rulesets datasource migration.
//
// State transformation is now handled by the provider's StateUpgraders (UpgradeState).
// The provider's UpgradeState handlers perform the actual state migration when
// Terraform detects a schema version mismatch.
//
// tf-migrate's role is limited to:
// - Transforming HCL configuration syntax (handled by TransformConfig)
//
// This delegation to the provider is the correct architectural pattern because:
// 1. The provider is the source of truth for state structure
// 2. Provider has access to proper schema definitions for type-safe parsing
// 3. Eliminates duplication of transformation logic
// 4. Ensures migrations work correctly with Terraform's state upgrade mechanisms
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// Return state unchanged - provider handles all state transformations
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this datasource uses provider-based state migration.
//
// When this returns true, tf-migrate knows that:
// - State transformation is delegated to the provider's StateUpgraders
// - The provider's UpgradeState handlers will perform the actual migration
// - tf-migrate should only handle configuration transformation
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
