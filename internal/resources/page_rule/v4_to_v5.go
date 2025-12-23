package page_rule

import (
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of Page Rule resources from v4 to v5
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the OLD (v4) resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_page_rule", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name (unchanged)
	return "cloudflare_page_rule"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_page_rule"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations will be done at HCL level
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Resource name doesn't change (cloudflare_page_rule in both v4 and v5)
	body := block.Body()

	// Step 0: Add status = "active" if not present (v4 default was "active", v5 default is "disabled")
	if body.GetAttribute("status") == nil {
		body.SetAttributeValue("status", cty.StringVal("active"))
	}

	// Step 1: Find and process actions block
	actionsBlock := tfhcl.FindBlockByType(body, "actions")
	if actionsBlock != nil {
		actionsBody := actionsBlock.Body()

		// Step 1a: Remove deprecated fields
		tfhcl.RemoveAttributes(actionsBody, "minify", "disable_railgun")

		// Step 1b: Transform cache_ttl_by_status (TypeSet blocks → MapAttribute)
		// MUST do this BEFORE converting actions block, while blocks still exist
		// v4: cache_ttl_by_status { codes = "200" ttl = 3600 }
		// v5: cache_ttl_by_status = { "200" = "3600" }
		m.transformCacheTTLByStatus(actionsBody)

		// Step 1c: Process nested forwarding_url block (if exists)
		// Convert forwarding_url TypeList MaxItems:1 block to SingleNestedAttribute
		if forwardingBlock := tfhcl.FindBlockByType(actionsBody, "forwarding_url"); forwardingBlock != nil {
			tfhcl.ConvertSingleBlockToAttribute(actionsBody, "forwarding_url", "forwarding_url")
		}

		// Step 1d: Process cache_key_fields nested structure (5 levels deep!)
		// Must process deepest blocks first, then parent
		if cacheKeyBlock := tfhcl.FindBlockByType(actionsBody, "cache_key_fields"); cacheKeyBlock != nil {
			cacheKeyBody := cacheKeyBlock.Body()

			// Convert deepest nested blocks first (TypeList MaxItems:1 → SingleNestedAttribute)
			tfhcl.ConvertSingleBlockToAttribute(cacheKeyBody, "cookie", "cookie")
			tfhcl.ConvertSingleBlockToAttribute(cacheKeyBody, "header", "header")
			tfhcl.ConvertSingleBlockToAttribute(cacheKeyBody, "host", "host")
			tfhcl.ConvertSingleBlockToAttribute(cacheKeyBody, "query_string", "query_string")
			tfhcl.ConvertSingleBlockToAttribute(cacheKeyBody, "user", "user")

			// Then convert cache_key_fields itself
			tfhcl.ConvertSingleBlockToAttribute(actionsBody, "cache_key_fields", "cache_key_fields")
		}
	}

	// Step 2: Convert actions block to attribute (must be LAST!)
	// Convert actions TypeList MaxItems:1 block to SingleNestedAttribute
	tfhcl.ConvertSingleBlockToAttribute(body, "actions", "actions")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()
	attrs := stateJSON.Get("attributes")

	if !attrs.Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// 1. Convert priority from int to float64
	if priority := attrs.Get("priority"); priority.Exists() {
		floatVal := state.ConvertToFloat64(priority)
		result, _ = sjson.Set(result, "attributes.priority", floatVal)
	}

	// 2. Handle status default change (v4: "active", v5: "disabled")
	// Preserve v4 behavior by explicitly setting "active" if missing
	statusField := attrs.Get("status")
	if !statusField.Exists() || statusField.String() == "" {
		result, _ = sjson.Set(result, "attributes.status", "active")
	}

	// 3. Transform actions array [{}] → object {}
	if actionsField := attrs.Get("actions"); actionsField.Exists() && actionsField.IsArray() {
		actions := actionsField.Array()
		if len(actions) > 0 {
			actionsObj := actions[0] // Take first element (MaxItems:1)
			result, _ = sjson.Set(result, "attributes.actions", actionsObj.Value())

			// Re-parse to get updated structure
			attrs = gjson.Parse(result).Get("attributes")

			// 3a. Remove deprecated fields
			result, _ = sjson.Delete(result, "attributes.actions.minify")
			result, _ = sjson.Delete(result, "attributes.actions.disable_railgun")

			// 3b. Transform nested MaxItems:1 arrays to objects
			result = m.transformActionsNestedArrays(result, attrs.Get("actions"))

			// 3c. Transform cache_ttl_by_status array → map
			result = m.transformCacheTTLByStatusState(result, attrs.Get("actions"))

			// 3d. Convert edge_cache_ttl from int to float64
			if edgeCacheTTL := attrs.Get("actions.edge_cache_ttl"); edgeCacheTTL.Exists() {
				floatVal := state.ConvertToFloat64(edgeCacheTTL)
				result, _ = sjson.Set(result, "attributes.actions.edge_cache_ttl", floatVal)
			}

			// 3e. Convert browser_cache_ttl from string to int64
			// v4 stores it as string, v5 expects int64
			if browserCacheTTL := attrs.Get("actions.browser_cache_ttl"); browserCacheTTL.Exists() {
				intVal := state.ConvertToInt64(browserCacheTTL)
				result, _ = sjson.Set(result, "attributes.actions.browser_cache_ttl", intVal)
			}

			// Re-parse attrs after all transformations to get final structure
			updatedAttrs := gjson.Parse(result).Get("attributes")

			// 3f. Transform empty values to null for action fields not explicitly set in config
			result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
				Ctx:              ctx,
				Result:           result,
				FieldPath:        "attributes.actions",
				FieldResult:      updatedAttrs.Get("actions"),
				ResourceName:     resourceName,
				HCLAttributePath: "actions",
				CanHandle:        m.CanHandle,
			})
		}
	}

	// 4. Always set schema_version
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// transformCacheTTLByStatus transforms cache_ttl_by_status blocks to map syntax
// v4: cache_ttl_by_status { codes = "200" ttl = 3600 }
// v5: cache_ttl_by_status = { "200" = "3600" }
func (m *V4ToV5Migrator) transformCacheTTLByStatus(body *hclwrite.Body) {
	// Find all cache_ttl_by_status blocks
	blocks := tfhcl.FindBlocksByType(body, "cache_ttl_by_status")
	if len(blocks) == 0 {
		return
	}

	// Collect all code->ttl mappings
	entries := make(map[string]string)
	for _, block := range blocks {
		blockBody := block.Body()

		// Extract codes and ttl from block
		codesAttr := blockBody.GetAttribute("codes")
		ttlAttr := blockBody.GetAttribute("ttl")

		if codesAttr != nil && ttlAttr != nil {
			codes := tfhcl.ExtractStringFromAttribute(codesAttr)

			// Extract TTL - it might be a number (not quoted)
			ttl := tfhcl.ExtractStringFromAttribute(ttlAttr)
			if ttl == "" {
				// Try getting the raw token value (for numbers)
				tokens := ttlAttr.Expr().BuildTokens(nil)
				for _, token := range tokens {
					if token.Type == hclsyntax.TokenNumberLit {
						ttl = string(token.Bytes)
						break
					}
				}
			}

			if codes != "" && ttl != "" {
				entries[codes] = ttl
			}
		}
	}

	// Remove all cache_ttl_by_status blocks BEFORE setting the attribute
	// This ensures the blocks don't interfere with the new attribute
	tfhcl.RemoveBlocksByType(body, "cache_ttl_by_status")

	// If we have entries, create map attribute
	if len(entries) > 0 {
		// Build map tokens: cache_ttl_by_status = { "200" = "3600", "404" = "300" }
		// Use TokensForObject to get properly formatted map
		var attrs []hclwrite.ObjectAttrTokens

		// Sort keys for deterministic output
		var sortedKeys []string
		for codes := range entries {
			sortedKeys = append(sortedKeys, codes)
		}
		sort.Strings(sortedKeys)

		for _, codes := range sortedKeys {
			ttl := entries[codes]

			// Create name tokens for the key (as a quoted string)
			nameTokens := hclwrite.Tokens{
				&hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")},
				&hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(codes)},
				&hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")},
			}

			// Create value tokens (as a quoted string)
			valueTokens := hclwrite.Tokens{
				&hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")},
				&hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(ttl)},
				&hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")},
			}

			attrs = append(attrs, hclwrite.ObjectAttrTokens{
				Name:  nameTokens,
				Value: valueTokens,
			})
		}

		// Use TokensForObject to create properly formatted object
		objTokens := hclwrite.TokensForObject(attrs)
		body.SetAttributeRaw("cache_ttl_by_status", objTokens)
	}
}

