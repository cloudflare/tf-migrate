// Package hcl provides utilities for transforming HCL configuration files
// during Terraform provider migrations. These utilities handle common patterns
// like renaming attributes, ensuring required fields exist, and restructuring
// resource configurations.
package hcl

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// EnsureAttribute ensures an attribute exists with a default value if not present.
// This is useful when a new provider version requires a field that was optional before.
//
// Example - Adding required TTL field to DNS records:
//
// Before:
//
//	resource "cloudflare_dns_record" "example" {
//	  zone_id = "abc123"
//	  name    = "test"
//	  type    = "A"
//	  content = "192.0.2.1"
//	}
//
// After calling EnsureAttribute(body, "ttl", 1):
//
//	resource "cloudflare_dns_record" "example" {
//	  zone_id = "abc123"
//	  name    = "test"
//	  type    = "A"
//	  content = "192.0.2.1"
//	  ttl     = 1
//	}
func EnsureAttribute(body *hclwrite.Body, attrName string, defaultValue interface{}) {
	if body.GetAttribute(attrName) == nil {
		tokens := TokensForSimpleValue(defaultValue)
		if tokens != nil {
			body.SetAttributeRaw(attrName, tokens)
		}
	}
}

// SetAttribute unconditionally sets an attribute to the specified value.
// Unlike EnsureAttribute, this will overwrite existing values.
// This is useful when migrating fields with changed defaults that need explicit values.
//
// Example - Explicitly setting fail_open during migration:
//
// Before:
//   deployment_configs {
//     production {
//       usage_model = "bundled"
//     }
//   }
//
// After calling SetAttribute(body, "fail_open", false):
//   deployment_configs {
//     production {
//       usage_model = "bundled"
//       fail_open   = false
//     }
//   }
func SetAttribute(body *hclwrite.Body, attrName string, value interface{}) {
	tokens := TokensForSimpleValue(value)
	if tokens != nil {
		body.SetAttributeRaw(attrName, tokens)
	}
}

// RenameAttribute renames an attribute from oldName to newName.
// Also updates any references to the old attribute name in lifecycle blocks (ignore_changes, replace_triggered_by).
// Returns true if the attribute was found and renamed, false otherwise.
//
// Example - Renaming 'value' to 'content' for DNS records:
//
// Before:
//
//	resource "cloudflare_dns_record" "example" {
//	  zone_id = "abc123"
//	  name    = "test"
//	  type    = "A"
//	  value   = "192.0.2.1"  # Old field name
//
//	  lifecycle {
//	    ignore_changes = [value]
//	  }
//	}
//
// After calling RenameAttribute(body, "value", "content"):
//
//	resource "cloudflare_dns_record" "example" {
//	  zone_id = "abc123"
//	  name    = "test"
//	  type    = "A"
//	  content = "192.0.2.1"  # New field name
//
//	  lifecycle {
//	    ignore_changes = [content]
//	  }
//	}
func RenameAttribute(body *hclwrite.Body, oldName, newName string) bool {
	renamed := false

	// Rename the attribute itself
	if attr := body.GetAttribute(oldName); attr != nil {
		tokens := attr.Expr().BuildTokens(nil)
		body.SetAttributeRaw(newName, tokens)
		body.RemoveAttribute(oldName)
		renamed = true
	}

	// Update references in lifecycle blocks
	for _, block := range body.Blocks() {
		if block.Type() == "lifecycle" {
			lifecycleBody := block.Body()

			// Update ignore_changes references
			if ignoreChangesAttr := lifecycleBody.GetAttribute("ignore_changes"); ignoreChangesAttr != nil {
				updated := updateAttributeListReferences(ignoreChangesAttr, oldName, newName)
				if updated != nil {
					lifecycleBody.SetAttributeRaw("ignore_changes", updated)
				}
			}

			// Update replace_triggered_by references
			if replaceAttr := lifecycleBody.GetAttribute("replace_triggered_by"); replaceAttr != nil {
				updated := updateAttributeListReferences(replaceAttr, oldName, newName)
				if updated != nil {
					lifecycleBody.SetAttributeRaw("replace_triggered_by", updated)
				}
			}
		}
	}

	return renamed
}

