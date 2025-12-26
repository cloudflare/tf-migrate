// Package hcl provides utilities for transforming HCL blocks and their structure
// during Terraform provider migrations.
package hcl

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// RenameResourceType renames a resource from oldType to newType.
// Returns true if the resource type was renamed, false otherwise.
//
// Example - Renaming legacy resource type:
//
// Before:
//
//	resource "cloudflare_record" "example" {
//	  zone_id = "abc123"
//	  name    = "test"
//	  type    = "A"
//	  value   = "192.0.2.1"
//	}
//
// After calling RenameResourceType(block, "cloudflare_record", "cloudflare_dns_record"):
//
//	resource "cloudflare_dns_record" "example" {
//	  zone_id = "abc123"
//	  name    = "test"
//	  type    = "A"
//	  value   = "192.0.2.1"
//	}
func RenameResourceType(block *hclwrite.Block, oldType, newType string) bool {
	labels := block.Labels()
	if len(labels) >= 2 && labels[0] == oldType {
		labels[0] = newType
		block.SetLabels(labels)
		return true
	}
	return false
}

// GetResourceType returns the resource type from a resource block
func GetResourceType(block *hclwrite.Block) string {
	if block.Type() != "resource" {
		return ""
	}
	labels := block.Labels()
	if len(labels) >= 1 {
		return labels[0]
	}
	return ""
}

// GetResourceName returns the resource name from a resource block
func GetResourceName(block *hclwrite.Block) string {
	if block.Type() != "resource" {
		return ""
	}
	labels := block.Labels()
	if len(labels) >= 2 {
		return labels[1]
	}
	return ""
}

// ConvertBlocksToAttribute converts all blocks of a certain type to an object attribute.
// The preProcess function is called on each block before conversion (can be nil).
//
// Example - Converting data blocks to attribute for CAA records:
//
// Before:
//
//	resource "cloudflare_dns_record" "caa" {
//	  zone_id = "abc123"
//	  name    = "example.com"
//	  type    = "CAA"
//
//	  data {
//	    flags   = "0"
//	    tag     = "issue"
//	    content = "letsencrypt.org"
//	  }
//	}
//
// After calling ConvertBlocksToAttribute(body, "data", "data", preProcess):
//
//	resource "cloudflare_dns_record" "caa" {
//	  zone_id = "abc123"
//	  name    = "example.com"
//	  type    = "CAA"
//	  data = {
//	    flags = "0"
//	    tag   = "issue"
//	    value = "letsencrypt.org"  # Renamed by preProcess
//	  }
//	}
func ConvertBlocksToAttribute(body *hclwrite.Body, blockType, attrName string, preProcess func(*hclwrite.Block)) {
	var blocksToRemove []*hclwrite.Block

	for _, block := range body.Blocks() {
		if block.Type() != blockType {
			continue
		}

		// Apply preprocessing if provided
		if preProcess != nil {
			preProcess(block)
		}

		// Convert block to object tokens
		objTokens := BuildObjectFromBlock(block)
		body.SetAttributeRaw(attrName, objTokens)
		blocksToRemove = append(blocksToRemove, block)
	}

	// Remove the converted blocks
	for _, block := range blocksToRemove {
		body.RemoveBlock(block)
	}
}

// HoistAttributeFromBlock copies an attribute from a nested block to the parent body.
// Returns true if the attribute was hoisted, false otherwise.
//
// Example - Hoisting priority from SRV record data block:
//
// Before:
//
//	resource "cloudflare_dns_record" "srv" {
//	  zone_id = "abc123"
//	  name    = "_sip._tcp"
//	  type    = "SRV"
//
//	  data {
//	    priority = 10
//	    weight   = 60
//	    port     = 5060
//	    target   = "sipserver.example.com"
//	  }
//	}
//
// After calling HoistAttributeFromBlock(body, "data", "priority"):
//
//	resource "cloudflare_dns_record" "srv" {
//	  zone_id  = "abc123"
//	  name     = "_sip._tcp"
//	  type     = "SRV"
//	  priority = 10  # Hoisted from data block
//
//	  data {
//	    weight = 60
//	    port   = 5060
//	    target = "sipserver.example.com"
//	  }
//	}
func HoistAttributeFromBlock(parentBody *hclwrite.Body, blockType, attrName string) bool {
	for _, block := range parentBody.Blocks() {
		if block.Type() != blockType {
			continue
		}
		if block.Body().GetAttribute(attrName) != nil {
			// Only hoist if parent doesn't already have this attribute
			if parentBody.GetAttribute(attrName) == nil {
				CopyAttribute(block.Body(), parentBody, attrName)
				return true
			}
		}
	}
	return false
}

