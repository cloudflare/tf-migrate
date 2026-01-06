package snippet_rules

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name (same as v5)
	internal.RegisterMigrator("cloudflare_snippet_rules", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name doesn't change
	return "cloudflare_snippet_rules"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_snippet_rules"
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_snippet_rules", "cloudflare_snippet_rules"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For snippet_rules, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// Main transformation: Convert rules blocks to attribute array
// CRITICAL: v4 default for enabled was true, v5 default is false
// We must explicitly set enabled = true when missing to preserve v4 behavior
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Convert rules blocks to attribute array with preprocessing to handle defaults
	// v4: rules { enabled = true ... }  OR  rules { ... } (enabled defaults to true in v4)
	// v5: rules = [{ enabled = true ... }]
	tfhcl.ConvertBlocksToAttributeList(body, "rules", func(ruleBlock *hclwrite.Block) {
		ruleBody := ruleBlock.Body()
		// If enabled field is missing, add it with v4's default value (true)
		// This prevents drift since v5's default is false
		tfhcl.EnsureAttribute(ruleBody, "enabled", true)
	})

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState transforms the JSON state from v4 to v5.
// Rules structure remains the same (already an array), but we must:
// 1. Set schema_version to 0
// 2. NOT add computed fields (id, last_updated)
// 3. Handle empty arrays correctly
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Get attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		// Even for invalid instances, set schema_version
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Handle empty rules array - delete it rather than keeping []
	rulesField := attrs.Get("rules")
	if rulesField.Exists() && rulesField.IsArray() {
		if len(rulesField.Array()) == 0 {
			result, _ = sjson.Delete(result, "attributes.rules")
		}
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
