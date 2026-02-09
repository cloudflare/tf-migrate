package mtls_certificate

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of mtls_certificate resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the OLD (v4) resource name
	internal.RegisterMigrator("cloudflare_mtls_certificate", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name - in this case, unchanged
	return "cloudflare_mtls_certificate"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_mtls_certificate"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for this simple migration
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	// No postprocessing needed
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_mtls_certificate doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_mtls_certificate", "cloudflare_mtls_certificate"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// NO-OP: Resource name and all fields are identical between v4 and v5
	// No transformations needed - just return the block as-is
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Early return for missing attributes
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// Remove new v5 computed fields if they somehow exist
	// Note: 'id' exists in both v4 and v5, so we preserve it
	// Only 'updated_at' is genuinely new in v5
	result = state.RemoveFields(result, "attributes", attrs, "updated_at")

	// All other fields remain unchanged:
	// - No field renames
	// - No type conversions
	// - Computed fields (issuer, signature, serial_number, uploaded_on, expires_on) preserved as-is

	// Always set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