// HoistAttributesFromBlock copies multiple attributes from nested blocks to parent
func HoistAttributesFromBlock(parentBody *hclwrite.Body, blockType string, attrNames ...string) int {
	hoisted := 0
	for _, attrName := range attrNames {
		if HoistAttributeFromBlock(parentBody, blockType, attrName) {
			hoisted++
		}
	}
	return hoisted
}

// FindBlockByType finds the first block of a given type
func FindBlockByType(body *hclwrite.Body, blockType string) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if block.Type() == blockType {
			return block
		}
	}
	return nil
}

// FindBlocksByType finds all blocks of a given type
func FindBlocksByType(body *hclwrite.Body, blockType string) []*hclwrite.Block {
	var blocks []*hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == blockType {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// RemoveBlocksByType removes all blocks of a given type
func RemoveBlocksByType(body *hclwrite.Body, blockType string) int {
	blocks := FindBlocksByType(body, blockType)
	for _, block := range blocks {
		body.RemoveBlock(block)
	}
	return len(blocks)
}

// ProcessBlocksOfType applies a function to all blocks of a given type
func ProcessBlocksOfType(body *hclwrite.Body, blockType string, processor func(*hclwrite.Block) error) error {
	for _, block := range body.Blocks() {
		if block.Type() == blockType {
			if err := processor(block); err != nil {
				return err
			}
		}
	}
	return nil
}

// ConvertSingleBlockToAttribute converts the first block of a type to an attribute
// This is useful when a resource changes from having a single block to an attribute
func ConvertSingleBlockToAttribute(body *hclwrite.Body, blockType, attrName string) bool {
	block := FindBlockByType(body, blockType)
	if block == nil {
		return false
	}

	objTokens := BuildObjectFromBlock(block)

	body.SetAttributeRaw(attrName, objTokens)
	body.RemoveBlock(block)
	return true
}

// ConvertBlocksToAttributeList converts multiple blocks of a certain type to an array attribute.
// The preProcess function is called on each block before conversion (can be nil).
//
// Example - Converting destinations blocks to array attribute:
//
// Before:
//
//	resource "cloudflare_zero_trust_access_application" "example" {
//	  name       = "App"
//	  account_id = "abc123"
//
//	  destinations {
//	    type = "public"
//	    uri  = "https://app.example.com"
//	  }
//
//	  destinations {
//	    type = "private"
//	    cidr = "10.0.0.0/24"
//	  }
//	}
//
// After calling ConvertBlocksToAttributeList(body, "destinations", nil):
//
//	resource "cloudflare_zero_trust_access_application" "example" {
//	  name       = "App"
//	  account_id = "abc123"
//
//	  destinations = [
//	    {
//	      type = "public"
//	      uri  = "https://app.example.com"
//	    },
//	    {
//	      type = "private"
//	      cidr = "10.0.0.0/24"
//	    }
//	  ]
//	}
func ConvertBlocksToAttributeList(body *hclwrite.Body, blockType string, preProcess func(*hclwrite.Block)) bool {
	blocks := FindBlocksByType(body, blockType)
	if len(blocks) == 0 {
		return false
	}

	var arrayTokens hclwrite.Tokens

	// Opening bracket
	arrayTokens = append(arrayTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrack,
		Bytes: []byte("["),
	})

	// Process each block
	for i, block := range blocks {
		// Apply preprocessing if provided
		if preProcess != nil {
			preProcess(block)
		}

		// Add comma and newline for all but first element
		if i > 0 {
			arrayTokens = append(arrayTokens, &hclwrite.Token{
				Type:  hclsyntax.TokenComma,
				Bytes: []byte(","),
			})
		}

		// Add newline before each object
		arrayTokens = append(arrayTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})

		// Convert block to object tokens
		objTokens := BuildObjectFromBlock(block)
		arrayTokens = append(arrayTokens, objTokens...)
	}

	// Closing newline and bracket
	arrayTokens = append(arrayTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})
	arrayTokens = append(arrayTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrack,
		Bytes: []byte("]"),
	})

	// Set the attribute with the array using the same name as the block type
	body.SetAttributeRaw(blockType, arrayTokens)

	// Remove the converted blocks
	for _, block := range blocks {
		body.RemoveBlock(block)
	}

	return true
}

