package load_balancer_pool

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_load_balancer_pool", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_load_balancer_pool"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer_pool"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Transform header blocks in origins to new format
	// header { header = "Host" values = [...] } -> header = { host = [...] }

	// Use regex to find and replace header blocks
	headerPattern := regexp.MustCompile(`(?s)header\s*\{\s*header\s*=\s*"Host"\s*values\s*=\s*(\[[^\]]*\])\s*\}`)
	content = headerPattern.ReplaceAllString(content, `header = {
      host = $1
    }`)

	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Handle dynamic origins blocks by converting them to for expressions
	m.transformDynamicOriginsBlocks(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) transformDynamicOriginsBlocks(body *hclwrite.Body) {
	var dynamicBlocks []*hclwrite.Block

	// Find all dynamic "origins" blocks
	for _, childBlock := range body.Blocks() {
		if childBlock.Type() == "dynamic" && len(childBlock.Labels()) > 0 && childBlock.Labels()[0] == "origins" {
			dynamicBlocks = append(dynamicBlocks, childBlock)
		}
	}

	if len(dynamicBlocks) == 0 {
		return
	}

	// Process each dynamic block
	for _, dynBlock := range dynamicBlocks {
		// Extract the for_each expression
		forEachAttr := dynBlock.Body().GetAttribute("for_each")
		if forEachAttr == nil {
			continue
		}

		// Get the iterator name (defaults to "origins")
		iteratorName := "origins"
		if iteratorAttr := dynBlock.Body().GetAttribute("iterator"); iteratorAttr != nil {
			tokens := iteratorAttr.Expr().BuildTokens(nil)
			if len(tokens) > 0 {
				iteratorName = string(tokens[0].Bytes)
			}
		}

		// Extract content block
		var contentBlock *hclwrite.Block
		for _, cb := range dynBlock.Body().Blocks() {
			if cb.Type() == "content" {
				contentBlock = cb
				break
			}
		}

		if contentBlock == nil {
			continue
		}

		// Build the for expression from the content block
		forExprStr := m.buildForExpressionFromContent(forEachAttr, contentBlock, iteratorName)

		if forExprStr != "" {
			// Parse and set the new origins attribute
			originsHCL := fmt.Sprintf("origins = %s", forExprStr)
			file, diags := hclwrite.ParseConfig([]byte(originsHCL), "", hcl.InitialPos)
			if !diags.HasErrors() {
				attr := file.Body().GetAttribute("origins")
				if attr != nil {
					body.SetAttributeRaw("origins", attr.Expr().BuildTokens(nil))
				}
			}
		}

		// Remove the dynamic block
		body.RemoveBlock(dynBlock)
	}
}

func (m *V4ToV5Migrator) buildForExpressionFromContent(forEachAttr *hclwrite.Attribute, contentBlock *hclwrite.Block, iteratorName string) string {
	// Get for_each expression as string
	forEachTokens := forEachAttr.Expr().BuildTokens(nil)
	var forEachStr string
	for _, token := range forEachTokens {
		forEachStr += string(token.Bytes)
	}

	// Build object with attributes from content block
	var attrStrs []string

	// Get all attributes except header blocks
	attrs := contentBlock.Body().Attributes()

	// Sort attribute names for consistent output
	var attrNames []string
	for attrName := range attrs {
		attrNames = append(attrNames, attrName)
	}
	// Use a fixed sort order
	sortAttributes(attrNames)

	for _, attrName := range attrNames {
		attr := attrs[attrName]
		tokens := attr.Expr().BuildTokens(nil)
		var valueStr string
		for _, token := range tokens {
			valueStr += string(token.Bytes)
		}

		// Replace iterator references
		// Handle both "origins.value" and "origins.value.field"
		if valueStr == iteratorName+".value" {
			valueStr = "value"
		} else {
			valueStr = strings.ReplaceAll(valueStr, iteratorName+".value.", "value.")
		}
		valueStr = strings.ReplaceAll(valueStr, iteratorName+".key", "key")

		attrStrs = append(attrStrs, fmt.Sprintf("    %s = %s", attrName, valueStr))
	}

	// Handle header blocks specially
	for _, childBlock := range contentBlock.Body().Blocks() {
		if childBlock.Type() == "header" {
			// Transform header block
			headerAttr := m.transformHeaderBlockToAttribute(childBlock, iteratorName)
			if headerAttr != "" {
				attrStrs = append(attrStrs, headerAttr)
			}
		}
	}

	// Build the for expression
	return fmt.Sprintf("[for key, value in %s : {\n%s\n  }]", forEachStr, strings.Join(attrStrs, "\n"))
}

func (m *V4ToV5Migrator) transformHeaderBlockToAttribute(headerBlock *hclwrite.Block, iteratorName string) string {
	// Get the header and values attributes
	headerAttr := headerBlock.Body().GetAttribute("header")
	valuesAttr := headerBlock.Body().GetAttribute("values")

	if headerAttr == nil || valuesAttr == nil {
		return ""
	}

	// Get the header name
	headerTokens := headerAttr.Expr().BuildTokens(nil)
	var headerName string
	for _, token := range headerTokens {
		headerName += string(token.Bytes)
	}
	headerName = strings.Trim(headerName, `"`)

	// Get the values
	valuesTokens := valuesAttr.Expr().BuildTokens(nil)
	var valuesStr string
	for _, token := range valuesTokens {
		valuesStr += string(token.Bytes)
	}

	// Replace iterator references
	valuesStr = strings.ReplaceAll(valuesStr, iteratorName+".value.", "value.")

	// Map header name to attribute name (only "Host" -> "host" is supported)
	attrName := strings.ToLower(headerName)

	return fmt.Sprintf("    header = {\n      %s = %s\n    }", attrName, valuesStr)
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// No state transformation needed
	return stateJSON.String(), nil
}

// sortAttributes sorts attribute names with a preferred order for origins
func sortAttributes(names []string) {
	// Define preferred order for common attributes
	order := map[string]int{
		"name":    0,
		"address": 1,
		"enabled": 2,
		"weight":  3,
	}

	// Sort with custom ordering
	sort.Slice(names, func(i, j int) bool {
		// Check if either has a defined order
		orderI, hasI := order[names[i]]
		orderJ, hasJ := order[names[j]]

		if hasI && hasJ {
			return orderI < orderJ
		}
		if hasI {
			return true
		}
		if hasJ {
			return false
		}
		// Neither has defined order, use alphabetical
		return names[i] < names[j]
	})
}
