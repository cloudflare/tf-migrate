package turnstile_widget

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_turnstile_widget from v4 to v5.
// Key transformations:
// 1. Config: Remove toset() wrapper from domains field (Set → List)
// 2. Config: Sort domains alphabetically to match API ordering
// 3. State: Handled by provider's StateUpgraders (UpgradeState)
//
// Field changes:
// - domains: SetAttribute in v4 → ListAttribute in v5 (remove toset wrapper, sort alphabetically)
//
// New v5 fields (will be populated by provider on first read):
// - sitekey: NEW in v5 (computed) - provider will populate
// - created_on: NEW in v5 (computed) - provider will populate
// - modified_on: NEW in v5 (computed) - provider will populate
// - clearance_level: NEW in v5 (optional computed) - provider will populate if configured
// - ephemeral_id: NEW in v5 (optional computed) - provider will populate if configured
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_turnstile_widget v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_turnstile_widget", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_turnstile_widget"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_turnstile_widget"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for turnstile_widget migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_turnstile_widget doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_turnstile_widget", "cloudflare_turnstile_widget"
}

// TransformConfig handles configuration file transformations.
// Main transformation: Remove toset() wrapper from domains field (Set → List)
// Also sorts domains alphabetically to match API ordering
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// v4: domains = toset(["example.com", "test.com"])
	// v5: domains = ["example.com", "test.com"]
	// Remove the toset() function wrapper
	tfhcl.RemoveFunctionWrapper(body, "domains", "toset")

	// Sort domains alphabetically to match API ordering
	// The Cloudflare API returns domains in alphabetical order, and since v5 uses
	// ListAttribute (ordered) instead of SetAttribute (unordered), we must sort
	// domains in both config and state to prevent drift
	tfhcl.SortStringArrayAttribute(body, "domains")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
// This function is a no-op for turnstile_widget migration.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// This function is a no-op for turnstile_widget migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
