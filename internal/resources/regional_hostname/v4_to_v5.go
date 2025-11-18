package regional_hostname

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// V4ToV5Migrator handles migration of regional hostname resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_regional_hostname", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_regional_hostname"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_regional_hostname"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For regional_hostname, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// Postprocess performs any post-processing after all transformations.
// Cross-file references are handled by global postprocessing.
func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_regional_hostname", "cloudflare_regional_hostname"
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// For regional_hostname, we need to remove the timeouts block if present.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Remove timeouts block - v5 Plugin Framework handles timeouts differently
	// than v4 SDKv2, and regional_hostname v5 doesn't support custom timeouts
	tfhcl.RemoveBlocksByType(body, "timeouts")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState transforms the JSON state from v4 to v5.
// For regional_hostname, we need to add the new v5 routing field with its default value
// to prevent plan changes after migration.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		// Even for invalid/incomplete instances, set schema_version for v5
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	instance := stateJSON.Get("attributes")

	// Add new v5 routing field with default value to match the schema
	// routing: default is "dns"
	result = state.EnsureField(result, "attributes", instance, "routing", "dns")

	// Remove timeouts field - v4 SDKv2 stores timeouts in state, but v5 Plugin Framework doesn't
	result = state.RemoveFields(result, "attributes", instance, "timeouts")

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
