package accounts

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_accounts datasource from v4 to v5.
// Key transformations:
// 1. Config block is structurally unchanged (name filter stays at top level) — no-op
// 2. Cross-file references: accounts → result (via GetAttributeRenames)
// 3. State transformation is a no-op (datasources are always re-read from the API)
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_accounts datasource v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with "data." prefix to distinguish from resource migration
	internal.RegisterMigrator("data.cloudflare_accounts", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 datasource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_accounts"
}

// GetAttributeRenames returns attribute renames for cross-file reference updates.
// The accounts datasource changed its output attribute from "accounts" to "result".
func (m *V4ToV5Migrator) GetAttributeRenames() []transform.AttributeRename {
	return []transform.AttributeRename{
		{
			ResourceType: "data.cloudflare_accounts",
			OldAttribute: "accounts",
			NewAttribute: "result",
		},
	}
}

// GetResourceRename implements the ResourceRenamer interface.
// cloudflare_accounts datasource doesn't rename, so return the same name.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"data.cloudflare_accounts"}, "data.cloudflare_accounts"
}

// CanHandle determines if this migrator can handle the given datasource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Only match datasource type (with "data." prefix)
	return resourceType == "data.cloudflare_accounts"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for accounts datasource migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
// The accounts datasource config block is structurally unchanged between v4 and v5:
// - name filter stays at top level
// - No attributes need to be moved, renamed, or restructured
// This is a no-op — the block is returned as-is.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func init() {
	// Register the migrator on package initialization
	NewV4ToV5Migrator()
}
