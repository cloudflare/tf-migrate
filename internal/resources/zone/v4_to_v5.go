package zone

import (
	"github.com/hashicorp/hcl/v2"

	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_zone"}, "cloudflare_zone"
}

// GetComputedAttributeMappings implements the ComputedAttributeMapper interface.
// In v4, the zone's domain was accessed as cloudflare_zone.example.zone.
// In v5, it's cloudflare_zone.example.name. This enables cross-file reference
// rewriting so that any resource referencing .zone gets updated to .name.
func (m *V4ToV5Migrator) GetComputedAttributeMappings() []transform.ComputedAttributeMapping {
	return []transform.ComputedAttributeMapping{
		{
			OldResourceType: "cloudflare_zone",
			OldAttribute:    "zone",
			NewResourceType: "cloudflare_zone",
			NewAttribute:    "name",
		},
	}
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

	// 3-4. Remove obsolete attributes.
	//
	// plan is computed and deprecated in v5, so removing it is correct, but the
	// zone's plan is no longer expressed in this resource at all. Silently
	// dropping it would leave the user believing Terraform still manages it.
	if body.GetAttribute("plan") != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf(
				"plan removed from cloudflare_zone.%s", tfhcl.GetResourceName(block)),
			Detail: "The plan attribute is computed and deprecated on cloudflare_zone in " +
				"v5, so it has been removed.\n\n" +
				"The zone's plan is no longer managed by this resource. If Terraform " +
				"should continue to manage it, use cloudflare_zone_subscription:\n\n" +
				"  resource \"cloudflare_zone_subscription\" \"example\" {\n" +
				"    zone_id   = cloudflare_zone.example.id\n" +
				"    rate_plan = { id = \"<plan>\" }\n" +
				"  }\n\n" +
				"Without it the plan stays as it is at the API and drifts untracked.",
		})
	}
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
