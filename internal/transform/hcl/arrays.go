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
	elements := []ArrayElement{} // non-nil: empty array returns [] not nil

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

		// Skip comments and bare newlines at the top level — they are not array
		// elements. Newlines inside strings or objects are handled by their own
		// tracking logic below and must not be skipped here.
		if token.Type == hclsyntax.TokenComment {
			continue
		}
		if token.Type == hclsyntax.TokenNewline && !inObject && !inQuotedString && templateDepth == 0 {
			continue
		}

		// Detect a for expression: [for ...] is a comprehension, not a static
		// list of elements. Return nil so callers know to treat it as opaque.
		if inArray && len(currentElement) == 0 && !inObject &&
			token.Type == hclsyntax.TokenIdent && string(token.Bytes) == "for" {
			return nil
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

	// Collect items from blocks (existing behaviour: blockType as HCL block).
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

	// Bug #001 fix: also handle blockType written as an HCL *attribute* rather than
	// a block.  This is valid HCL and used in production configs in two patterns:
	//
	//   Pattern A — opaque expression (local ref, for-expression, concat(), …):
	//     items_with_description = local.my_tunnels
	//   Pattern B — inline object list:
	//     items_with_description = [{ value = "x", description = "y" }, …]
	//
	// When blockType != arrayAttrName we need to look for it as an attribute too.
	if blockType != arrayAttrName {
		if blockAttr := body.GetAttribute(blockType); blockAttr != nil {
			exprTokens := blockAttr.Expr().BuildTokens(nil)

			// Determine whether the expression starts with "[" (array literal) or not.
			isArrayLiteral := false
			for _, tok := range exprTokens {
				if tok.Type == hclsyntax.TokenNewline || tok.Type == hclsyntax.TokenComment {
					continue
				}
				if tok.Type == hclsyntax.TokenOBrack {
					isArrayLiteral = true
				}
				break
			}

			if !isArrayLiteral {
				// Pattern A: opaque expression (local ref, for-expression, function call, …).
				// Cannot be statically evaluated — rename the attribute verbatim to
				// outputAttrName and return immediately.  Merging with arrayAttrName
				// items is not possible without evaluating the expression.
				body.RemoveAttribute(blockType)
				body.SetAttributeRaw(outputAttrName, exprTokens)
				return true
			}

			// Pattern B: inline object list.
			// Use a dedicated token-level parser that correctly extracts all fields
			// (including multi-token values like resource references) without the
			// lossy heuristics in ParseArrayAttribute.
			parsedItems := parseInlineObjectListTokens(exprTokens, primaryField, optionalFields)
			for _, itemTokens := range parsedItems {
				blockItemTokens = append(blockItemTokens, itemTokens)
				modified = true
			}

			body.RemoveAttribute(blockType)
			if !modified {
				// Empty inline list — nothing added, but attribute was removed.
				modified = true
			}
		}
	}

	// Collect items from the primary array attribute (arrayAttrName, e.g. "items").
	var arrayItems []cty.Value
	var arrayItemTokens []map[string]hclwrite.Tokens
	if attr := body.GetAttribute(arrayAttrName); attr != nil {
		elements := ParseArrayAttribute(attr)

		// nil means the array is a for expression (or other opaque expression).
		// Leave it completely untouched — don't remove it, don't rewrite it.
		if elements == nil {
			return modified
		}

		// Empty slice can mean either an empty array literal ([]) OR an array that
		// ParseArrayAttribute couldn't decompose (e.g. already-migrated object list
		// like items = [{ description = null, value = "x" }, ...]).
		// Distinguish the two by checking whether the raw token sequence starts with
		// "[" immediately followed by "{" (object list) vs "[" followed by a quote or
		// ident (simple string/reference array) vs just "[]" (empty).
		hasBlockItems := len(blockItems) > 0 || len(blockItemTokens) > 0

		if len(elements) == 0 {
			// Check whether the existing attr is an already-migrated object list or
			// a genuine empty array.
			rawTokens := attr.Expr().BuildTokens(nil)
			isObjectList := false
			for _, t := range rawTokens {
				if t.Type == hclsyntax.TokenNewline || t.Type == hclsyntax.TokenComment {
					continue
				}
				if t.Type == hclsyntax.TokenOBrack {
					continue
				}
				if t.Type == hclsyntax.TokenOBrace {
					isObjectList = true
				}
				break
			}

			if isObjectList {
				if !hasBlockItems {
					// Nothing to prepend/append — leave the already-migrated attr untouched.
					return modified
				}
				// We have block/blockAttr items to prepend (blocksFirst) or append.
				// Preserve the existing object list as raw token items so they appear in
				// the merged output.  Parse them with parseInlineObjectListTokens.
				existingItems := parseInlineObjectListTokens(rawTokens, primaryField, optionalFields)
				for _, itemTokens := range existingItems {
					arrayItemTokens = append(arrayItemTokens, itemTokens)
				}
				body.RemoveAttribute(arrayAttrName)
				modified = true
			} else {
				// Genuine empty array — remove it (same as before); nothing to merge.
				body.RemoveAttribute(arrayAttrName)
				modified = true
			}
		} else {
			// Non-empty slice of elements. Accumulate simple strings and references.
			// Track whether any element was actually parseable as a simple value.
			parsedAny := false
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
					parsedAny = true
				} else if tokens, ok := extractValueTokensFromArrayElement(elem); ok {
					// If we can't extract a plain string, preserve the raw tokens
					itemTokens := make(map[string]hclwrite.Tokens)
					itemTokens[primaryField] = tokens

					// Add null tokens for optional fields
					for _, fieldName := range optionalFields {
						itemTokens[fieldName] = hclwrite.Tokens{
							{Type: hclsyntax.TokenIdent, Bytes: []byte("null")},
						}
					}

					arrayItemTokens = append(arrayItemTokens, itemTokens)
					parsedAny = true
				}
			}

			if !parsedAny {
				// ParseArrayAttribute returned non-nil but unparseable elements — this
				// happens for already-migrated object lists like
				//   items = [{ description = null, value = "x" }, ...]
				// where each element has type="object" but no extractable tokens.
				// Treat the same as the isObjectList case above.
				if !hasBlockItems {
					return modified
				}
				rawTokens := attr.Expr().BuildTokens(nil)
				existingItems := parseInlineObjectListTokens(rawTokens, primaryField, optionalFields)
				for _, itemTokens := range existingItems {
					// Append to the SECOND set of items (respecting blocksFirst).
					// We add to arrayItemTokens so they are placed after blockItemTokens
					// when blocksFirst=true, or before when blocksFirst=false.
					arrayItemTokens = append(arrayItemTokens, itemTokens)
				}
				body.RemoveAttribute(arrayAttrName)
				modified = true
			} else {
				body.RemoveAttribute(arrayAttrName)
				modified = true
			}
		}
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

