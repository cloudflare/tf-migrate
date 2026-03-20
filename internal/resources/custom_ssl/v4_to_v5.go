package custom_ssl

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of cloudflare_custom_ssl resources from v4 to v5.
//
// Key structural change: v4 groups certificate configuration inside a
// "custom_ssl_options" TypeList MaxItems:1 block. v5 exposes all those fields
// flat at the top level. The migration unpacks that block.
//
// Summary of changes:
//   - custom_ssl_options { certificate, private_key, bundle_method, type, geo_restrictions } (block)
//     → flat top-level attributes (certificate, private_key, bundle_method, type)
//   - geo_restrictions: plain string "us" → SingleNestedAttribute { label = "us" }
//   - custom_ssl_priority: removed (write-only reprioritization field, not in v5)
//   - priority: TypeInt (computed) → Float64 (handled by provider StateUpgrader)
//
// State transformation is handled by the provider's UpgradeState mechanism.
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_custom_ssl", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_custom_ssl"
}

// CanHandle returns true if this migrator handles the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_custom_ssl"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing is needed for custom_ssl — all transforms are done via HCL helpers.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements ResourceRenamer — resource is NOT renamed.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_custom_ssl"}, "cloudflare_custom_ssl"
}

// TransformConfig handles HCL configuration transformations.
//
// Steps:
//  1. Hoist flat attributes (certificate, private_key, bundle_method, type) from the
//     custom_ssl_options block to the resource root using HoistAttributesFromBlock.
//  2. Transform geo_restrictions: extract the string value from inside the block and
//     emit a SingleNestedAttribute object { label = "..." } at the root.
//  3. Remove the now-empty custom_ssl_options block.
//  4. Remove the custom_ssl_priority block (write-only, not in v5).
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Step 1 & 2: Process the custom_ssl_options block.
	m.transformCustomSSLOptionsBlock(body)

	// Step 3: Remove custom_ssl_priority blocks (write-only reprioritization, not in v5).
	tfhcl.RemoveBlocksByType(body, "custom_ssl_priority")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// transformCustomSSLOptionsBlock unpacks the custom_ssl_options block:
//   - Hoists certificate, private_key, bundle_method, type to root.
//   - Transforms geo_restrictions from a plain string to { label = "..." }.
//   - Removes the custom_ssl_options block.
func (m *V4ToV5Migrator) transformCustomSSLOptionsBlock(body *hclwrite.Body) {
	optionsBlock := tfhcl.FindBlockByType(body, "custom_ssl_options")
	if optionsBlock == nil {
		return
	}

	blockBody := optionsBlock.Body()

	// Hoist flat string attributes.
	tfhcl.HoistAttributesFromBlock(body, "custom_ssl_options", "certificate", "private_key", "bundle_method", "type")

	// Transform geo_restrictions: string inside block → { label = "..." } at root.
	if geoAttr := blockBody.GetAttribute("geo_restrictions"); geoAttr != nil {
		// Get the value tokens (the quoted string expression).
		valueTokens := geoAttr.Expr().BuildTokens(nil)

		// Build a SingleNestedAttribute object: { label = <value> }
		objTokens := buildGeoRestrictionsObject(valueTokens)
		body.SetAttributeRaw("geo_restrictions", objTokens)
	}

	// Remove the custom_ssl_options block now that its contents have been hoisted.
	body.RemoveBlock(optionsBlock)
}

// buildGeoRestrictionsObject builds the HCL tokens for:
//
//	geo_restrictions = {
//	  label = <labelTokens>
//	}
func buildGeoRestrictionsObject(labelValueTokens hclwrite.Tokens) hclwrite.Tokens {
	nameTokens := hclwrite.TokensForIdentifier("label")

	attrs := []hclwrite.ObjectAttrTokens{
		{
			Name:  nameTokens,
			Value: labelValueTokens,
		},
	}
	return hclwrite.TokensForObject(attrs)
}
