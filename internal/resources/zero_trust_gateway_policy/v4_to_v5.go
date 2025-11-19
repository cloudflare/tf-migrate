package zero_trust_gateway_policy

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of Zero Trust Gateway Policy resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the OLD (v4) resource name: cloudflare_teams_rule
	internal.RegisterMigrator("cloudflare_teams_rule", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_gateway_policy"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_teams_rule"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations can be done with HCL helpers
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_teams_rule", "cloudflare_zero_trust_gateway_policy"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename resource type: cloudflare_teams_rule → cloudflare_zero_trust_gateway_policy
	tfhcl.RenameResourceType(block, "cloudflare_teams_rule", "cloudflare_zero_trust_gateway_policy")

	body := block.Body()

	// Process rule_settings if it exists
	if ruleSettingsBlock := tfhcl.FindBlockByType(body, "rule_settings"); ruleSettingsBlock != nil {
		m.processRuleSettingsBlock(ruleSettingsBlock)
	}

	// Convert rule_settings block to attribute syntax
	// This must be done AFTER processing nested blocks
	tfhcl.ConvertSingleBlockToAttribute(body, "rule_settings", "rule_settings")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// processRuleSettingsBlock processes all nested structures within rule_settings
func (m *V4ToV5Migrator) processRuleSettingsBlock(ruleSettingsBlock *hclwrite.Block) {
	ruleSettingsBody := ruleSettingsBlock.Body()

	// Rename fields at rule_settings level
	tfhcl.RenameAttribute(ruleSettingsBody, "block_page_reason", "block_reason")

	// Convert all nested MaxItems:1 blocks to attributes
	// These blocks need to be converted to attribute syntax with =
	nestedBlocks := []string{
		"audit_ssh",
		"l4override",
		"biso_admin_controls",
		"check_session",
		"egress",
		"untrusted_cert",
		"payload_log",
		"notification_settings",
		"dns_resolvers",
		"resolve_dns_internally",
	}

	for _, blockName := range nestedBlocks {
		// For notification_settings, rename message → msg BEFORE converting
		if blockName == "notification_settings" {
			if notifBlock := tfhcl.FindBlockByType(ruleSettingsBody, "notification_settings"); notifBlock != nil {
				tfhcl.RenameAttribute(notifBlock.Body(), "message", "msg")
			}
		}

		// Convert block to attribute syntax
		tfhcl.ConvertSingleBlockToAttribute(ruleSettingsBody, blockName, blockName)
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// This function receives a single instance and needs to return the transformed instance JSON
	result := stateJSON.String()
	// Get attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Convert precedence from int to float64
	if precedence := attrs.Get("precedence"); precedence.Exists() {
		floatVal := state.ConvertToFloat64(precedence)
		result, _ = sjson.Set(result, "attributes.precedence", floatVal)
	}

	// Convert version from int to float64
	if version := attrs.Get("version"); version.Exists() {
		floatVal := state.ConvertToFloat64(version)
		result, _ = sjson.Set(result, "attributes.version", floatVal)
	}

	// Transform rule_settings if it exists
	ruleSettings := attrs.Get("rule_settings")
	if ruleSettings.Exists() {
		result = m.transformRuleSettings(result, ruleSettings)
	}

	// Always set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// transformRuleSettings handles the transformation of rule_settings from array to object
func (m *V4ToV5Migrator) transformRuleSettings(result string, ruleSettings gjson.Result) string {
	// Check if rule_settings is an array (v4 format: [{...}])
	if ruleSettings.IsArray() {
		// Get the first element of the array (v4 always has exactly one element)
		arr := ruleSettings.Array()
		if len(arr) > 0 {
			settingsObj := arr[0]

			// Convert the settings object
			transformedSettings := m.transformRuleSettingsObject(settingsObj)

			// Set as object (not array)
			result, _ = sjson.SetRaw(result, "attributes.rule_settings", transformedSettings)
		}
	} else if ruleSettings.IsObject() {
		// Already an object, just transform it
		transformedSettings := m.transformRuleSettingsObject(ruleSettings)
		result, _ = sjson.SetRaw(result, "attributes.rule_settings", transformedSettings)
	}

	return result
}

// transformRuleSettingsObject transforms a single rule_settings object
func (m *V4ToV5Migrator) transformRuleSettingsObject(settings gjson.Result) string {
	result := settings.String()

	// Rename block_page_reason → block_reason
	if settings.Get("block_page_reason").Exists() {
		value := settings.Get("block_page_reason").Value()
		result, _ = sjson.Set(result, "block_reason", value)
		result, _ = sjson.Delete(result, "block_page_reason")
	}

	// Transform nested structures (convert arrays to objects)
	nestedStructures := []string{
		"audit_ssh",
		"l4override",
		"biso_admin_controls",
		"check_session",
		"egress",
		"untrusted_cert",
		"payload_log",
		"notification_settings",
		"dns_resolvers",
		"resolve_dns_internally",
	}

	for _, structName := range nestedStructures {
		nested := settings.Get(structName)

		// Skip if field doesn't exist or is not an array
		if !nested.Exists() || !nested.IsArray() {
			continue
		}

		// Handle empty array - remove it (v5 uses optional SingleNestedAttribute, so null is correct)
		arr := nested.Array()
		if len(arr) == 0 {
			result, _ = sjson.Delete(result, structName)
			continue
		}

		// Convert from array to object (get first element)
		nestedObj := arr[0]

		// Special handling for specific structures
		if structName == "notification_settings" {
			// Rename message → msg
			nestedResult := nestedObj.String()
			if nestedObj.Get("message").Exists() {
				value := nestedObj.Get("message").Value()
				nestedResult, _ = sjson.Set(nestedResult, "msg", value)
				nestedResult, _ = sjson.Delete(nestedResult, "message")
			}
			result, _ = sjson.SetRaw(result, structName, nestedResult)
		} else if structName == "l4override" {
			// Convert port to float64
			nestedResult := nestedObj.String()
			if port := nestedObj.Get("port"); port.Exists() {
				nestedResult, _ = sjson.Set(nestedResult, "port", state.ConvertToFloat64(port))
			}
			result, _ = sjson.SetRaw(result, structName, nestedResult)
		} else if structName == "dns_resolvers" {
			// Handle dns_resolvers specially - it has ipv4/ipv6 arrays with port fields
			result = m.transformDnsResolvers(result, nestedObj)
		} else if structName == "biso_admin_controls" {
			// Remove deprecated v1-only disable_* attributes that were removed in v5
			// These were replaced with new attributes (e.g., disable_printing → printing with values "enabled"/"disabled")
			nestedResult := nestedObj.String()
			nestedResult, _ = sjson.Delete(nestedResult, "disable_clipboard_redirection")
			nestedResult, _ = sjson.Delete(nestedResult, "disable_printing")
			nestedResult, _ = sjson.Delete(nestedResult, "disable_copy_paste")
			nestedResult, _ = sjson.Delete(nestedResult, "disable_download")
			nestedResult, _ = sjson.Delete(nestedResult, "disable_keyboard")
			nestedResult, _ = sjson.Delete(nestedResult, "disable_upload")
			result, _ = sjson.SetRaw(result, structName, nestedResult)
		} else {
			// Just convert array to object
			result, _ = sjson.SetRaw(result, structName, nestedObj.String())
		}
	}

	return result
}

// transformDnsResolvers handles the transformation of dns_resolvers with port conversions
func (m *V4ToV5Migrator) transformDnsResolvers(result string, dnsResolvers gjson.Result) string {
	resolversResult := dnsResolvers.String()

	// Transform ipv4 array - convert each port to float64
	if ipv4 := dnsResolvers.Get("ipv4"); ipv4.Exists() && ipv4.IsArray() {
		ipv4.ForEach(func(key, value gjson.Result) bool {
			if port := value.Get("port"); port.Exists() {
				resolversResult, _ = sjson.Set(resolversResult, "ipv4."+key.String()+".port", state.ConvertToFloat64(port))
			}
			return true
		})
	}

	// Transform ipv6 array - convert each port to float64
	if ipv6 := dnsResolvers.Get("ipv6"); ipv6.Exists() && ipv6.IsArray() {
		ipv6.ForEach(func(key, value gjson.Result) bool {
			if port := value.Get("port"); port.Exists() {
				resolversResult, _ = sjson.Set(resolversResult, "ipv6."+key.String()+".port", state.ConvertToFloat64(port))
			}
			return true
		})
	}

	result, _ = sjson.SetRaw(result, "dns_resolvers", resolversResult)
	return result
}
