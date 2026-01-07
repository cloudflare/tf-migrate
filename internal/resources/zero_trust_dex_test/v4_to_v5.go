package zero_trust_dex_test

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	tfstate "github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of Zero Trust DEX Test resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_device_dex_test", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_dex_test", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_dex_test"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_device_dex_test" || resourceType == "cloudflare_zero_trust_dex_test"
}

// Preprocess - no preprocessing needed, transformation happens in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_device_dex_test", "cloudflare_zero_trust_dex_test"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// rename cloudflare_device_dex_test → cloudflare_zero_trust_dex_test
	tfhcl.RenameResourceType(block, "cloudflare_device_dex_test", "cloudflare_zero_trust_dex_test")

	body := block.Body()

	// Transform data field: TypeList MaxItems:1 → SingleNestedAttribute
	tfhcl.ConvertSingleBlockToAttribute(body, "data", "data")

	// Remove computed timestamp fields if they exist in config (unlikely but possible)
	tfhcl.RemoveAttributes(body, "updated", "created")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	attrs := stateJSON.Get("attributes")
	if attrs.Exists() {
		// Set test_id from id (v5 requires test_id as computed field)
		idValue := attrs.Get("id")
		if idValue.Exists() {
			result, _ = sjson.Set(result, "attributes.test_id", idValue.String())
		}

		// Transform data field: TypeList MaxItems:1 → SingleNestedAttribute
		result = tfstate.TransformFieldArrayToObject(result, "attributes", attrs, "data", tfstate.ArrayToObjectOptions{})

		// Clean up empty method fields
		dataObj := gjson.Parse(result).Get("attributes.data")
		if dataObj.Exists() && dataObj.IsObject() {
			methodField := dataObj.Get("method")
			if methodField.Exists() && methodField.String() == "" {
				result, _ = sjson.Delete(result, "attributes.data.method")
			}
		}

		// Remove computed timestamp fields that don't exist in v5
		result = tfstate.RemoveFields(result, "attributes", attrs, "updated", "created")
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