// RenameAndWrapInArray renames an attribute and wraps its value in an array.
// This is useful when a field changes from scalar to list type during migration.
//
// Example - Converting certificate string to list:
//
// Before:
//
//	config {
//	  issuer_url      = "https://saml.example.com"
//	  idp_public_cert = "MIIDpDCCAoygAwIBAgIGAV..."
//	}
//
// After calling RenameAndWrapInArray(configBody, "idp_public_cert", "idp_public_certs"):
//
//	config {
//	  issuer_url       = "https://saml.example.com"
//	  idp_public_certs = ["MIIDpDCCAoygAwIBAgIGAV..."]
//	}
func RenameAndWrapInArray(body *hclwrite.Body, oldName, newName string) bool {
	attr := body.GetAttribute(oldName)
	if attr == nil {
		return false
	}

	// Get the original value tokens
	valueTokens := attr.Expr().BuildTokens(nil)

	// Wrap in array: [value]
	arrayTokens := hclwrite.TokensForTuple([]hclwrite.Tokens{valueTokens})

	// Set as new attribute name
	body.SetAttributeRaw(newName, arrayTokens)

	// Remove old attribute
	body.RemoveAttribute(oldName)

	return true
}

// updateAttributeListReferences updates references to oldName in a list attribute (like ignore_changes)
// Returns updated tokens if any replacements were made, nil otherwise
func updateAttributeListReferences(attr *hclwrite.Attribute, oldName, newName string) hclwrite.Tokens {
	tokens := attr.Expr().BuildTokens(nil)
	modified := false

	for i, token := range tokens {
		if token.Type == hclsyntax.TokenIdent && string(token.Bytes) == oldName {
			tokens[i] = &hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte(newName),
			}
			modified = true
		}
	}

	if modified {
		return tokens
	}
	return nil
}

// RemoveAttributes removes multiple attributes from a body.
// Returns the count of attributes actually removed.
//
// Example - Removing deprecated fields:
//
// Before:
//
//	resource "cloudflare_dns_record" "example" {
//	  zone_id         = "abc123"
//	  name            = "test"
//	  type            = "A"
//	  content         = "192.0.2.1"
//	  allow_overwrite = true      # Deprecated in v5
//	  hostname        = "test.com" # Deprecated in v5
//	}
//
// After calling RemoveAttributes(body, "allow_overwrite", "hostname"):
//
//	resource "cloudflare_dns_record" "example" {
//	  zone_id = "abc123"
//	  name    = "test"
//	  type    = "A"
//	  content = "192.0.2.1"
//	}
func RemoveAttributes(body *hclwrite.Body, attrNames ...string) int {
	removed := 0
	for _, attrName := range attrNames {
		if body.GetAttribute(attrName) != nil {
			body.RemoveAttribute(attrName)
			removed++
		}
	}
	return removed
}

// ExtractStringFromAttribute extracts a string value from an HCL attribute.
// Handles both quoted literals and identifiers.
//
// Example usage:
//
//	typeAttr := body.GetAttribute("type")
//	recordType := ExtractStringFromAttribute(typeAttr)
//	// Returns "A" from: type = "A"
//	// Returns "var.record_type" from: type = var.record_type
func ExtractStringFromAttribute(attr *hclwrite.Attribute) string {
	if attr == nil {
		return ""
	}

	tokens := attr.Expr().BuildTokens(nil)
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenQuotedLit {
			// Remove quotes from quoted literal
			return strings.Trim(string(token.Bytes), "\"")
		} else if token.Type == hclsyntax.TokenIdent {
			// Return identifier as-is
			return string(token.Bytes)
		}
	}
	return ""
}

// ExtractBoolFromAttribute extracts a boolean value from an HCL attribute.
// Returns the boolean value and true if successful, or false and false if not found/invalid.
//
// Example usage:
//   enabledAttr := body.GetAttribute("enabled")
//   value, ok := ExtractBoolFromAttribute(enabledAttr)
//   // Returns (true, true) from: enabled = true
//   // Returns (false, true) from: enabled = false
//   // Returns (false, false) from: enabled = null or missing
func ExtractBoolFromAttribute(attr *hclwrite.Attribute) (bool, bool) {
	if attr == nil {
		return false, false
	}

	tokens := attr.Expr().BuildTokens(nil)
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenIdent {
			val := string(token.Bytes)
			if val == "true" {
				return true, true
			}
			if val == "false" {
				return false, true
			}
		}
	}
	return false, false
}

// HasAttribute checks if an attribute exists in the body
func HasAttribute(body *hclwrite.Body, attrName string) bool {
	return body.GetAttribute(attrName) != nil
}

// CopyAndRenameAttribute copies an attribute with a new name
func CopyAndRenameAttribute(from, to *hclwrite.Body, oldName, newName string) bool {
	if attr := from.GetAttribute(oldName); attr != nil {
		tokens := attr.Expr().BuildTokens(nil)
		to.SetAttributeRaw(newName, tokens)
		return true
	}
	return false
}

// AttributeRenameMap represents a mapping of old attribute names to new ones
type AttributeRenameMap map[string]string

// ApplyAttributeRenames applies multiple attribute renames based on a map
func ApplyAttributeRenames(body *hclwrite.Body, renames AttributeRenameMap) int {
	renamed := 0
	for oldName, newName := range renames {
		if RenameAttribute(body, oldName, newName) {
			renamed++
		}
	}
	return renamed
}

