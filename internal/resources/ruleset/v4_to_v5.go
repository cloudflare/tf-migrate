package ruleset

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_ruleset", "cloudflare_ruleset"
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
		}
	}

	// Define which nested block types should always be converted to arrays (even with 1 element)
	// These correspond to ListNestedAttribute fields in the v5 schema
	alwaysArrayFields := map[string]bool{
		"rules":            true, // Can refer to both top-level rules OR action_parameters.overrides.rules
		"categories":       true, // action_parameters.overrides.categories is always an array
		"status_code_ttl":  true, // action_parameters.edge_ttl.status_code_ttl is always an array
		"algorithms":       true, // action_parameters.algorithms is always an array
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

// transformQueryStringInState transforms query_string within cache_key.custom_key in state
// v4: include = ["param1"] or include = ["*"]
// v5: include = { list = ["param1"] } or include = { all = true }
func transformQueryStringInState(jsonStr string, actionParamsPath string) string {
	result := jsonStr
	queryStringPath := actionParamsPath + ".cache_key.custom_key.query_string"
	queryString := gjson.Parse(result).Get(queryStringPath)

	if !queryString.Exists() || !queryString.IsObject() {
		return result
	}

	// Transform include if present
	if includeVal := queryString.Get("include"); includeVal.Exists() {
		if includeVal.IsArray() {
			includeArray := includeVal.Array()

			// Check for wildcard case: ["*"]
			if len(includeArray) == 1 && includeArray[0].String() == "*" {
				// Convert to { all = true }
				// Only set the field that has a value to avoid ExactlyOneOf validation issues
				newInclude := map[string]interface{}{
					"all": true,
				}
				result, _ = sjson.Set(result, queryStringPath+".include", newInclude)
			} else {
				// Convert to { list = [...] }
				// Only set the field that has a value to avoid ExactlyOneOf validation issues
				newInclude := map[string]interface{}{
					"list": includeVal.Value(),
				}
				result, _ = sjson.Set(result, queryStringPath+".include", newInclude)
			}
		} else if includeVal.String() == "*" {
			// Handle wildcard as string: "*" -> { all = true }
			newInclude := map[string]interface{}{
				"all": true,
			}
			result, _ = sjson.Set(result, queryStringPath+".include", newInclude)
		}
	}

	// Transform exclude if present
	if excludeVal := queryString.Get("exclude"); excludeVal.Exists() && excludeVal.IsArray() {
		// Convert to { list = [...] }
		// Note: exclude only has 'list', not 'all', so we just set the list
		newExclude := map[string]interface{}{"list": excludeVal.Value()}
		result, _ = sjson.Set(result, queryStringPath+".exclude", newExclude)
	}

	return result
}

// transformCookieFields transforms cookie_fields from string array to object array
// v4: ["session_id", "user_token"]
// v5: [{name: "session_id"}, {name: "user_token"}]
func transformCookieFields(jsonStr string, actionParamsPath string) string {
	return transformFieldNameArray(jsonStr, actionParamsPath+".cookie_fields")
}

// transformRequestFields transforms request_fields from string array to object array
// v4: ["cf.bot_score", "http.user_agent"]
// v5: [{name: "cf.bot_score"}, {name: "http.user_agent"}]
func transformRequestFields(jsonStr string, actionParamsPath string) string {
	return transformFieldNameArray(jsonStr, actionParamsPath+".request_fields")
}

// transformResponseFields transforms response_fields from string array to object array
// v4: ["status_code", "content_type"]
// v5: [{name: "status_code"}, {name: "content_type"}]
func transformResponseFields(jsonStr string, actionParamsPath string) string {
	return transformFieldNameArray(jsonStr, actionParamsPath+".response_fields")
}

// transformFieldNameArray is a helper function that converts a string array to an object array with name field
// v4: ["value1", "value2"]
// v5: [{name: "value1"}, {name: "value2"}]
func transformFieldNameArray(jsonStr string, fieldPath string) string {
	result := jsonStr
	fieldValue := gjson.Parse(result).Get(fieldPath)

	if !fieldValue.Exists() || !fieldValue.IsArray() {
		return result
	}

	// Convert string array to object array with name field
	var newFields []map[string]interface{}
	for _, item := range fieldValue.Array() {
		if item.Type == gjson.String {
			newFields = append(newFields, map[string]interface{}{
				"name": item.String(),
			})
		}
	}

	// Only update if we actually converted something
	if len(newFields) > 0 {
		result, _ = sjson.Set(result, fieldPath, newFields)
	}

	return result
}

// transformStatusCodeTTLNumericFields ensures numeric fields in status_code_ttl are float64
// Handles edge_ttl.status_code_ttl array items with status_code, value, and status_code_range
func transformStatusCodeTTLNumericFields(jsonStr string, actionParamsPath string) string {
	result := jsonStr
	edgeTTLPath := actionParamsPath + ".edge_ttl.status_code_ttl"
	statusCodeTTL := gjson.Parse(result).Get(edgeTTLPath)

	if !statusCodeTTL.Exists() || !statusCodeTTL.IsArray() {
		return result
	}

	// Process each item in the status_code_ttl array
	for idx, item := range statusCodeTTL.Array() {
		itemPath := fmt.Sprintf("%s.%d", edgeTTLPath, idx)

		// Convert status_code to float64 if present
		if statusCode := item.Get("status_code"); statusCode.Exists() {
			result, _ = sjson.Set(result, itemPath+".status_code", statusCode.Float())
		}

		// Convert value to float64 if present
		if value := item.Get("value"); value.Exists() {
			result, _ = sjson.Set(result, itemPath+".value", value.Float())
		}

		// Convert status_code_range from array to object if it's an array
		// In v4, status_code_range was a block (MaxItems:1) stored as single-element array
		// In v5, status_code_range is a SingleNestedAttribute stored as object
		if scRange := item.Get("status_code_range"); scRange.Exists() {
			if scRange.IsArray() && len(scRange.Array()) > 0 {
				// Convert single-element array to object
				result, _ = sjson.Set(result, itemPath+".status_code_range", scRange.Array()[0].Value())
			}
		}

		// Re-parse to get the updated status_code_range (now as object)
		item = gjson.Parse(result).Get(itemPath)

		// Convert status_code_range.from and .to to float64 if present
		if scRange := item.Get("status_code_range"); scRange.Exists() && scRange.IsObject() {
			if from := scRange.Get("from"); from.Exists() {
				result, _ = sjson.Set(result, itemPath+".status_code_range.from", from.Float())
			}
			if to := scRange.Get("to"); to.Exists() {
				result, _ = sjson.Set(result, itemPath+".status_code_range.to", to.Float())
			}
		}
	}

	return result
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string, resourceName string) (string, error) {
	result := instance.String()

	// Parse attributes
	attrs := instance.Get("attributes")
	if !attrs.Exists() {
		return result, nil
	}

	// Process rules array
	rulesArray := attrs.Get("rules")
	if rulesArray.Exists() && rulesArray.IsArray() {
		rulesArray.ForEach(func(key, rule gjson.Result) bool {
			basePath := "attributes.rules." + key.String()

			// Convert action_parameters from array to object (MaxItems:1 in v4)
			if actionParams := rule.Get("action_parameters"); actionParams.Exists() && actionParams.IsArray() && len(actionParams.Array()) > 0 {
				actionParamsObj := actionParams.Array()[0]
				result, _ = sjson.Set(result, basePath+".action_parameters", actionParamsObj.Value())

				// Now recursively convert nested single-element arrays to objects within action_parameters
				// This handles all MaxItems:1 blocks like autominify, uri, etc.
				actionParamsPath := basePath + ".action_parameters"
				result = convertNestedSingleElementArraysToObjects(result, actionParamsPath)

				// Remove disable_railgun attribute (removed in v5)
				if gjson.Parse(result).Get(actionParamsPath + ".disable_railgun").Exists() {
					result, _ = sjson.Delete(result, actionParamsPath+".disable_railgun")
				}

				// Process headers within action_parameters
				// In v4, headers is an array of objects with "name" field
				// In v5, headers is a map where the key is the header name
				if headers := gjson.Parse(result).Get(actionParamsPath + ".headers"); headers.Exists() && headers.IsArray() {
					headersMap := make(map[string]interface{})
					for _, header := range headers.Array() {
						if name := header.Get("name"); name.Exists() {
							headerName := name.String()
							// Create object without the "name" field
							headerObj := make(map[string]interface{})
							header.ForEach(func(k, v gjson.Result) bool {
								if k.String() != "name" {
									headerObj[k.String()] = v.Value()
								}
								return true
							})
							headersMap[headerName] = headerObj
						}
					}
					if len(headersMap) > 0 {
						result, _ = sjson.Set(result, actionParamsPath+".headers", headersMap)
					} else {
						result, _ = sjson.Delete(result, actionParamsPath+".headers")
					}
				}

				// Transform custom log fields (cookie_fields, request_fields, response_fields)
				// In v4: ["field1", "field2"]
				// In v5: [{name: "field1"}, {name: "field2"}]
				result = transformCookieFields(result, actionParamsPath)
				result = transformRequestFields(result, actionParamsPath)
				result = transformResponseFields(result, actionParamsPath)

				// Transform query_string within cache_key.custom_key
				// In v4: include = ["param1", "param2"] or include = ["*"]
				// In v5: include = { list = ["param1", "param2"] } or include = { all = true }
				result = transformQueryStringInState(result, actionParamsPath)

				// Transform status_code_ttl numeric fields within edge_ttl
				result = transformStatusCodeTTLNumericFields(result, actionParamsPath)
			}

			// Convert ratelimit from array to object
			if ratelimit := rule.Get("ratelimit"); ratelimit.Exists() && ratelimit.IsArray() && len(ratelimit.Array()) > 0 {
				ratelimitObj := ratelimit.Array()[0]
				result, _ = sjson.Set(result, basePath+".ratelimit", ratelimitObj.Value())

				// Convert numeric fields to float64
				if period := ratelimitObj.Get("period"); period.Exists() {
					result, _ = sjson.Set(result, basePath+".ratelimit.period", period.Float())
				}
				if requestsPerPeriod := ratelimitObj.Get("requests_per_period"); requestsPerPeriod.Exists() {
					result, _ = sjson.Set(result, basePath+".ratelimit.requests_per_period", requestsPerPeriod.Float())
				}
				if mitigationTimeout := ratelimitObj.Get("mitigation_timeout"); mitigationTimeout.Exists() {
					result, _ = sjson.Set(result, basePath+".ratelimit.mitigation_timeout", mitigationTimeout.Float())
				}
			}

			// Convert exposed_credential_check from array to object
			if credCheck := rule.Get("exposed_credential_check"); credCheck.Exists() && credCheck.IsArray() && len(credCheck.Array()) > 0 {
				credCheckObj := credCheck.Array()[0]
				result, _ = sjson.Set(result, basePath+".exposed_credential_check", credCheckObj.Value())
			}

			// Delete empty arrays
			if ratelimit := gjson.Parse(result).Get(basePath + ".ratelimit"); ratelimit.Exists() && ratelimit.IsArray() && len(ratelimit.Array()) == 0 {
				result, _ = sjson.Delete(result, basePath+".ratelimit")
			}
			if credCheck := gjson.Parse(result).Get(basePath + ".exposed_credential_check"); credCheck.Exists() && credCheck.IsArray() && len(credCheck.Array()) == 0 {
				result, _ = sjson.Delete(result, basePath+".exposed_credential_check")
			}

			// Apply recursive conversion to the entire rule to handle fields like logging
			result = convertNestedSingleElementArraysToObjects(result, basePath)

			return true
		})
	}

	// Update schema_version to 0 (v5 starts at 0)
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// convertNestedSingleElementArraysToObjects recursively converts single-element arrays to objects
// This handles MaxItems:1 blocks that are stored as arrays in v4 but need to be objects in v5
func convertNestedSingleElementArraysToObjects(jsonStr string, path string) string {
	result := jsonStr
	data := gjson.Parse(result).Get(path)

	if !data.Exists() {
		return result
	}

	// Recursively process each field
	data.ForEach(func(key, value gjson.Result) bool {
		fieldName := key.String()
		fieldPath := path + "." + fieldName

		// Skip certain fields that should remain as arrays, but still recurse into their items
		if fieldName == "rules" || fieldName == "categories" || fieldName == "headers" || fieldName == "status_code_ttl" || fieldName == "algorithms" || fieldName == "cookie_fields" || fieldName == "request_fields" || fieldName == "response_fields" {
			// These fields should remain as arrays, but we still need to recurse into their items
			// to convert nested single-element arrays (like status_code_range within status_code_ttl items)
			if value.IsArray() {
				for idx := range value.Array() {
					itemPath := fmt.Sprintf("%s.%d", fieldPath, idx)
					result = convertNestedSingleElementArraysToObjects(result, itemPath)
				}
			}
			return true
		}

		// If it's an array, handle conversion based on length
		if value.IsArray() {
			arrLen := len(value.Array())
			if arrLen == 1 {
				// Single-element array: convert to object
				result, _ = sjson.Set(result, fieldPath, value.Array()[0].Value())
				// Recursively process the converted object
				result = convertNestedSingleElementArraysToObjects(result, fieldPath)
			} else if arrLen == 0 {
				// Empty array: convert to null
				result, _ = sjson.Set(result, fieldPath, nil)
			}
			// Arrays with multiple elements remain as arrays
		} else if value.IsObject() {
			// Recursively process nested objects
			result = convertNestedSingleElementArraysToObjects(result, fieldPath)
		}

		return true
	})

	return result
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
			result = append(result, token)      // iterator name
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
