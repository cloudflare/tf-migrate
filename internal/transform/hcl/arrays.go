// Package hcl provides utilities for transforming HCL arrays and collections
package hcl

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// ArrayElement represents a parsed element from an array
type ArrayElement struct {
	// Tokens for the entire element
	Tokens hclwrite.Tokens
	// Type of element (string, object, etc)
	Type string
	// Parsed fields if this is an object
	Fields map[string]hclwrite.Tokens
}

// ParseArrayAttribute parses an array attribute and extracts its elements
// Returns a slice of ArrayElement structs containing the parsed elements
//
// Example - Parsing a simple string array:
//
//	items = ["value1", "value2"]
//
// Returns two ArrayElement structs with Type="string"
//
// Example - Parsing an object array:
//
//	items = [
//	  { value = "val1", description = "desc1" },
//	  { value = "val2" }
//	]
//
// Returns two ArrayElement structs with Type="object" and Fields populated
func ParseArrayAttribute(attr *hclwrite.Attribute) []ArrayElement {
	if attr == nil {
		return nil
	}

	tokens := attr.Expr().BuildTokens(nil)
	var elements []ArrayElement

	// Find the opening bracket
	inArray := false
	inObject := false
	objectDepth := 0
	currentElement := hclwrite.Tokens{}
	currentObject := make(map[string]hclwrite.Tokens)
	currentFieldName := ""
	inFieldValue := false

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		// Handle array boundaries
		if token.Type == hclsyntax.TokenOBrack {
			inArray = true
			continue
		}
		if token.Type == hclsyntax.TokenCBrack {
			// End of array - save any pending element
			if len(currentElement) > 0 {
				elem := ArrayElement{
					Tokens: currentElement,
					Type:   determineElementType(currentElement),
				}
				elements = append(elements, elem)
			}
			break
		}

		if !inArray {
			continue
		}

		// Handle object boundaries
		if token.Type == hclsyntax.TokenOBrace {
			if !inObject {
				inObject = true
				currentObject = make(map[string]hclwrite.Tokens)
				currentElement = hclwrite.Tokens{}
			}
			objectDepth++
			if objectDepth > 1 {
				currentElement = append(currentElement, token)
			}
			continue
		}

		if token.Type == hclsyntax.TokenCBrace {
			objectDepth--
			if objectDepth == 0 && inObject {
				// End of object
				elem := ArrayElement{
					Type:   "object",
					Fields: currentObject,
				}
				elements = append(elements, elem)
				inObject = false
				currentObject = make(map[string]hclwrite.Tokens)
				currentElement = hclwrite.Tokens{}
			} else if objectDepth > 0 {
				currentElement = append(currentElement, token)
			}
			continue
		}

		// Handle commas (element separators)
		if token.Type == hclsyntax.TokenComma && !inObject {
			if len(currentElement) > 0 {
				elem := ArrayElement{
					Tokens: currentElement,
					Type:   determineElementType(currentElement),
				}
				elements = append(elements, elem)
				currentElement = hclwrite.Tokens{}
			}
			continue
		}

		// Inside an object, parse field names and values
		if inObject && objectDepth == 1 {
			// Skip whitespace and newlines
			if token.Type == hclsyntax.TokenNewline ||
				(token.Type == hclsyntax.TokenIdent && strings.TrimSpace(string(token.Bytes)) == "") {
				continue
			}

			// Handle field name
			if token.Type == hclsyntax.TokenIdent && !inFieldValue {
				currentFieldName = string(token.Bytes)
				continue
			}

			// Handle equals sign
			if token.Type == hclsyntax.TokenEqual {
				inFieldValue = true
				currentElement = hclwrite.Tokens{}
				continue
			}

			// Handle comma (field separator)
			if token.Type == hclsyntax.TokenComma {
				if inFieldValue && currentFieldName != "" {
					currentObject[currentFieldName] = currentElement
					currentFieldName = ""
					inFieldValue = false
					currentElement = hclwrite.Tokens{}
				}
				continue
			}

			// Collect field value tokens
			if inFieldValue {
				currentElement = append(currentElement, token)

				// Check if we've reached the end of this field value
				// (next token is comma, close brace, or newline followed by identifier)
				if i+1 < len(tokens) {
					nextToken := tokens[i+1]
					if nextToken.Type == hclsyntax.TokenComma ||
						nextToken.Type == hclsyntax.TokenCBrace ||
						(nextToken.Type == hclsyntax.TokenNewline && i+2 < len(tokens) &&
							tokens[i+2].Type == hclsyntax.TokenIdent) {
						// Save this field
						currentObject[currentFieldName] = currentElement
						currentFieldName = ""
						inFieldValue = false
						currentElement = hclwrite.Tokens{}
					}
				}
			}
		} else if !inObject {
			// Regular array element - just collect tokens
			currentElement = append(currentElement, token)
		}
	}

	return elements
}

// determineElementType determines the type of an array element based on its tokens
func determineElementType(tokens hclwrite.Tokens) string {
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenOBrace {
			return "object"
		}
		if token.Type == hclsyntax.TokenQuotedLit {
			return "string"
		}
		if token.Type == hclsyntax.TokenNumberLit {
			return "number"
		}
		if token.Type == hclsyntax.TokenIdent {
			ident := string(token.Bytes)
			if ident == "true" || ident == "false" {
				return "bool"
			}
			return "reference"
		}
	}
	return "unknown"
}

