package zero_trust_gateway_settings

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zero Trust Gateway Settings from v4 to v5
// v4: cloudflare_teams_account
// v5: cloudflare_zero_trust_gateway_settings
type V4ToV5Migrator struct {
	oldType string
	newType string
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType: "cloudflare_teams_account",
		newType: "cloudflare_zero_trust_gateway_settings",
	}
	internal.RegisterMigrator("cloudflare_teams_account", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return m.newType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == m.oldType
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_teams_account", "cloudflare_zero_trust_gateway_settings"
}

// Preprocess - no preprocessing needed, all transformations done in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename cloudflare_teams_account to cloudflare_zero_trust_gateway_settings
	tfhcl.RenameResourceType(block, "cloudflare_teams_account", "cloudflare_zero_trust_gateway_settings")

	_ = block.Body() // Will be used in implementation

	// TODO: Implement major restructuring
	// 1. Create settings wrapper
	// 2. Move flat booleans into nested structures
	// 3. Convert MaxItems:1 blocks to attributes
	// 4. Nest everything under settings
	// 5. Apply field renames
	// 6. Remove deprecated fields

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState transforms the state JSON from v4 to v5
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	result := instance.String()

	// Get attributes
	attrs := instance.Get("attributes")
	if !attrs.Exists() {
		return result, nil
	}

	// TODO: Implement state transformation
	// 1. Create settings object
	// 2. Move flat booleans into nested structures
	// 3. Convert MaxItems:1 arrays to objects
	// 4. Nest all settings
	// 5. Apply field renames
	// 6. Remove deprecated fields

	return result, nil
}
