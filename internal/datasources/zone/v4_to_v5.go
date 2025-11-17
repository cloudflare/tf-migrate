package zone

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zone datasource from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_zone", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zone"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zone"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - schemas are identical
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	// No postprocessing needed
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Zone datasource name is unchanged between v4 and v5
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	// No rename - return empty strings
	return "", ""
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// v4 to v5 changes:
	// 1. v4 uses direct attributes: name, account_id
	// 2. v5 uses nested filter attribute
	// 3. v4: name = "example.com" -> v5: filter = { name = "example.com" }
	// 4. v4: account_id = "123" -> v5: filter = { account = { id = "123" } }

	body := block.Body()

	// Check for v4-style direct attributes
	nameAttr := body.GetAttribute("name")
	accountIDAttr := body.GetAttribute("account_id")

	// If we have v4-style direct attributes, convert them to v5 filter
	if nameAttr != nil || accountIDAttr != nil {
		// Build filter object attributes
		var filterAttrs []hclwrite.ObjectAttrTokens

		if nameAttr != nil {
			filterAttrs = append(filterAttrs, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForIdentifier("name"),
				Value: nameAttr.Expr().BuildTokens(nil),
			})
			body.RemoveAttribute("name")
		}

		if accountIDAttr != nil {
			// Create nested account object with id field
			accountAttrs := []hclwrite.ObjectAttrTokens{
				{
					Name:  hclwrite.TokensForIdentifier("id"),
					Value: accountIDAttr.Expr().BuildTokens(nil),
				},
			}
			accountTokens := hclwrite.TokensForObject(accountAttrs)

			filterAttrs = append(filterAttrs, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForIdentifier("account"),
				Value: accountTokens,
			})
			body.RemoveAttribute("account_id")
		}

		// Create filter attribute with all collected attributes
		filterTokens := hclwrite.TokensForObject(filterAttrs)
		body.SetAttributeRaw("filter", filterTokens)
	}

	// Handle any existing filter blocks (convert block to attribute syntax)
	// This handles cases where the config already uses v5-style but with block syntax
	filterBlock := tfhcl.FindBlockByType(body, "filter")
	if filterBlock != nil {
		// Convert nested account block to attribute syntax
		tfhcl.ConvertSingleBlockToAttribute(filterBlock.Body(), "account", "account")
		// Then convert the filter block itself to attribute syntax
		tfhcl.ConvertSingleBlockToAttribute(body, "filter", "filter")
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	// Start with the instance JSON
	stateJSON := instance.Raw

	// The only change needed is to ensure schema_version is set to 0 for v5
	// All other fields are identical between v4 and v5
	stateJSON, _ = sjson.Set(stateJSON, "schema_version", 0)

	return stateJSON, nil
}
