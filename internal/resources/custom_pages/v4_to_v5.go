package custom_pages

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_custom_pages from v4 to v5.
// Key transformations:
// 1. type → identifier (field rename)
// 2. state: Optional → Required (ensure field exists with default)
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_custom_pages v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name
	internal.RegisterMigrator("cloudflare_custom_pages", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_custom_pages"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_custom_pages"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for custom_pages migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_custom_pages doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_custom_pages", "cloudflare_custom_pages"
}

// TransformConfig handles configuration file transformations.
// Transformations:
// 1. type → identifier
// 2. Ensure state field exists (add default if missing)
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Rename type → identifier
	tfhcl.RenameAttribute(body, "type", "identifier")

	// 2. Ensure state field exists (required in v5, optional in v4)
	// Note: If state already exists, EnsureAttribute won't overwrite it
	tfhcl.EnsureAttribute(body, "state", "default")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State passthrough - provider handles all state migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
// This tells tf-migrate that the provider's StateUpgraders (UpgradeState) handle all state
// transformation, not tf-migrate's TransformState function.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
