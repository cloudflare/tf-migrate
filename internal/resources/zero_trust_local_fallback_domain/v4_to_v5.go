package zero_trust_local_fallback_domain

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

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
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Note: This is complex because we have conditional renaming
// We'll return the default profile name here, but actual renaming happens in TransformConfig
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	// Return one of the mappings - the actual logic is in TransformConfig
	return "cloudflare_zero_trust_local_fallback_domain", m.newTypeDefault
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
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
	tfhcl.RenameResourceType(block, currentType, newResourceType)

	// Remove policy_id attribute if it's null (default profile doesn't accept policy_id)
	if !hasPolicyID && policyIDAttr != nil {
		body.RemoveAttribute("policy_id")
	}

	// Convert domains blocks to attribute array
	tfhcl.ConvertBlocksToAttributeList(body, "domains", nil)

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
		m.lastTransformedType = m.newTypeDefault
		return result, nil
	}

	// Check if policy_id exists and is not null
	policyID := attrs.Get("policy_id")
	hasPolicyID := policyID.Exists() && policyID.Type != gjson.Null
	if hasPolicyID {
		m.lastTransformedType = m.newTypeCustom
	} else {
		m.lastTransformedType = m.newTypeDefault
	}

	transform.SetStateTypeRename(ctx, resourceName, m.oldType, m.lastTransformedType)
	transform.SetStateTypeRename(ctx, resourceName, m.oldTypeDeprecated, m.lastTransformedType)

	// Remove policy_id from state if it's null (default profile doesn't have policy_id)
	if !hasPolicyID && policyID.Exists() {
		result, _ = sjson.Delete(result, "attributes.policy_id")
	}

	// Transform domains (TypeSet → ListNestedAttribute/SetNestedAttribute)
	// The state structure is the same for both v5 variants, just different collection semantics
	// v4: domains is a set of objects
	// v5: domains is either a list (default profile) or set (custom profile) of objects
	// The JSON structure is identical, so no transformation needed
	domains := attrs.Get("domains")
	if domains.Exists() && domains.IsArray() {
		if len(domains.Array()) == 0 {
			result, _ = sjson.Delete(result, "attributes.domains")
		}
	}

	// Handle dns_server empty arrays within each domain
	if domains.Exists() && domains.IsArray() {
		domains.ForEach(func(key, value gjson.Result) bool {
			dnsServer := value.Get("dns_server")
			if dnsServer.Exists() && dnsServer.IsArray() && len(dnsServer.Array()) == 0 {
				result, _ = sjson.Delete(result, "attributes.domains."+key.String()+".dns_server")
			}
			return true
		})
	}

	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
