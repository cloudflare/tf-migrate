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
		newBlocks = append(newBlocks, m.createSmartRoutingBlock(block, resourceName)...)
	} else if smartRoutingAttr != nil && tieredCachingAttr != nil {
		// Both smart_routing and tiered_caching
		newBlocks = append(newBlocks, m.createSmartRoutingBlock(block, resourceName)...)
		newBlocks = append(newBlocks, m.createTieredCachingBlock(block, resourceName+"_tiered")...)
	} else if smartRoutingAttr != nil {
		// Only smart_routing
		newBlocks = append(newBlocks, m.createSmartRoutingBlock(block, resourceName)...)
	} else {
		// Only tiered_caching
		newBlocks = append(newBlocks, m.createTieredCachingBlock(block, resourceName)...)
	}

	return &transform.TransformResult{
		Blocks:         newBlocks,
		RemoveOriginal: true,
	}, nil
}

func (m *V4ToV5Migrator) createSmartRoutingBlock(originalBlock *hclwrite.Block, resourceName string) []*hclwrite.Block {
	newBlocks := make([]*hclwrite.Block, 0)

	newBlock := tfhcl.CreateDerivedBlock(originalBlock, "cloudflare_argo_smart_routing", resourceName, tfhcl.AttributeTransform{
		Copy:              []string{"zone_id"},
		Rename:            map[string]string{"smart_routing": "value"},
		CopyMetaArguments: true,
	})

	tfhcl.EnsureAttribute(newBlock.Body(), "value", "off")

	newBlocks = append(newBlocks, newBlock)

	movedBlock := m.createMovedBlock(tfhcl.GetResourceName(originalBlock), resourceName, "cloudflare_argo", "cloudflare_argo_smart_routing")
	newBlocks = append(newBlocks, movedBlock)

	return newBlocks
}

func (m *V4ToV5Migrator) createTieredCachingBlock(originalBlock *hclwrite.Block, resourceName string) []*hclwrite.Block {
	newBlocks := make([]*hclwrite.Block, 0)

	newBlock := tfhcl.CreateDerivedBlock(originalBlock, "cloudflare_argo_tiered_caching", resourceName, tfhcl.AttributeTransform{
		Copy:              []string{"zone_id"},
		Rename:            map[string]string{"tiered_caching": "value"},
		CopyMetaArguments: true,
	})
	newBlocks = append(newBlocks, newBlock)

	movedBlock := m.createMovedBlock(tfhcl.GetResourceName(originalBlock), resourceName, "cloudflare_argo", "cloudflare_argo_tiered_caching")
	newBlocks = append(newBlocks, movedBlock)

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
	if !attrs.Exists() {
		return result, nil
	}

	// Determine which resource type this stateJSON should become based on the presence of attributes
	// Note: When both attributes exist, this is called once and we transform to smart_routing
	// The tiered_caching resource would need to be created separately (managed by the HCL transformation)
	hasSmartRouting := attrs.Get("smart_routing").Exists()
	hasTieredCaching := attrs.Get("tiered_caching").Exists()

	var targetType string
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

	// Common transformations for both resource types

	// Update ID to be zone_id instead of checksum
	if attrs.Get("zone_id").Exists() {
		zoneID := attrs.Get("zone_id").String()
		result, _ = sjson.Set(result, "attributes.id", zoneID)
	}

	// Add computed fields with reasonable defaults
	result, _ = sjson.Set(result, "attributes.editable", true)
	result, _ = sjson.Set(result, "attributes.modified_on", nil)

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Store the determined resource type in context metadata for the handler to use
	// Use a key that combines resource path prefix (without the stateJSON index) to handle multiple resources
	// The handler will read this after processing the first stateJSON and update the resource type
	metadataKey := fmt.Sprintf("argo_resource_type:%s", resourceName)
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]interface{})
	}
	ctx.Metadata[metadataKey] = targetType

	return result, nil
}
