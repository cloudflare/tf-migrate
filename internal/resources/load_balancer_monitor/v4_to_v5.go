package load_balancer_monitor

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of load balancer monitor resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_load_balancer_monitor", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return v5 resource name (unchanged from v4)
	return "cloudflare_load_balancer_monitor"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer_monitor"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done with HCL helpers
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_load_balancer_monitor doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_load_balancer_monitor"}, "cloudflare_load_balancer_monitor"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Transform header blocks to map attribute
	// v4: header { header = "Host" values = ["example.com"] }
	// v5: header = { "Host" = ["example.com"] }
	headerTokens, err := m.buildHeaderMapTokens(body)
	if err != nil {
		return nil, err
	}
	if headerTokens != nil {
		body.SetAttributeRaw("header", headerTokens)
		// Remove the old header blocks
		tfhcl.RemoveBlocksByType(body, "header")
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// buildHeaderMapTokens converts v4 header blocks to v5 header map tokens
// v4: header { header = "Host" values = ["example.com"] }
// v5: header = { "Host" = ["example.com"] }
func (m *V4ToV5Migrator) buildHeaderMapTokens(body *hclwrite.Body) (hclwrite.Tokens, error) {
	// Find all header blocks
	headerBlocks := tfhcl.FindBlocksByType(body, "header")
	if len(headerBlocks) == 0 {
		return nil, nil
	}

	// Build a list of object attributes for the header map
	var headerAttrs []hclwrite.ObjectAttrTokens

	for _, block := range headerBlocks {
		blockBody := block.Body()

		// Get the header name
		headerAttr := blockBody.GetAttribute("header")
		if headerAttr == nil {
			continue
		}

		// Get the values
		valuesAttr := blockBody.GetAttribute("values")
		if valuesAttr == nil {
			continue
		}

		// Use the header value as the map key and values as the map value
		nameTokens := headerAttr.Expr().BuildTokens(nil)
		valueTokens := valuesAttr.Expr().BuildTokens(nil)

		headerAttrs = append(headerAttrs, hclwrite.ObjectAttrTokens{
			Name:  nameTokens,
			Value: valueTokens,
		})
	}

	if len(headerAttrs) == 0 {
		return nil, nil
	}

	// Create the object tokens for the header map
	return hclwrite.TokensForObject(headerAttrs), nil
}

func init() {
	// Register the migrator when the package is imported
	NewV4ToV5Migrator()
}
