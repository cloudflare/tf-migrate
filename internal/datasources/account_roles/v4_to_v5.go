package account_roles

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_account_roles datasource from v4 to v5.
// Key transformations:
// 1. Config block is structurally unchanged — no-op
// 2. Cross-file references: .roles → .result (via GetAttributeRenames)
// 3. State transformation is a no-op (datasources are always re-read from the API)
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_account_roles datasource v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("data.cloudflare_account_roles", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 datasource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_account_roles"
}

// GetAttributeRenames returns attribute renames for cross-file reference updates.
// The account_roles datasource changed its output attribute from "roles" to "result".
func (m *V4ToV5Migrator) GetAttributeRenames() []transform.AttributeRename {
	return []transform.AttributeRename{
		{
			ResourceType: "data.cloudflare_account_roles",
			OldAttribute: "roles",
			NewAttribute: "result",
		},
	}
}

// GetResourceRename implements the ResourceRenamer interface.
// cloudflare_account_roles datasource doesn't rename, so return the same name.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"data.cloudflare_account_roles"}, "data.cloudflare_account_roles"
}

// CanHandle determines if this migrator can handle the given datasource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "data.cloudflare_account_roles"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for account_roles datasource migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
// The account_roles datasource config block is structurally unchanged between v4 and v5
// (only account_id is required). This is a no-op — the block is returned as-is.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func init() {
	NewV4ToV5Migrator()
}
