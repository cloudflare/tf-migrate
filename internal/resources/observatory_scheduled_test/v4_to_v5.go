package observatory_scheduled_test

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles migration of observatory_scheduled_test resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name (no rename for this resource)
	internal.RegisterMigrator("cloudflare_observatory_scheduled_test", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name doesn't change
	return "cloudflare_observatory_scheduled_test"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_observatory_scheduled_test"
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_observatory_scheduled_test", "cloudflare_observatory_scheduled_test"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For observatory_scheduled_test, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// Postprocess performs any string-level transformations after HCL generation.
func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// For observatory_scheduled_test: No config transformation needed and no resource rename.
//
// All fields remain unchanged:
// - v4: region and frequency are Required
// - v5: region and frequency are Computed+Optional (relaxation, not a breaking change)
// - User-provided values are preserved in config
//
// No moved blocks generated since resource name doesn't change.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Resource not renamed: return block unchanged, no moved block needed
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is a no-op for observatory_scheduled_test.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
// The provider automatically migrates v4 state to v5 when users run terraform apply.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// This function is a no-op for observatory_scheduled_test migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
