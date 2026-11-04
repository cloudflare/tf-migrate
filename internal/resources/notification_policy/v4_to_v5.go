package notification_policy

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_notification_policy"}, "cloudflare_notification_policy"
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
	m.restructureIntegrationFields(ctx, body, tfhcl.GetResourceName(block))

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
func (m *V4ToV5Migrator) restructureIntegrationFields(ctx *transform.Context, body *hclwrite.Body, resourceName string) {
	// v4 integration block type -> v5 mechanisms key, in a fixed order so the
	// generated object is deterministic.
	integrations := []struct{ blockType, key string }{
		{"email_integration", "email"},
		{"webhooks_integration", "webhooks"},
		{"pagerduty_integration", "pagerduty"},
	}

	type pending struct {
		key    string
		tokens hclwrite.Tokens
	}
	var found []pending
	unconvertible := map[string]string{}
	skippedByType := map[string]int{}

	for _, integration := range integrations {
		staticBlocks := tfhcl.FindBlocksByType(body, integration.blockType)
		dynBlock := tfhcl.FindDynamicBlockFor(body, integration.blockType)

		switch {
		case dynBlock != nil && len(staticBlocks) > 0:
			// Splicing a for expression into a literal list would need
			// concat(); refuse rather than emit something subtly wrong.
			unconvertible[integration.blockType] = "it is mixed with static " +
				integration.blockType + " block(s)"

		case dynBlock != nil:
			tokens, ok, reason := tfhcl.BuildDynamicForListTokens(dynBlock, integration.blockType)
			if !ok {
				unconvertible[integration.blockType] = reason
				continue
			}
			found = append(found, pending{integration.key, tokens})
			body.RemoveBlock(dynBlock)

		case len(staticBlocks) > 0:
			arrayTokens, skipped := m.buildIntegrationArray(staticBlocks)
			if skipped > 0 {
				skippedByType[integration.blockType] = skipped
			}
			found = append(found, pending{integration.key, arrayTokens})
		}
	}

	if len(found) == 0 && len(unconvertible) == 0 {
		return
	}

	// Remove old integration blocks before writing the attribute.
	for _, integration := range integrations {
		tfhcl.RemoveBlocksByType(body, integration.blockType)
	}

	// An empty object would positively declare "no destinations", so only write
	// mechanisms when something actually converted.
	if len(found) > 0 {
		mechanismsTokens := make([]hclwrite.ObjectAttrTokens, 0, len(found))
		for _, p := range found {
			mechanismsTokens = append(mechanismsTokens, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForIdentifier(p.key),
				Value: p.tokens,
			})
		}
		body.SetAttributeRaw("mechanisms", hclwrite.TokensForObject(mechanismsTokens))
	}

	// A destination that cannot be represented is dropped, which silences the
	// alert for it, so it must be reported.
	for _, integration := range integrations {
		skipped, ok := skippedByType[integration.blockType]
		if !ok {
			continue
		}
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf(
				"Notification destination(s) dropped from cloudflare_notification_policy.%s",
				resourceName),
			Detail: fmt.Sprintf(
				"%d %s block(s) had no id attribute and have been dropped from "+
					"mechanisms.\n\n"+
					"v5 identifies a destination only by id, so these could not be "+
					"migrated. The policy will no longer notify them until the missing "+
					"id(s) are added to mechanisms by hand.\n\n"+
					"Note that the v4 name attribute is also dropped for destinations "+
					"that did migrate, because v5 accepts only id. That is cosmetic and "+
					"needs no action.",
				skipped, integration.blockType),
		})
	}

	// A dynamic block left in place has no v5 equivalent, so it must be reported.
	for _, integration := range integrations {
		reason, ok := unconvertible[integration.blockType]
		if !ok {
			continue
		}
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf(
				"Dynamic %s block not migrated on cloudflare_notification_policy.%s",
				integration.blockType, resourceName),
			Detail: fmt.Sprintf(
				"The dynamic %q block could not be converted automatically because %s.\n\n"+
					"It has been left in place, but v5 has no %s block: the policy will "+
					"not deliver to these destinations until it is moved into mechanisms "+
					"by hand, for example:\n\n"+
					"  mechanisms = {\n"+
					"    %s = [for k, v in <for_each> : { id = v }]\n"+
					"  }",
				integration.blockType, reason, integration.blockType, integration.key),
		})
	}
}

// buildIntegrationArray converts integration blocks to an array of objects with only id field
func (m *V4ToV5Migrator) buildIntegrationArray(blocks []*hclwrite.Block) (hclwrite.Tokens, int) {
	var arrayElements []hclwrite.Tokens
	skipped := 0

	for _, block := range blocks {
		blockBody := block.Body()

		// Extract id attribute
		idAttr := blockBody.GetAttribute("id")
		if idAttr == nil {
			// v5 mechanisms identify a destination only by id, so a block
			// without one cannot be represented and is dropped.
			skipped++
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
	return hclwrite.TokensForTuple(arrayElements), skipped
}