// ConditionalRenameAttribute renames an attribute only if a condition is met
func ConditionalRenameAttribute(body *hclwrite.Body, oldName, newName string, condition func(*hclwrite.Attribute) bool) bool {
	if attr := body.GetAttribute(oldName); attr != nil {
		if condition(attr) {
			tokens := attr.Expr().BuildTokens(nil)
			body.SetAttributeRaw(newName, tokens)
			body.RemoveAttribute(oldName)
			return true
		}
	}
	return false
}

// UpdateResourceReferences updates all references to a resource type in the content.
// This is useful when renaming resource types to ensure cross-resource references are updated.
//
// Example - Updating references from cloudflare_record to cloudflare_dns_record:
//
// Before:
//
//	content = "${cloudflare_record.example_a.name}.${var.domain_name}"
//
// After calling UpdateResourceReferences(content, "cloudflare_record", "cloudflare_dns_record"):
//
//	content = "${cloudflare_dns_record.example_a.name}.${var.domain_name}"
func UpdateResourceReferences(content, oldType, newType string) string {
	// Replace resource references in string interpolations
	// Pattern: ${oldType. -> ${newType.
	content = strings.ReplaceAll(content, "${"+oldType+".", "${"+newType+".")
	// Pattern: oldType. (for non-interpolated references)
	// We need to be careful to only replace when followed by a dot and identifier
	// This is a simpler approach that works for most cases
	content = strings.ReplaceAll(content, oldType+".", newType+".")
	return content
}

// AttributeValueContainsKey checks if an attribute's value is an object/map
// and contains the specified key as a top-level key in that object.
//
// Example - Checking if an object contains a specific key:
//
// Given:
//
//	config = {
//	  key1 = "value1"
//	  key2 = "value2"
//	}
//
// AttributeValueContainsKey(attr, "key1") returns true
// AttributeValueContainsKey(attr, "value1") returns false (it's a value, not a key)
// AttributeValueContainsKey(attr, "missing") returns false
func AttributeValueContainsKey(attr *hclwrite.Attribute, key string) bool {
	if attr == nil {
		return false
	}

	valueTokens := attr.BuildTokens(hclwrite.Tokens{})

	// Track object nesting depth - we only want to check top-level keys (depth 1)
	depth := 0
	for i, token := range valueTokens {
		// Check if we're entering an object
		if token.Type == hclsyntax.TokenOBrace {
			depth++
			continue
		}

		// Check if we're leaving an object
		if token.Type == hclsyntax.TokenCBrace {
			depth--
			continue
		}

		// If we're at depth 1 (top-level of the object) and find an identifier, check if it's a key
		// (a key is an identifier followed by an equals sign)
		if depth == 1 && token.Type == hclsyntax.TokenIdent && string(token.Bytes) == key {
			// Look ahead to see if this identifier is followed by an equals sign
			for j := i + 1; j < len(valueTokens); j++ {
				nextToken := valueTokens[j]
				// Skip whitespace and newlines
				if nextToken.Type == hclsyntax.TokenNewline || nextToken.Type == hclsyntax.TokenComment {
					continue
				}
				// If we find an equals sign, this is a key
				if nextToken.Type == hclsyntax.TokenEqual {
					return true
				}
				// If we find something else, this is not a key
				break
			}
		}
	}
	return false
}

// AttributeInfo holds an attribute name and its corresponding Attribute object
type AttributeInfo struct {
	Name      string
	Attribute *hclwrite.Attribute
}

// AttributesOrdered returns attributes from a body in their original order
// This is important when generating HCL that needs to maintain specific field ordering
func AttributesOrdered(body *hclwrite.Body) []AttributeInfo {
	// Get all attributes as a map for lookup
	attrMap := body.Attributes()

	// Get tokens to find the original order
	tokens := body.BuildTokens(nil)

	var orderedAttrs []AttributeInfo
	seenAttrs := make(map[string]bool)

	// Scan through tokens to find attribute names in order
	for i := range tokens {
		token := tokens[i]

		// Look for identifier tokens that could be attribute names
		if token.Type == hclsyntax.TokenIdent && i+1 < len(tokens) {
			// Check if the next token is an equals sign
			nextToken := tokens[i+1]
			if nextToken.Type == hclsyntax.TokenEqual {
				attrName := string(token.Bytes)

				// Check if this is actually an attribute and we haven't seen it yet
				if attr, exists := attrMap[attrName]; exists && !seenAttrs[attrName] {
					orderedAttrs = append(orderedAttrs, AttributeInfo{
						Name:      attrName,
						Attribute: attr,
					})
					seenAttrs[attrName] = true
				}
			}
		}
	}

	return orderedAttrs
}

