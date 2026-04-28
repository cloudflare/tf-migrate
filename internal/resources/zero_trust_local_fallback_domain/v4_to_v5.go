package zero_trust_local_fallback_domain

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
	oldType             string
	oldTypeDeprecated   string
	newTypeDefault      string
	newTypeCustom       string
	lastTransformedType string
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType:             "cloudflare_zero_trust_local_fallback_domain",
		oldTypeDeprecated:   "cloudflare_fallback_domain",
		newTypeDefault:      "cloudflare_zero_trust_device_default_profile_local_domain_fallback",
		newTypeCustom:       "cloudflare_zero_trust_device_custom_profile_local_domain_fallback",
		lastTransformedType: "",
	}

	internal.RegisterMigrator("cloudflare_zero_trust_local_fallback_domain", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_fallback_domain", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the type used in the last transformation
	// If not set, default to default profile
	if m.lastTransformedType != "" {
		return m.lastTransformedType
	}
	return m.newTypeDefault
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == m.oldType || resourceType == m.oldTypeDeprecated
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Update resource references for migrated device profiles
	// Device profiles migrate to either:
	// - cloudflare_zero_trust_device_custom_profile (if they have match AND precedence)
	// - cloudflare_zero_trust_device_default_profile (otherwise)
	//
	// We need to update policy_id references like:
	//   policy_id = cloudflare_zero_trust_device_profiles.example.id
	// to:
	//   policy_id = cloudflare_zero_trust_device_custom_profile.example.id
	//
	// Strategy: Parse HCL to find device profile resources, determine their type, and replace references

	// Parse HCL to find device profile resources
	file, diags := hclwrite.ParseConfig([]byte(content), "", hcl.InitialPos)
	if diags.HasErrors() {
		// If parsing fails, return content unchanged
		return content
	}

	// Build a map of resource reference → new resource type
	resourceTypeMap := make(map[string]string)

	for _, block := range file.Body().Blocks() {
		if block.Type() != "resource" {
			continue
		}

		labels := block.Labels()
		if len(labels) != 2 {
			continue
		}

		oldResourceType := labels[0]
		resourceName := labels[1]

		// Only process device profile resources
		if oldResourceType != "cloudflare_zero_trust_device_profiles" && oldResourceType != "cloudflare_device_settings_policy" {
			continue
		}

		body := block.Body()

		// Determine if this is a custom profile (has match AND precedence)
		matchAttr := body.GetAttribute("match")
		precedenceAttr := body.GetAttribute("precedence")
		defaultAttr := body.GetAttribute("default")

		hasMatch := matchAttr != nil
		hasPrecedence := precedenceAttr != nil

		// Check if default is explicitly set to true
		isExplicitDefault := false
		if defaultAttr != nil {
			exprTokens := defaultAttr.Expr().BuildTokens(nil)
			if len(exprTokens) == 1 && string(exprTokens[0].Bytes) == "true" {
				isExplicitDefault = true
			}
		}

		// If default=true explicitly, it's a default profile (even if match/precedence present)
		// Otherwise, if it has match AND precedence, it's a custom profile
		isCustomProfile := !isExplicitDefault && hasMatch && hasPrecedence

		var newResourceType string
		if isCustomProfile {
			newResourceType = "cloudflare_zero_trust_device_custom_profile"
		} else {
			newResourceType = "cloudflare_zero_trust_device_default_profile"
		}

		// Store mapping for both old resource type names
		resourceTypeMap[oldResourceType+"."+resourceName] = newResourceType + "." + resourceName
	}

	// Replace all references in the content
	result := content
	for oldRef, newRef := range resourceTypeMap {
		result = strings.ReplaceAll(result, oldRef, newRef)
	}

	return result
}

// GetResourceRename implements the ResourceRenamer interface
// Note: This is complex because we have conditional renaming
// We'll return the default profile name here, but actual renaming happens in TransformConfig
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	// Return one of the mappings - the actual logic is in TransformConfig
	return []string{"cloudflare_zero_trust_local_fallback_domain", "cloudflare_fallback_domain"}, m.newTypeDefault
}

