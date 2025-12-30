package managed_transforms

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the OLD (v4) resource name
	internal.RegisterMigrator("cloudflare_managed_headers", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_managed_transforms"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_managed_headers"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - block to attribute conversion handled in TransformConfig
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Returns rename from cloudflare_managed_headers to cloudflare_managed_transforms
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_managed_headers", "cloudflare_managed_transforms"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename resource type
	tfhcl.RenameResourceType(block, "cloudflare_managed_headers", "cloudflare_managed_transforms")

	body := block.Body()

	// Convert managed_request_headers blocks to array attribute
	// Set empty array if no blocks found (since v5 requires this field)
	tfhcl.ConvertBlocksToArrayAttribute(body, "managed_request_headers", true)

	// Convert managed_response_headers blocks to array attribute
	// Set empty array if no blocks found (since v5 requires this field)
	tfhcl.ConvertBlocksToArrayAttribute(body, "managed_response_headers", true)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string, resourceName string) (string, error) {
	result := stateJSON.String()

	// Get the attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		return result, nil
	}

	// Handle managed_request_headers - ensure it's an array (even if empty)
	// v5 requires this field, v4 allowed null
	if !attrs.Get("managed_request_headers").Exists() || attrs.Get("managed_request_headers").Raw == "null" {
		result, _ = sjson.Set(result, "attributes.managed_request_headers", []interface{}{})
	}

	// Handle managed_response_headers - ensure it's an array (even if empty)
	// v5 requires this field, v4 allowed null
	if !attrs.Get("managed_response_headers").Exists() || attrs.Get("managed_response_headers").Raw == "null" {
		result, _ = sjson.Set(result, "attributes.managed_response_headers", []interface{}{})
	}

	// Set schema_version to 0 for v5
	result = state.SetSchemaVersion(result, 0)

	return result, nil
}
