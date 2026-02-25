package zones

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_zones datasource from v4 to v5.
// Key transformations:
// 1. Remove filter block wrapper - flatten fields to top-level
// 2. filter.account_id → account = { id = "..." }
// 3. filter.name → name (hoist to top-level)
// 4. filter.status → status (hoist to top-level)
// 5. Drop: filter.lookup_type, filter.match, filter.paused (not supported in v5)
// 6. Cross-file references: zones → result (via GetAttributeRenames)
// 7. State transformation is a no-op (datasources are always re-read from the API)
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_zones datasource v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with "data." prefix to distinguish from resource migration
	internal.RegisterMigrator("data.cloudflare_zones", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 datasource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zones"
}

// GetAttributeRenames returns attribute renames for cross-file reference updates.
// The zones datasource changed its output attribute from "zones" to "result".
func (m *V4ToV5Migrator) GetAttributeRenames() []transform.AttributeRename {
	return []transform.AttributeRename{
		{
			ResourceType: "data.cloudflare_zones",
			OldAttribute: "zones",
			NewAttribute: "result",
		},
	}
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_zones datasource doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "data.cloudflare_zones", "data.cloudflare_zones"
}

// CanHandle determines if this migrator can handle the given datasource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Only match datasource type (with "data." prefix)
	return resourceType == "data.cloudflare_zones"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for zones datasource migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
// Main transformation: Remove filter block and flatten/restructure fields
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Find filter block
	filterBlocks := tfhcl.FindBlocksByType(body, "filter")
	if len(filterBlocks) == 0 {
		// No filter block - nothing to transform
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	filterBlock := filterBlocks[0]
	filterBody := filterBlock.Body()

	// Extract filter attributes
	accountIdAttr := filterBody.GetAttribute("account_id")
	nameAttr := filterBody.GetAttribute("name")
	statusAttr := filterBody.GetAttribute("status")
	lookupTypeAttr := filterBody.GetAttribute("lookup_type")
	matchAttr := filterBody.GetAttribute("match")
	pausedAttr := filterBody.GetAttribute("paused")

	// Transform account_id → account = { id = "..." }
	if accountIdAttr != nil {
		m.setAccountAttribute(body, accountIdAttr)
	}

	// Hoist name to top-level
	if nameAttr != nil {
		body.SetAttributeRaw("name", nameAttr.Expr().BuildTokens(nil))
	}

	// Hoist status to top-level
	if statusAttr != nil {
		body.SetAttributeRaw("status", statusAttr.Expr().BuildTokens(nil))
	}

	// Warn about dropped fields - these cannot be automatically migrated
	// Users will need to manually adjust their configs if they relied on these fields
	if lookupTypeAttr != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Field 'filter.lookup_type' cannot be automatically migrated",
			Detail:   "In v5, use name filter operators directly in the name field (e.g., 'contains:example', 'starts_with:test'). See v5 documentation for available operators.",
		})
	}
	if matchAttr != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Field 'filter.match' has been dropped",
			Detail:   "v4's filter.match was a regex pattern for client-side filtering. v5's 'match' attribute has a completely different meaning (any/all combinator for search requirements). Manual migration required.",
		})
	}
	if pausedAttr != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Field 'filter.paused' is not supported in v5",
			Detail:   "The paused filter is not available in the v5 zones datasource. You will need to filter paused zones in your Terraform code after fetching the results.",
		})
	}

	// Remove filter block
	tfhcl.RemoveBlocksByType(body, "filter")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// setAccountAttribute creates account = { id = "..." } from v4's account_id
// Uses manual HCL token construction since account is a SingleNestedAttribute
func (m *V4ToV5Migrator) setAccountAttribute(body *hclwrite.Body, accountIdAttr *hclwrite.Attribute) {
	// Build: account = { id = <value> }
	accountIdTokens := accountIdAttr.Expr().BuildTokens(nil)

	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("  id")},
		{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
	}
	tokens = append(tokens, accountIdTokens...)
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
	)

	body.SetAttributeRaw("account", tokens)
}

// TransformState is a no-op for cloudflare_zones datasource migration.
//
// Datasources are always re-read from the API on the next plan/apply, so state
// transformation is unnecessary. tf-migrate's role for datasources is limited to
// transforming HCL configuration syntax (handled by TransformConfig) and updating
// cross-file attribute references (handled by GetAttributeRenames).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	return instance.String(), nil
}

func init() {
	// Register the migrator on package initialization
	NewV4ToV5Migrator()
}
