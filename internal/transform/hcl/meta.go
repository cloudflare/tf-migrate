// Package hcl provides utilities for transforming HCL configuration files
package hcl

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// MetaArguments holds Terraform meta-arguments that should be preserved during migrations.
// These are special arguments that affect resource behavior but aren't part of the resource schema.
type MetaArguments struct {
	// Count defines how many instances of this resource to create
	Count *hclwrite.Attribute
	// ForEach creates an instance for each item in a map or set
	ForEach *hclwrite.Attribute
	// Lifecycle controls resource lifecycle behavior
	Lifecycle *hclwrite.Block
	// DependsOn explicitly defines dependencies
	DependsOn *hclwrite.Attribute
	// Provider selects which provider configuration to use
	Provider *hclwrite.Attribute
	// Timeouts customizes operation timeouts
	Timeouts *hclwrite.Block
}

// ExtractMetaArguments extracts all meta-arguments from a resource block.
// Returns a MetaArguments struct with pointers to the attributes/blocks found.
// Fields will be nil if the corresponding meta-argument doesn't exist.
//
// Example:
//
//	resource "cloudflare_zone_setting" "example" {
//	  zone_id = "abc123"
//	  count   = 5
//
//	  lifecycle {
//	    ignore_changes = [modified_on]
//	  }
//	}
//
//	meta := ExtractMetaArguments(block)
//	// meta.Count is non-nil
//	// meta.Lifecycle is non-nil
//	// meta.ForEach is nil
func ExtractMetaArguments(block *hclwrite.Block) *MetaArguments {
	body := block.Body()
	return &MetaArguments{
		Count:     body.GetAttribute("count"),
		ForEach:   body.GetAttribute("for_each"),
		Lifecycle: FindBlockByType(body, "lifecycle"),
		DependsOn: body.GetAttribute("depends_on"),
		Provider:  body.GetAttribute("provider"),
		Timeouts:  FindBlockByType(body, "timeouts"),
	}
}

// CopyMetaArgumentsToBlock copies meta-arguments to a destination block.
// Only non-nil meta-arguments are copied.
//
// Example - Creating a derived resource with preserved meta-arguments:
//
//	originalBlock := ... // has count and lifecycle
//	meta := ExtractMetaArguments(originalBlock)
//
//	newBlock := hclwrite.NewBlock("resource", []string{"cloudflare_new_type", "example"})
//	CopyMetaArgumentsToBlock(newBlock, meta)
//	// newBlock now has count and lifecycle
func CopyMetaArgumentsToBlock(dstBlock *hclwrite.Block, meta *MetaArguments) {
	if meta == nil {
		return
	}

	dstBody := dstBlock.Body()

	if meta.Count != nil {
		dstBody.SetAttributeRaw("count", meta.Count.Expr().BuildTokens(nil))
	}
	if meta.ForEach != nil {
		dstBody.SetAttributeRaw("for_each", meta.ForEach.Expr().BuildTokens(nil))
	}
	if meta.DependsOn != nil {
		dstBody.SetAttributeRaw("depends_on", meta.DependsOn.Expr().BuildTokens(nil))
	}
	if meta.Provider != nil {
		dstBody.SetAttributeRaw("provider", meta.Provider.Expr().BuildTokens(nil))
	}

	// Copy blocks
	if meta.Lifecycle != nil {
		lifecycleBlock := dstBody.AppendNewBlock("lifecycle", nil)
		// Copy all attributes from the lifecycle block
		for name, attr := range meta.Lifecycle.Body().Attributes() {
			lifecycleBlock.Body().SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
		}
	}
	if meta.Timeouts != nil {
		timeoutsBlock := dstBody.AppendNewBlock("timeouts", nil)
		// Copy all attributes from the timeouts block
		for name, attr := range meta.Timeouts.Body().Attributes() {
			timeoutsBlock.Body().SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
		}
	}
}

// CopyMetaArgumentsToImport copies compatible meta-arguments to an import block.
// Import blocks support for_each but not count, and don't support lifecycle/timeouts.
//
// Example - Creating import block with for_each:
//
//	meta := ExtractMetaArguments(resourceBlock)
//	importBlock := hclwrite.NewBlock("import", nil)
//	CopyMetaArgumentsToImport(importBlock, meta)
//	// importBlock has for_each if original had it
func CopyMetaArgumentsToImport(importBlock *hclwrite.Block, meta *MetaArguments) {
	if meta == nil {
		return
	}

	body := importBlock.Body()

	// Import blocks can have for_each but not count
	if meta.ForEach != nil {
		body.SetAttributeRaw("for_each", meta.ForEach.Expr().BuildTokens(nil))
	}

	// provider is also supported on import blocks
	if meta.Provider != nil {
		body.SetAttributeRaw("provider", meta.Provider.Expr().BuildTokens(nil))
	}
}