// CreateMovedBlock creates a moved block for resource migration
// This is used when resources are renamed or restructured between provider versions
func CreateMovedBlock(from, to string) *hclwrite.Block {
	block := hclwrite.NewBlock("moved", nil)
	body := block.Body()

	// Create traversals for from and to
	fromParts := strings.Split(from, ".")
	toParts := strings.Split(to, ".")

	// Build from traversal
	fromTraversal := hcl.Traversal{}
	for i, part := range fromParts {
		if i == 0 {
			fromTraversal = append(fromTraversal, hcl.TraverseRoot{Name: part})
		} else {
			fromTraversal = append(fromTraversal, hcl.TraverseAttr{Name: part})
		}
	}

	// Build to traversal
	toTraversal := hcl.Traversal{}
	for i, part := range toParts {
		if i == 0 {
			toTraversal = append(toTraversal, hcl.TraverseRoot{Name: part})
		} else {
			toTraversal = append(toTraversal, hcl.TraverseAttr{Name: part})
		}
	}

	body.SetAttributeTraversal("from", fromTraversal)
	body.SetAttributeTraversal("to", toTraversal)

	return block
}

// CreateImportBlock creates an import block for a resource
// Used for generating import blocks when transforming resources
func CreateImportBlock(resourceType, resourceName, importID string) *hclwrite.Block {
	block := hclwrite.NewBlock("import", nil)
	body := block.Body()

	// Build the "to" value: resource_type.resource_name
	toTokens := BuildResourceReference(resourceType, resourceName)
	body.SetAttributeRaw("to", toTokens)

	// Set the import ID
	body.SetAttributeValue("id", cty.StringVal(importID))

	return block
}

// CreateImportBlockWithTokens creates an import block using raw tokens for the ID
// This variant is useful when the import ID needs to be a template expression
func CreateImportBlockWithTokens(resourceType, resourceName string, idTokens hclwrite.Tokens) *hclwrite.Block {
	block := hclwrite.NewBlock("import", nil)
	body := block.Body()

	// Build the "to" value: resource_type.resource_name
	toTokens := BuildResourceReference(resourceType, resourceName)
	body.SetAttributeRaw("to", toTokens)

	// Set the ID using raw tokens
	body.SetAttributeRaw("id", idTokens)

	return block
}

// BuildObjectFromBlock creates object tokens from a block's attributes
// Useful for converting block syntax to object syntax
func BuildObjectFromBlock(block *hclwrite.Block) hclwrite.Tokens {
	// Get attributes in their original order
	orderedAttrs := AttributesOrdered(block.Body())

	// Build a list of attribute tokens preserving the original order
	var attrs []hclwrite.ObjectAttrTokens

	for _, attrInfo := range orderedAttrs {
		// Create tokens for the attribute name (as a simple identifier)
		nameTokens := hclwrite.TokensForIdentifier(attrInfo.Name)

		// Get the value tokens from the attribute's expression
		valueTokens := attrInfo.Attribute.Expr().BuildTokens(nil)

		attrs = append(attrs, hclwrite.ObjectAttrTokens{
			Name:  nameTokens,
			Value: valueTokens,
		})
	}

	// Use the built-in TokensForObject function to create properly formatted object tokens
	return hclwrite.TokensForObject(attrs)
}

// RemoveEmptyBlocks removes blocks with no attributes or nested blocks
func RemoveEmptyBlocks(body *hclwrite.Body, blockType string) {
	var blocksToRemove []*hclwrite.Block

	for _, block := range body.Blocks() {
		if block.Type() == blockType {
			blockBody := block.Body()
			if len(blockBody.Attributes()) == 0 && len(blockBody.Blocks()) == 0 {
				blocksToRemove = append(blocksToRemove, block)
			}
		}
	}

	for _, block := range blocksToRemove {
		body.RemoveBlock(block)
	}
}

// AttributeTransform defines how to transform attributes from an original block to a new block
type AttributeTransform struct {
	// Copy specifies attributes to copy as-is from original to new block
	Copy []string
	// Rename specifies attributes to copy with a new name: map[oldName]newName
	Rename map[string]string
	// Set specifies new attributes to set with default values: map[name]value
	Set map[string]interface{}
	// CopyMetaArguments specifies whether to copy lifecycle and other meta-argument blocks
	CopyMetaArguments bool
}