// isOpaqueForEach returns true when the for_each token stream does not look like
// a simple literal collection (e.g. toset([...]) or [...]).  References such as
// local.some_var or var.domains are considered opaque because we cannot verify
// the iterator content statically.
func isOpaqueForEach(tokens hclwrite.Tokens) bool {
	// Strip surrounding whitespace/newline tokens to get the first meaningful token.
	for _, tok := range tokens {
		switch tok.Type {
		case hclsyntax.TokenNewline, hclsyntax.TokenIdent:
			name := strings.TrimSpace(string(tok.Bytes))
			// Literal function calls that produce inline collections are safe.
			// Everything else (var, local, module references) is opaque.
			switch name {
			case "toset", "tolist", "concat", "flatten", "":
				return false
			}
			// An identifier that is not a known safe built-in → opaque.
			return true
		case hclsyntax.TokenOBrack: // starts with "[" → inline tuple → safe
			return false
		}
	}
	return false
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceName := tfhcl.GetResourceName(block)
	body := block.Body()

	// Check if policy_id exists and is not null
	policyIDAttr := body.GetAttribute("policy_id")
	hasPolicyID := false
	if policyIDAttr != nil {
		// Check if the value is not null
		exprTokens := policyIDAttr.Expr().BuildTokens(nil)
		// If the expression is just "null", treat it as not having policy_id
		if len(exprTokens) != 1 || string(exprTokens[0].Bytes) != "null" {
			hasPolicyID = true
		}
	}

	var newResourceType string
	if hasPolicyID {
		newResourceType = m.newTypeCustom
	} else {
		newResourceType = m.newTypeDefault
	}
	m.lastTransformedType = newResourceType

	// Rename resource type to appropriate v5 resource
	currentType := tfhcl.GetResourceType(block)
	oldType := currentType // Store original type for moved block
	tfhcl.RenameResourceType(block, currentType, newResourceType)

	// Remove policy_id attribute if it's null (default profile doesn't accept policy_id)
	if !hasPolicyID && policyIDAttr != nil {
		body.RemoveAttribute("policy_id")
	}

	// Convert dynamic "domains" blocks to for-expressions before static block conversion.
	// Many users write:
	//   dynamic "domains" { for_each = toset([...]) content { suffix = domains.value } }
	// The v5 provider expects:
	//   domains = [for value in toset([...]) : { suffix = value }]
	//
	// When multiple dynamic "domains" blocks exist, they are automatically merged
	// via concat() by ConvertDynamicBlocksToForExpression.
	dynamicDomainsCount := 0
	for _, dynBlock := range tfhcl.FindBlocksByType(body, "dynamic") {
		labels := dynBlock.Labels()
		if len(labels) == 0 || labels[0] != "domains" {
			continue
		}
		dynamicDomainsCount++
		// Detect whether the for_each expression is opaque (not a literal).
		// An opaque for_each references a variable or local that cannot be
		// statically resolved — we still attempt conversion but warn the user.
		dynBody := dynBlock.Body()
		if forEachAttr := dynBody.GetAttribute("for_each"); forEachAttr != nil {
			tokens := forEachAttr.Expr().BuildTokens(nil)
			if isOpaqueForEach(tokens) {
				ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  fmt.Sprintf("Dynamic 'domains' block requires manual verification: %s.%s", newResourceType, resourceName),
					Detail: `A dynamic "domains" block with a non-literal for_each expression has been converted to a for-expression.
Please verify the generated output is correct for your configuration.

The v5 provider uses 'domains' as a list attribute instead of blocks.
Expected output:
  domains = [for value in <expr> : {
    suffix      = value.suffix
    description = value.description
  }]`,
				})
			}
		}
	}

	if dynamicDomainsCount > 1 {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Multiple dynamic 'domains' blocks merged via concat(): %s.%s", newResourceType, resourceName),
			Detail: fmt.Sprintf(`%d dynamic "domains" blocks were found and merged into a single attribute using concat().
Please verify the generated output preserves all intended domain entries and ordering.`, dynamicDomainsCount),
		})
	}

	tfhcl.ConvertDynamicBlocksToForExpression(body, "domains")

	// Convert static domains blocks to attribute array.
	// If dynamic blocks already produced a "domains" attribute, we need to merge
	// the static blocks into it via concat() rather than overwriting.
	staticDomainsBlocks := tfhcl.FindBlocksByType(body, "domains")
	hasDynamicDomains := dynamicDomainsCount > 0 && body.GetAttribute("domains") != nil

	if hasDynamicDomains && len(staticDomainsBlocks) > 0 {
		// Mixed case: merge dynamic for-expression(s) and static blocks via concat().
		existingTokens := body.GetAttribute("domains").Expr().BuildTokens(nil)
		tfhcl.MergeStaticBlocksIntoAttribute(body, "domains", existingTokens)

		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Mixed static and dynamic 'domains' blocks merged via concat(): %s.%s", newResourceType, resourceName),
			Detail:   "Both static domains blocks and dynamic domains blocks were found. They have been merged into a single attribute using concat(). Please verify the generated output.",
		})
	} else {
		tfhcl.ConvertBlocksToAttributeList(body, "domains", nil)
	}

	// Generate moved block
	from := oldType + "." + resourceName
	to := newResourceType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}
