package zero_trust_device_posture_integration

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of device posture integration resources from v4 to v5
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register BOTH v4 resource names (deprecated and current)
	internal.RegisterMigrator("cloudflare_device_posture_integration", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_device_posture_integration", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_device_posture_integration"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_device_posture_integration" ||
		resourceType == "cloudflare_zero_trust_device_posture_integration"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done at HCL level
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Returns rename from cloudflare_device_posture_integration to cloudflare_zero_trust_device_posture_integration
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_device_posture_integration", "cloudflare_zero_trust_device_posture_integration"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename both possible v4 resource names to v5 name
	tfhcl.RenameResourceType(block, "cloudflare_device_posture_integration", "cloudflare_zero_trust_device_posture_integration")

	body := block.Body()

	// Ensure interval is present (required in v5, optional in v4)
	// Default to "24h" if missing
	if !tfhcl.HasAttribute(body, "interval") {
		tfhcl.EnsureAttribute(body, "interval", "24h")
	}

	// Remove deprecated identifier field
	tfhcl.RemoveAttributes(body, "identifier")

	// Convert config block to attribute syntax
	// v4: config { ... }  (block syntax)
	// v5: config = { ... } (attribute syntax)
	if configBlock := tfhcl.FindBlockByType(body, "config"); configBlock != nil {
		tfhcl.ConvertBlocksToAttribute(body, "config", "config", func(cb *hclwrite.Block) {
			// No nested transformations needed - config fields remain the same
		})
	}

	// If config doesn't exist at all, add empty config (required in v5)
	if !tfhcl.HasAttribute(body, "config") && tfhcl.FindBlockByType(body, "config") == nil {
		// Create empty config object
		tfhcl.EnsureAttribute(body, "config", map[string]any{})
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Transform config from array to object
	// v4: "config": [{ ... }]
	// v5: "config": { ... }
	configField := attrs.Get("config")
	if configField.Exists() {
		if configField.IsArray() && len(configField.Array()) > 0 {
			// Extract first element from array and set as object
			configObj := configField.Array()[0]
			result, _ = sjson.Set(result, "attributes.config", configObj.Value())

			// Transform empty values to null for fields not explicitly set in config
			// v4 sets optional fields to empty string, v5 uses null
			updatedAttrs := gjson.Parse(result).Get("attributes")
			updatedConfigField := updatedAttrs.Get("config")
			result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
				Ctx:              ctx,
				Result:           result,
				FieldPath:        "attributes.config",
				FieldResult:      updatedConfigField,
				ResourceName:     resourceName,
				HCLAttributePath: "config",
				CanHandle:        m.CanHandle,
			})
		} else if configField.IsArray() && len(configField.Array()) == 0 {
			// Empty array -> set empty object
			result, _ = sjson.Set(result, "attributes.config", map[string]any{})
		}
	}

	// Add interval with default if missing (required in v5)
	intervalField := attrs.Get("interval")
	if !intervalField.Exists() || intervalField.String() == "" {
		result, _ = sjson.Set(result, "attributes.interval", "24h")
	}

	// Remove deprecated identifier field
	if attrs.Get("identifier").Exists() {
		result, _ = sjson.Delete(result, "attributes.identifier")
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
