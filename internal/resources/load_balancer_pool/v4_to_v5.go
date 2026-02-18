package load_balancer_pool

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of load balancer pool resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_load_balancer_pool", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return v5 resource name (unchanged from v4)
	return "cloudflare_load_balancer_pool"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer_pool"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done with HCL helpers
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_load_balancer_pool doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_load_balancer_pool", "cloudflare_load_balancer_pool"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Pre-process origins blocks to transform nested header blocks
	// v4: header { header = "Host" values = [...] }
	// v5: header = { host = [...] }
	originsBlocks := tfhcl.FindBlocksByType(body, "origins")
	for _, originBlock := range originsBlocks {
		transformHeaderBlock(originBlock.Body())
	}

	// Pre-process dynamic origins blocks to transform nested header blocks
	dynamicBlocks := tfhcl.FindBlocksByType(body, "dynamic")
	for _, dynamicBlock := range dynamicBlocks {
		labels := dynamicBlock.Labels()
		if len(labels) > 0 && labels[0] == "origins" {
			// Find the content block within the dynamic block
			contentBlock := tfhcl.FindBlockByType(dynamicBlock.Body(), "content")
			if contentBlock != nil {
				transformHeaderBlock(contentBlock.Body())
			}
		}
	}

	// Convert dynamic "origins" blocks to for expressions
	// v4: dynamic "origins" { for_each = ... content { ... } }
	// v5: origins = [for value in ... : { ... }]
	tfhcl.ConvertDynamicBlocksToForExpression(body, "origins")

	// Transform origins blocks to origins attribute array
	// v4: origins { name = "origin1" address = "1.2.3.4" }
	// v5: origins = [{ name = "origin1" address = "1.2.3.4" }]
	tfhcl.ConvertBlocksToArrayAttribute(body, "origins", false)

	// Transform load_shedding block to attribute (MaxItems:1)
	// v4: load_shedding { ... }
	// v5: load_shedding = { ... }
	tfhcl.ConvertSingleBlockToAttribute(body, "load_shedding", "load_shedding")

	// Transform origin_steering block to attribute (MaxItems:1)
	// v4: origin_steering { ... }
	// v5: origin_steering = { ... }
	tfhcl.ConvertSingleBlockToAttribute(body, "origin_steering", "origin_steering")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// transformHeaderBlock transforms a header block from v4 to v5 format
// v4: header { header = "Host" values = [...] }
// v5: header = { host = [...] }
func transformHeaderBlock(body *hclwrite.Body) {
	headerBlocks := tfhcl.FindBlocksByType(body, "header")
	if len(headerBlocks) == 0 {
		return
	}

	// For each header block, transform it to the v5 format
	for _, headerBlock := range headerBlocks {
		headerBody := headerBlock.Body()

		// Get the header name (e.g., "Host")
		headerAttr := headerBody.GetAttribute("header")
		if headerAttr == nil {
			continue
		}

		// Get the values
		valuesAttr := headerBody.GetAttribute("values")
		if valuesAttr == nil {
			continue
		}

		// Extract header name from attribute
		headerName := ""
		headerTokens := headerAttr.Expr().BuildTokens(nil)
		for _, token := range headerTokens {
			if token.Type == hclsyntax.TokenQuotedLit {
				headerName = string(token.Bytes)
				break
			}
		}

		if headerName == "" {
			continue
		}

		// Convert header name to lowercase for the key
		headerKey := strings.ToLower(headerName)

		// Get the values tokens
		valuesTokens := valuesAttr.Expr().BuildTokens(nil)

		// Build the new header object: { host = [...] }
		var headerObjTokens hclwrite.Tokens
		headerObjTokens = append(headerObjTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte("{"),
		})
		headerObjTokens = append(headerObjTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(headerKey),
		})
		headerObjTokens = append(headerObjTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte(" = "),
		})
		headerObjTokens = append(headerObjTokens, valuesTokens...)
		headerObjTokens = append(headerObjTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte(" }"),
		})

		// Replace the header block with a header attribute
		body.SetAttributeRaw("header", headerObjTokens)
		body.RemoveBlock(headerBlock)
	}
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is now handled by the provider's StateUpgraders (UpgradeState)
	// The provider's migration/v500 package handles all state transformations:
	// - load_shedding: array[0] → object
	// - origin_steering: array[0] → object
	// - origins.header: array → object with structure change
	// - check_regions: Set → List
	//
	// This function is a no-op for cloudflare_load_balancer_pool migration.
	// The provider automatically applies state upgrades when users run `terraform apply`.
	return stateJSON.String(), nil
}