// transformActionsNestedArrays transforms nested MaxItems:1 arrays to objects
// Handles: forwarding_url, cache_key_fields and its nested fields
// Empty arrays are left as [] so TransformEmptyValuesToNull can convert them to null
func (m *V4ToV5Migrator) transformActionsNestedArrays(result string, actionsField gjson.Result) string {
	// Transform forwarding_url [{}] → {}
	if forwardingURL := actionsField.Get("forwarding_url"); forwardingURL.Exists() && forwardingURL.IsArray() {
		arr := forwardingURL.Array()
		if len(arr) > 0 {
			result, _ = sjson.Set(result, "attributes.actions.forwarding_url", arr[0].Value())
		}
		// Keep empty arrays as [] - TransformEmptyValuesToNull will convert to null
	}

	// Transform cache_key_fields [{}] → {} and its nested fields
	if cacheKeyFields := actionsField.Get("cache_key_fields"); cacheKeyFields.Exists() && cacheKeyFields.IsArray() {
		arr := cacheKeyFields.Array()
		if len(arr) > 0 {
			result, _ = sjson.Set(result, "attributes.actions.cache_key_fields", arr[0].Value())

			// Re-parse to get updated structure
			updatedActions := gjson.Parse(result).Get("attributes.actions")
			cacheKeyObj := updatedActions.Get("cache_key_fields")

			// Transform nested arrays within cache_key_fields
			for _, field := range []string{"cookie", "header", "host", "query_string", "user"} {
				if fieldVal := cacheKeyObj.Get(field); fieldVal.Exists() && fieldVal.IsArray() {
					fieldArr := fieldVal.Array()
					if len(fieldArr) > 0 {
						result, _ = sjson.Set(result, "attributes.actions.cache_key_fields."+field, fieldArr[0].Value())
					} else {
						// Convert empty arrays to null immediately
						result, _ = sjson.Set(result, "attributes.actions.cache_key_fields."+field, nil)
					}
				}
			}

			// Re-parse to get updated cache_key_fields structure
			updatedActions = gjson.Parse(result).Get("attributes.actions")
			cacheKeyObj = updatedActions.Get("cache_key_fields")

			// Clean up empty arrays within the nested objects (cookie, query_string, etc.)
			result = m.cleanupCacheKeyFieldsNestedArrays(result, cacheKeyObj)
		} else {
			// Convert empty cache_key_fields array to null
			result, _ = sjson.Set(result, "attributes.actions.cache_key_fields", nil)
		}
	}

	return result
}

