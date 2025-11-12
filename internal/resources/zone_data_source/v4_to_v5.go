package zone_data_source

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zone datasource from v4 to v5
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register datasource with the OLD (v4) name
	// Note: Datasources use the same name in v4 and v5
	internal.RegisterMigrator("cloudflare_zone", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) datasource name (unchanged)
	return "cloudflare_zone"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) datasource name
	return resourceType == "cloudflare_zone"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for zone datasource
	// All transformations can be done at HCL level
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No resource type rename needed (stays cloudflare_zone)
	body := block.Body()

	// Get zone_id, name, and account_id attributes
	zoneIDAttr := body.GetAttribute("zone_id")
	nameAttr := body.GetAttribute("name")
	accountIDAttr := body.GetAttribute("account_id")

	// If zone_id is present, just remove account_id (no filter needed)
	if zoneIDAttr != nil {
		if accountIDAttr != nil {
			tfhcl.RemoveAttributes(body, "account_id")
		}
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	// If name or account_id is present, create filter attribute
	// Note: We pass the attributes before removal so createFilterAttribute can extract their values
	if nameAttr != nil || accountIDAttr != nil {
		m.createFilterAttribute(body, nameAttr, accountIDAttr)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// createFilterAttribute creates the filter attribute with nested structure
// filter = {
//   name = "example.com"
//   account = {
//     id = "abc123"
//   }
// }
func (m *V4ToV5Migrator) createFilterAttribute(body *hclwrite.Body, nameAttr, accountIDAttr *hclwrite.Attribute) {
	// Build the filter object tokens manually
	tokens := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}},
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
	}

	// Add name field if present
	if nameAttr != nil {
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("name")},
			&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte{'='}},
		)
		// Copy the value tokens from the original name attribute
		tokens = append(tokens, nameAttr.Expr().BuildTokens(nil)...)
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})

		// Remove the original name attribute
		body.RemoveAttribute("name")
	}

	// Add account block if account_id is present
	if accountIDAttr != nil {
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("account")},
			&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte{'='}},
			&hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}},
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
			&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("id")},
			&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte{'='}},
		)
		// Copy the value tokens from the original account_id attribute
		tokens = append(tokens, accountIDAttr.Expr().BuildTokens(nil)...)
		tokens = append(tokens,
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
			&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}},
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
		)

		// Remove the original account_id attribute
		body.RemoveAttribute("account_id")
	}

	// Close the filter object
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})

	// Set the filter attribute
	body.SetAttributeRaw("filter", tokens)
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result, nil
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Note: Most fields in v5 are computed, so minimal state transformation needed
	// The provider will populate computed fields on next refresh

	return result, nil
}
