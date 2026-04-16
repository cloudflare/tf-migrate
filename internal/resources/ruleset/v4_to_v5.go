package ruleset

import (
	"sort"
	"strings"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates and registers a new V4ToV5 migrator for cloudflare_ruleset
func NewV4ToV5Migrator() {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_ruleset", "v4", "v5", migrator)
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_ruleset"
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_ruleset"
}

func (m *V4ToV5Migrator) GetMinSupportedVersion() string {
	return "4.0.0"
}

func (m *V4ToV5Migrator) GetMaxSupportedVersion() string {
	return "4.999.999"
}

func (m *V4ToV5Migrator) GetTargetVersion() string {
	return "5.0.0"
}

// Preprocess implements the ConfigPreprocessor interface
// No preprocessing needed for ruleset
func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for ruleset
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_ruleset doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_ruleset"}, "cloudflare_ruleset"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Handle dynamic "rules" blocks - convert to for expression
	// v4: dynamic "rules" { for_each = var.rules content { ... } }
	// v5: rules = [for rule in var.rules : { ... }]
	dynamicBlocks := hcl.FindBlocksByType(body, "dynamic")
	for _, dynBlock := range dynamicBlocks {
		labels := dynBlock.Labels()
		if len(labels) > 0 && labels[0] == "rules" {
			convertDynamicRulesToForExpression(body, dynBlock)
		}
	}

	// First, handle special case for headers blocks within action_parameters
	// In v5, headers is a MapNestedAttribute where the "name" field becomes the map key
	rulesBlocks := hcl.FindBlocksByType(body, "rules")
	for _, ruleBlock := range rulesBlocks {
		ruleBody := ruleBlock.Body()
		actionParamsBlocks := hcl.FindBlocksByType(ruleBody, "action_parameters")
		for _, actionParamsBlock := range actionParamsBlocks {
			actionParamsBody := actionParamsBlock.Body()
			convertHeadersBlocksToMap(actionParamsBody)

			// Handle query_string blocks within cache_key.custom_key
			// Multiple query_string blocks need to be merged into a single query_string attribute
			cacheKeyBlocks := hcl.FindBlocksByType(actionParamsBody, "cache_key")
			for _, cacheKeyBlock := range cacheKeyBlocks {
				cacheKeyBody := cacheKeyBlock.Body()
				customKeyBlocks := hcl.FindBlocksByType(cacheKeyBody, "custom_key")
				for _, customKeyBlock := range customKeyBlocks {
					customKeyBody := customKeyBlock.Body()
					mergeQueryStringBlocks(customKeyBody)
				}
			}

			// Transform log custom fields: cookie_fields, request_fields, response_fields
			// v4: ["field1", "field2"] -> v5: [{name = "field1"}, {name = "field2"}]
			convertStringArrayToNameObjectArray(actionParamsBody, "cookie_fields")
			convertStringArrayToNameObjectArray(actionParamsBody, "request_fields")
			convertStringArrayToNameObjectArray(actionParamsBody, "response_fields")

			// Transform action_parameters.rules map values from comma-separated strings to lists
			// v4: rules = { ruleset_id = "rule1,rule2,rule3" }
			// v5: rules = { ruleset_id = ["rule1", "rule2", "rule3"] }
			convertRulesMapValuesToLists(actionParamsBody)
		}
	}

	// Define which nested block types should always be converted to arrays (even with 1 element)
	// These correspond to ListNestedAttribute fields in the v5 schema
	alwaysArrayFields := map[string]bool{
		"rules":           true, // Can refer to both top-level rules OR action_parameters.overrides.rules
		"categories":      true, // action_parameters.overrides.categories is always an array
		"status_code_ttl": true, // action_parameters.edge_ttl.status_code_ttl is always an array
		"algorithms":      true, // action_parameters.algorithms is always an array
	}

	// Convert all nested blocks to attributes recursively.
	// This will handle:
	// 1. Top-level rules blocks -> rules = [...]
	// 2. Nested action_parameters blocks inside each rule -> action_parameters = {...}
	// 3. Nested ratelimit, exposed_credential_check blocks
	// 4. Deeply nested blocks like overrides.rules and overrides.categories
	// Note: headers is NOT in alwaysArrayFields because it's already been converted to a map above
	hcl.ConvertBlockToAttributeWithNestedAndArrays(body, "rules", alwaysArrayFields)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// convertHeadersBlocksToMap converts headers blocks to map syntax
// v4: headers { name = "X-Custom" operation = "set" value = "val" }
// v5: headers = { "X-Custom" = { operation = "set" value = "val" } }
func convertHeadersBlocksToMap(body *hclwrite.Body) {
	headersBlocks := hcl.FindBlocksByType(body, "headers")
	if len(headersBlocks) == 0 {
		return
	}

	// Build a map from headers blocks
	headersMap := make(map[string]*hclwrite.Block)
	for _, headerBlock := range headersBlocks {
		// Extract the "name" attribute
		nameAttr := headerBlock.Body().GetAttribute("name")
		if nameAttr == nil {
			continue
		}

		// Get the header name value (remove quotes if present)
		nameTokens := nameAttr.Expr().BuildTokens(nil)
		headerName := string(nameTokens.Bytes())

		// Store the block for this header name
		headersMap[headerName] = headerBlock
	}

	// Build the map syntax tokens
	// headers = {
	//   "X-Custom" = { operation = "set", value = "val" }
	//   "X-Other" = { operation = "remove" }
	// }
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}},
		{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
	}

	// Sort header names for deterministic output
	var headerNames []string
	for headerName := range headersMap {
		headerNames = append(headerNames, headerName)
	}
	sort.Strings(headerNames)

	for _, headerName := range headerNames {
		headerBlock := headersMap[headerName]
		// Add map key (header name)
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(headerName)})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})

		// Build object for this header (excluding the name attribute)
		headerBody := headerBlock.Body()
		objTokens := hclwrite.Tokens{
			{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}},
			{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
		}

		// Add all attributes except "name" in sorted order for consistency
		attrs := headerBody.Attributes()
		var attrNames []string
		for attrName := range attrs {
			if attrName != "name" {
				attrNames = append(attrNames, attrName)
			}
		}
		// Sort attribute names for consistent output
		sort.Strings(attrNames)

		for _, attrName := range attrNames {
			attr := attrs[attrName]
			objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("      ")}) // indent (6 spaces)
			objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(attrName)})
			objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})
			objTokens = append(objTokens, attr.Expr().BuildTokens(nil)...)
			objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
		}

		objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent (4 spaces)
		objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})

		tokens = append(tokens, objTokens...)
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")}) // indent
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})

	// Set the headers attribute with the map syntax
	body.SetAttributeRaw("headers", tokens)

	// Remove all headers blocks
	for _, headerBlock := range headersBlocks {
		body.RemoveBlock(headerBlock)
	}
}

