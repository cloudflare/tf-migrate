// Package hcl provides utilities for transforming HCL blocks and their structure
// during Terraform provider migrations.
package hcl

import (
	"sort"
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

// ConvertBlockToAttributeWithNested converts a block to attribute format, handling nested blocks
// MaxItems:1 blocks → single object
// TypeList blocks (multiple with same name) → array of objects
// This recursively processes all nested blocks
func ConvertBlockToAttributeWithNested(body *hclwrite.Body, blockName string) {
	ConvertBlockToAttributeWithNestedAndArrays(body, blockName, nil)
}

// ConvertBlockToAttributeWithNestedAndArrays converts blocks to attributes with explicit array field specification
// alwaysArrayFields: map of block types that should always be arrays (even with 1 element)
func ConvertBlockToAttributeWithNestedAndArrays(body *hclwrite.Body, blockName string, alwaysArrayFields map[string]bool) {
	blocks := FindBlocksByType(body, blockName)
	if len(blocks) == 0 {
		return
	}

	// Check if this block type should always be an array (even with 1 element)
	forceArray := alwaysArrayFields != nil && alwaysArrayFields[blockName]

	// Group blocks by their type to identify TypeList vs MaxItems:1
	// For ruleset, we know action_parameters is MaxItems:1, but nested blocks vary
	if len(blocks) == 1 && !forceArray {
		// MaxItems:1 - convert to single object
		block := blocks[0]
		tokens := buildObjectFromBlockRecursiveWithArrays(block.Body(), 0, alwaysArrayFields)
		body.SetAttributeRaw(blockName, tokens)
		body.RemoveBlock(block)
	} else {
		// TypeList - convert to array of objects
		var arrayElements []hclwrite.Tokens
		for _, block := range blocks {
			objTokens := buildObjectFromBlockRecursiveWithArrays(block.Body(), 0, alwaysArrayFields)
			arrayElements = append(arrayElements, objTokens)
		}

		// Build array tokens
		tokens := hclwrite.Tokens{
			{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
			{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		}
		for i, elem := range arrayElements {
			// Add indentation
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")})
			tokens = append(tokens, elem...)
			if i < len(arrayElements)-1 {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
			}
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		}
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})

		body.SetAttributeRaw(blockName, tokens)
		for _, block := range blocks {
			body.RemoveBlock(block)
		}
	}
}

// buildObjectFromBlockRecursive builds object tokens from a block body,
// recursively converting nested blocks to either objects or arrays
func buildObjectFromBlockRecursive(body *hclwrite.Body, indentLevel int) hclwrite.Tokens {
	return buildObjectFromBlockRecursiveWithArrays(body, indentLevel, nil)
}

// buildObjectFromBlockRecursiveWithArrays builds object tokens with explicit array field specification
func buildObjectFromBlockRecursiveWithArrays(body *hclwrite.Body, indentLevel int, alwaysArrayFields map[string]bool) hclwrite.Tokens {
	indent := strings.Repeat("  ", indentLevel)
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	}

	// First, add all attributes
	orderedAttrs := AttributesOrdered(body)
	for _, attrInfo := range orderedAttrs {
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + "  " + attrInfo.Name)})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
		tokens = append(tokens, attrInfo.Attribute.Expr().BuildTokens(nil)...)
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
	}

	// Then, handle nested blocks - group by type
	blocksByType := make(map[string][]*hclwrite.Block)
	for _, block := range body.Blocks() {
		blocksByType[block.Type()] = append(blocksByType[block.Type()], block)
	}

	// Sort block types for deterministic output
	var blockTypes []string
	for blockType := range blocksByType {
		blockTypes = append(blockTypes, blockType)
	}
	sort.Strings(blockTypes)

	// Process each block type in sorted order
	for _, blockType := range blockTypes {
		blocks := blocksByType[blockType]
		// Check if this block type should always be an array (even with 1 element)
		forceArray := alwaysArrayFields != nil && alwaysArrayFields[blockType]

		if len(blocks) == 1 && !forceArray {
			// MaxItems:1 - single nested object
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + "  " + blockType)})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
			nestedTokens := buildObjectFromBlockRecursiveWithArrays(blocks[0].Body(), indentLevel+1, alwaysArrayFields)
			tokens = append(tokens, nestedTokens...)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		} else {
			// TypeList - array of objects (or forceArray is true)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + "  " + blockType)})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

			for i, block := range blocks {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + "    ")})
				nestedTokens := buildObjectFromBlockRecursiveWithArrays(block.Body(), indentLevel+1, alwaysArrayFields)
				tokens = append(tokens, nestedTokens...)
				if i < len(blocks)-1 {
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
				}
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			}

			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + "  ]")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		}
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + "}")})
	return tokens
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

