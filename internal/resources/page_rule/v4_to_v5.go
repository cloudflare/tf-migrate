package page_rule

import (
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
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

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_page_rule doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_page_rule", "cloudflare_page_rule"
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

			// Step 1d.1: Ensure user block has all required fields
			// v5 schema has device_type, geo, lang (all default false)
			// v4 configs may only have device_type and geo
			// Cloudflare API returns all three, so add lang = false to prevent drift
			if userBlock := tfhcl.FindBlockByType(cacheKeyBody, "user"); userBlock != nil {
				userBody := userBlock.Body()
				if userBody.GetAttribute("lang") == nil {
					// Add lang = false if not present
					userBody.SetAttributeValue("lang", cty.BoolVal(false))
				}
			}

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
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// TransformConfig handles config-level transformations (block → attribute conversions)
	// This function is a no-op for page_rule migration
	return stateJSON.String(), nil
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

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}

