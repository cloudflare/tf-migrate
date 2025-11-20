package argo

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/hcl"
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

	// Create smart_routing resource if attribute exists or neither attribute exists (default case)
	if smartRoutingAttr != nil || (smartRoutingAttr == nil && tieredCachingAttr == nil) {
		smartRoutingBlock := m.createSmartRoutingBlock(block, resourceName, smartRoutingAttr)
		newBlocks = append(newBlocks, smartRoutingBlock)

		// Create moved block for smart_routing
		movedBlock := m.createMovedBlock(resourceName, resourceName, "cloudflare_argo", "cloudflare_argo_smart_routing")
		newBlocks = append(newBlocks, movedBlock)
	}

	// Create tiered_caching resource if attribute exists
	if tieredCachingAttr != nil {
		// Determine resource name for tiered_caching
		tieredResourceName := resourceName
		// Add "_tiered" suffix if both attributes exist to avoid naming conflict
		if smartRoutingAttr != nil {
			tieredResourceName = resourceName + "_tiered"
		}

		tieredCachingBlock := m.createTieredCachingBlock(block, tieredResourceName, tieredCachingAttr)
		newBlocks = append(newBlocks, tieredCachingBlock)

		// Create moved block for tiered_caching
		movedBlock := m.createMovedBlock(resourceName, tieredResourceName, "cloudflare_argo", "cloudflare_argo_tiered_caching")
		newBlocks = append(newBlocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         newBlocks,
		RemoveOriginal: true,
	}, nil
}

// createSmartRoutingBlock creates a new cloudflare_argo_smart_routing resource block
func (m *V4ToV5Migrator) createSmartRoutingBlock(originalBlock *hclwrite.Block, resourceName string, smartRoutingAttr *hclwrite.Attribute) *hclwrite.Block {
	newBlock := hclwrite.NewBlock("resource", []string{"cloudflare_argo_smart_routing", resourceName})
	newBody := newBlock.Body()
	originalBody := originalBlock.Body()

	// Copy zone_id
	if zoneIDAttr := originalBody.GetAttribute("zone_id"); zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		newBody.SetAttributeRaw("zone_id", tokens)
	}

	// Set value from smart_routing attribute, or default to "off"
	if smartRoutingAttr != nil {
		tokens := smartRoutingAttr.Expr().BuildTokens(nil)
		newBody.SetAttributeRaw("value", tokens)
	} else {
		// Default to "off" when neither attribute exists
		newBody.SetAttributeValue("value", cty.StringVal("off"))
	}

	// Copy lifecycle and other meta-arguments
	m.copyMetaArguments(originalBlock, newBlock)

	return newBlock
}

// createTieredCachingBlock creates a new cloudflare_argo_tiered_caching resource block
func (m *V4ToV5Migrator) createTieredCachingBlock(originalBlock *hclwrite.Block, resourceName string, tieredCachingAttr *hclwrite.Attribute) *hclwrite.Block {
	newBlock := hclwrite.NewBlock("resource", []string{"cloudflare_argo_tiered_caching", resourceName})
	newBody := newBlock.Body()
	originalBody := originalBlock.Body()

	// Copy zone_id
	if zoneIDAttr := originalBody.GetAttribute("zone_id"); zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		newBody.SetAttributeRaw("zone_id", tokens)
	}

	// Set value from tiered_caching attribute
	tokens := tieredCachingAttr.Expr().BuildTokens(nil)
	newBody.SetAttributeRaw("value", tokens)

	// Copy lifecycle and other meta-arguments
	m.copyMetaArguments(originalBlock, newBlock)

	return newBlock
}

// copyMetaArguments copies lifecycle and other meta-arguments from original block to new block
func (m *V4ToV5Migrator) copyMetaArguments(originalBlock, newBlock *hclwrite.Block) {
	originalBody := originalBlock.Body()
	newBody := newBlock.Body()

	// Copy lifecycle block if it exists
	for _, block := range originalBody.Blocks() {
		if block.Type() == "lifecycle" {
			// Clone the lifecycle block
			lifecycleBlock := newBody.AppendNewBlock("lifecycle", nil)
			lifecycleBody := lifecycleBlock.Body()

			// Copy all attributes from the original lifecycle block
			// We need to update attribute names if they reference fields that changed
			for name, attr := range block.Body().Attributes() {
				// For ignore_changes, we need to rename smart_routing/tiered_caching to value
				if name == "ignore_changes" {
					// Get the expression and modify it if it references smart_routing or tiered_caching
					tokens := attr.Expr().BuildTokens(nil)
					// Note: This is a simplified approach. In production, we'd need to parse
					// the expression and replace specific identifiers
					lifecycleBody.SetAttributeRaw(name, tokens)
				} else {
					tokens := attr.Expr().BuildTokens(nil)
					lifecycleBody.SetAttributeRaw(name, tokens)
				}
			}
		}
	}

	// Copy other blocks (provider, etc.) if any
	for _, block := range originalBody.Blocks() {
		if block.Type() != "lifecycle" && block.Type() != "resource" {
			// Copy non-lifecycle, non-resource blocks as-is
			newBody.AppendBlock(block)
		}
	}
}

// createMovedBlock creates a moved block for state migration tracking
func (m *V4ToV5Migrator) createMovedBlock(fromName, toName, fromType, toType string) *hclwrite.Block {
	from := fmt.Sprintf("%s.%s", fromType, fromName)
	to := fmt.Sprintf("%s.%s", toType, toName)
	return hcl.CreateMovedBlock(from, to)
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result, nil
	}

	// Determine which resource type this instance should become based on the presence of attributes
	// Note: When both attributes exist, this is called once and we transform to smart_routing
	// The tiered_caching resource would need to be created separately (managed by the HCL transformation)

	hasSmartRouting := attrs.Get("smart_routing").Exists()
	hasTieredCaching := attrs.Get("tiered_caching").Exists()

	// Determine the target resource type based on attributes
	var targetType string

	// Transform to tiered_caching ONLY if tiered_caching exists and smart_routing does NOT
	if hasTieredCaching && !hasSmartRouting {
		targetType = "cloudflare_argo_tiered_caching"

		// Rename field: tiered_caching → value
		result = state.RenameField(result, "attributes", attrs, "tiered_caching", "value")

		// Remove smart_routing field (if any)
		result = state.RemoveFields(result, "attributes", attrs, "smart_routing")

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
	} else {
		targetType = "cloudflare_argo_smart_routing"

		// Transform to smart_routing (either it exists, or neither attribute exists)
		// Rename field: smart_routing → value
		if attrs.Get("smart_routing").Exists() {
			result = state.RenameField(result, "attributes", attrs, "smart_routing", "value")
		} else {
			// Set default value when neither attribute exists
			result, _ = sjson.Set(result, "attributes.value", "off")
		}

		// Remove tiered_caching field
		result = state.RemoveFields(result, "attributes", attrs, "tiered_caching")

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
	}

	// Store the determined resource type in context metadata for the handler to use
	// Use a key that combines resource path prefix (without the instance index) to handle multiple resources
	// The handler will read this after processing the first instance and update the resource type
	metadataKey := fmt.Sprintf("argo_resource_type:%s", resourceName)
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]interface{})
	}
	ctx.Metadata[metadataKey] = targetType

	return result, nil
}
