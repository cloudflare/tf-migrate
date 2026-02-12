package url_normalization_settings

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_url_normalization_settings from v4 to v5.
// This is one of the simplest migrations - no field changes, no resource rename.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_url_normalization_settings v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_url_normalization_settings", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_url_normalization_settings"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_url_normalization_settings"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for url_normalization_settings migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_url_normalization_settings doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_url_normalization_settings", "cloudflare_url_normalization_settings"
}

// TransformConfig handles configuration file transformations.
// No transformations needed - all fields map directly and resource name is unchanged.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// State transformation is handled by the provider's StateUpgraders (UpgradeState)
// This function is a no-op for url_normalization_settings migration
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
