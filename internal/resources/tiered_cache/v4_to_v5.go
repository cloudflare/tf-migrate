package tiered_cache

import (
	"fmt"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// V4ToV5Migrator handles the migration of cloudflare_tiered_cache from v4 to v5.
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_tiered_cache v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name
	internal.RegisterMigrator("cloudflare_tiered_cache", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_tiered_cache"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_tiered_cache"
}

// Preprocess handles string-level transformations before HCL parsing.
// This transforms cache_type values to prepare for HCL transformation.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	blocks := make([]*hclwrite.Block, 0)
	removeOriginal := false

	// rename cache_type to value
	tfhcl.RenameAttribute(body, "cache_type", "value")

	resourceName := tfhcl.GetResourceName(block)
	value := tfhcl.ExtractStringFromAttribute(body.GetAttribute("value"))
	if value == "smart" {
		// cache_type="smart" → value="on"
		tfhcl.SetAttribute(body, "value", "on")
		blocks = append(blocks, block)
	} else if value == "off" {
		// cache_type="off" → value="off"
		tfhcl.SetAttribute(body, "value", "off")
		blocks = append(blocks, block)
	} else if value == "generic" {
		newBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_argo_tiered_caching",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id"},
				Set:               map[string]interface{}{"value": "on"},
				CopyMetaArguments: true,
			},
		)

		movedBlock := tfhcl.CreateMovedBlock(
			"cloudflare_tiered_cache."+resourceName,
			"cloudflare_argo_tiered_caching."+resourceName,
		)

		blocks = append(blocks, newBlock, movedBlock)
		removeOriginal = true
	} else {
		blocks = append(blocks, block)
	}

	return &transform.TransformResult{
		Blocks:         blocks,
		RemoveOriginal: removeOriginal,
	}, nil
}

func (m *V4ToV5Migrator) createMovedBlock(fromName, toName, fromType, toType string) *hclwrite.Block {
	from := fmt.Sprintf("%s.%s", fromType, fromName)
	to := fmt.Sprintf("%s.%s", toType, toName)
	return tfhcl.CreateMovedBlock(from, to)
}

// TransformState handles state file transformations.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}
	attrs := stateJSON.Get("attributes")

	// Check if we have cache_type attribute
	cacheTypeField := attrs.Get("cache_type")
	if !cacheTypeField.Exists() {
		// Already migrated or no cache_type, just set schema version
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	cacheTypeValue := cacheTypeField.String()

	// Determine target resource type based on cache_type value
	var targetType string

	// Transform based on cache_type value
	if cacheTypeValue == "generic" {
		// Transform to argo_tiered_caching
		targetType = "cloudflare_argo_tiered_caching"
		result, _ = sjson.Delete(result, "attributes.cache_type")
		result, _ = sjson.Set(result, "attributes.value", "on")
	} else if cacheTypeValue == "smart" {
		// Keep as tiered_cache, transform smart → on
		targetType = "cloudflare_tiered_cache"
		result, _ = sjson.Delete(result, "attributes.cache_type")
		result, _ = sjson.Set(result, "attributes.value", "on")
	} else if cacheTypeValue == "off" {
		// Keep as tiered_cache, cache_type → value (no value change)
		targetType = "cloudflare_tiered_cache"
		result, _ = sjson.Delete(result, "attributes.cache_type")
		result, _ = sjson.Set(result, "attributes.value", "off")
	} else {
		// Unknown value (variables, expressions), just rename the field
		targetType = "cloudflare_tiered_cache"
		result = state.RenameField(result, "attributes", attrs, "cache_type", "value")
	}

	// Set schema version to 0
	result, _ = sjson.Set(result, "schema_version", 0)

	transform.SetStateTypeRename(ctx, resourceName, "cloudflare_tiered_cache", targetType)

	return result, nil
}
