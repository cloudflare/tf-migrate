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
	inQuotedString := false
	templateDepth := 0
	objectDepth := 0
	parenDepth := 0
	bracketDepth := 0
	currentElement := hclwrite.Tokens{}
	currentObject := make(map[string]hclwrite.Tokens)
	currentFieldName := ""
	inFieldValue := false

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		// Handle array boundaries
		if token.Type == hclsyntax.TokenOBrack && !inArray {
			inArray = true
			continue
		}
		if token.Type == hclsyntax.TokenCBrack && inArray && bracketDepth == 0 && templateDepth == 0 && !inQuotedString {
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

		// Track quoted string boundaries to avoid stopping on commas inside strings
		if token.Type == hclsyntax.TokenOQuote {
			inQuotedString = true
			currentElement = append(currentElement, token)
			continue
		}
		if token.Type == hclsyntax.TokenCQuote && templateDepth == 0 {
			inQuotedString = false
			currentElement = append(currentElement, token)
			continue
		}

		// Track template interpolation depth (${...})
		if token.Type == hclsyntax.TokenTemplateInterp {
			templateDepth++
			currentElement = append(currentElement, token)
			continue
		}
		if token.Type == hclsyntax.TokenTemplateSeqEnd {
			templateDepth--
			currentElement = append(currentElement, token)
			continue
		}

		// If we're in a quoted string (but not inside template interpolation), just collect all tokens
		if inQuotedString && templateDepth == 0 {
			currentElement = append(currentElement, token)
			continue
		}

		// Track parentheses depth (for function calls)
		if token.Type == hclsyntax.TokenOParen {
			parenDepth++
			currentElement = append(currentElement, token)
			continue
		}
		if token.Type == hclsyntax.TokenCParen {
			parenDepth--
			currentElement = append(currentElement, token)
			continue
		}

		// Track bracket depth (for nested arrays and index expressions)
		if token.Type == hclsyntax.TokenOBrack {
			bracketDepth++
			currentElement = append(currentElement, token)
			continue
		}
		if token.Type == hclsyntax.TokenCBrack {
			bracketDepth--
			currentElement = append(currentElement, token)
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

		// Handle commas (element separators) - only when at depth 0
		if token.Type == hclsyntax.TokenComma && !inObject && templateDepth == 0 && parenDepth == 0 && bracketDepth == 0 {
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

			// Handle comma (field separator) - only at depth 0
			if token.Type == hclsyntax.TokenComma && templateDepth == 0 && parenDepth == 0 && bracketDepth == 0 {
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
	hasQuote := false
	hasTemplateInterp := false

	for _, token := range tokens {
		if token.Type == hclsyntax.TokenOBrace {
			return "object"
		}
		if token.Type == hclsyntax.TokenOQuote || token.Type == hclsyntax.TokenCQuote {
			hasQuote = true
		}
		// TokenTemplateInterp is ${, TokenTemplateSeqEnd is }
		if token.Type == hclsyntax.TokenTemplateInterp || token.Type == hclsyntax.TokenTemplateSeqEnd {
			hasTemplateInterp = true
		}
		if token.Type == hclsyntax.TokenQuotedLit && !hasTemplateInterp {
			// Only consider it a plain string if there's no template interpolation
			continue
		}
		if token.Type == hclsyntax.TokenNumberLit {
			return "number"
		}
		if token.Type == hclsyntax.TokenIdent && !hasQuote {
			ident := string(token.Bytes)
			if ident == "true" || ident == "false" {
				return "bool"
			}
			return "reference"
		}
	}

	if hasTemplateInterp {
		return "template"
	}
	if hasQuote {
		return "string"
	}
	return "unknown"
}

// cleanTokens normalizes tokens by resetting spacing metadata
// This ensures consistent formatting when building new token sequences
func cleanTokens(tokens hclwrite.Tokens) hclwrite.Tokens {
	cleaned := make(hclwrite.Tokens, len(tokens))
	for i, token := range tokens {
		cleanedToken := &hclwrite.Token{
			Type:  token.Type,
			Bytes: token.Bytes,
		}
		// Clear the SpacesBefore field to ensure consistent spacing in generated output
		cleanedToken.SpacesBefore = 0
		cleaned[i] = cleanedToken
	}
	return cleaned
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

// extractTokensFromBlock extracts raw tokens from a block attribute
// This preserves dynamic references, interpolations, and function calls
func extractTokensFromBlock(block *hclwrite.Block, fieldName string) (hclwrite.Tokens, bool) {
	if attr := block.Body().GetAttribute(fieldName); attr != nil {
		tokens := attr.Expr().BuildTokens(nil)
		if len(tokens) > 0 {
			return cleanTokens(tokens), true
		}
	}
	return nil, false
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

// extractValueTokensFromArrayElement extracts the raw tokens representing a value from an array element
// This preserves interpolations, references, and function calls instead of trying to parse them
func extractValueTokensFromArrayElement(elem ArrayElement) (hclwrite.Tokens, bool) {
	if len(elem.Tokens) == 0 {
		return nil, false
	}

	// Don't trim anything - just return all tokens
	// Template strings, function calls, and index expressions need all their tokens preserved
	// But we need to clean them to remove any ANSI codes
	return cleanTokens(elem.Tokens), true
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
	var blockItemTokens []map[string]hclwrite.Tokens

	for _, block := range body.Blocks() {
		if block.Type() != blockType {
			continue
		}

		// Try to extract plain string values first
		itemAttrs := make(map[string]cty.Value)
		itemTokens := make(map[string]hclwrite.Tokens)
		hasPlainValues := false
		hasTokenValues := false

		// Extract primary field
		if val, ok := extractStringFieldFromBlock(block, primaryField); ok {
			itemAttrs[primaryField] = cty.StringVal(val)
			hasPlainValues = true
		} else if tokens, ok := extractTokensFromBlock(block, primaryField); ok {
			itemTokens[primaryField] = tokens
			hasTokenValues = true
		}

		// Extract optional fields
		for _, fieldName := range optionalFields {
			if val, ok := extractStringFieldFromBlock(block, fieldName); ok {
				itemAttrs[fieldName] = cty.StringVal(val)
				hasPlainValues = true
			} else if tokens, ok := extractTokensFromBlock(block, fieldName); ok {
				itemTokens[fieldName] = tokens
				hasTokenValues = true
			} else if hasPlainValues {
				itemAttrs[fieldName] = cty.NullVal(cty.String)
			} else if hasTokenValues {
				itemTokens[fieldName] = hclwrite.Tokens{
					{Type: hclsyntax.TokenIdent, Bytes: []byte("null")},
				}
			}
		}

		// Add the item using the appropriate format
		if hasPlainValues && !hasTokenValues {
			blockItems = append(blockItems, cty.ObjectVal(itemAttrs))
			modified = true
		} else if hasTokenValues {
			// If we have any tokens (dynamic references), use token-based representation
			// Convert any plain values to tokens
			for field, val := range itemAttrs {
				if _, exists := itemTokens[field]; !exists {
					itemTokens[field] = buildValueTokens(val)
				}
			}
			blockItemTokens = append(blockItemTokens, itemTokens)
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
	var arrayItemTokens []map[string]hclwrite.Tokens
	if attr := body.GetAttribute(arrayAttrName); attr != nil {
		elements := ParseArrayAttribute(attr)

		for _, elem := range elements {
			// First try to extract a plain string value
			if val, ok := extractStringFieldFromArrayElement(elem); ok {
				itemAttrs := make(map[string]cty.Value)
				itemAttrs[primaryField] = cty.StringVal(val)

				// Add null values for optional fields
				for _, fieldName := range optionalFields {
					itemAttrs[fieldName] = cty.NullVal(cty.String)
				}

				arrayItems = append(arrayItems, cty.ObjectVal(itemAttrs))
			} else if tokens, ok := extractValueTokensFromArrayElement(elem); ok {
				// If we can't extract a plain string, preserve the raw tokens (interpolations, references, etc)
				itemTokens := make(map[string]hclwrite.Tokens)
				itemTokens[primaryField] = tokens

				// Add null tokens for optional fields
				for _, fieldName := range optionalFields {
					itemTokens[fieldName] = hclwrite.Tokens{
						{Type: hclsyntax.TokenIdent, Bytes: []byte("null")},
					}
				}

				arrayItemTokens = append(arrayItemTokens, itemTokens)
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
	// If we have any items with raw tokens (interpolations/references), we need to build the attribute manually
	if len(arrayItemTokens) > 0 || len(blockItemTokens) > 0 {
		// Build the array using tokens to preserve expressions
		var tokens hclwrite.Tokens
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte("["),
		})

		// Decide which items to add first based on blocksFirst parameter
		var firstTokenItems, secondTokenItems []map[string]hclwrite.Tokens
		var firstValueItems, secondValueItems []cty.Value

		if blocksFirst {
			firstTokenItems = blockItemTokens
			firstValueItems = blockItems
			secondTokenItems = arrayItemTokens
			secondValueItems = arrayItems
		} else {
			firstTokenItems = arrayItemTokens
			firstValueItems = arrayItems
			secondTokenItems = blockItemTokens
			secondValueItems = blockItems
		}

		// Add first set of token-based items
		for i, itemTokens := range firstTokenItems {
			if i > 0 {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte(","),
				})
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			} else {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			}
			tokens = append(tokens, buildObjectTokensFromMap(itemTokens, primaryField, optionalFields)...)
		}

		// Add first set of value-based items (cty.Values)
		for i, item := range firstValueItems {
			if i > 0 || len(firstTokenItems) > 0 {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte(","),
				})
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			} else {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			}
			tokens = append(tokens, buildObjectTokens(item, primaryField, optionalFields)...)
		}

		// Add second set of token-based items
		for i, itemTokens := range secondTokenItems {
			if i > 0 || len(firstTokenItems) > 0 || len(firstValueItems) > 0 {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte(","),
				})
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			} else {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			}
			tokens = append(tokens, buildObjectTokensFromMap(itemTokens, primaryField, optionalFields)...)
		}

		// Add second set of value-based items
		for i, item := range secondValueItems {
			if i > 0 || len(firstTokenItems) > 0 || len(firstValueItems) > 0 || len(secondTokenItems) > 0 {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte(","),
				})
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			} else {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			}
			tokens = append(tokens, buildObjectTokens(item, primaryField, optionalFields)...)
		}

		// Close the array
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrack,
			Bytes: []byte("]"),
		})

		body.SetAttributeRaw(outputAttrName, tokens)
	} else if len(allItems) > 0 {
		// No items with raw tokens, can use SetAttributeValue
		// Use TupleVal to preserve order (SetVal would reorder elements)
		itemsVal := cty.TupleVal(allItems)
		body.SetAttributeValue(outputAttrName, itemsVal)
	}

	return modified
}

