package zero_trust_gateway_policy

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
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
	// Get resource name before transformation for moved block
	resourceName := tfhcl.GetResourceName(block)

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

	// Generate moved block for resource rename
	oldType, newType := m.GetResourceRename()
	from := oldType + "." + resourceName
	to := newType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
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

		// For biso_admin_controls, rename v1 fields BEFORE converting
		// v1 fields were renamed from disable_* to shortened versions (dp, dd, dcp, dk, du)
		if blockName == "biso_admin_controls" {
			if bisoBlock := tfhcl.FindBlockByType(ruleSettingsBody, "biso_admin_controls"); bisoBlock != nil {
				bisoBody := bisoBlock.Body()
				// Rename v1 fields to shortened versions
				tfhcl.RenameAttribute(bisoBody, "disable_printing", "dp")
				tfhcl.RenameAttribute(bisoBody, "disable_copy_paste", "dcp")
				tfhcl.RenameAttribute(bisoBody, "disable_download", "dd")
				tfhcl.RenameAttribute(bisoBody, "disable_keyboard", "dk")
				tfhcl.RenameAttribute(bisoBody, "disable_upload", "du")
				// Remove disable_clipboard_redirection (no v5 equivalent)
				tfhcl.RemoveAttributes(bisoBody, "disable_clipboard_redirection")
			}
		}

		// Convert block to attribute syntax
		tfhcl.ConvertSingleBlockToAttribute(ruleSettingsBody, blockName, blockName)
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState)
	// The moved block generated in TransformConfig triggers the provider's migration logic
	// This function is a no-op for zero_trust_gateway_policy migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
