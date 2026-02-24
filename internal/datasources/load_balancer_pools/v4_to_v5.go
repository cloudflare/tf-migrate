package load_balancer_pools

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

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

// TransformState is a no-op for load_balancer_pools datasource migration.
//
// State transformation is now handled by the provider's StateUpgraders (UpgradeState).
// The provider's UpgradeState handlers perform the actual state migration when
// Terraform detects a schema version mismatch.
//
// tf-migrate's role is limited to:
// - Transforming HCL configuration syntax (handled by TransformConfig)
// - Updating cross-file attribute references (handled by GetAttributeRenames)
//
// This delegation to the provider is the correct architectural pattern because:
// 1. The provider is the source of truth for state structure
// 2. Provider has access to proper schema definitions for type-safe parsing
// 3. Eliminates duplication of transformation logic
// 4. Ensures migrations work correctly with Terraform's state upgrade mechanisms
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	// Return state unchanged - provider handles all state transformations
	return instance.String(), nil
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

func init() {
	// Register the migrator on package initialization
	NewV4ToV5Migrator()
}
