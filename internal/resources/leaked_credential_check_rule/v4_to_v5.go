package leaked_credential_check_rule

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_leaked_credential_check_rule from v4 to v5.
// This is a pass-through migration as the resource name and all fields remain unchanged between v4 and v5.
// The only changes are validation-level (username and password changed from Required to Optional in v5).
type V4ToV5Migrator struct {
}

// NewV4ToV5Migrator creates a new migrator for cloudflare_leaked_credential_check_rule v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register the migrator with the internal registry
	// Resource name is identical in v4 and v5
	internal.RegisterMigrator("cloudflare_leaked_credential_check_rule", "v4", "v5", migrator)

	return migrator
}

// GetResourceType returns the resource type this migrator handles (v5 name).
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_leaked_credential_check_rule"
}

// CanHandle determines if this migrator can handle the given resource type.
// Returns true for cloudflare_leaked_credential_check_rule (resource name is the same in v4 and v5).
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_leaked_credential_check_rule"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing is needed for this migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// This resource does not rename, so we return the same name for both old and new.
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_leaked_credential_check_rule", "cloudflare_leaked_credential_check_rule"
}

// TransformConfig handles configuration file transformations.
// For leaked_credential_check_rule, no HCL transformations are needed because:
// - Resource name is identical in v4 and v5
// - All v4 fields exist in v5 with the same names and types
// - No fields were deprecated or renamed
// - The only changes are validation-level (username and password became optional)
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// For leaked_credential_check_rule, no state transformations are needed because:
// - All v4 fields exist in v5 with the same names and types
// - No fields were deprecated or renamed
// - No type conversions are required (all StringAttribute)
// - All fields maintain their semantics
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Validate that this is a valid leaked_credential_check_rule instance
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// Validate that zone_id exists (required field)
	if !attrs.Get("zone_id").Exists() {
		return result, nil
	}

	// No transformations needed - all fields pass through unchanged
	return result, nil
}