// buildObjectTokens builds HCL tokens for an object from a cty.Value
func buildObjectTokens(item cty.Value, primaryField string, optionalFields []string) hclwrite.Tokens {
	var tokens hclwrite.Tokens

	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte("{"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})

	// Add primary field
	if item.Type().IsObjectType() && item.Type().HasAttribute(primaryField) {
		val := item.GetAttr(primaryField)
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(primaryField),
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte(" = "),
		})
		tokens = append(tokens, buildValueTokens(val)...)
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
	}

	// Add optional fields
	for _, fieldName := range optionalFields {
		if item.Type().IsObjectType() && item.Type().HasAttribute(fieldName) {
			val := item.GetAttr(fieldName)
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte(fieldName),
			})
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenEqual,
				Bytes: []byte(" = "),
			})
			tokens = append(tokens, buildValueTokens(val)...)
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte("\n"),
			})
		}
	}

	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrace,
		Bytes: []byte("}"),
	})

	return tokens
}

// buildObjectTokensFromMap builds HCL tokens for an object from a map of field tokens
func buildObjectTokensFromMap(itemTokens map[string]hclwrite.Tokens, primaryField string, optionalFields []string) hclwrite.Tokens {
	var tokens hclwrite.Tokens

	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte("{"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})

	// Add primary field
	if fieldTokens, ok := itemTokens[primaryField]; ok {
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(primaryField),
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte(" = "),
		})
		tokens = append(tokens, cleanTokens(fieldTokens)...)
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
	}

	// Add optional fields
	for _, fieldName := range optionalFields {
		if fieldTokens, ok := itemTokens[fieldName]; ok {
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte(fieldName),
			})
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenEqual,
				Bytes: []byte(" = "),
			})
			tokens = append(tokens, cleanTokens(fieldTokens)...)
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte("\n"),
			})
		}
	}

	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrace,
		Bytes: []byte("}"),
	})

	return tokens
}

// buildValueTokens builds HCL tokens for a cty.Value
func buildValueTokens(val cty.Value) hclwrite.Tokens {
	var tokens hclwrite.Tokens

	if val.IsNull() {
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("null"),
		})
	} else if val.Type() == cty.String {
		str := val.AsString()
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenOQuote,
			Bytes: []byte("\""),
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenQuotedLit,
			Bytes: []byte(str),
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenCQuote,
			Bytes: []byte("\""),
		})
	}

	return tokens
}

// BuildArrayFromObjects creates array tokens from multiple object tokens
// Useful for converting multiple blocks to an array attribute
func BuildArrayFromObjects(objects []hclwrite.Tokens) hclwrite.Tokens {
	if len(objects) == 0 {
		return TokensForEmptyArray()
	}
	return hclwrite.TokensForTuple(objects)
}