// SetAttributeValue is a helper that sets an attribute value based on its Go type
// It automatically converts common Go types to their cty equivalents
func SetAttributeValue(body *hclwrite.Body, name string, val interface{}) {
	switch v := val.(type) {
	case string:
		body.SetAttributeValue(name, cty.StringVal(v))
	case int:
		body.SetAttributeValue(name, cty.NumberIntVal(int64(v)))
	case int64:
		body.SetAttributeValue(name, cty.NumberIntVal(v))
	case float64:
		body.SetAttributeValue(name, cty.NumberFloatVal(v))
	case bool:
		body.SetAttributeValue(name, cty.BoolVal(v))
	case []string:
		if len(v) == 0 {
			body.SetAttributeValue(name, cty.ListValEmpty(cty.String))
		} else {
			values := make([]cty.Value, len(v))
			for i, s := range v {
				values[i] = cty.StringVal(s)
			}
			body.SetAttributeValue(name, cty.ListVal(values))
		}
	case map[string]string:
		values := make(map[string]cty.Value)
		for k, v := range v {
			values[k] = cty.StringVal(v)
		}
		body.SetAttributeValue(name, cty.ObjectVal(values))
	default:
		// For complex types, caller should use SetAttributeRaw with tokens
		// or SetAttributeValue with a properly constructed cty.Value
	}
}

// CopyAttribute copies an attribute from one body to another, preserving its expression
func CopyAttribute(from, to *hclwrite.Body, attrName string) {
	if attr := from.GetAttribute(attrName); attr != nil {
		tokens := attr.Expr().BuildTokens(nil)
		to.SetAttributeRaw(attrName, tokens)
	}
}

// CreateNestedAttributeFromFields creates a nested attribute (object) from a map of field names to tokens.
// This is useful for restructuring flat attributes into nested objects (e.g., http_config, tcp_config).
//
// Example - Creating http_config from collected HTTP fields:
//
// Before:
//
//	resource "cloudflare_healthcheck" "example" {
//	  zone_id = "abc123"
//	  type    = "HTTP"
//	  port    = 80
//	  path    = "/health"
//	  method  = "GET"
//	}
//
// After collecting fields and calling CreateNestedAttributeFromFields(body, "http_config", fields):
//
//	resource "cloudflare_healthcheck" "example" {
//	  zone_id = "abc123"
//	  type    = "HTTP"
//	  http_config = {
//	    port   = 80
//	    path   = "/health"
//	    method = "GET"
//	  }
//	}
func CreateNestedAttributeFromFields(body *hclwrite.Body, attrName string, fields map[string]hclwrite.Tokens) {
	if len(fields) == 0 {
		return
	}

	// Build object attribute tokens from the fields map
	var attrs []hclwrite.ObjectAttrTokens

	// Sort keys for consistent output (optional but nice for testing)
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create object attributes
	for _, fieldName := range keys {
		nameTokens := hclwrite.TokensForIdentifier(fieldName)
		valueTokens := fields[fieldName]

		attrs = append(attrs, hclwrite.ObjectAttrTokens{
			Name:  nameTokens,
			Value: valueTokens,
		})
	}

	// Create the object tokens and set the attribute
	objTokens := hclwrite.TokensForObject(attrs)
	body.SetAttributeRaw(attrName, objTokens)
}

// MoveAttributesToNestedObject moves multiple attributes from the body into a nested object attribute.
// This is the most common pattern for flat-to-nested migrations.
// Returns the number of attributes that were moved.
//
// Example - Moving HTTP fields into http_config:
//
// Before:
//
//	resource "cloudflare_healthcheck" "example" {
//	  zone_id = "abc123"
//	  type    = "HTTP"
//	  port    = 80
//	  path    = "/health"
//	  method  = "GET"
//	}
//
// After calling MoveAttributesToNestedObject(body, "http_config", []string{"port", "path", "method"}):
//
//	resource "cloudflare_healthcheck" "example" {
//	  zone_id = "abc123"
//	  type    = "HTTP"
//	  http_config = {
//	    port   = 80
//	    path   = "/health"
//	    method = "GET"
//	  }
//	}
func MoveAttributesToNestedObject(body *hclwrite.Body, nestedAttrName string, fieldNames []string) int {
	fields := make(map[string]hclwrite.Tokens)

	// Collect the field tokens
	for _, fieldName := range fieldNames {
		if attr := body.GetAttribute(fieldName); attr != nil {
			fields[fieldName] = attr.Expr().BuildTokens(nil)
		}
	}

	if len(fields) == 0 {
		return 0
	}

	// Create the nested attribute
	CreateNestedAttributeFromFields(body, nestedAttrName, fields)

	// Remove the original attributes
	for fieldName := range fields {
		body.RemoveAttribute(fieldName)
	}

	return len(fields)
}
