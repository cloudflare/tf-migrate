package zero_trust_gateway_settings

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zero Trust Gateway Settings from v4 to v5
// v4: cloudflare_teams_account
// v5: cloudflare_zero_trust_gateway_settings
type V4ToV5Migrator struct {
	oldType string
	newType string
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType: "cloudflare_teams_account",
		newType: "cloudflare_zero_trust_gateway_settings",
	}
	internal.RegisterMigrator(migrator.oldType, "v4", "v5", migrator)
	internal.RegisterMigrator(migrator.newType, "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return m.newType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_teams_account" || resourceType == "cloudflare_zero_trust_gateway_settings"
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_teams_account", "cloudflare_zero_trust_gateway_settings"
}

// Preprocess - no preprocessing needed, all transformations done in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Get the original resource name for creating related resources
	resourceName := tfhcl.GetResourceName(block)

	// Track additional resources to create
	newResourceBlocks := []*hclwrite.Block{}

	body := block.Body()

	// Create new cloudflare_zero_trust_gateway_logging resource: logging -> cloudflare_zero_trust_gateway_logging
	if loggingBlock := tfhcl.FindBlockByType(body, "logging"); loggingBlock != nil {
		accountIDAttr := body.GetAttribute("account_id")
		newLoggingResource := m.createLoggingResource(resourceName, accountIDAttr, loggingBlock)
		if newLoggingResource != nil {
			newResourceBlocks = append(newResourceBlocks, newLoggingResource)
		}
	}

	// Create new cloudflare_zero_trust_device_settings resource: proxy -> cloudflare_zero_trust_device_settings
	if proxyBlock := tfhcl.FindBlockByType(body, "proxy"); proxyBlock != nil {
		if newDeviceSettingsResource := m.createDeviceSettingsResource(resourceName, body.GetAttribute("account_id"), proxyBlock); newDeviceSettingsResource != nil {
			newResourceBlocks = append(newResourceBlocks, newDeviceSettingsResource)
		}
	}

	// Remove blocks that we created new resources from
	tfhcl.RemoveBlocksByType(body, "logging")
	tfhcl.RemoveBlocksByType(body, "proxy")

	// TODO - Confirm these don't need to be recreated elsewhere
	tfhcl.RemoveBlocksByType(body, "ssh_session_log")
	tfhcl.RemoveBlocksByType(body, "payload_log")

	// Track rename before modifying block type
	wasRenamed := tfhcl.GetResourceType(block) == "cloudflare_teams_account"
	if wasRenamed {
		tfhcl.RenameResourceType(block, "cloudflare_teams_account", "cloudflare_zero_trust_gateway_settings")
	}

	// Compile all other settings into settings block
	settingsTokens := m.buildSettingsBlock(block)

	var allBlocks []*hclwrite.Block

	// When we have additional resources, create a fresh block for gateway settings to avoid formatting issues
	removeOriginal := len(newResourceBlocks) > 0 || wasRenamed
	if len(newResourceBlocks) > 0 {
		newGatewayBlock := m.createFreshGatewaySettingsBlock(block, settingsTokens)
		allBlocks = []*hclwrite.Block{newGatewayBlock}
		allBlocks = append(allBlocks, newResourceBlocks...)
	} else {
		allBlocks = []*hclwrite.Block{block}
	}

	// Generate moved block when the resource was renamed from cloudflare_teams_account
	if wasRenamed {
		oldType, newType := m.GetResourceRename()
		from := oldType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)
		allBlocks = append(allBlocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         allBlocks,
		RemoveOriginal: removeOriginal,
	}, nil
}

