package load_balancer_pool

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

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

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Transform load_shedding from array to object (or null if empty)
	// v4: "load_shedding": [{ ... }] or []
	// v5: "load_shedding": { ... } or null
	loadShedding := stateJSON.Get("attributes.load_shedding")
	if loadShedding.Exists() && loadShedding.IsArray() {
		if len(loadShedding.Array()) > 0 {
			firstElement := loadShedding.Array()[0]
			result, _ = sjson.Set(result, "attributes.load_shedding", firstElement.Value())
		} else {
			// Empty array -> null
			result, _ = sjson.Set(result, "attributes.load_shedding", nil)
		}
	}

	// Transform origin_steering from array to object (or null if empty)
	// v4: "origin_steering": [{ ... }] or []
	// v5: "origin_steering": { ... } or null
	originSteering := stateJSON.Get("attributes.origin_steering")
	if originSteering.Exists() && originSteering.IsArray() {
		if len(originSteering.Array()) > 0 {
			firstElement := originSteering.Array()[0]
			result, _ = sjson.Set(result, "attributes.origin_steering", firstElement.Value())
		} else {
			// Empty array -> null
			result, _ = sjson.Set(result, "attributes.origin_steering", nil)
		}
	}

	// Transform header field inside each origin from array to object/null
	// v4: origins[*].header = [] or [{ ... }]
	// v5: origins[*].header = {} or null (provider expects object, not array)
	// Re-parse to get updated state after previous transformations
	updatedState := gjson.Parse(result)
	origins := updatedState.Get("attributes.origins")
	if origins.Exists() && origins.IsArray() {
		originsArray := origins.Array()
		for i, origin := range originsArray {
			header := origin.Get("header")
			if header.Exists() && header.IsArray() {
				if len(header.Array()) == 0 {
					// Empty array -> empty object (v5 provider expects object type)
					result, _ = sjson.Set(result, fmt.Sprintf("attributes.origins.%d.header", i), map[string]interface{}{})
				} else {
					// Non-empty array -> convert first element to object
					// This handles the case where v4 had header as array of objects
					firstElement := header.Array()[0]
					result, _ = sjson.Set(result, fmt.Sprintf("attributes.origins.%d.header", i), firstElement.Value())
				}
			}
		}
	}

	return result, nil
}