// parseInlineObjectListTokens parses a token sequence of the form
// [ { field = value, … }, { field = value, … }, … ] and returns one
// map[fieldName]tokens entry per object element.
//
// This is used instead of ParseArrayAttribute for the items_with_description
// attribute case because ParseArrayAttribute's object-field extraction is lossy:
// it relies on a lookahead heuristic that misses string-literal field values.
// This parser consumes the full value token sequence for each field correctly.
//
// primaryField and optionalFields are used to populate null tokens for absent
// optional fields, ensuring output objects have a consistent shape.
func parseInlineObjectListTokens(
	tokens hclwrite.Tokens,
	primaryField string,
	optionalFields []string,
) []map[string]hclwrite.Tokens {
	var result []map[string]hclwrite.Tokens

	// State machine:
	//   outside → inside [ → inside { → field name → = → field value tokens → , or }
	inArray := false
	inObject := false
	objectDepth := 0
	arrayBrackDepth := 0
	inQuote := false
	templateDepth := 0

	currentObject := make(map[string]hclwrite.Tokens)
	currentFieldName := ""
	var currentValueTokens hclwrite.Tokens
	inFieldValue := false

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		// Outermost "[" opens the array.
		if !inArray {
			if tok.Type == hclsyntax.TokenOBrack {
				inArray = true
			}
			continue
		}

		// Outermost "]" closes the array.
		if tok.Type == hclsyntax.TokenCBrack && !inObject && arrayBrackDepth == 0 && !inQuote && templateDepth == 0 {
			break
		}

		// Track string quoting.
		if tok.Type == hclsyntax.TokenOQuote {
			inQuote = true
			if inFieldValue {
				currentValueTokens = append(currentValueTokens, tok)
			}
			continue
		}
		if tok.Type == hclsyntax.TokenCQuote && templateDepth == 0 {
			inQuote = false
			if inFieldValue {
				currentValueTokens = append(currentValueTokens, tok)
			}
			continue
		}
		if inQuote && templateDepth == 0 {
			if inFieldValue {
				currentValueTokens = append(currentValueTokens, tok)
			}
			continue
		}

		// Track template interpolation depth inside strings.
		if tok.Type == hclsyntax.TokenTemplateInterp {
			templateDepth++
			if inFieldValue {
				currentValueTokens = append(currentValueTokens, tok)
			}
			continue
		}
		if tok.Type == hclsyntax.TokenTemplateSeqEnd {
			templateDepth--
			if inFieldValue {
				currentValueTokens = append(currentValueTokens, tok)
			}
			continue
		}

		// Object open "{".
		if tok.Type == hclsyntax.TokenOBrace {
			if !inObject {
				inObject = true
				objectDepth = 1
				currentObject = make(map[string]hclwrite.Tokens)
				currentFieldName = ""
				currentValueTokens = nil
				inFieldValue = false
			} else {
				// Nested brace inside a field value.
				objectDepth++
				if inFieldValue {
					currentValueTokens = append(currentValueTokens, tok)
				}
			}
			continue
		}

		// Object close "}".
		if tok.Type == hclsyntax.TokenCBrace {
			if inObject {
				objectDepth--
				if objectDepth == 0 {
					// Flush any pending field value.
					if inFieldValue && currentFieldName != "" {
						currentObject[currentFieldName] = trimTrailingWhitespace(currentValueTokens)
					}
					// Save the completed object.
					result = append(result, buildNormalisedItemTokens(currentObject, primaryField, optionalFields))
					inObject = false
					currentFieldName = ""
					currentValueTokens = nil
					inFieldValue = false
				} else {
					// Nested brace closing inside a field value.
					if inFieldValue {
						currentValueTokens = append(currentValueTokens, tok)
					}
				}
			}
			continue
		}

		if !inObject {
			// Between objects (commas, newlines) — skip.
			continue
		}

		// Inside an object at depth 1.
		if objectDepth == 1 {
			// A newline while reading a field value may signal end-of-field when
			// the next non-whitespace token is another identifier (next field name).
			// We look ahead to decide: if so, flush current field and switch to
			// field-name mode. If the next non-whitespace token is NOT an ident
			// (e.g. it's a quote or another token type), the newline is part of the
			// value and we continue accumulating.
			if tok.Type == hclsyntax.TokenNewline && inFieldValue && !inQuote && templateDepth == 0 {
				// Look ahead past any additional newlines to the next meaningful token.
				nextMeaningfulIdx := -1
				for j := i + 1; j < len(tokens); j++ {
					tt := tokens[j].Type
					if tt != hclsyntax.TokenNewline && tt != hclsyntax.TokenComment {
						nextMeaningfulIdx = j
						break
					}
				}
				if nextMeaningfulIdx >= 0 {
					nextType := tokens[nextMeaningfulIdx].Type
					// If next token is an ident or a closing brace, this newline ends the field.
					if nextType == hclsyntax.TokenIdent || nextType == hclsyntax.TokenCBrace {
						currentObject[currentFieldName] = trimTrailingWhitespace(currentValueTokens)
						currentFieldName = ""
						currentValueTokens = nil
						inFieldValue = false
						continue
					}
				}
				// Otherwise the newline is interior to the value (e.g. heredoc) — skip it.
				continue
			}

			// Skip bare newlines and whitespace between fields.
			if tok.Type == hclsyntax.TokenNewline && !inFieldValue {
				continue
			}

			// Field name identifier (when not yet reading a value).
			if tok.Type == hclsyntax.TokenIdent && !inFieldValue {
				currentFieldName = string(tok.Bytes)
				continue
			}

			// "=" separator starts the value.
			if tok.Type == hclsyntax.TokenEqual && currentFieldName != "" && !inFieldValue {
				inFieldValue = true
				currentValueTokens = nil
				continue
			}

			// Comma at depth 1 separates fields inside the object.
			if tok.Type == hclsyntax.TokenComma && inFieldValue && objectDepth == 1 && !inQuote && templateDepth == 0 {
				currentObject[currentFieldName] = trimTrailingWhitespace(currentValueTokens)
				currentFieldName = ""
				currentValueTokens = nil
				inFieldValue = false
				continue
			}

			// Collect field value tokens.
			if inFieldValue {
				// Skip leading whitespace/newline at the very start of a value.
				if len(currentValueTokens) == 0 &&
					(tok.Type == hclsyntax.TokenNewline || tok.Type == hclsyntax.TokenComment) {
					continue
				}
				currentValueTokens = append(currentValueTokens, tok)
			}
		} else {
			// Inside a nested brace — just accumulate tokens.
			if inFieldValue {
				currentValueTokens = append(currentValueTokens, tok)
			}
		}
	}

	return result
}

// buildNormalisedItemTokens builds a map[fieldName]tokens from a parsed object,
// inserting null tokens for optional fields that were absent.
func buildNormalisedItemTokens(
	obj map[string]hclwrite.Tokens,
	primaryField string,
	optionalFields []string,
) map[string]hclwrite.Tokens {
	result := make(map[string]hclwrite.Tokens)

	if fieldTokens, ok := obj[primaryField]; ok {
		result[primaryField] = cleanTokens(fieldTokens)
	}
	for _, fieldName := range optionalFields {
		if fieldTokens, ok := obj[fieldName]; ok {
			result[fieldName] = cleanTokens(fieldTokens)
		} else {
			result[fieldName] = hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte("null")},
			}
		}
	}

	return result
}

// trimTrailingWhitespace removes trailing newline/whitespace tokens from a token slice.
func trimTrailingWhitespace(tokens hclwrite.Tokens) hclwrite.Tokens {
	end := len(tokens)
	for end > 0 {
		tt := tokens[end-1].Type
		if tt == hclsyntax.TokenNewline || tt == hclsyntax.TokenComment {
			end--
		} else {
			break
		}
	}
	return tokens[:end]
}
