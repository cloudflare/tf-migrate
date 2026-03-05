package leaked_credential_check

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_leaked_credential_check from v4 to v5.
// This is a Framework→Framework migration with minimal changes:
// - enabled field changed from Required to Optional
// - No schema transformations needed
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_leaked_credential_check v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 - no rename)
	internal.RegisterMigrator("cloudflare_leaked_credential_check", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_leaked_credential_check"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_leaked_credential_check"
}

// GetResourceRename implements the ResourceRenamer interface.
// cloudflare_leaked_credential_check doesn't rename, so return the same name.
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_leaked_credential_check", "cloudflare_leaked_credential_check"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for this migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
// No config transformations needed - v4 and v5 syntax is identical.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No transformations needed - just return the block unchanged
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders.
	// This function intentionally returns the original state unchanged.
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
