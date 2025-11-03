package argo

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles migration of Argo resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_argo", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// This is a special case - argo splits into two different resource types
	// We'll return smart_routing as the primary type
	return "cloudflare_argo_smart_routing"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_argo"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for Argo
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := block.Labels()[1]

	var newBlocks []*hclwrite.Block

	// Check for smart_routing attribute
	smartRoutingAttr := body.GetAttribute("smart_routing")
	if smartRoutingAttr != nil {
		// Create cloudflare_argo_smart_routing resource
		smartRoutingBlock := hclwrite.NewBlock("resource", []string{"cloudflare_argo_smart_routing", resourceName})
		smartRoutingBody := smartRoutingBlock.Body()

		// Copy zone_id
		if zoneIdAttr := body.GetAttribute("zone_id"); zoneIdAttr != nil {
			tokens := zoneIdAttr.Expr().BuildTokens(nil)
			smartRoutingBody.SetAttributeRaw("zone_id", tokens)
		}

		// Rename smart_routing to value
		tokens := smartRoutingAttr.Expr().BuildTokens(nil)
		smartRoutingBody.SetAttributeRaw("value", tokens)

		// Copy lifecycle and other nested blocks
		for _, nestedBlock := range body.Blocks() {
			m.copyBlock(smartRoutingBody, nestedBlock)
		}

		newBlocks = append(newBlocks, smartRoutingBlock)

		// Create moved block for smart_routing
		movedBlockSmart := m.createMovedBlock(
			"cloudflare_argo."+resourceName,
			"cloudflare_argo_smart_routing."+resourceName,
		)
		newBlocks = append(newBlocks, movedBlockSmart)
	}

	// Check for tiered_caching attribute
	tieredCachingAttr := body.GetAttribute("tiered_caching")
	if tieredCachingAttr != nil {
		// Create cloudflare_argo_tiered_caching resource with a different name to avoid conflicts
		// when both smart_routing and tiered_caching exist
		tieredResourceName := resourceName
		if smartRoutingAttr != nil {
			// Only append suffix if we have both attributes to avoid name collision
			tieredResourceName = resourceName + "_tiered"
		}

		tieredCachingBlock := hclwrite.NewBlock("resource", []string{"cloudflare_argo_tiered_caching", tieredResourceName})
		tieredCachingBody := tieredCachingBlock.Body()

		// Copy zone_id
		if zoneIdAttr := body.GetAttribute("zone_id"); zoneIdAttr != nil {
			tokens := zoneIdAttr.Expr().BuildTokens(nil)
			tieredCachingBody.SetAttributeRaw("zone_id", tokens)
		}

		// Rename tiered_caching to value
		tokens := tieredCachingAttr.Expr().BuildTokens(nil)
		tieredCachingBody.SetAttributeRaw("value", tokens)

		// Copy lifecycle and other nested blocks
		for _, nestedBlock := range body.Blocks() {
			m.copyBlock(tieredCachingBody, nestedBlock)
		}

		newBlocks = append(newBlocks, tieredCachingBlock)

		// Create moved block for tiered_caching
		movedBlockTiered := m.createMovedBlock(
			"cloudflare_argo."+resourceName,
			"cloudflare_argo_tiered_caching."+tieredResourceName,
		)
		newBlocks = append(newBlocks, movedBlockTiered)
	}

	// If neither attribute exists, just create a smart_routing resource with value = "off"
	if smartRoutingAttr == nil && tieredCachingAttr == nil {
		smartRoutingBlock := hclwrite.NewBlock("resource", []string{"cloudflare_argo_smart_routing", resourceName})
		smartRoutingBody := smartRoutingBlock.Body()

		// Copy zone_id
		if zoneIdAttr := body.GetAttribute("zone_id"); zoneIdAttr != nil {
			tokens := zoneIdAttr.Expr().BuildTokens(nil)
			smartRoutingBody.SetAttributeRaw("zone_id", tokens)
		}

		// Default to "off"
		smartRoutingBody.SetAttributeValue("value", cty.StringVal("off"))

		// Copy lifecycle and other nested blocks
		for _, nestedBlock := range body.Blocks() {
			m.copyBlock(smartRoutingBody, nestedBlock)
		}

		newBlocks = append(newBlocks, smartRoutingBlock)

		// Create moved block
		movedBlock := m.createMovedBlock(
			"cloudflare_argo."+resourceName,
			"cloudflare_argo_smart_routing."+resourceName,
		)
		newBlocks = append(newBlocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         newBlocks,
		RemoveOriginal: true,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// Argo doesn't have state transformations - the resource split is handled at the config level
	// and Terraform's moved blocks handle the state migration
	return stateJSON.String(), nil
}

// copyBlock copies a block to a new body
func (m *V4ToV5Migrator) copyBlock(targetBody *hclwrite.Body, sourceBlock *hclwrite.Block) {
	newBlock := targetBody.AppendNewBlock(sourceBlock.Type(), sourceBlock.Labels())
	newBlockBody := newBlock.Body()

	// Copy all attributes
	for name, attr := range sourceBlock.Body().Attributes() {
		tokens := attr.Expr().BuildTokens(nil)
		newBlockBody.SetAttributeRaw(name, tokens)
	}

	// Recursively copy nested blocks
	for _, nestedBlock := range sourceBlock.Body().Blocks() {
		m.copyBlock(newBlockBody, nestedBlock)
	}
}

// createMovedBlock creates a moved block for state migration
func (m *V4ToV5Migrator) createMovedBlock(from, to string) *hclwrite.Block {
	movedBlock := hclwrite.NewBlock("moved", nil)
	movedBody := movedBlock.Body()

	// Create traversal tokens for 'from' and 'to'
	movedBody.SetAttributeTraversal("from", m.parseTraversal(from))
	movedBody.SetAttributeTraversal("to", m.parseTraversal(to))

	return movedBlock
}

// parseTraversal parses a string like "cloudflare_argo.example" into a traversal
func (m *V4ToV5Migrator) parseTraversal(path string) hcl.Traversal {
	parts := strings.Split(path, ".")
	traversal := hcl.Traversal{}

	if len(parts) > 0 {
		traversal = append(traversal, hcl.TraverseRoot{
			Name: parts[0],
		})
	}

	for i := 1; i < len(parts); i++ {
		traversal = append(traversal, hcl.TraverseAttr{
			Name: parts[i],
		})
	}

	return traversal
}
