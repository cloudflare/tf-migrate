package ruleset

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/hcl"
	hclv2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
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

// alwaysArrayFields lists the nested block types that become list attributes in
// v5 even when only one block is present. These correspond to
// ListNestedAttribute fields in the v5 schema. Every other nested block type
// becomes a single object.
var alwaysArrayFields = map[string]bool{
	"rules":           true, // Can refer to both top-level rules OR action_parameters.overrides.rules
	"categories":      true, // action_parameters.overrides.categories is always an array
	"status_code_ttl": true, // action_parameters.edge_ttl.status_code_ttl is always an array
	"algorithms":      true, // action_parameters.algorithms is always an array
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := hcl.GetResourceName(block)

	// Static rules blocks and the content of dynamic "rules" blocks get the
	// same treatment, so a rule migrates identically however it was written.
	for _, ruleBlock := range hcl.FindBlocksByType(body, "rules") {
		transformRuleBody(ctx, ruleBlock.Body(), resourceName)
	}

	if hcl.HasDynamicBlockFor(body, "rules") {
		// v4: dynamic "rules" { for_each = var.rules content { ... } }
		// v5: rules = [for rule in var.rules : { ... }]
		convertRulesWithDynamicBlocks(ctx, body, resourceName)
	} else {
		// Convert all nested blocks to attributes recursively.
		// This will handle:
		// 1. Top-level rules blocks -> rules = [...]
		// 2. Nested action_parameters blocks inside each rule -> action_parameters = {...}
		// 3. Nested ratelimit, exposed_credential_check blocks
		// 4. Deeply nested blocks like overrides.rules and overrides.categories
		// Note: headers is NOT in alwaysArrayFields because it's already been converted to a map above
		hcl.ConvertBlockToAttributeWithNestedAndArrays(body, "rules", alwaysArrayFields)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// transformRuleBody applies the v5 rewrites that a single rule needs before it
// is turned into an object, whether it came from a static rules block or from
// the content of a dynamic "rules" block.
func transformRuleBody(ctx *transform.Context, ruleBody *hclwrite.Body, resourceName string) {
	// First, handle special case for headers blocks within action_parameters
	// In v5, headers is a MapNestedAttribute where the "name" field becomes the map key
	actionParamsBlocks := hcl.FindBlocksByType(ruleBody, "action_parameters")
	for _, actionParamsBlock := range actionParamsBlocks {
		actionParamsBody := actionParamsBlock.Body()
		convertDynamicHeadersToMap(ctx, actionParamsBody, resourceName)
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

	// Blocks inside a rule, such as ratelimit or action_parameters, become
	// object attributes in v5, and a dynamic block cannot appear inside an
	// object value.
	convertNestedDynamicBlocks(ctx, ruleBody, resourceName)
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
// Also handles wildcard: include/exclude = ["*"] -> include/exclude = { all = true }
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
		if isWildcardList(includeTokens) {
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

		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("      ")}) // indent
		if isWildcardList(excludeTokens) {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("all")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("true")})
		} else {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("list")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" = ")})
			tokens = append(tokens, excludeTokens...)
		}
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

