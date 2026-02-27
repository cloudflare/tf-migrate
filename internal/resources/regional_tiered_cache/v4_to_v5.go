package regional_tiered_cache

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_regional_tiered_cache from v4 to v5.
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_regional_tiered_cache v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 name in this case)
	internal.RegisterMigrator("cloudflare_regional_tiered_cache", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name doesn't change
	return "cloudflare_regional_tiered_cache"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_regional_tiered_cache"
}

// Preprocess handles string-level transformations before HCL parsing.
// Not needed for regional_tiered_cache - config is pass-through.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// cloudflare_regional_tiered_cache doesn't rename, so return the same name.
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_regional_tiered_cache", "cloudflare_regional_tiered_cache"
}

// TransformConfig handles configuration file transformations.
// For regional_tiered_cache, config is pass-through - no transformations needed.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// NO transformations needed!
	// Resource name stays the same, all fields stay the same.
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is a no-op for regional_tiered_cache.
// State migration is now handled by the provider's StateUpgraders.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	// No-op: return state unchanged
	// The provider's StateUpgraders handle all state transformations
	return instance.String(), nil
}

// UsesProviderStateUpgrader indicates this resource uses provider-side state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
