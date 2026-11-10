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
		prepareBody(ctx, ruleBlock.Body(), "rules", resourceName)
	}

	if hcl.HasDynamicBlockFor(body, "rules") {
		// v4: dynamic "rules" { for_each = var.rules content { ... } }
		// v5: rules = [for rule in var.rules : { ... }]
		convertBlocksOfType(ctx, body, "rules", resourceName)
	} else {
		for _, ruleBlock := range hcl.FindBlocksByType(body, "rules") {
			warnUnconvertedDynamicBlocks(ctx, ruleBlock.Body(), resourceName)
		}
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

// prepareBody applies the v5 rewrites that the body of a blockType block needs
// before it is turned into an object, then does the same for every block
// nested in it. body is either a static block's body or the content of a
// dynamic block, so a block migrates identically however it was written: a
// dynamic "action_parameters" gets the same headers, query_string and log
// field rewrites as a static one.
func prepareBody(ctx *transform.Context, body *hclwrite.Body, blockType, resourceName string) {
	switch blockType {
	case "action_parameters":
		// In v5, headers is a MapNestedAttribute where the "name" field becomes the map key
		convertHeadersToMap(ctx, body, resourceName)

		// Transform log custom fields: cookie_fields, request_fields, response_fields
		// v4: ["field1", "field2"] -> v5: [{name = "field1"}, {name = "field2"}]
		convertStringArrayToNameObjectArray(body, "cookie_fields")
		convertStringArrayToNameObjectArray(body, "request_fields")
		convertStringArrayToNameObjectArray(body, "response_fields")

		// Transform action_parameters.rules map values from comma-separated strings to lists
		// v4: rules = { ruleset_id = "rule1,rule2,rule3" }
		// v5: rules = { ruleset_id = ["rule1", "rule2", "rule3"] }
		convertRulesMapValuesToLists(body)
	case "custom_key":
		// Multiple query_string blocks within cache_key.custom_key are merged
		// into a single query_string attribute
		mergeQueryStringBlocks(body)
	case "query_string":
		// Only reached for the content of a dynamic "query_string" block;
		// static ones were merged by their custom_key above.
		wrapQueryStringFilters(body)
	}

	for _, block := range body.Blocks() {
		if block.Type() != "dynamic" {
			prepareBody(ctx, block.Body(), block.Type(), resourceName)
		}
	}

	// Nested blocks become object attributes in v5, and a dynamic block
	// cannot appear inside an object value. A dynamic block's content is
	// prepared when it is converted.
	convertNestedDynamicBlocks(ctx, body, resourceName)
}

// wrapQueryStringFilters rewrites the include and exclude lists of a single
// query_string body to the v5 object form, as mergeQueryStringBlocks does for
// static blocks: ["*"] becomes { all = true } and any other value becomes
// { list = <value> }.
func wrapQueryStringFilters(body *hclwrite.Body) {
	for _, name := range []string{"include", "exclude"} {
		attr := body.GetAttribute(name)
		if attr == nil {
			continue
		}
		value := attr.Expr().BuildTokens(nil)
		if isWildcardList(value) {
			body.SetAttributeRaw(name, relex(hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte("{ all = true }")},
			}))
			continue
		}
		wrapped := hclwrite.Tokens{{Type: hclsyntax.TokenIdent, Bytes: []byte("{ list = ")}}
		wrapped = append(wrapped, value...)
		wrapped = append(wrapped, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" }")})
		body.SetAttributeRaw(name, relex(wrapped))
	}
}

