package notification_policy

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_notification_policy", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_notification_policy"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_notification_policy"
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_notification_policy", "cloudflare_notification_policy"
}

// Preprocess performs any string-level transformations before HCL parsing.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Validate alert types and add diagnostics warnings for deprecated types
	m.validateAlertTypes(ctx, body)

	// Convert filters block to attribute (MaxItems:1 → SingleNestedAttribute)
	tfhcl.ConvertBlocksToAttribute(body, "filters", "filters", func(block *hclwrite.Block) {})

	// Restructure integration fields into mechanisms
	m.restructureIntegrationFields(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// validateAlertTypes checks for deprecated alert types and adds diagnostics warnings
func (m *V4ToV5Migrator) validateAlertTypes(ctx *transform.Context, body *hclwrite.Body) {
	alertTypeAttr := body.GetAttribute("alert_type")
	if alertTypeAttr == nil {
		return
	}

	alertType := tfhcl.ExtractStringFromAttribute(alertTypeAttr)
	if alertType == "" {
		return
	}

	deprecatedTypes := map[string]string{
		"weekly_account_overview": "This alert type was removed in v5. Please contact Cloudflare support for alternative notification options or choose from the available v5 alert types.",
		"workers_alert":           "This alert type was removed in v5. Please update to use a more specific alert type from the v5 provider. Refer to the Cloudflare API documentation for available alert types.",
	}

	if detail, isDeprecated := deprecatedTypes[alertType]; isDeprecated {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Alert type '%s' is not supported in v5", alertType),
			Detail:   fmt.Sprintf("%s Migration will complete successfully, but Terraform validation will fail until the alert_type is updated to a valid v5 value.", detail),
		})
	}
}

// restructureIntegrationFields converts v4 integration fields to v5 mechanisms structure
func (m *V4ToV5Migrator) restructureIntegrationFields(body *hclwrite.Body) {
	// Collect all integration blocks
	emailBlocks := tfhcl.FindBlocksByType(body, "email_integration")
	webhooksBlocks := tfhcl.FindBlocksByType(body, "webhooks_integration")
	pagerdutyBlocks := tfhcl.FindBlocksByType(body, "pagerduty_integration")

	if len(emailBlocks) == 0 && len(webhooksBlocks) == 0 && len(pagerdutyBlocks) == 0 {
		return
	}

	// Build mechanisms structure
	mechanismsTokens := []hclwrite.ObjectAttrTokens{}

	// Add email integrations
	if len(emailBlocks) > 0 {
		emailArray := m.buildIntegrationArray(emailBlocks)
		mechanismsTokens = append(mechanismsTokens, hclwrite.ObjectAttrTokens{
			Name:  hclwrite.TokensForIdentifier("email"),
			Value: emailArray,
		})
	}

	// Add webhooks integrations
	if len(webhooksBlocks) > 0 {
		webhooksArray := m.buildIntegrationArray(webhooksBlocks)
		mechanismsTokens = append(mechanismsTokens, hclwrite.ObjectAttrTokens{
			Name:  hclwrite.TokensForIdentifier("webhooks"),
			Value: webhooksArray,
		})
	}

	// Add pagerduty integrations
	if len(pagerdutyBlocks) > 0 {
		pagerdutyArray := m.buildIntegrationArray(pagerdutyBlocks)
		mechanismsTokens = append(mechanismsTokens, hclwrite.ObjectAttrTokens{
			Name:  hclwrite.TokensForIdentifier("pagerduty"),
			Value: pagerdutyArray,
		})
	}

	// Remove old integration blocks first
	tfhcl.RemoveBlocksByType(body, "email_integration")
	tfhcl.RemoveBlocksByType(body, "webhooks_integration")
	tfhcl.RemoveBlocksByType(body, "pagerduty_integration")

	// Create mechanisms attribute after removal
	body.SetAttributeRaw("mechanisms", hclwrite.TokensForObject(mechanismsTokens))
}

// buildIntegrationArray converts integration blocks to an array of objects with only id field
func (m *V4ToV5Migrator) buildIntegrationArray(blocks []*hclwrite.Block) hclwrite.Tokens {
	var arrayElements []hclwrite.Tokens

	for _, block := range blocks {
		blockBody := block.Body()

		// Extract id attribute
		idAttr := blockBody.GetAttribute("id")
		if idAttr == nil {
			continue
		}

		// Create object with only id field (name field is dropped)
		objTokens := hclwrite.TokensForObject([]hclwrite.ObjectAttrTokens{
			{
				Name:  hclwrite.TokensForIdentifier("id"),
				Value: idAttr.Expr().BuildTokens(nil),
			},
		})

		arrayElements = append(arrayElements, objTokens)
	}

	// Build array tokens using TokensForTuple (which creates arrays in HCL)
	return hclwrite.TokensForTuple(arrayElements)
}

// TransformState is a no-op for notification_policy.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
// The provider's UpgradeFromV4 function handles:
// - filters: MaxItems:1 array → SingleNestedAttribute object
// - Three integration Sets → single mechanisms nested object
// - Integration items: drop "name" field, keep only "id"
// - Filter fields: Set → List conversion (~35 fields)
// - Timestamps: String → RFC3339
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders
	// This function is a no-op for notification_policy migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
