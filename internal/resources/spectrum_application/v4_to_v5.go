package spectrum_application

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name
	internal.RegisterMigrator("cloudflare_spectrum_application", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name doesn't change
	return "cloudflare_spectrum_application"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_spectrum_application"
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_spectrum_application", "cloudflare_spectrum_application"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For spectrum_application, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Remove id attribute if present (defensive cleanup - id is computed-only in both versions)
	tfhcl.RemoveAttributes(body, "id")

	// 2. Convert MaxItems:1 blocks to attributes
	// dns: required block -> attribute
	tfhcl.ConvertBlocksToAttribute(body, "dns", "dns", func(block *hclwrite.Block) {})

	// origin_dns: optional block -> attribute
	tfhcl.ConvertBlocksToAttribute(body, "origin_dns", "origin_dns", func(block *hclwrite.Block) {})

	// edge_ips: optional block -> attribute
	tfhcl.ConvertBlocksToAttribute(body, "edge_ips", "edge_ips", func(block *hclwrite.Block) {})

	// 3. Convert origin_port_range block to origin_port string
	m.convertOriginPortRange(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// convertOriginPortRange converts origin_port_range block to origin_port string attribute
func (m *V4ToV5Migrator) convertOriginPortRange(body *hclwrite.Body) {
	// Find origin_port_range blocks
	for _, block := range body.Blocks() {
		if block.Type() == "origin_port_range" {
			blockBody := block.Body()

			// Extract start and end attributes
			startAttr := blockBody.GetAttribute("start")
			endAttr := blockBody.GetAttribute("end")

			if startAttr != nil && endAttr != nil {
				// Get the raw token values
				startTokens := startAttr.Expr().BuildTokens(nil)
				endTokens := endAttr.Expr().BuildTokens(nil)

				if len(startTokens) >= 1 && len(endTokens) >= 1 {
					startValue := string(startTokens[0].Bytes)
					endValue := string(endTokens[0].Bytes)

					// Create the range string: "start-end"
					rangeValue := startValue + "-" + endValue

					// Set origin_port attribute with quoted string
					body.SetAttributeValue("origin_port", cty.StringVal(rangeValue))
				}
			}

			// Remove the origin_port_range block
			body.RemoveBlock(block)
			break // Only handle the first one (MaxItems: 1)
		}
	}
}

// TransformState handles state file transformations.
// State transformation is handled by the provider's StateUpgraders (UpgradeState)
// This function is a no-op for spectrum_application migration
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
