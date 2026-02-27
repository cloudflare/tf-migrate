package zero_trust_local_fallback_domain

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

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
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	// Return one of the mappings - the actual logic is in TransformConfig
	return "cloudflare_zero_trust_local_fallback_domain", m.newTypeDefault
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

	// Convert domains blocks to attribute array
	tfhcl.ConvertBlocksToAttributeList(body, "domains", nil)

	// Generate moved block
	from := oldType + "." + resourceName
	to := newResourceType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState)
	// The moved block generated in TransformConfig triggers the provider's migration logic
	// This function is a no-op for zero_trust_local_fallback_domain migration
	return stateJSON.String(), nil
}