// CreateDerivedBlock creates a new block derived from an existing block with attribute transformations.
// This is useful when splitting resources or creating related resources during migration.
//
// Parameters:
//   - original: The source block to derive from
//   - newResourceType: The resource type for the new block (e.g., "cloudflare_argo_smart_routing")
//   - newResourceName: The resource name for the new block
//   - transform: Specification of how to transform attributes
//
// Example - Creating smart_routing block from argo block:
//
//	Before (original block):
//	  resource "cloudflare_argo" "main" {
//	    zone_id        = "abc123"
//	    smart_routing  = "on"
//	    tiered_caching = "on"
//	    lifecycle {
//	      ignore_changes = [smart_routing]
//	    }
//	  }
//
//	Call:
//	  newBlock := CreateDerivedBlock(originalBlock, "cloudflare_argo_smart_routing", "main",
//	    AttributeTransform{
//	      Copy:   []string{"zone_id"},
//	      Rename: map[string]string{"smart_routing": "value"},
//	      CopyMetaArguments: true,
//	    })
//
//	After (new block):
//	  resource "cloudflare_argo_smart_routing" "main" {
//	    zone_id = "abc123"
//	    value   = "on"
//	    lifecycle {
//	      ignore_changes = [smart_routing]
//	    }
//	  }
func CreateDerivedBlock(original *hclwrite.Block, newResourceType, newResourceName string, transform AttributeTransform) *hclwrite.Block {
	// Create new block with the specified type and name
	newBlock := hclwrite.NewBlock("resource", []string{newResourceType, newResourceName})
	newBody := newBlock.Body()
	originalBody := original.Body()

	// Copy specified attributes as-is
	for _, attrName := range transform.Copy {
		CopyAttribute(originalBody, newBody, attrName)
	}

	// Copy and rename specified attributes
	for oldName, newName := range transform.Rename {
		CopyAndRenameAttribute(originalBody, newBody, oldName, newName)
	}

	// Set new attributes with default values
	for name, value := range transform.Set {
		tokens := TokensForSimpleValue(value)
		if tokens != nil {
			newBody.SetAttributeRaw(name, tokens)
		}
	}

	// Copy meta-arguments (lifecycle, provider, etc.) if requested
	if transform.CopyMetaArguments {
		// Compute valid attributes for the new resource (for lifecycle filtering)
		validAttributes := make([]string, 0)
		validAttributes = append(validAttributes, transform.Copy...)
		for _, newName := range transform.Rename {
			validAttributes = append(validAttributes, newName)
		}
		for name := range transform.Set {
			validAttributes = append(validAttributes, name)
		}

		copyMetaArguments(original, newBlock, transform.Rename, validAttributes)
	}

	return newBlock
}

// applyAttributeRenames applies attribute renames to tokens (e.g., in lifecycle ignore_changes lists)
func applyAttributeRenames(tokens hclwrite.Tokens, renames map[string]string) hclwrite.Tokens {
	result := make(hclwrite.Tokens, len(tokens))
	for i, token := range tokens {
		// Check if this is an identifier token that needs to be renamed
		if token.Type == hclsyntax.TokenIdent {
			oldName := string(token.Bytes)
			if newName, ok := renames[oldName]; ok {
				// Create a new token with the renamed identifier
				result[i] = &hclwrite.Token{
					Type:  token.Type,
					Bytes: []byte(newName),
				}
				continue
			}
		}
		// Keep the original token
		result[i] = token
	}
	return result
}

