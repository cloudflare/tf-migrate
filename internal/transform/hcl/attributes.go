// Package hcl provides utilities for transforming HCL configuration files
// during Terraform provider migrations. These utilities handle common patterns
// like renaming attributes, ensuring required fields exist, and restructuring
// resource configurations.
package hcl

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/hcl"
)

// EnsureAttribute ensures an attribute exists with a default value if not present.
// This is useful when a new provider version requires a field that was optional before.
//
// Example - Adding required TTL field to DNS records:
//
// Before:
//   resource "cloudflare_dns_record" "example" {
//     zone_id = "abc123"
//     name    = "test"
//     type    = "A"
//     content = "192.0.2.1"
//   }
//
// After calling EnsureAttribute(body, "ttl", 1):
//   resource "cloudflare_dns_record" "example" {
//     zone_id = "abc123"
//     name    = "test"
//     type    = "A"
//     content = "192.0.2.1"
//     ttl     = 1
//   }
func EnsureAttribute(body *hclwrite.Body, attrName string, defaultValue interface{}) {
	if body.GetAttribute(attrName) == nil {
		tokens := hcl.TokensForSimpleValue(defaultValue)
		if tokens != nil {
			body.SetAttributeRaw(attrName, tokens)
		}
	}
}

// RenameAttribute renames an attribute from oldName to newName.
// Returns true if the attribute was found and renamed, false otherwise.
//
// Example - Renaming 'value' to 'content' for DNS records:
//
// Before:
//   resource "cloudflare_dns_record" "example" {
//     zone_id = "abc123"
//     name    = "test"
//     type    = "A"
//     value   = "192.0.2.1"  # Old field name
//   }
//
// After calling RenameAttribute(body, "value", "content"):
//   resource "cloudflare_dns_record" "example" {
//     zone_id = "abc123"
//     name    = "test"
//     type    = "A"
//     content = "192.0.2.1"  # New field name
//   }
func RenameAttribute(body *hclwrite.Body, oldName, newName string) bool {
	if attr := body.GetAttribute(oldName); attr != nil {
		tokens := attr.Expr().BuildTokens(nil)
		body.SetAttributeRaw(newName, tokens)
		body.RemoveAttribute(oldName)
		return true
	}
	return false
}

// RemoveAttributes removes multiple attributes from a body.
// Returns the count of attributes actually removed.
//
// Example - Removing deprecated fields:
//
// Before:
//   resource "cloudflare_dns_record" "example" {
//     zone_id         = "abc123"
//     name            = "test"
//     type            = "A"
//     content         = "192.0.2.1"
//     allow_overwrite = true      # Deprecated in v5
//     hostname        = "test.com" # Deprecated in v5
//   }
//
// After calling RemoveAttributes(body, "allow_overwrite", "hostname"):
//   resource "cloudflare_dns_record" "example" {
//     zone_id = "abc123"
//     name    = "test"
//     type    = "A"
//     content = "192.0.2.1"
//   }
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
//   typeAttr := body.GetAttribute("type")
//   recordType := ExtractStringFromAttribute(typeAttr)
//   // Returns "A" from: type = "A"
//   // Returns "var.record_type" from: type = var.record_type
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

// CreateNestedAttributeFromFields creates a nested attribute (object) from a map of field names to tokens.
// This is useful for restructuring flat attributes into nested objects (e.g., http_config, tcp_config).
//
// Example - Creating http_config from collected HTTP fields:
//
// Before:
//   resource "cloudflare_healthcheck" "example" {
//     zone_id = "abc123"
//     type    = "HTTP"
//     port    = 80
//     path    = "/health"
//     method  = "GET"
//   }
//
// After collecting fields and calling CreateNestedAttributeFromFields(body, "http_config", fields):
//   resource "cloudflare_healthcheck" "example" {
//     zone_id = "abc123"
//     type    = "HTTP"
//     http_config = {
//       port   = 80
//       path   = "/health"
//       method = "GET"
//     }
//   }
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
//   resource "cloudflare_healthcheck" "example" {
//     zone_id = "abc123"
//     type    = "HTTP"
//     port    = 80
//     path    = "/health"
//     method  = "GET"
//   }
//
// After calling MoveAttributesToNestedObject(body, "http_config", []string{"port", "path", "method"}):
//   resource "cloudflare_healthcheck" "example" {
//     zone_id = "abc123"
//     type    = "HTTP"
//     http_config = {
//       port   = 80
//       path   = "/health"
//       method = "GET"
//     }
//   }
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