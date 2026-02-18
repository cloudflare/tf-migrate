package argo

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
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

// GetResourceRename implements the ResourceRenamer interface
// Argo is special - it splits into multiple resources (argo_smart_routing, argo_tiered_caching)
// We use the old name for both to indicate it doesn't have a 1:1 rename
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_argo", "cloudflare_argo"
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

// TransformState is a no-op for argo migration
// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState)
// The moved blocks generated in TransformConfig trigger the provider's migration logic
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is delegated to the provider
	// Provider handles:
	// - Resource type change (cloudflare_argo → cloudflare_argo_smart_routing or cloudflare_argo_tiered_caching)
	// - Field transformations (smart_routing/tiered_caching → value)
	// - ID transformation (checksum → zone_id)
	// - Computed field initialization (editable, modified_on)
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
// When true, tf-migrate will not perform state transformation - the provider handles it
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
