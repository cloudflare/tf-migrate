package page_rule

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_page_rule"}, "cloudflare_page_rule"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Resource name doesn't change (cloudflare_page_rule in both v4 and v5)
	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	// Step 0: Add status = "active" if not present (v4 default was "active", v5 default is "disabled")
	if body.GetAttribute("status") == nil {
		body.SetAttributeValue("status", cty.StringVal("active"))
	}

	// Step 1: Find and process actions block
	actionsBlock := tfhcl.FindBlockByType(body, "actions")
	if actionsBlock != nil {
		actionsBody := actionsBlock.Body()

		// Step 1a: Remove deprecated fields and warn
		var removedFields []string
		if actionsBody.GetAttribute("minify") != nil {
			removedFields = append(removedFields, "minify")
		}
		if actionsBody.GetAttribute("disable_railgun") != nil {
			removedFields = append(removedFields, "disable_railgun")
		}
		if actionsBody.GetAttribute("server_side_exclude") != nil {
			removedFields = append(removedFields, "server_side_exclude")
		}
		if len(removedFields) > 0 {
			ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
				Severity: transform.DiagInfo,
				Summary:  fmt.Sprintf("Deprecated fields removed: cloudflare_page_rule.%s", resourceName),
				Detail: fmt.Sprintf(`The following deprecated fields have been removed during migration:
  %s

These fields are no longer supported in the v5 provider:
  - minify: Use cloudflare_zone_setting with setting_id = "minify" instead
  - disable_railgun: Railgun has been discontinued
  - server_side_exclude: Removed from page_rule actions in v5`, strings.Join(removedFields, ", ")),
			})
		}
		tfhcl.RemoveAttributes(actionsBody, "minify", "disable_railgun", "server_side_exclude")

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

			// Translate query_string.ignore to v5 include/exclude fields.
			if queryStringBlock := tfhcl.FindBlockByType(cacheKeyBody, "query_string"); queryStringBlock != nil {
				queryStringBody := queryStringBlock.Body()
				if ignoreAttr := queryStringBody.GetAttribute("ignore"); ignoreAttr != nil {
					if ignore, ok := tfhcl.ExtractBoolFromAttribute(ignoreAttr); ok {
						queryStringBody.RemoveAttribute("ignore")
						if ignore {
							queryStringBody.SetAttributeValue("include", cty.ListValEmpty(cty.String))
							queryStringBody.SetAttributeValue("exclude", cty.ListVal([]cty.Value{cty.StringVal("*")}))
						} else {
							queryStringBody.SetAttributeValue("include", cty.ListVal([]cty.Value{cty.StringVal("*")}))
							queryStringBody.SetAttributeValue("exclude", cty.ListValEmpty(cty.String))
						}
					}
				}
			}

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

// quotedStringTokens renders s as a quoted HCL string literal.
func quotedStringTokens(s string) hclwrite.Tokens {
	return hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")},
		&hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(s)},
		&hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")},
	}
}

// cloneTokens makes an independent copy of tokens so they stay valid after the
// block they came from is removed from the body.
func cloneTokens(tokens hclwrite.Tokens) hclwrite.Tokens {
	out := make(hclwrite.Tokens, 0, len(tokens))
	for _, t := range tokens {
		b := make([]byte, len(t.Bytes))
		copy(b, t.Bytes)
		out = append(out, &hclwrite.Token{Type: t.Type, Bytes: b, SpacesBefore: t.SpacesBefore})
	}
	return out
}

// literalScalar reports whether the attribute is a plain literal scalar that can
// be folded into the map, and returns its string form. It deliberately tests for
// a literal rather than against a list of known expression prefixes: anything it
// cannot prove is a literal (traversal, function call, interpolation) must have
// its tokens preserved verbatim instead of being flattened.
func literalScalar(attr *hclwrite.Attribute) (string, bool) {
	if attr == nil {
		return "", false
	}

	tokens := attr.Expr().BuildTokens(nil)

	// Quoted string, joined across literal runs (HCL splits them at '%' and '$').
	if val, ok := tfhcl.JoinQuotedLiteral(tokens); ok {
		return val, true
	}

	// Bare number, e.g. ttl = 3600. v5 represents the map value as a string.
	var number string
	for _, token := range tokens {
		switch token.Type {
		case hclsyntax.TokenNumberLit:
			if number != "" {
				return "", false
			}
			number = string(token.Bytes)
		case hclsyntax.TokenNewline, hclsyntax.TokenComment:
			// ignore
		default:
			return "", false
		}
	}
	if number != "" {
		return number, true
	}

	return "", false
}