// filterLifecycleTokens removes attribute identifiers from lifecycle expressions that are not in validAttributes
// This is useful when splitting resources - each new resource should only reference its own attributes
func filterLifecycleTokens(tokens hclwrite.Tokens, validAttributes []string) hclwrite.Tokens {
	if len(validAttributes) == 0 {
		return tokens
	}

	// Create a set for faster lookup
	validSet := make(map[string]bool)
	for _, attr := range validAttributes {
		validSet[attr] = true
	}

	// Track if we're inside brackets (array/list)
	var result hclwrite.Tokens
	inArray := false
	skipNext := false

	for i, token := range tokens {
		// Track array boundaries
		if token.Type == hclsyntax.TokenOBrack {
			inArray = true
			result = append(result, token)
			continue
		}
		if token.Type == hclsyntax.TokenCBrack {
			inArray = false
			result = append(result, token)
			continue
		}

		// Skip if we marked this token to skip (e.g., comma before removed identifier)
		if skipNext {
			skipNext = false
			continue
		}

		// Check if this is an identifier inside an array
		if inArray && token.Type == hclsyntax.TokenIdent {
			attrName := string(token.Bytes)
			if !validSet[attrName] {
				// This attribute should be filtered out
				// Also skip the following comma if present
				if i+1 < len(tokens) && tokens[i+1].Type == hclsyntax.TokenComma {
					skipNext = true
				}
				// Or skip the preceding comma if this is the last item
				if len(result) > 0 && result[len(result)-1].Type == hclsyntax.TokenComma {
					// Check if this was the only remaining item after the comma
					hasMoreItems := false
					for j := i + 1; j < len(tokens); j++ {
						if tokens[j].Type == hclsyntax.TokenIdent {
							hasMoreItems = true
							break
						}
						if tokens[j].Type == hclsyntax.TokenCBrack {
							break
						}
					}
					if !hasMoreItems {
						// Remove the trailing comma
						result = result[:len(result)-1]
					}
				}
				continue
			}
		}

		result = append(result, token)
	}

	return result
}

// copyMetaArguments copies lifecycle and other meta-argument blocks from original to new block
// and applies attribute renames within lifecycle blocks, filtering out attributes not in validAttributes
func copyMetaArguments(original, newBlock *hclwrite.Block, attributeRenames map[string]string, validAttributes []string) {
	originalBody := original.Body()
	newBody := newBlock.Body()

	// Copy lifecycle block if it exists
	for _, block := range originalBody.Blocks() {
		if block.Type() == "lifecycle" {
			// Clone the lifecycle block
			lifecycleBlock := newBody.AppendNewBlock("lifecycle", nil)
			lifecycleBody := lifecycleBlock.Body()

			// Copy all attributes from the original lifecycle block
			for name, attr := range block.Body().Attributes() {
				tokens := attr.Expr().BuildTokens(nil)

				// Apply attribute renames within lifecycle attributes (e.g., ignore_changes)
				if len(attributeRenames) > 0 {
					tokens = applyAttributeRenames(tokens, attributeRenames)
				}

				// Filter out invalid attributes from lifecycle blocks
				if len(validAttributes) > 0 {
					tokens = filterLifecycleTokens(tokens, validAttributes)
				}

				lifecycleBody.SetAttributeRaw(name, tokens)
			}
		}
	}

	// Copy other meta-argument blocks (provider, etc.) if any
	for _, block := range originalBody.Blocks() {
		// Skip lifecycle (already handled) and resource blocks
		if block.Type() != "lifecycle" && block.Type() != "resource" {
			newBody.AppendBlock(block)
		}
	}
}

// ConvertBlocksToArrayAttribute converts multiple blocks to an array attribute
// This is useful when migrating from v4 block syntax to v5 array attribute syntax
//
// Example - Converting managed headers:
//
// Before:
//   managed_request_headers {
//     id      = "header_1"
//     enabled = true
//   }
//   managed_request_headers {
//     id      = "header_2"
//     enabled = false
//   }
//
// After calling ConvertBlocksToArrayAttribute(body, "managed_request_headers"):
//   managed_request_headers = [
//     { id = "header_1", enabled = true },
//     { id = "header_2", enabled = false }
//   ]
//
// If no blocks are found and emptyIfNone is true, sets an empty array [].
func ConvertBlocksToArrayAttribute(body *hclwrite.Body, blockType string, emptyIfNone bool) {
	blocks := FindBlocksByType(body, blockType)

	if len(blocks) == 0 {
		if emptyIfNone {
			body.SetAttributeRaw(blockType, TokensForEmptyArray())
		}
		return
	}

	// Convert each block to object tokens
	var objectTokens []hclwrite.Tokens
	for _, block := range blocks {
		objTokens := BuildObjectFromBlock(block)
		objectTokens = append(objectTokens, objTokens)
	}

	// Build array tokens from the objects and set as attribute
	arrayTokens := BuildArrayFromObjects(objectTokens)
	body.SetAttributeRaw(blockType, arrayTokens)

	// Remove all original blocks
	RemoveBlocksByType(body, blockType)
}