// ConvertDynamicBlocksToForExpression converts dynamic blocks to for expressions
// This handles the case where v4 used dynamic blocks and v5 uses array attributes with for expressions
//
// Example:
//
// Before:
//   dynamic "origins" {
//     for_each = local.origin_configs
//     content {
//       name    = origins.value.name
//       address = origins.value.address
//     }
//   }
//
// After:
//   origins = [for value in local.origin_configs : {
//     name    = value.name
//     address = value.address
//   }]
func ConvertDynamicBlocksToForExpression(body *hclwrite.Body, targetBlockType string) {
	// Find all dynamic blocks
	dynamicBlocks := FindBlocksByType(body, "dynamic")

	for _, dynamicBlock := range dynamicBlocks {
		// Check if this dynamic block's label matches our target
		labels := dynamicBlock.Labels()
		if len(labels) == 0 || labels[0] != targetBlockType {
			continue
		}

		dynamicBody := dynamicBlock.Body()

		// Get the for_each expression
		forEachAttr := dynamicBody.GetAttribute("for_each")
		if forEachAttr == nil {
			continue
		}
		forEachTokens := forEachAttr.Expr().BuildTokens(nil)

		// Get the iterator name (default is the block label)
		iteratorName := targetBlockType
		if iteratorAttr := dynamicBody.GetAttribute("iterator"); iteratorAttr != nil {
			// Extract the iterator name from the attribute
			iterTokens := iteratorAttr.Expr().BuildTokens(nil)
			for _, token := range iterTokens {
				if token.Type == hclsyntax.TokenIdent {
					iteratorName = string(token.Bytes)
					break
				}
			}
		}

		// Find the content block
		contentBlock := FindBlockByType(dynamicBody, "content")
		if contentBlock == nil {
			continue
		}

		// Before building the object, convert any nested blocks to attributes
		contentBody := contentBlock.Body()

		// Convert nested blocks to attributes (e.g., header block -> header attribute)
		for _, nestedBlock := range contentBody.Blocks() {
			nestedBlockType := nestedBlock.Type()
			objTokens := BuildObjectFromBlock(nestedBlock)
			contentBody.SetAttributeRaw(nestedBlockType, objTokens)
			contentBody.RemoveBlock(nestedBlock)
		}

		// Build the for expression
		var forExprTokens hclwrite.Tokens

		// Opening bracket
		forExprTokens = append(forExprTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte("["),
		})

		// "for value in"
		forExprTokens = append(forExprTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("for"),
		})
		forExprTokens = append(forExprTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("value"),
		})
		forExprTokens = append(forExprTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("in"),
		})

		// Add the for_each expression
		forExprTokens = append(forExprTokens, forEachTokens...)

		// Add colon
		forExprTokens = append(forExprTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenColon,
			Bytes: []byte(":"),
		})

		// Build the object from content block
		objTokens := BuildObjectFromBlock(contentBlock)

		// Replace iterator references (e.g., origins.value -> value)
		objTokens = replaceIteratorReferences(objTokens, iteratorName)

		forExprTokens = append(forExprTokens, objTokens...)

		// Closing bracket
		forExprTokens = append(forExprTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrack,
			Bytes: []byte("]"),
		})

		// Set the attribute with the for expression
		body.SetAttributeRaw(targetBlockType, forExprTokens)

		// Remove the dynamic block
		body.RemoveBlock(dynamicBlock)
	}
}

// replaceIteratorReferences replaces iterator references in tokens
// For example, replaces "origins.value.name" with "value.name"
func replaceIteratorReferences(tokens hclwrite.Tokens, iteratorName string) hclwrite.Tokens {
	result := make(hclwrite.Tokens, 0, len(tokens))
	i := 0

	for i < len(tokens) {
		token := tokens[i]

		// Look for pattern: iteratorName DOT value DOT ...
		// We want to remove the "iteratorName DOT" part
		if token.Type == hclsyntax.TokenIdent && string(token.Bytes) == iteratorName {
			// Check if next token is a dot
			if i+1 < len(tokens) && tokens[i+1].Type == hclsyntax.TokenDot {
				// Check if token after dot is "value"
				if i+2 < len(tokens) && tokens[i+2].Type == hclsyntax.TokenIdent && string(tokens[i+2].Bytes) == "value" {
					// Skip the iterator name and the first dot, keep "value"
					// This converts "origins.value.name" to "value.name"
					i++ // Skip iterator name
					i++ // Skip first dot
					// Continue with "value"
					continue
				}
			}
		}

		result = append(result, token)
		i++
	}

	return result
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
//
// NOTE: This function does NOT automatically convert dynamic blocks to for expressions.
// If you need to convert dynamic blocks, call ConvertDynamicBlocksToForExpression separately before this function.
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
