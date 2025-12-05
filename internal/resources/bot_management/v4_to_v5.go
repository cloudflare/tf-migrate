package bot_management

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_bot_management", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the resource type this migrator handles (v5 name).
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_bot_management"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_bot_management"
}

// Preprocess handles string-level transformations before HCL parsing.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_bot_management", "cloudflare_bot_management"
}

// TransformConfig handles configuration file transformations.
// For bot_management, no HCL transformations are needed because:
// - All v4 fields exist in v5 with the same names and semantics
// - No fields were deprecated or renamed
// - New v5 fields are optional and should not be added during migration
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// For bot_management, no state transformations are needed because:
// - All v4 fields exist in v5 with the same names and types
// - No fields were deprecated or renamed
// - No type conversions are required
// - New v5 fields are optional/computed and should not be added during migration
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")
	if !attrs.Get("zone_id").Exists() {
		return result, nil
	}

	return result, nil
}