// mergeQueryStringBlocks merges multiple query_string blocks into a single query_string attribute
// v4: query_string { include = [...] } query_string { exclude = [...] }
// v5: query_string = { include = { list = [...] }, exclude = { list = [...] } }
// Also handles wildcard: include = ["*"] -> include = { all = true }
func mergeQueryStringBlocks(body *hclwrite.Body) {
	queryStringBlocks := hcl.FindBlocksByType(body, "query_string")
	if len(queryStringBlocks) == 0 {
		return
	}

	// Collect include and exclude from all query_string blocks
	var includeTokens hclwrite.Tokens
	var excludeTokens hclwrite.Tokens

	for _, qsBlock := range queryStringBlocks {
		qsBody := qsBlock.Body()

		if includeAttr := qsBody.GetAttribute("include"); includeAttr != nil {
			includeTokens = includeAttr.Expr().BuildTokens(nil)
		}

		if excludeAttr := qsBody.GetAttribute("exclude"); excludeAttr != nil {
			excludeTokens = excludeAttr.Expr().BuildTokens(nil)
		}
	}

	// Build the merged query_string attribute
	// query_string = {
	//   include = { ... }
	//   exclude = { ... }
	// }
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}},
		{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
	}

	// Add include if present
	if len(includeTokens) > 0 {
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("include")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})

		// Check if this is the wildcard case: ["*"]
		includeStr := string(hclwrite.Format(includeTokens.Bytes()))
		if includeStr == "[\"*\"]" || includeStr == "[\n  \"*\",\n]" || includeStr == "[\"*\",]" {
			// Wildcard: convert to { all = true }
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("      ")}) // indent
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("all")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("true")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})
		} else {
			// Non-wildcard: wrap in { list = [...] }
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("      ")}) // indent
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("list")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})
			tokens = append(tokens, includeTokens...)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})
		}
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
	}

	// Add exclude if present
	if len(excludeTokens) > 0 {
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("exclude")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})

		// Wrap in { list = [...] }
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("      ")}) // indent
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("list")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})
		tokens = append(tokens, excludeTokens...)
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")}) // indent
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})

	// Set the merged query_string attribute
	body.SetAttributeRaw("query_string", tokens)

	// Remove all query_string blocks
	for _, qsBlock := range queryStringBlocks {
		body.RemoveBlock(qsBlock)
	}
}

