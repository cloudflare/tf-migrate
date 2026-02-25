package zone

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_zone datasource from v4 to v5.
// Key transformations:
// 1. account_id and/or name → filter = { ... } (wrap in filter attribute)
// 2. zone_id lookups remain unchanged
// 3. State transformation is a no-op (datasources are always re-read from the API)
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_zone datasource v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with "data." prefix to distinguish from resource migration
	internal.RegisterMigrator("data.cloudflare_zone", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 datasource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zone"
}

// CanHandle determines if this migrator can handle the given datasource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Only match datasource type (with "data." prefix)
	return resourceType == "data.cloudflare_zone"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for zone datasource migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_zone datasource doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "data.cloudflare_zone", "data.cloudflare_zone"
}

// TransformConfig handles configuration file transformations.
// Scenarios:
// 1. zone_id only → No changes
// 2. name only → filter = { name = "..." }
// 3. account_id only → filter = { account = { id = "..." } }
// 4. name + account_id → filter = { name = "...", account = { id = "..." } }
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Check if we have name or account_id (which need to move to filter)
	nameAttr := body.GetAttribute("name")
	accountIdAttr := body.GetAttribute("account_id")

	// Scenario 1: zone_id lookup - no changes needed
	if nameAttr == nil && accountIdAttr == nil {
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	// Scenarios 2-4: Create filter attribute
	m.setFilterAttribute(body, nameAttr, accountIdAttr)

	// Remove old attributes
	if nameAttr != nil {
		body.RemoveAttribute("name")
	}
	if accountIdAttr != nil {
		body.RemoveAttribute("account_id")
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// setFilterAttribute creates the filter = { ... } attribute structure
// Handles 3 cases:
// 1. name only: filter = { name = "..." }
// 2. account_id only: filter = { account = { id = "..." } }
// 3. both: filter = { name = "...", account = { id = "..." } }
func (m *V4ToV5Migrator) setFilterAttribute(body *hclwrite.Body, nameAttr, accountIdAttr *hclwrite.Attribute) {
	// Start building filter object: filter = {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	}

	// Add name field if present
	if nameAttr != nil {
		nameTokens := nameAttr.Expr().BuildTokens(nil)
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  name")},
			&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
		)
		tokens = append(tokens, nameTokens...)
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		)
	}

	// Add account = { id = "..." } if present
	if accountIdAttr != nil {
		accountIdTokens := accountIdAttr.Expr().BuildTokens(nil)
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  account")},
			&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
			&hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    id")},
			&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
		)
		tokens = append(tokens, accountIdTokens...)
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")},
			&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		)
	}

	// Close filter object
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
	)

	// Set the filter attribute
	body.SetAttributeRaw("filter", tokens)
}

// TransformState is a no-op for cloudflare_zone datasource migration.
//
// Datasources are always re-read from the API on the next plan/apply, so state
// transformation is unnecessary. tf-migrate's role for datasources is limited to
// transforming HCL configuration syntax (handled by TransformConfig).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}