func isWildcardList(tokens hclwrite.Tokens) bool {
	expression, diagnostics := hclsyntax.ParseExpression(tokens.Bytes(), "", hclv2.InitialPos)
	if diagnostics.HasErrors() {
		return false
	}

	tuple, ok := expression.(*hclsyntax.TupleConsExpr)
	if !ok || len(tuple.Exprs) != 1 {
		return false
	}

	value, diagnostics := tuple.Value(nil)
	if diagnostics.HasErrors() || !value.IsKnown() {
		return false
	}

	element := value.Index(cty.NumberIntVal(0))
	return element.RawEquals(cty.StringVal("*"))
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

// convertRulesWithDynamicBlocks sets the v5 rules attribute when at least one
// rule comes from a dynamic block.
//
// Terraform expands a dynamic block in place among its static siblings, so the
// rules keep their source order: each dynamic block becomes a for expression,
// each run of static blocks becomes a list, and more than one part is joined
// with concat(). Rule order matters in a ruleset, and so does keeping every
// rule: setting the attribute once per block would keep only the last.
//
//	rules { ... }
//	dynamic "rules" { for_each = var.rules content { ... } }
//
// becomes:
//
//	rules = concat(
//	  [{ ... }],
//	  [for rules in var.rules : { ... }],
//	)
func convertRulesWithDynamicBlocks(ctx *transform.Context, body *hclwrite.Body, resourceName string) {
	var parts []hclwrite.Tokens
	var static []hclwrite.Tokens
	var converted []*hclwrite.Block

	flushStatic := func() {
		if len(static) > 0 {
			parts = append(parts, hcl.BuildArrayFromObjects(static))
			static = nil
		}
	}

	for _, block := range body.Blocks() {
		switch {
		case block.Type() == "rules":
			static = append(static, hcl.BuildObjectFromBodyWithArrays(block.Body(), alwaysArrayFields))
			converted = append(converted, block)
		case block.Type() == "dynamic" && len(block.Labels()) > 0 && block.Labels()[0] == "rules":
			tokens, ok, reason := buildDynamicBlockForExpr(ctx, block, "rules", resourceName)
			if !ok {
				warnDynamicBlockNotMigrated(ctx, resourceName, "rules", reason)
				// Converting the rest would drop this rule from the list, so
				// leave every rule as written for the user to finish by hand.
				return
			}
			flushStatic()
			parts = append(parts, tokens)
			converted = append(converted, block)
		}
	}
	flushStatic()

	if len(parts) == 1 {
		body.SetAttributeRaw("rules", relex(parts[0]))
	} else {
		body.SetAttributeRaw("rules", relex(hcl.BuildConcatExpression(parts)))
	}
	for _, block := range converted {
		body.RemoveBlock(block)
	}
}

// convertNestedDynamicBlocks rewrites every dynamic block nested in body, at
// any depth, as an attribute holding a for expression. A block type listed in
// alwaysArrayFields becomes a list; any other becomes a single object, and is
// wrapped in one() because v4 allowed at most one such block:
//
//	dynamic "ratelimit" {
//	  for_each = [rules.value.ratelimit]
//	  content {
//	    period = ratelimit.value.period
//	  }
//	}
//
// becomes:
//
//	ratelimit = one([for ratelimit in [rules.value.ratelimit] : {
//	  period = ratelimit.period
//	}])
//
// one() returns null for an empty list, which matches a dynamic block that
// generated no block. dynamic "headers" is left to convertDynamicHeadersToMap.
func convertNestedDynamicBlocks(ctx *transform.Context, body *hclwrite.Body, resourceName string) {
	for _, block := range body.Blocks() {
		if block.Type() != "dynamic" {
			convertNestedDynamicBlocks(ctx, block.Body(), resourceName)
			continue
		}

		labels := block.Labels()
		if len(labels) == 0 || labels[0] == "headers" {
			continue
		}
		blockType := labels[0]

		if len(hcl.FindBlocksByType(body, blockType)) > 0 {
			warnDynamicBlockNotMigrated(ctx, resourceName, blockType, "it is mixed with static "+blockType+" block(s)")
			continue
		}

		tokens, ok, reason := buildDynamicBlockForExpr(ctx, block, blockType, resourceName)
		if !ok {
			warnDynamicBlockNotMigrated(ctx, resourceName, blockType, reason)
			continue
		}
		if !alwaysArrayFields[blockType] {
			tokens = append(hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte("one")},
				{Type: hclsyntax.TokenOParen, Bytes: []byte("(")},
			}, append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})...)
		}
		body.SetAttributeRaw(blockType, relex(tokens))
		body.RemoveBlock(block)
	}
}

// relex re-reads hand-built expression tokens through the HCL lexer, so a
// token that holds several lexical tokens, such as a quoted map key built as
// one identifier, comes back split the way a parsed file would hold it. The
// tokens are returned unchanged if they do not parse.
func relex(tokens hclwrite.Tokens) hclwrite.Tokens {
	file, diags := hclwrite.ParseConfig(append([]byte("x = "), tokens.Bytes()...), "", hclv2.InitialPos)
	if diags.HasErrors() {
		return tokens
	}
	attr := file.Body().GetAttribute("x")
	if attr == nil {
		return tokens
	}
	return attr.Expr().BuildTokens(nil)
}