// convertHeadersToMap sets the v5 headers map from the static and dynamic
// headers blocks in an action_parameters body. Static blocks form one map
// literal and each dynamic block a for expression; more than one part is
// joined with merge(), so every header is kept:
//
//	headers = merge(
//	  { "x-foo" = { operation = "remove" } },
//	  { for k, v in var.extra : k => { operation = "set", value = v } },
//	)
//
// A dynamic block that cannot be converted is left in place with a warning.
func convertHeadersToMap(ctx *transform.Context, body *hclwrite.Body, resourceName string) {
	convertHeadersBlocksToMap(body)

	var parts []hclwrite.Tokens
	if existing := body.GetAttribute("headers"); existing != nil {
		parts = append(parts, existing.Expr().BuildTokens(nil))
	}
	for {
		tokens, ok := convertDynamicHeadersToMap(ctx, body, resourceName)
		if !ok {
			break
		}
		parts = append(parts, tokens)
	}

	switch {
	case len(parts) == 0:
		return
	case len(parts) == 1:
		body.SetAttributeRaw("headers", parts[0])
	default:
		merged := hclwrite.Tokens{
			{Type: hclsyntax.TokenIdent, Bytes: []byte("merge")},
			{Type: hclsyntax.TokenOParen, Bytes: []byte("(")},
			{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		}
		for _, part := range parts {
			merged = append(merged, part...)
			merged = append(merged,
				&hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")},
				&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		}
		merged = append(merged, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})
		body.SetAttributeRaw("headers", relex(merged))
	}
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

// convertNestedDynamicBlocks converts, for each block type that has at least
// one dynamic block directly in body, all of that type's blocks into a single
// attribute with convertBlocksOfType. Nested bodies are handled by
// prepareBody. dynamic "headers" is left to convertDynamicHeadersToMap.
func convertNestedDynamicBlocks(ctx *transform.Context, body *hclwrite.Body, resourceName string) {
	seen := map[string]bool{}
	for _, block := range body.Blocks() {
		if block.Type() != "dynamic" || len(block.Labels()) == 0 {
			continue
		}
		blockType := block.Labels()[0]
		if blockType == "headers" || seen[blockType] {
			continue
		}
		seen[blockType] = true
		if body.GetAttribute(blockType) != nil {
			// An earlier rewrite already merged the static blocks of this
			// type into an attribute (query_string); setting it again would
			// silently drop them.
			warnDynamicBlockNotMigrated(ctx, resourceName, blockType,
				"it is mixed with static "+blockType+" block(s), which were merged into one "+blockType+" value")
			continue
		}
		convertBlocksOfType(ctx, body, blockType, resourceName)
	}
}

// convertBlocksOfType replaces every blockType block in body, static and
// dynamic, with one blockType attribute, when at least one of them is dynamic.
// The static blocks must already have been through prepareBody.
//
// Terraform expands a dynamic block in place among its static siblings, so the
// blocks keep their source order: each dynamic block becomes a for expression,
// each run of static blocks becomes a list, and more than one part is joined
// with concat(). Order matters for rules, and every block must be kept:
// setting the attribute once per block would keep only the last.
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
//
// A type listed in alwaysArrayFields stays a list. Any other type is a single
// object in v5 and allowed at most one block in v4, so the list is wrapped in
// one(), which returns null for an empty list (no block generated) and fails
// for more than one element, as v4 did:
//
//	ratelimit = one([for ratelimit in [rules.value.ratelimit] : {
//	  period = ratelimit.period
//	}])
func convertBlocksOfType(ctx *transform.Context, body *hclwrite.Body, blockType, resourceName string) {
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
		case block.Type() == blockType:
			static = append(static, buildObject(ctx, block.Body(), resourceName))
			converted = append(converted, block)
		case block.Type() == "dynamic" && len(block.Labels()) > 0 && block.Labels()[0] == blockType:
			tokens, ok, reason := buildDynamicBlockForExpr(ctx, block, blockType, resourceName)
			if !ok {
				warnDynamicBlockNotMigrated(ctx, resourceName, blockType, reason)
				// Converting the rest would drop this block from the list, so
				// leave every block of this type as written for the user to
				// finish by hand.
				return
			}
			flushStatic()
			parts = append(parts, tokens)
			converted = append(converted, block)
		}
	}
	flushStatic()

	expr := parts[0]
	if len(parts) > 1 {
		expr = hcl.BuildConcatExpression(parts)
	}
	if !alwaysArrayFields[blockType] {
		expr = append(hclwrite.Tokens{
			{Type: hclsyntax.TokenIdent, Bytes: []byte("one")},
			{Type: hclsyntax.TokenOParen, Bytes: []byte("(")},
		}, append(expr, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})...)
	}
	body.SetAttributeRaw(blockType, relex(expr))
	for _, block := range converted {
		body.RemoveBlock(block)
	}
}

// buildObject builds the object tokens for a prepared block body, warning
// about any dynamic block still inside it: it would otherwise be written out
// as an attribute named "dynamic", which v5 rejects.
func buildObject(ctx *transform.Context, body *hclwrite.Body, resourceName string) hclwrite.Tokens {
	warnUnconvertedDynamicBlocks(ctx, body, resourceName)
	return hcl.BuildObjectFromBodyWithArrays(body, alwaysArrayFields)
}

