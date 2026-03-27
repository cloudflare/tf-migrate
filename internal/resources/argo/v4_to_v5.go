package argo

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

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
	// Returns empty string because this resource routes to TWO different types based on attribute presence:
	// - smart_routing attribute → cloudflare_argo_smart_routing (via moved blocks in TransformConfig)
	// - tiered_caching attribute → cloudflare_argo_tiered_caching (via moved blocks in TransformConfig)
	// The actual type is determined dynamically in TransformConfig based on configuration attributes.
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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_argo"}, "cloudflare_argo"
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
		// Both smart_routing and tiered_caching.
		// smart_routing gets a moved block (primary resource, state carries over).
		// tiered_caching is a brand-new resource with no existing state entry —
		// the user must import the existing zone setting before applying.
		tieredName := resourceName + "_tiered"
		newBlocks = append(newBlocks, m.createSmartRoutingBlock(block, resourceName, true)...)
		newBlocks = append(newBlocks, m.createTieredCachingBlock(block, tieredName, false)...)

		// Extract zone_id from the original block so the import block and
		// diagnostic message use the real value when it is a literal string.
		zoneID := "<zone_id>"
		if zoneAttr := body.GetAttribute("zone_id"); zoneAttr != nil {
			extracted := tfhcl.ExtractStringFromAttribute(zoneAttr)
			if len(extracted) == 32 {
				zoneID = extracted
			}
		}

		// Generate an import block so the user only needs to fill in the
		// zone_id placeholder (already resolved when it's a literal).
		importBlock := tfhcl.CreateImportBlock("cloudflare_argo_tiered_caching", tieredName, zoneID)
		newBlocks = append(newBlocks, importBlock)

		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Action required: import tiered caching resource for cloudflare_argo.%s", resourceName),
			Detail: fmt.Sprintf(`The cloudflare_argo resource has been split into two separate resources in v5:
  - cloudflare_argo_smart_routing.%s (migrated via moved block — no action needed)
  - cloudflare_argo_tiered_caching.%s (NEW — import block generated)

An import {} block has been added with the zone_id as the import ID.
Before running terraform apply, verify the import block:

  import {
    to = cloudflare_argo_tiered_caching.%s
    id = "%s"
  }

If zone_id is a variable reference, replace <zone_id> with the actual zone ID.`, resourceName, tieredName, tieredName, zoneID),
		})
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
