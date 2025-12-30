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

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_tiered_cache doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_tiered_cache", "cloudflare_tiered_cache"
}

// TransformConfig handles configuration file transformations.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	blocks := make([]*hclwrite.Block, 0)

	// rename cache_type to value in the original block
	tfhcl.RenameAttribute(body, "cache_type", "value")

	resourceName := tfhcl.GetResourceName(block)
	valueAttr := body.GetAttribute("value")

	// Try to get the actual value
	var value string
	if tfhcl.IsExpressionAttribute(valueAttr) {
		// It's a variable reference - look up the actual value in state
		value = state.GetResourceAttribute(ctx.StateJSON, "cloudflare_tiered_cache", resourceName, "cache_type")
	} else {
		// It's a literal value
		value = tfhcl.ExtractStringFromAttribute(valueAttr)
	}

	if value == "smart" {
		// cache_type="smart" → value="on" for both resources
		// Create tiered_cache resource with value="on"
		tfhcl.SetAttribute(body, "value", "on")
		tieredCacheBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_tiered_cache",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id", "value"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, tieredCacheBlock)

		// Create argo_tiered_caching resource with value="on"
		argoBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_argo_tiered_caching",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id"},
				Set:               map[string]interface{}{"value": "on"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, argoBlock)
	} else if value == "off" {
		// cache_type="off" → value="off" for both resources
		// Create tiered_cache resource with value="off"
		tfhcl.SetAttribute(body, "value", "off")
		tieredCacheBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_tiered_cache",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id", "value"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, tieredCacheBlock)

		// Create argo_tiered_caching resource with value="off"
		argoBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_argo_tiered_caching",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id"},
				Set:               map[string]interface{}{"value": "off"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, argoBlock)
	} else if value == "generic" {
		// cache_type="generic" → tiered_cache value="off", argo_tiered_caching value="on"
		// Create tiered_cache resource with value="off"
		tfhcl.SetAttribute(body, "value", "off")
		tieredCacheBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_tiered_cache",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id", "value"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, tieredCacheBlock)

		// Create argo_tiered_caching resource with value="on"
		argoBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_argo_tiered_caching",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id"},
				Set:               map[string]interface{}{"value": "on"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, argoBlock)
	} else {
		// For variables or other expressions, just rename the attribute and copy as-is
		tieredCacheBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_tiered_cache",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id", "value"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, tieredCacheBlock)
	}

	return &transform.TransformResult{
		Blocks:         blocks,
		RemoveOriginal: true,
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

	// Transform based on cache_type value
	// All resources stay as cloudflare_tiered_cache in the state
	// The argo_tiered_caching resources in the config are new and will be created on next apply
	if cacheTypeValue == "generic" {
		// For generic, the tiered_cache resource gets value="off"
		result, _ = sjson.Delete(result, "attributes.cache_type")
		result, _ = sjson.Set(result, "attributes.value", "off")
	} else if cacheTypeValue == "smart" {
		// For smart, the tiered_cache resource gets value="on"
		result, _ = sjson.Delete(result, "attributes.cache_type")
		result, _ = sjson.Set(result, "attributes.value", "on")
	} else if cacheTypeValue == "off" {
		// For off, the tiered_cache resource gets value="off"
		result, _ = sjson.Delete(result, "attributes.cache_type")
		result, _ = sjson.Set(result, "attributes.value", "off")
	} else {
		// Unknown value (variables, expressions), just rename the field
		result = state.RenameField(result, "attributes", attrs, "cache_type", "value")
	}

	// Set schema version to 0
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