// extractStringFieldFromBlock extracts a string field value from a block's attributes
func extractStringFieldFromBlock(block *hclwrite.Block, fieldName string) (string, bool) {
	if attr := block.Body().GetAttribute(fieldName); attr != nil {
		tokens := attr.Expr().BuildTokens(nil)
		for _, token := range tokens {
			if token.Type == hclsyntax.TokenQuotedLit {
				val := string(token.Bytes)
				return strings.Trim(val, "\""), true
			}
		}
	}
	return "", false
}

// extractStringFieldFromArrayElement extracts a string value from an array element
func extractStringFieldFromArrayElement(elem ArrayElement) (string, bool) {
	if elem.Type == "string" {
		for _, token := range elem.Tokens {
			if token.Type == hclsyntax.TokenQuotedLit {
				val := string(token.Bytes)
				return strings.Trim(val, "\""), true
			}
		}
	}
	return "", false
}

// MergeAttributeAndBlocksToObjectArray merges an array attribute and blocks into a single array of objects
//
// This is a generic function that can handle the pattern of:
// - An array attribute with simple values (e.g., items = ["val1", "val2"])
// - Blocks with attributes (e.g., items_with_description { value = "val1", description = "desc1" })
// - Merging them into a single array of objects with consistent fields
//
// Parameters:
//   - body: The HCL body to modify
//   - arrayAttrName: Name of the array attribute to merge (e.g., "items")
//   - blockType: Type of blocks to merge (e.g., "items_with_description")
//   - outputAttrName: Name of the output attribute (e.g., "items")
//   - primaryField: Name of the primary/required field (e.g., "value")
//   - optionalFields: Names of optional fields that should be null if not present (e.g., ["description"])
//   - blocksFirst: If true, blocks are processed first (to match API order), otherwise array elements first
//
// # Returns true if any modifications were made
//
// Example usage for Zero Trust List migration:
//
//	Before:
//	  resource "cloudflare_teams_list" "example" {
//	    items = ["192.168.1.1", "10.0.0.0/8"]
//	    items_with_description {
//	      value       = "172.16.0.0/12"
//	      description = "Private network"
//	    }
//	  }
//
//	Call:
//	  MergeAttributeAndBlocksToObjectArray(body, "items", "items_with_description", "items",
//	                                       "value", []string{"description"}, true)
//
//	After:
//	  resource "cloudflare_zero_trust_list" "example" {
//	    items = [{
//	      description = "Private network"
//	      value       = "172.16.0.0/12"
//	    }, {
//	      description = null
//	      value       = "192.168.1.1"
//	    }, {
//	      description = null
//	      value       = "10.0.0.0/8"
//	    }]
//	  }
//
// Example usage for other resources:
//
//	// Merge tags array and tag_with_metadata blocks
//	MergeAttributeAndBlocksToObjectArray(body, "tags", "tag_with_metadata", "tags",
//	                                     "name", []string{"metadata"}, false)
//
//	// Merge rules array and rule_with_priority blocks with multiple optional fields
//	MergeAttributeAndBlocksToObjectArray(body, "rules", "rule_with_priority", "rules",
//	                                     "expression", []string{"priority", "description"}, true)
func MergeAttributeAndBlocksToObjectArray(
	body *hclwrite.Body,
	arrayAttrName string,
	blockType string,
	outputAttrName string,
	primaryField string,
	optionalFields []string,
	blocksFirst bool,
) bool {
	modified := false
	var allItems []cty.Value

	// Collect items from blocks
	var blocksToRemove []*hclwrite.Block
	var blockItems []cty.Value

	for _, block := range body.Blocks() {
		if block.Type() != blockType {
			continue
		}

		itemAttrs := make(map[string]cty.Value)

		// Extract primary field
		if val, ok := extractStringFieldFromBlock(block, primaryField); ok {
			itemAttrs[primaryField] = cty.StringVal(val)
		}

		// Extract optional fields
		for _, fieldName := range optionalFields {
			if val, ok := extractStringFieldFromBlock(block, fieldName); ok {
				itemAttrs[fieldName] = cty.StringVal(val)
			} else {
				itemAttrs[fieldName] = cty.NullVal(cty.String)
			}
		}

		if len(itemAttrs) > 0 {
			blockItems = append(blockItems, cty.ObjectVal(itemAttrs))
			modified = true
		}
		blocksToRemove = append(blocksToRemove, block)
	}

	// Remove blocks
	for _, block := range blocksToRemove {
		body.RemoveBlock(block)
	}

	// Collect items from array attribute
	var arrayItems []cty.Value
	if attr := body.GetAttribute(arrayAttrName); attr != nil {
		elements := ParseArrayAttribute(attr)

		for _, elem := range elements {
			if val, ok := extractStringFieldFromArrayElement(elem); ok {
				itemAttrs := make(map[string]cty.Value)
				itemAttrs[primaryField] = cty.StringVal(val)

				// Add null values for optional fields
				for _, fieldName := range optionalFields {
					itemAttrs[fieldName] = cty.NullVal(cty.String)
				}

				arrayItems = append(arrayItems, cty.ObjectVal(itemAttrs))
			}
		}

		body.RemoveAttribute(arrayAttrName)
		modified = true
	}

	// Combine items in the requested order
	if blocksFirst {
		allItems = append(allItems, blockItems...)
		allItems = append(allItems, arrayItems...)
	} else {
		allItems = append(allItems, arrayItems...)
		allItems = append(allItems, blockItems...)
	}

	// Create the output attribute with all items as objects
	// Use TupleVal to preserve order (SetVal would reorder elements)
	if len(allItems) > 0 {
		itemsVal := cty.TupleVal(allItems)
		body.SetAttributeValue(outputAttrName, itemsVal)
	}

	return modified
}