// convertStringArrayToNameObjectArray converts a string array attribute to an array of objects with name field
// v4: field = ["value1", "value2"]
// v5: field = [{name = "value1"}, {name = "value2"}]
func convertStringArrayToNameObjectArray(body *hclwrite.Body, fieldName string) {
	attr := body.GetAttribute(fieldName)
	if attr == nil {
		return
	}

	// Get the current expression tokens
	exprTokens := attr.Expr().BuildTokens(nil)
	exprStr := strings.TrimSpace(string(exprTokens.Bytes()))

	// Check if it's an array expression
	if !strings.HasPrefix(exprStr, "[") || !strings.HasSuffix(exprStr, "]") {
		return
	}

	// Parse the array content (remove brackets and split by comma)
	arrayContent := strings.TrimPrefix(exprStr, "[")
	arrayContent = strings.TrimSuffix(arrayContent, "]")
	arrayContent = strings.TrimSpace(arrayContent)

	if arrayContent == "" {
		return
	}

	// Split by comma and handle quoted strings
	var items []string
	inQuotes := false
	currentItem := ""
	for i := 0; i < len(arrayContent); i++ {
		ch := arrayContent[i]
		if ch == '"' {
			inQuotes = !inQuotes
			currentItem += string(ch)
		} else if ch == ',' && !inQuotes {
			items = append(items, strings.TrimSpace(currentItem))
			currentItem = ""
		} else {
			currentItem += string(ch)
		}
	}
	if currentItem != "" {
		items = append(items, strings.TrimSpace(currentItem))
	}

	// Sort items alphabetically to match API ordering (API reorders these fields)
	sort.Strings(items)

	// Build the new array of objects
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte{'['}},
		{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
	}

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		// Build object: { name = "value" }
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" name = ")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(item)})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte{','}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")}) // indent
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte{']'}})

	// Set the new attribute value
	body.SetAttributeRaw(fieldName, tokens)
}

