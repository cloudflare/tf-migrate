package zone

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_zone from v4 to v5.
// Key transformations:
// 1. zone → name (field rename)
// 2. account_id (string) → account = { id = "..." } (structure change)
// 3. Remove jump_start (no v5 equivalent)
// 4. Remove plan (becomes computed-only in v5)
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_zone v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name
	internal.RegisterMigrator("cloudflare_zone", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zone"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zone"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for zone migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_zone doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_zone", "cloudflare_zone"
}

// TransformConfig handles configuration file transformations.
// Transformations:
// 1. zone → name
// 2. account_id (string) → account = { id = "..." }
// 3. Remove jump_start
// 4. Remove plan
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Rename zone → name
	tfhcl.RenameAttribute(body, "zone", "name")

	// 2. Transform account_id → account = { id = "..." }
	if accountIdAttr := body.GetAttribute("account_id"); accountIdAttr != nil {
		// Get the expression tokens (preserves variables, literals, etc.)
		accountIdTokens := accountIdAttr.Expr().BuildTokens(nil)

		// Build: account = { id = <expr> }
		// We need to create the nested object structure manually
		m.setAccountNestedAttribute(body, accountIdTokens)

		// Remove old account_id attribute
		body.RemoveAttribute("account_id")
	}

	// 3-4. Remove obsolete attributes
	tfhcl.RemoveAttributes(body, "jump_start", "plan")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// setAccountNestedAttribute creates the nested account = { id = <expr> } structure
func (m *V4ToV5Migrator) setAccountNestedAttribute(body *hclwrite.Body, accountIdTokens hclwrite.Tokens) {
	// Build tokens for: { id = <accountIdTokens> }
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("  id")},
		{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
	}

	// Append the account ID expression tokens
	tokens = append(tokens, accountIdTokens...)

	// Close the object
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
	)

	// Set the account attribute with the nested object
	body.SetAttributeRaw("account", tokens)
}

// TransformState handles state file transformations.
// This function receives a single resource instance and returns the transformed instance JSON.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Check if it's a valid zone instance
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		// Even for invalid instances, set schema_version for v5
		result = state.SetSchemaVersion(result, 0)
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// 1. zone → name
	if zoneField := attrs.Get("zone"); zoneField.Exists() {
		result, _ = sjson.Set(result, "attributes.name", zoneField.Value())
		result, _ = sjson.Delete(result, "attributes.zone")
	}

	// 2. account_id → account = { id = "..." }
	if accountIdField := attrs.Get("account_id"); accountIdField.Exists() {
		result, _ = sjson.Set(result, "attributes.account", map[string]interface{}{
			"id": accountIdField.Value(),
		})
		result, _ = sjson.Delete(result, "attributes.account_id")
	}

	// 3-4. Remove obsolete fields
	result = state.RemoveFieldsIfExist(result, "attributes", attrs, "jump_start", "plan")

	// Set schema_version to 0 for v5
	result = state.SetSchemaVersion(result, 0)

	return result, nil
}