// cacheTTLEntry is one code->ttl pair awaiting rendering into the v5 map.
type cacheTTLEntry struct {
	nameTokens  hclwrite.Tokens
	valueTokens hclwrite.Tokens
}

// transformCacheTTLByStatus transforms cache_ttl_by_status blocks to map syntax
// v4: cache_ttl_by_status { codes = "200" ttl = 3600 }
// v5: cache_ttl_by_status = { "200" = "3600" }
//
// Literal codes and ttl values are folded into the map directly. Values that are
// not literals (variables, locals, function calls) are carried over verbatim
// instead, because collapsing them loses the reference entirely: ExtractString-
// FromAttribute returns only the first identifier of an expression, so
// "ttl = var.cache_ttl" used to render as the literal string "var" and
// "ttl = max(300, var.x)" as "max". An expression key is parenthesised, which
// Terraform requires for a computed object key.
func (m *V4ToV5Migrator) transformCacheTTLByStatus(body *hclwrite.Body) {
	// Find all cache_ttl_by_status blocks
	blocks := tfhcl.FindBlocksByType(body, "cache_ttl_by_status")
	if len(blocks) == 0 {
		return
	}

	// Literal keys are deduplicated (last block wins, as before) and sorted for
	// deterministic output. Expression keys cannot be compared for equality, so
	// they are kept in source order and appended after the literal entries.
	literalEntries := make(map[string]hclwrite.Tokens)
	var exprEntries []cacheTTLEntry

	for _, block := range blocks {
		blockBody := block.Body()

		codesAttr := blockBody.GetAttribute("codes")
		ttlAttr := blockBody.GetAttribute("ttl")
		if codesAttr == nil || ttlAttr == nil {
			continue
		}

		// Resolve the value first: literal ttl becomes a quoted string, anything
		// else is preserved as-is. Terraform converts a numeric expression to the
		// string the v5 map expects.
		var valueTokens hclwrite.Tokens
		if ttl, ok := literalScalar(ttlAttr); ok {
			if ttl == "" {
				continue
			}
			valueTokens = quotedStringTokens(ttl)
		} else {
			valueTokens = cloneTokens(ttlAttr.Expr().BuildTokens(nil))
		}

		if codes, ok := literalScalar(codesAttr); ok {
			if codes == "" {
				continue
			}
			literalEntries[codes] = valueTokens
			continue
		}

		// Expression key: wrap in parentheses so Terraform evaluates it rather
		// than reading it as a literal attribute name.
		nameTokens := hclwrite.Tokens{
			&hclwrite.Token{Type: hclsyntax.TokenOParen, Bytes: []byte("(")},
		}
		nameTokens = append(nameTokens, cloneTokens(codesAttr.Expr().BuildTokens(nil))...)
		nameTokens = append(nameTokens, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})

		exprEntries = append(exprEntries, cacheTTLEntry{
			nameTokens:  nameTokens,
			valueTokens: valueTokens,
		})
	}

	// Remove all cache_ttl_by_status blocks BEFORE setting the attribute
	// This ensures the blocks don't interfere with the new attribute
	tfhcl.RemoveBlocksByType(body, "cache_ttl_by_status")

	if len(literalEntries) == 0 && len(exprEntries) == 0 {
		return
	}

	// Build map tokens: cache_ttl_by_status = { "200" = "3600", "404" = "300" }
	var attrs []hclwrite.ObjectAttrTokens

	// Sort literal keys for deterministic output
	var sortedKeys []string
	for codes := range literalEntries {
		sortedKeys = append(sortedKeys, codes)
	}
	sort.Strings(sortedKeys)

	for _, codes := range sortedKeys {
		attrs = append(attrs, hclwrite.ObjectAttrTokens{
			Name:  quotedStringTokens(codes),
			Value: literalEntries[codes],
		})
	}

	for _, entry := range exprEntries {
		attrs = append(attrs, hclwrite.ObjectAttrTokens{
			Name:  entry.nameTokens,
			Value: entry.valueTokens,
		})
	}

	// Use TokensForObject to create properly formatted object
	objTokens := hclwrite.TokensForObject(attrs)
	body.SetAttributeRaw("cache_ttl_by_status", objTokens)
}