// convertRulesMapValuesToLists converts action_parameters.rules map values from strings
// to lists of strings.
// v4: rules = { efb7b8c9... = "rule1,rule2,rule3" } or rules = { efb7b8c9... = "rule1" }
// v5: rules = { efb7b8c9... = ["rule1", "rule2", "rule3"] } or rules = { efb7b8c9... = ["rule1"] }
// In v4, action_parameters.rules was a map(string) where values could be single or comma-separated rule IDs.
// In v5, it changed to map(list(string)) where every value must be a list.
func convertRulesMapValuesToLists(body *hclwrite.Body) {
	rulesAttr := body.GetAttribute("rules")
	if rulesAttr == nil {
		return
	}

	// Get the expression tokens and check if it looks like a map literal
	exprTokens := rulesAttr.Expr().BuildTokens(nil)
	exprStr := strings.TrimSpace(string(exprTokens.Bytes()))

	// Only process if it starts with { (a map literal)
	if !strings.HasPrefix(exprStr, "{") {
		return
	}

	// Check if this map has any string values (as opposed to already being lists or other types).
	// We need to distinguish map keys (quoted strings before =) from map values (quoted strings after =).
	// Track whether we've seen an = sign to know if a quoted string is a key or value.
	hasStringValues := false
	afterEqual := false
	for _, token := range exprTokens {
		if token.Type == hclsyntax.TokenEqual {
			afterEqual = true
			continue
		}
		if afterEqual && token.Type == hclsyntax.TokenOQuote {
			hasStringValues = true
			break
		}
		if token.Type == hclsyntax.TokenNewline || token.Type == hclsyntax.TokenOBrace || token.Type == hclsyntax.TokenCBrace {
			afterEqual = false
		}
	}

	if !hasStringValues {
		return
	}

	// Build new tokens converting all map string values to lists.
	// We track whether we're after an = sign to only convert values (not keys).
	var newTokens hclwrite.Tokens
	afterEq := false
	for i := 0; i < len(exprTokens); i++ {
		token := exprTokens[i]

		if token.Type == hclsyntax.TokenEqual {
			afterEq = true
			newTokens = append(newTokens, token)
			continue
		}

		// Reset afterEq on newline or braces (next line is a new key-value pair)
		if token.Type == hclsyntax.TokenNewline || token.Type == hclsyntax.TokenCBrace {
			afterEq = false
			newTokens = append(newTokens, token)
			continue
		}

		// Look for pattern: TokenOQuote TokenQuotedLit TokenCQuote after an = sign
		// These are map values that need to be converted to lists
		if afterEq && token.Type == hclsyntax.TokenOQuote &&
			i+2 < len(exprTokens) &&
			exprTokens[i+1].Type == hclsyntax.TokenQuotedLit &&
			exprTokens[i+2].Type == hclsyntax.TokenCQuote {

			quotedVal := string(exprTokens[i+1].Bytes)

			// Split by comma (handles both single values and comma-separated)
			parts := strings.Split(quotedVal, ",")
			newTokens = append(newTokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})

			for j, part := range parts {
				part = strings.TrimSpace(part)
				if j > 0 {
					newTokens = append(newTokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
					newTokens = append(newTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
				}
				newTokens = append(newTokens, &hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")})
				newTokens = append(newTokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(part)})
				newTokens = append(newTokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")})
			}

			newTokens = append(newTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})

			// Skip the original 3 tokens (OQuote, QuotedLit, CQuote)
			i += 2
			afterEq = false
			continue
		}

		newTokens = append(newTokens, token)
	}

	body.SetAttributeRaw("rules", newTokens)
}

// convertDynamicRulesToForExpression converts dynamic "rules" blocks to rules = [for ...] syntax
// v4: dynamic "rules" { for_each = var.rules content { ... } }
// v5: rules = [for rule in var.rules : { ... }]
func convertDynamicRulesToForExpression(body *hclwrite.Body, dynamicBlock *hclwrite.Block) {
	dynBody := dynamicBlock.Body()

	// Extract for_each expression
	forEachAttr := dynBody.GetAttribute("for_each")
	if forEachAttr == nil {
		return
	}
	forEachExpr := forEachAttr.Expr().BuildTokens(nil)

	// Get iterator name from block label (defaults to the block type name, which is "rules")
	iteratorName := "rules"
	if iteratorAttr := dynBody.GetAttribute("iterator"); iteratorAttr != nil {
		// If iterator is specified, use that name
		iteratorTokens := iteratorAttr.Expr().BuildTokens(nil)
		iteratorName = strings.TrimSpace(string(iteratorTokens.Bytes()))
	}

	// Find the content block
	contentBlocks := hcl.FindBlocksByType(dynBody, "content")
	if len(contentBlocks) == 0 {
		return
	}
	contentBlock := contentBlocks[0]
	contentBody := contentBlock.Body()

	// Build the for expression: [for <iterator> in <for_each> : { ... }]
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte{'['}},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("for ")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte(iteratorName)},
		{Type: hclsyntax.TokenIdent, Bytes: []byte(" in ")},
	}
	tokens = append(tokens, forEachExpr...)
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" : ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})

	// Add all attributes from the content block
	// Need to transform references from iterator.value.field to iterator.field
	attrs := contentBody.Attributes()
	var attrNames []string
	for attrName := range attrs {
		attrNames = append(attrNames, attrName)
	}
	sort.Strings(attrNames)

	for _, attrName := range attrNames {
		attr := attrs[attrName]
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")}) // indent
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(attrName)})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})

		// Transform the expression to remove .value
		// e.g., rules.value.expression -> rules.expression
		exprTokens := attr.Expr().BuildTokens(nil)
		transformedTokens := transformIteratorValueReferences(exprTokens, iteratorName)
		tokens = append(tokens, transformedTokens...)

		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
	}

	// Handle nested blocks in content (like action_parameters)
	for _, nestedBlock := range contentBody.Blocks() {
		tokens = append(tokens, buildNestedBlockTokens(nestedBlock, "    ", iteratorName)...)
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")}) // indent
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte{']'}})

	// Set the rules attribute with the for expression
	body.SetAttributeRaw("rules", tokens)

	// Remove the dynamic block
	body.RemoveBlock(dynamicBlock)
}