// cleanupCacheKeyFieldsNestedArrays converts empty arrays within cache_key_fields nested objects to null
// This handles fields like cookie.include, query_string.exclude, etc.
func (m *V4ToV5Migrator) cleanupCacheKeyFieldsNestedArrays(result string, cacheKeyFieldsObj gjson.Result) string {
	if !cacheKeyFieldsObj.Exists() || !cacheKeyFieldsObj.IsObject() {
		return result
	}

	// Fields that should be checked within each cache_key_fields sub-object
	nestedFieldsToCheck := map[string][]string{
		"cookie":       {"check_presence", "include"},
		"header":       {"check_presence", "exclude", "include"},
		"query_string": {"exclude", "include"},
	}

	for parentField, childFields := range nestedFieldsToCheck {
		parentObj := cacheKeyFieldsObj.Get(parentField)
		if !parentObj.Exists() || !parentObj.IsObject() {
			continue
		}

		for _, childField := range childFields {
			childValue := parentObj.Get(childField)
			if childValue.Exists() && childValue.IsArray() && len(childValue.Array()) == 0 {
				// Convert empty array to null
				result, _ = sjson.Set(result, "attributes.actions.cache_key_fields."+parentField+"."+childField, nil)
			}
		}
	}

	return result
}

// transformCacheTTLByStatusState transforms cache_ttl_by_status from array to map
// v4 state: [{"codes": "200", "ttl": 3600}, {"codes": "404", "ttl": 300}]
// v5 state: {"200": "3600", "404": "300"}
func (m *V4ToV5Migrator) transformCacheTTLByStatusState(result string, actionsField gjson.Result) string {
	cacheTTLField := actionsField.Get("cache_ttl_by_status")

	// Check if cache_ttl_by_status exists and is an array
	if !cacheTTLField.Exists() || !cacheTTLField.IsArray() {
		return result
	}

	// Convert array to map: [{"codes": "200", "ttl": 3600}] → {"200": "3600"}
	cacheTTLMap := make(map[string]string)

	for _, item := range cacheTTLField.Array() {
		codes := item.Get("codes").String()
		ttl := item.Get("ttl")

		// Convert TTL to string (v5 uses MapAttribute with string values)
		var ttlStr string
		switch ttl.Type {
		case gjson.Number:
			ttlStr = ttl.String()
		case gjson.String:
			ttlStr = ttl.String()
		default:
			// Skip invalid entries
			continue
		}

		if codes != "" && ttlStr != "" {
			cacheTTLMap[codes] = ttlStr
		}
	}

	// Set the map (or delete if empty)
	if len(cacheTTLMap) > 0 {
		result, _ = sjson.Set(result, "attributes.actions.cache_ttl_by_status", cacheTTLMap)
	} else {
		result, _ = sjson.Delete(result, "attributes.actions.cache_ttl_by_status")
	}

	return result
}
