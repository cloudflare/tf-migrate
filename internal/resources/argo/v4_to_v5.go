package argo

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of Argo resources from v4 to v5
// The v4 cloudflare_argo resource is split into two separate resources in v5:
// - cloudflare_argo_smart_routing
// - cloudflare_argo_tiered_caching
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_argo", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return empty string because this resource splits into TWO different types
	// based on instance attributes. The type is determined dynamically in TransformState.
	return ""
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_argo"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	// Check which attributes exist
	smartRoutingAttr := body.GetAttribute("smart_routing")
	tieredCachingAttr := body.GetAttribute("tiered_caching")

	var newBlocks []*hclwrite.Block
	if smartRoutingAttr == nil && tieredCachingAttr == nil {
		// Neither smart_routing or tiered_caching - Default to smart_routing with value = off
		newBlocks = append(newBlocks, m.createSmartRoutingBlock(block, resourceName, true)...)
	} else if smartRoutingAttr != nil && tieredCachingAttr != nil {
		// Both smart_routing and tiered_caching
		// Only smart_routing gets moved block (primary resource)
		// tiered_caching is created as new (users must import it)
		newBlocks = append(newBlocks, m.createSmartRoutingBlock(block, resourceName, true)...)
		newBlocks = append(newBlocks, m.createTieredCachingBlock(block, resourceName+"_tiered", false)...)
	} else if smartRoutingAttr != nil {
		// Only smart_routing
		newBlocks = append(newBlocks, m.createSmartRoutingBlock(block, resourceName, true)...)
	} else {
		// Only tiered_caching
		newBlocks = append(newBlocks, m.createTieredCachingBlock(block, resourceName, true)...)
	}

	return &transform.TransformResult{
		Blocks:         newBlocks,
		RemoveOriginal: true,
	}, nil
}

func (m *V4ToV5Migrator) createSmartRoutingBlock(originalBlock *hclwrite.Block, resourceName string, includeMovedBlock bool) []*hclwrite.Block {
	newBlocks := make([]*hclwrite.Block, 0)

	newBlock := tfhcl.CreateDerivedBlock(originalBlock, "cloudflare_argo_smart_routing", resourceName, tfhcl.AttributeTransform{
		Copy:              []string{"zone_id"},
		Rename:            map[string]string{"smart_routing": "value"},
		CopyMetaArguments: true,
	})

	tfhcl.EnsureAttribute(newBlock.Body(), "value", "off")

	newBlocks = append(newBlocks, newBlock)

	if includeMovedBlock {
		movedBlock := m.createMovedBlock(tfhcl.GetResourceName(originalBlock), resourceName, "cloudflare_argo", "cloudflare_argo_smart_routing")
		newBlocks = append(newBlocks, movedBlock)
	}

	return newBlocks
}

func (m *V4ToV5Migrator) createTieredCachingBlock(originalBlock *hclwrite.Block, resourceName string, includeMovedBlock bool) []*hclwrite.Block {
	newBlocks := make([]*hclwrite.Block, 0)

	newBlock := tfhcl.CreateDerivedBlock(originalBlock, "cloudflare_argo_tiered_caching", resourceName, tfhcl.AttributeTransform{
		Copy:              []string{"zone_id"},
		Rename:            map[string]string{"tiered_caching": "value"},
		CopyMetaArguments: true,
	})
	newBlocks = append(newBlocks, newBlock)

	if includeMovedBlock {
		movedBlock := m.createMovedBlock(tfhcl.GetResourceName(originalBlock), resourceName, "cloudflare_argo", "cloudflare_argo_tiered_caching")
		newBlocks = append(newBlocks, movedBlock)
	}

	return newBlocks
}

// createMovedBlock creates a moved block for state migration tracking
func (m *V4ToV5Migrator) createMovedBlock(fromName, toName, fromType, toType string) *hclwrite.Block {
	from := fmt.Sprintf("%s.%s", fromType, fromName)
	to := fmt.Sprintf("%s.%s", toType, toName)
	return tfhcl.CreateMovedBlock(from, to)
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	attrs := stateJSON.Get("attributes")

	// Determine which resource type this stateJSON should become based on the presence of attributes
	// Note: When both attributes exist, this is called once and we transform to smart_routing
	// The tiered_caching resource would need to be created separately (managed by the HCL transformation)
	var targetType string

	if !attrs.Exists() {
		// No attributes block - default to smart_routing
		targetType = "cloudflare_argo_smart_routing"
	} else {
		// Check if attributes exist AND are not null (null attributes in state should be treated as not present)
		hasSmartRouting := attrs.Get("smart_routing").Exists() && attrs.Get("smart_routing").Type != gjson.Null && attrs.Get("smart_routing").String() != ""
		hasTieredCaching := attrs.Get("tiered_caching").Exists() && attrs.Get("tiered_caching").Type != gjson.Null && attrs.Get("tiered_caching").String() != ""

		if hasTieredCaching && !hasSmartRouting {
			// Transform to tiered_caching ONLY if tiered_caching exists and smart_routing does NOT
			targetType = "cloudflare_argo_tiered_caching"

			result = state.RenameField(result, "attributes", attrs, "tiered_caching", "value")
			result = state.RemoveFields(result, "attributes", attrs, "smart_routing")
		} else {
			// Transform to smart_routing (either it exists, or neither attribute exists)
			targetType = "cloudflare_argo_smart_routing"

			if attrs.Get("smart_routing").Exists() {
				result = state.RenameField(result, "attributes", attrs, "smart_routing", "value")
			} else {
				result, _ = sjson.Set(result, "attributes.value", "off")
			}

			// Remove tiered_caching field
			result = state.RemoveFields(result, "attributes", attrs, "tiered_caching")
		}

		// Common transformations for both resource types (only when attributes exist)

		// Update ID to be zone_id instead of checksum
		if attrs.Get("zone_id").Exists() {
			zoneID := attrs.Get("zone_id").String()
			result, _ = sjson.Set(result, "attributes.id", zoneID)
		}

		// Add computed fields with reasonable defaults
		result, _ = sjson.Set(result, "attributes.editable", true)
		result, _ = sjson.Set(result, "attributes.modified_on", nil)
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Store the determined resource type in context metadata for the handler to use
	// The handler will read this after processing the instance and update the resource-level type
	metadataKey := fmt.Sprintf("argo_resource_type:%s", resourceName)
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]interface{})
	}
	ctx.Metadata[metadataKey] = targetType

	return result, nil
}
