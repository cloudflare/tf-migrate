package zero_trust_access_application

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zero Trust Access Application resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the OLD (v4) resource name: cloudflare_access_application
	internal.RegisterMigrator("cloudflare_access_application", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_access_application"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_access_application"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done with HCL manipulation
	return content
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_application", "cloudflare_zero_trust_access_application"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// 1. Rename resource type
	tfhcl.RenameResourceType(block, "cloudflare_access_application", "cloudflare_zero_trust_access_application")

	body := block.Body()

	// 2. Remove deprecated attributes
	if body.GetAttribute("domain_type") != nil {
		body.RemoveAttribute("domain_type")
	}

	// 3. Convert blocks to map attributes (MaxItems:1 blocks become object attributes)
	// These blocks have MaxItems:1 and should be converted to single object attributes
	blocksToConvertToMap := []string{
		"saas_app",
		"cors_headers",
		"hybrid_and_implicit_options",
		"refresh_token_options",
		"scim_config",
		"authentication",
		"operations",
	}

	for _, blockType := range blocksToConvertToMap {
		tfhcl.ConvertSingleBlockToAttribute(body, blockType, blockType)
	}

	// 4. Convert blocks to list attributes
	blocksToConvertToList := []string{
		"footer_links",
		"landing_page_design",
		"custom_attribute",
		"source",
		"custom_claim",
		"mappings",
	}

	for _, blockType := range blocksToConvertToList {
		tfhcl.ConvertBlocksToArrayAttribute(body, blockType, false)
	}

	return &transform.TransformResult{}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Get attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		// Set schema_version even for invalid instances
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// 1. Remove deprecated attributes
	result, _ = sjson.Delete(result, "attributes.domain_type")

	// 2. Convert MaxItems:1 array attributes to single objects
	// saas_app: [{...}] → {...}
	maxItems1Attributes := []string{
		"saas_app",
		"cors_headers",
		"hybrid_and_implicit_options",
		"refresh_token_options",
		"scim_config",
		"authentication",
		"operations",
	}

	for _, attr := range maxItems1Attributes {
		if attrs.Get(attr).IsArray() {
			arr := attrs.Get(attr).Array()
			if len(arr) > 0 {
				// Convert first element of array to single object
				val := arr[0].Value()
				result, _ = sjson.Set(result, "attributes."+attr, val)
			}
		}
	}

	// 3. Update schema version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

func init() {
	_ = NewV4ToV5Migrator()
}
