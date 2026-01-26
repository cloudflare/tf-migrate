package zero_trust_device_managed_networks

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of device managed networks resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	internal.RegisterMigrator("cloudflare_device_managed_networks", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_device_managed_networks", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_device_managed_networks"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_device_managed_networks" || resourceType == "cloudflare_zero_trust_device_managed_networks"
}

// Preprocess - no preprocessing needed, transformation happens in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Returns rename from cloudflare_device_managed_networks to cloudflare_zero_trust_device_managed_networks
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_device_managed_networks", "cloudflare_zero_trust_device_managed_networks"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	tfhcl.RenameResourceType(block, "cloudflare_device_managed_networks", "cloudflare_zero_trust_device_managed_networks")

	body := block.Body()

	if configBlock := tfhcl.FindBlockByType(body, "config"); configBlock != nil {
		tfhcl.ConvertBlocksToAttribute(body, "config", "config", func(configBlock *hclwrite.Block) {})
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

	// Transform config from array to object (TypeList MaxItems:1 → SingleNestedAttribute)
	configField := attrs.Get("config")
	if configField.Exists() && configField.IsArray() {
		result = state.TransformFieldArrayToObject(
			result,
			"attributes",
			attrs,
			"config",
			state.ArrayToObjectOptions{},
		)
	}

	// Copy id to network_id (v5 provider needs both fields populated)
	// v4 stored network_id as the id field, v5 has them as separate fields
	idField := attrs.Get("id")
	if idField.Exists() {
		result, _ = sjson.Set(result, "attributes.network_id", idField.String())
	}

	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