// buildDynamicBlockForExpr returns the list for expression that replaces a
// dynamic block: [for <iterator> in <for_each> : { <content> }].
//
// The loop variable takes the iterator's name, so <iterator>.value.x becomes
// <iterator>.x and a nested dynamic block shadows an outer one exactly as it
// did in v4. <iterator>.key, when used, becomes a <iterator>_key variable.
// The content gets the same rewrites as a static block, including any dynamic
// blocks nested inside it.
func buildDynamicBlockForExpr(ctx *transform.Context, dynBlock *hclwrite.Block, blockType, resourceName string) (hclwrite.Tokens, bool, string) {
	dynBody := dynBlock.Body()

	forEach := dynBody.GetAttribute("for_each")
	if forEach == nil {
		return nil, false, "the dynamic block has no for_each"
	}
	contentBlock := hcl.FindBlockByType(dynBody, "content")
	if contentBlock == nil {
		return nil, false, "the dynamic block has no content block"
	}
	if dynBody.GetAttribute("labels") != nil {
		return nil, false, "it sets labels, which nested blocks in a ruleset do not take"
	}

	iterator := blockType
	if it := dynBody.GetAttribute("iterator"); it != nil {
		iterator = strings.Trim(strings.TrimSpace(string(it.Expr().BuildTokens(nil).Bytes())), `"`)
	}

	contentBody := contentBlock.Body()
	if blockType == "rules" {
		transformRuleBody(ctx, contentBody, resourceName)
	} else {
		convertNestedDynamicBlocks(ctx, contentBody, resourceName)
	}

	keyVar := iterator + "_key"
	usesKey := rewriteIteratorRefsInBody(contentBody, iterator, keyVar, iterator)

	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("for")},
	}
	if usesKey {
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(keyVar), SpacesBefore: 1},
			&hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
	}
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(iterator), SpacesBefore: 1},
		&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("in"), SpacesBefore: 1},
		&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	// for_each is evaluated in the enclosing scope, so an outer dynamic
	// block's references in it are rewritten along with the outer content.
	tokens = append(tokens, forEach.Expr().BuildTokens(nil)...)
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" : ")})
	tokens = append(tokens, hcl.BuildObjectFromBodyWithArrays(contentBody, alwaysArrayFields)...)
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})

	return tokens, true, ""
}

// rewriteIteratorRefsInBody rewrites <iterator>.key and <iterator>.value in
// every attribute of body and its nested blocks, and reports whether any
// <iterator>.key reference was found.
func rewriteIteratorRefsInBody(body *hclwrite.Body, iterator, keyVar, valVar string) bool {
	usesKey := false
	for name, attr := range body.Attributes() {
		tokens := attr.Expr().BuildTokens(nil)
		if referencesIteratorKey(tokens, iterator) {
			usesKey = true
		}
		body.SetAttributeRaw(name, hcl.RewriteDynamicIteratorRefs(tokens, iterator, keyVar, valVar))
	}
	for _, block := range body.Blocks() {
		if rewriteIteratorRefsInBody(block.Body(), iterator, keyVar, valVar) {
			usesKey = true
		}
	}
	return usesKey
}

// referencesIteratorKey reports whether tokens contain <iterator>.key, matching
// the references hcl.RewriteDynamicIteratorRefs rewrites.
func referencesIteratorKey(tokens hclwrite.Tokens, iterator string) bool {
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenIdent && string(tokens[i].Bytes) == iterator &&
			(i == 0 || tokens[i-1].Type != hclsyntax.TokenDot) &&
			tokens[i+1].Type == hclsyntax.TokenDot &&
			tokens[i+2].Type == hclsyntax.TokenIdent && string(tokens[i+2].Bytes) == "key" {
			return true
		}
	}
	return false
}

func warnDynamicBlockNotMigrated(ctx *transform.Context, resourceName, blockType, reason string) {
	ctx.Diagnostics = append(ctx.Diagnostics, &hclv2.Diagnostic{
		Severity: hclv2.DiagWarning,
		Summary: fmt.Sprintf(
			"Dynamic %s block not migrated on cloudflare_ruleset.%s", blockType, resourceName),
		Detail: fmt.Sprintf(
			"The dynamic %q block could not be converted automatically because %s.\n\n"+
				"%s is an attribute in v5, and a block cannot appear inside an object value, "+
				"so this must be rewritten by hand as a for expression, for example:\n\n"+
				"  %s = [for %s in <for_each> : { ... }]",
			blockType, reason, blockType, blockType, blockType),
	})
}