// warnUnconvertedDynamicBlocks emits warnDynamicBlockNotMigrated for each
// dynamic block left anywhere in body, unless a warning for that block type
// on this resource was already emitted when the conversion was skipped.
func warnUnconvertedDynamicBlocks(ctx *transform.Context, body *hclwrite.Body, resourceName string) {
	for _, block := range body.Blocks() {
		if block.Type() == "dynamic" && len(block.Labels()) > 0 {
			blockType := block.Labels()[0]
			if !hasDynamicBlockWarning(ctx, resourceName, blockType) {
				warnDynamicBlockNotMigrated(ctx, resourceName, blockType,
					"tf-migrate does not support a dynamic "+blockType+" block in this position")
			}
			continue
		}
		warnUnconvertedDynamicBlocks(ctx, block.Body(), resourceName)
	}
}

func dynamicBlockWarningSummary(resourceName, blockType string) string {
	return fmt.Sprintf("Dynamic %s block not migrated on cloudflare_ruleset.%s", blockType, resourceName)
}

func hasDynamicBlockWarning(ctx *transform.Context, resourceName, blockType string) bool {
	summary := dynamicBlockWarningSummary(resourceName, blockType)
	for _, diag := range ctx.Diagnostics {
		if diag.Summary == summary {
			return true
		}
	}
	return false
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
	prepareBody(ctx, contentBody, blockType, resourceName)

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
	tokens = append(tokens, buildObject(ctx, contentBody, resourceName)...)
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
	example := fmt.Sprintf("%s = [for %s in <for_each> : { ... }]", blockType, blockType)
	if !alwaysArrayFields[blockType] {
		example = fmt.Sprintf("%s = one([for %s in <for_each> : { ... }])", blockType, blockType)
	}
	ctx.Diagnostics = append(ctx.Diagnostics, &hclv2.Diagnostic{
		Severity: hclv2.DiagWarning,
		Summary:  dynamicBlockWarningSummary(resourceName, blockType),
		Detail: fmt.Sprintf(
			"The dynamic %q block could not be converted automatically because %s.\n\n"+
				"%s is an attribute in v5, and a block cannot appear inside an object value, "+
				"so this must be rewritten by hand as a for expression, for example:\n\n"+
				"  %s",
			blockType, reason, blockType, example),
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
//
// It converts the first dynamic "headers" block in body, removes it, and
// returns the map expression, so the caller can repeat until none is left. A
// block that cannot be converted is left in place with a warning, and false is
// returned.
func convertDynamicHeadersToMap(ctx *transform.Context, body *hclwrite.Body, resourceName string) (hclwrite.Tokens, bool) {
	const keyVar, valVar = "k", "v"

	dynBlock := hcl.FindDynamicBlockFor(body, "headers")
	if dynBlock == nil {
		return nil, false
	}

	warn := func(reason string) {
		ctx.Diagnostics = append(ctx.Diagnostics, &hclv2.Diagnostic{
			Severity: hclv2.DiagWarning,
			Summary:  dynamicBlockWarningSummary(resourceName, "headers"),
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

	dynBody := dynBlock.Body()
	forEach := dynBody.GetAttribute("for_each")
	if forEach == nil {
		warn("the dynamic block has no for_each")
		return nil, false
	}
	contentBlock := hcl.FindBlockByType(dynBody, "content")
	if contentBlock == nil {
		warn("the dynamic block has no content block")
		return nil, false
	}
	if len(contentBlock.Body().Blocks()) > 0 {
		warn("the content block contains nested blocks")
		return nil, false
	}

	iterator := "headers"
	if it := dynBody.GetAttribute("iterator"); it != nil {
		iterator = strings.Trim(strings.TrimSpace(string(it.Expr().BuildTokens(nil).Bytes())), `"`)
	}

	// The v4 name attribute supplies the v5 map key.
	nameAttr := contentBlock.Body().GetAttribute("name")
	if nameAttr == nil {
		warn("the content block has no name attribute to use as the map key")
		return nil, false
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

	body.RemoveBlock(dynBlock)
	return tokens, true
}