// Build the settings object with all nested content
// This is a complex transformation - all v4 settings must be nested under settings in v5
func (m *V4ToV5Migrator) buildSettingsBlock(block *hclwrite.Block) []hclwrite.ObjectAttrTokens {
	body := block.Body()
	settingsTokens := []hclwrite.ObjectAttrTokens{}

	// activity_log_enabled -> settings.activity_log.enabled
	if attr := body.GetAttribute("activity_log_enabled"); attr != nil {
		settingsTokens = append(settingsTokens, m.createEnabledNestedObject("activity_log", attr.Expr().BuildTokens(nil)))
	}

	// tls_decrypt_enabled -> settings.tls_decrypt.enabled
	if attr := body.GetAttribute("tls_decrypt_enabled"); attr != nil {
		settingsTokens = append(settingsTokens, m.createEnabledNestedObject("tls_decrypt", attr.Expr().BuildTokens(nil)))
	}

	// protocol_detection_enabled -> settings.protocol_detection.enabled
	if attr := body.GetAttribute("protocol_detection_enabled"); attr != nil {
		settingsTokens = append(settingsTokens, m.createEnabledNestedObject("protocol_detection", attr.Expr().BuildTokens(nil)))
	}

	// Browser isolation settings (two fields combined)
	// Always create browser_isolation block and add v4 defaults for missing fields
	// V4 had Default: false for both fields, v5 has no defaults
	browserIsolationTokens := []hclwrite.ObjectAttrTokens{}

	// url_browser_isolation_enabled: use explicit value or default to false
	if attr := body.GetAttribute("url_browser_isolation_enabled"); attr != nil {
		browserIsolationTokens = append(browserIsolationTokens, hclwrite.ObjectAttrTokens{
			Name:  hclwrite.TokensForIdentifier("url_browser_isolation_enabled"),
			Value: attr.Expr().BuildTokens(nil),
		})
	} else {
		// Add v4 default: false
		browserIsolationTokens = append(browserIsolationTokens, hclwrite.ObjectAttrTokens{
			Name:  hclwrite.TokensForIdentifier("url_browser_isolation_enabled"),
			Value: hclwrite.TokensForValue(cty.BoolVal(false)),
		})
	}

	// Rename: non_identity_browser_isolation_enabled -> non_identity_enabled
	// Use explicit value or default to false
	if attr := body.GetAttribute("non_identity_browser_isolation_enabled"); attr != nil {
		browserIsolationTokens = append(browserIsolationTokens, hclwrite.ObjectAttrTokens{
			Name:  hclwrite.TokensForIdentifier("non_identity_enabled"),
			Value: attr.Expr().BuildTokens(nil),
		})
	} else {
		// Add v4 default: false
		browserIsolationTokens = append(browserIsolationTokens, hclwrite.ObjectAttrTokens{
			Name:  hclwrite.TokensForIdentifier("non_identity_enabled"),
			Value: hclwrite.TokensForValue(cty.BoolVal(false)),
		})
	}

	// Always add browser_isolation block
	settingsTokens = append(settingsTokens, hclwrite.ObjectAttrTokens{
		Name:  hclwrite.TokensForIdentifier("browser_isolation"),
		Value: hclwrite.TokensForObject(browserIsolationTokens),
	})

	// 2. Convert and add MaxItems:1 blocks as nested attributes under settings
	blockNames := []string{
		"block_page",
		"body_scanning",
		"fips",
		"antivirus",
		"extended_email_matching",
		"custom_certificate",
		"certificate",
	}

	for _, blockName := range blockNames {
		blocks := tfhcl.FindBlocksByType(body, blockName)
		if len(blocks) > 0 {
			// Get the first block (MaxItems:1 means there's only one)
			block := blocks[0]

			// Special handling for antivirus - has nested notification_settings
			if blockName == "antivirus" {
				antivirusBody := block.Body()

				// First rename the field in the block
				notificationBlocks := tfhcl.FindBlocksByType(antivirusBody, "notification_settings")
				if len(notificationBlocks) > 0 {
					notifBody := notificationBlocks[0].Body()
					tfhcl.RenameAttribute(notifBody, "message", "msg")
				}

				// Convert block to attribute (handles finding, converting, and removing)
				tfhcl.ConvertSingleBlockToAttribute(antivirusBody, "notification_settings", "notification_settings")
			}

			// Build tokens from block using helper
			blockTokens := tfhcl.BuildObjectFromBlock(block)

			settingsTokens = append(settingsTokens, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForIdentifier(blockName),
				Value: blockTokens,
			})
		}
	}

	// Remove old top-level fields that are now under settings
	tfhcl.RemoveAttributes(body,
		"activity_log_enabled",
		"tls_decrypt_enabled",
		"protocol_detection_enabled",
		"url_browser_isolation_enabled",
		"non_identity_browser_isolation_enabled",
	)

	// Remove MaxItems:1 blocks
	for _, blockName := range blockNames {
		tfhcl.RemoveBlocksByType(body, blockName)
	}

	// Create the settings wrapper
	if len(settingsTokens) > 0 {
		body.SetAttributeRaw("settings", hclwrite.TokensForObject(settingsTokens))
	}

	return settingsTokens
}

// createEnabledNestedObject creates a nested object with a single "enabled" field.
// This is used for fields like activity_log_enabled -> activity_log { enabled = true }
func (m *V4ToV5Migrator) createEnabledNestedObject(parentName string, enabledValue hclwrite.Tokens) hclwrite.ObjectAttrTokens {
	return hclwrite.ObjectAttrTokens{
		Name: hclwrite.TokensForIdentifier(parentName),
		Value: hclwrite.TokensForObject([]hclwrite.ObjectAttrTokens{
			{
				Name:  hclwrite.TokensForIdentifier("enabled"),
				Value: enabledValue,
			},
		}),
	}
}