// convertDynamicHeadersToMap rewrites a dynamic "headers" block inside
// action_parameters into the v5 map form.
//
// action_parameters is an object attribute in v5, and a block cannot appear
// inside an object value, so the dynamic block cannot simply be left alone. In
// v5 headers is keyed by header name, so the v4 name attribute becomes the map
// key and the remaining attributes form the value:
//
//	dynamic "headers" {
//	  for_each = var.response_headers
//	  content {
//	    name      = headers.key
//	    operation = "set"
//	    value     = headers.value
//	  }
//	}
//
// becomes:
//
//	headers = { for k, v in var.response_headers : k => {
//	  operation = "set"
//	  value     = v
//	} }
func convertDynamicHeadersToMap(ctx *transform.Context, body *hclwrite.Body, resourceName string) {
	const keyVar, valVar = "k", "v"

	dynBlock := hcl.FindDynamicBlockFor(body, "headers")
	if dynBlock == nil {
		return
	}

	warn := func(reason string) {
		ctx.Diagnostics = append(ctx.Diagnostics, &hclv2.Diagnostic{
			Severity: hclv2.DiagWarning,
			Summary: fmt.Sprintf(
				"Dynamic headers block not migrated on cloudflare_ruleset.%s", resourceName),
			Detail: fmt.Sprintf(
				"The dynamic \"headers\" block could not be converted automatically "+
					"because %s.\n\n"+
					"v5 expects action_parameters.headers to be a map keyed by header "+
					"name, and a block cannot appear inside an object value, so this "+
					"must be rewritten by hand, for example:\n\n"+
					"  headers = { for k, v in <for_each> : k => {\n"+
					"    operation = \"set\"\n"+
					"    value     = v\n"+
					"  } }",
				reason),
		})
	}

	if len(hcl.FindBlocksByType(body, "headers")) > 0 {
		warn("it is mixed with static headers block(s)")
		return
	}

	dynBody := dynBlock.Body()
	forEach := dynBody.GetAttribute("for_each")
	if forEach == nil {
		warn("the dynamic block has no for_each")
		return
	}
	contentBlock := hcl.FindBlockByType(dynBody, "content")
	if contentBlock == nil {
		warn("the dynamic block has no content block")
		return
	}
	if len(contentBlock.Body().Blocks()) > 0 {
		warn("the content block contains nested blocks")
		return
	}

	iterator := "headers"
	if it := dynBody.GetAttribute("iterator"); it != nil {
		iterator = strings.Trim(strings.TrimSpace(string(it.Expr().BuildTokens(nil).Bytes())), `"`)
	}

	// The v4 name attribute supplies the v5 map key.
	nameAttr := contentBlock.Body().GetAttribute("name")
	if nameAttr == nil {
		warn("the content block has no name attribute to use as the map key")
		return
	}
	keyTokens := hcl.RewriteDynamicIteratorRefs(
		nameAttr.Expr().BuildTokens(nil), iterator, keyVar, valVar)

	ident := func(str string) *hclwrite.Token {
		return &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(str)}
	}
	newline := func() *hclwrite.Token {
		return &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}}
	}

	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}},
		ident(" for " + keyVar + ", " + valVar + " in "),
	}
	tokens = append(tokens, forEach.Expr().BuildTokens(nil)...)
	tokens = append(tokens, ident(" : "))
	tokens = append(tokens, keyTokens...)
	tokens = append(tokens, ident(" => "),
		&hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}}, newline())

	for _, attrInfo := range hcl.AttributesOrdered(contentBlock.Body()) {
		if attrInfo.Name == "name" {
			continue // became the map key
		}
		tokens = append(tokens, ident(attrInfo.Name), ident(" = "))
		tokens = append(tokens, hcl.RewriteDynamicIteratorRefs(
			attrInfo.Attribute.Expr().BuildTokens(nil), iterator, keyVar, valVar)...)
		tokens = append(tokens, newline())
	}

	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}},
		ident(" "),
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})

	body.SetAttributeRaw("headers", tokens)
	body.RemoveBlock(dynBlock)
}