// transformIteratorValueReferences transforms iterator.value.field to iterator.field
func transformIteratorValueReferences(tokens hclwrite.Tokens, iteratorName string) hclwrite.Tokens {
	result := hclwrite.Tokens{}
	i := 0
	for i < len(tokens) {
		token := tokens[i]

		// Look for pattern: <iteratorName> . value . <field>
		if i+4 < len(tokens) &&
			string(token.Bytes) == iteratorName &&
			tokens[i+1].Type == hclsyntax.TokenDot &&
			string(tokens[i+2].Bytes) == "value" &&
			tokens[i+3].Type == hclsyntax.TokenDot {
			// Skip the ".value." part (keep iterator and first dot, skip "value" and second dot)
			result = append(result, token)       // iterator name
			result = append(result, tokens[i+1]) // first dot
			// Skip tokens[i+2] (value) and tokens[i+3] (second dot)
			i += 4
			continue
		}

		result = append(result, token)
		i++
	}
	return result
}

// buildNestedBlockTokens recursively builds tokens for nested blocks
func buildNestedBlockTokens(block *hclwrite.Block, indent string, iteratorName string) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent)})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(block.Type())})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})

	blockBody := block.Body()
	attrs := blockBody.Attributes()
	var attrNames []string
	for attrName := range attrs {
		attrNames = append(attrNames, attrName)
	}
	sort.Strings(attrNames)

	for _, attrName := range attrNames {
		attr := attrs[attrName]
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + "  ")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(attrName)})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})

		exprTokens := attr.Expr().BuildTokens(nil)
		transformedTokens := transformIteratorValueReferences(exprTokens, iteratorName)
		tokens = append(tokens, transformedTokens...)

		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
	}

	// Handle nested blocks recursively
	for _, nestedBlock := range blockBody.Blocks() {
		tokens = append(tokens, buildNestedBlockTokens(nestedBlock, indent+"  ", iteratorName)...)
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent)})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})

	return tokens
}