// TransformState is a no-op for zero_trust_gateway_settings.
// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState).
// The moved block generated in TransformConfig triggers the provider's migration logic.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	return instance.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}

// createLoggingResource creates a new cloudflare_zero_trust_gateway_logging resource
func (m *V4ToV5Migrator) createLoggingResource(baseName string, accountId *hclwrite.Attribute, loggingBlock *hclwrite.Block) *hclwrite.Block {
	if accountId == nil {
		return nil
	}

	resourceName := baseName + "_logging"
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zero_trust_gateway_logging", resourceName})
	body := block.Body()

	body.SetAttributeRaw("account_id", accountId.Expr().BuildTokens(nil))
	loggingBody := loggingBlock.Body()

	// Transform settings_by_rule_type from MaxItems:1 block to attribute
	settingsByRuleTypeBlocks := tfhcl.FindBlocksByType(loggingBody, "settings_by_rule_type")
	if len(settingsByRuleTypeBlocks) > 0 {
		settingsBlock := settingsByRuleTypeBlocks[0]
		settingsBody := settingsBlock.Body()

		// Build the settings_by_rule_type object
		settingsTokens := []hclwrite.ObjectAttrTokens{}

		// Convert dns, http, l4 blocks to attributes
		for _, ruleType := range []string{"dns", "http", "l4"} {
			ruleBlocks := tfhcl.FindBlocksByType(settingsBody, ruleType)
			if len(ruleBlocks) > 0 {
				ruleBlock := ruleBlocks[0]

				// Build the rule type object from its attributes using helper
				ruleTokens := tfhcl.BuildObjectFromBlock(ruleBlock)

				settingsTokens = append(settingsTokens, hclwrite.ObjectAttrTokens{
					Name:  hclwrite.TokensForIdentifier(ruleType),
					Value: ruleTokens,
				})
			}
		}

		if len(settingsTokens) > 0 {
			body.SetAttributeRaw("settings_by_rule_type", hclwrite.TokensForObject(settingsTokens))
		}
	}

	// Copy redact_pii if present
	tfhcl.CopyAttribute(loggingBody, body, "redact_pii")

	return block
}

// createFreshGatewaySettingsBlock creates a fresh cloudflare_zero_trust_gateway_settings block
func (m *V4ToV5Migrator) createFreshGatewaySettingsBlock(originalBlock *hclwrite.Block, settingsTokens []hclwrite.ObjectAttrTokens) *hclwrite.Block {
	// Get resource name from original block
	resourceName := ""
	if len(originalBlock.Labels()) >= 2 {
		resourceName = originalBlock.Labels()[1]
	}

	// Create fresh block
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zero_trust_gateway_settings", resourceName})
	body := block.Body()

	// Copy account_id from original
	originalBody := originalBlock.Body()
	tfhcl.CopyAttribute(originalBody, body, "account_id")
	body.AppendNewline() // Add blank line after account_id

	// Add settings if present
	if len(settingsTokens) > 0 {
		body.SetAttributeRaw("settings", hclwrite.TokensForObject(settingsTokens))
	}

	return block
}

// createDeviceSettingsResource creates a new cloudflare_zero_trust_device_settings resource
func (m *V4ToV5Migrator) createDeviceSettingsResource(baseName string, accountIDAttr *hclwrite.Attribute, proxyBlock *hclwrite.Block) *hclwrite.Block {
	if accountIDAttr == nil {
		return nil
	}

	resourceName := baseName + "_device_settings"
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zero_trust_device_settings", resourceName})
	body := block.Body()

	// Set account_id (reuse the expression from the main resource)
	body.SetAttributeRaw("account_id", accountIDAttr.Expr().BuildTokens(nil))

	proxyBody := proxyBlock.Body()

	// Map V4 proxy fields to V5 device settings fields
	fieldMappings := []struct {
		v4Name string
		v5Name string
	}{
		{"tcp", "gateway_proxy_enabled"},
		{"udp", "gateway_udp_proxy_enabled"},
		{"root_ca", "root_certificate_installation_enabled"},
		{"virtual_ip", "use_zt_virtual_ip"},
		{"disable_for_time", "disable_for_time"},
	}

	for _, mapping := range fieldMappings {
		if attr := proxyBody.GetAttribute(mapping.v4Name); attr != nil {
			body.SetAttributeRaw(mapping.v5Name, attr.Expr().BuildTokens(nil))
		}
	}

	return block
}
