package zones_data_source

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zones datasource from v4 to v5
// This is the PLURAL/LIST datasource that returns multiple zones
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register datasource with the OLD (v4) name
	// Note: Datasources use the same name in v4 and v5
	internal.RegisterMigrator("cloudflare_zones", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) datasource name (unchanged)
	return "cloudflare_zones"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) datasource name
	return resourceType == "cloudflare_zones"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for zones datasource
	// All transformations can be done at HCL level
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No resource type rename needed (stays cloudflare_zones)
	body := block.Body()

	// Find the filter block (v4 uses TypeList MaxItems:1 which creates a block)
	filterBlocks := tfhcl.FindBlocksByType(body, "filter")

	if len(filterBlocks) == 0 {
		// No filter block - nothing to transform
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	// Get the first (and only) filter block
	filterBlock := filterBlocks[0]
	filterBody := filterBlock.Body()

	// Extract attributes from filter block
	nameAttr := filterBody.GetAttribute("name")
	statusAttr := filterBody.GetAttribute("status")
	accountIDAttr := filterBody.GetAttribute("account_id")
	matchAttr := filterBody.GetAttribute("match")
	lookupTypeAttr := filterBody.GetAttribute("lookup_type")
	pausedAttr := filterBody.GetAttribute("paused")

	// Warn about non-migratable fields using HCL diagnostics
	if matchAttr != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Non-migratable field: filter.match",
			Detail:   "The filter.match field (RE2 regex) cannot be migrated to v5. In v5, the 'match' field has a different meaning ('any'/'all' for combining filters). This field will be removed during migration.",
		})
	}
	if lookupTypeAttr != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Non-migratable field: filter.lookup_type",
			Detail:   "The filter.lookup_type field ('contains'/'exact') cannot be migrated to v5. In v5, use filter operators in the name field itself (e.g., name = 'contains:example'). This field will be removed during migration.",
		})
	}
	if pausedAttr != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Non-migratable field: filter.paused",
			Detail:   "The filter.paused field cannot be migrated to v5 as there is no equivalent input filter in v5. The 'paused' field is only available in the output/result data. This field will be removed during migration.",
		})
	}

	// Hoist simple attributes to root level
	if nameAttr != nil {
		tokens := nameAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("name", tokens)
	}

	if statusAttr != nil {
		tokens := statusAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("status", tokens)
	}

	// Transform account_id to nested account.id structure
	if accountIDAttr != nil {
		m.createAccountAttribute(body, accountIDAttr)
	}

	// Remove the filter block
	body.RemoveBlock(filterBlock)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// createAccountAttribute creates the account attribute with nested id structure
// account = {
//   id = "abc123"
// }
func (m *V4ToV5Migrator) createAccountAttribute(body *hclwrite.Body, accountIDAttr *hclwrite.Attribute) {
	// Build the account object tokens manually
	tokens := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}},
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
		&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("id")},
		&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte{'='}},
	}

	// Copy the value tokens from the original account_id attribute
	tokens = append(tokens, accountIDAttr.Expr().BuildTokens(nil)...)

	// Close the account object
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}},
	)

	// Set the account attribute
	body.SetAttributeRaw("account", tokens)
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result, nil
	}

	// 1. Rename zones → result
	result = state.RenameField(result, "attributes", attrs, "zones", "result")

	// 2. Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Note: The filter attribute in state is kept as-is
	// The provider will handle state refresh and populate new computed fields
	// Most new v5 fields (account, activated_on, etc.) in the result array are computed
	// and will be populated by the provider on first refresh

	return result, nil
}
