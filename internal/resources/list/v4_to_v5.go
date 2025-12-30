package list

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/resources/list_item"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_list", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_list"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_list"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Check if this is JSON state content (starts with '{')
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") {
		// Process cross-resource state migrations (merge list_item into list, remove list_items)
		return list_item.ProcessCrossResourceStateMigration(content)
	}
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_list doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_list", "cloudflare_list"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Process cross-resource migrations (merge list_item resources into this list)
	// This is idempotent - if called multiple times, subsequent calls are no-ops
	if ctx.CFGFile != nil {
		list_item.ProcessCrossResourceConfigMigration(ctx.CFGFile)
	}

	kind := tfhcl.ExtractStringFromAttribute(body.GetAttribute("kind"))
	if kind == "" {
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	itemBlocks := tfhcl.FindBlocksByType(body, "item")
	dynamicBlocks := findDynamicItemBlocks(body)

	if len(dynamicBlocks) > 0 {
		transformListWithDynamicBlocks(body, itemBlocks, dynamicBlocks, kind)
	} else if len(itemBlocks) > 0 {
		transformStaticItemBlocks(body, itemBlocks, kind)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()
	attrs := stateJSON.Get("attributes")

	if !attrs.Exists() {
		result = state.SetSchemaVersion(result, 0)
		return result, nil
	}

	kind := attrs.Get("kind").String()

	if itemsField := attrs.Get("item"); itemsField.Exists() && itemsField.IsArray() {
		var transformedItems []map[string]interface{}

		for _, item := range itemsField.Array() {
			transformedItem := transformStateItem(item, kind)
			if transformedItem != nil {
				transformedItems = append(transformedItems, transformedItem)
			}
		}

		if len(transformedItems) > 0 {
			result, _ = sjson.Set(result, "attributes.items", transformedItems)
		}

		result, _ = sjson.Delete(result, "attributes.item")
	}

	if numItems := attrs.Get("num_items"); numItems.Exists() {
		floatVal := state.ConvertToFloat64(numItems)
		result, _ = sjson.Set(result, "attributes.num_items", floatVal)
	}

	result = state.SetSchemaVersion(result, 0)

	return result, nil
}

func findDynamicItemBlocks(body *hclwrite.Body) []*hclwrite.Block {
	var dynamicBlocks []*hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == "dynamic" && len(block.Labels()) > 0 && block.Labels()[0] == "item" {
			dynamicBlocks = append(dynamicBlocks, block)
		}
	}
	return dynamicBlocks
}

func transformStaticItemBlocks(body *hclwrite.Body, itemBlocks []*hclwrite.Block, kind string) {
	if len(itemBlocks) == 0 {
		return
	}

	// Check if any item block contains non-literal expressions (like each.key, each.value)
	// If so, use the string-based expression building approach to preserve them
	hasExpressions := false
	for _, itemBlock := range itemBlocks {
		if itemBlockHasExpressions(itemBlock) {
			hasExpressions = true
			break
		}
	}

	if hasExpressions {
		// Use string-based approach to preserve expressions
		itemsExpr := buildStaticItemsExpressionStringPreserving(itemBlocks, kind)
		for _, itemBlock := range itemBlocks {
			body.RemoveBlock(itemBlock)
		}
		if itemsExpr != "" {
			setItemsAttributeFromString(body, itemsExpr)
		}
		return
	}

	// For simple literal values, use cty.Value approach
	var itemObjects []cty.Value

	for _, itemBlock := range itemBlocks {
		itemObj := buildItemObjectFromBlock(itemBlock, kind)
		if itemObj != cty.NilVal {
			itemObjects = append(itemObjects, itemObj)
		}
	}

	for _, itemBlock := range itemBlocks {
		body.RemoveBlock(itemBlock)
	}

	if len(itemObjects) > 0 {
		body.SetAttributeValue("items", cty.TupleVal(itemObjects))
	}
}

func buildItemObjectFromBlock(itemBlock *hclwrite.Block, kind string) cty.Value {
	itemBody := itemBlock.Body()
	itemMap := make(map[string]cty.Value)

	if commentAttr := itemBody.GetAttribute("comment"); commentAttr != nil {
		commentValue := tfhcl.ExtractStringFromAttribute(commentAttr)
		if commentValue != "" {
			itemMap["comment"] = cty.StringVal(commentValue)
		}
	}

	for _, valueBlock := range itemBody.Blocks() {
		if valueBlock.Type() == "value" {
			valueBody := valueBlock.Body()

			switch kind {
			case "ip":
				if ipAttr := valueBody.GetAttribute("ip"); ipAttr != nil {
					ipValue := tfhcl.ExtractStringFromAttribute(ipAttr)
					if ipValue != "" {
						itemMap["ip"] = cty.StringVal(normalizeIPAddress(ipValue))
					}
				}

			case "asn":
				if asnAttr := valueBody.GetAttribute("asn"); asnAttr != nil {
					asnTokens := asnAttr.Expr().BuildTokens(nil)
					for _, token := range asnTokens {
						if token.Type == hclsyntax.TokenNumberLit {
							itemMap["asn"] = cty.NumberIntVal(parseNumber(string(token.Bytes)))
							break
						}
					}
				}

			case "hostname":
				for _, hostnameBlock := range valueBody.Blocks() {
					if hostnameBlock.Type() == "hostname" {
						hostnameBody := hostnameBlock.Body()
						hostnameMap := make(map[string]cty.Value)

						if urlAttr := hostnameBody.GetAttribute("url_hostname"); urlAttr != nil {
							urlValue := tfhcl.ExtractStringFromAttribute(urlAttr)
							if urlValue != "" {
								hostnameMap["url_hostname"] = cty.StringVal(urlValue)
							}
						}

						if len(hostnameMap) > 0 {
							itemMap["hostname"] = cty.ObjectVal(hostnameMap)
						}
					}
				}

			case "redirect":
				for _, redirectBlock := range valueBody.Blocks() {
					if redirectBlock.Type() == "redirect" {
						redirectBody := redirectBlock.Body()
						redirectMap := make(map[string]cty.Value)

						if sourceAttr := redirectBody.GetAttribute("source_url"); sourceAttr != nil {
							sourceValue := tfhcl.ExtractStringFromAttribute(sourceAttr)
							if sourceValue != "" {
								sourceValue = ensureSourceURLHasPath(sourceValue)
								redirectMap["source_url"] = cty.StringVal(sourceValue)
							}
						}

						if targetAttr := redirectBody.GetAttribute("target_url"); targetAttr != nil {
							targetValue := tfhcl.ExtractStringFromAttribute(targetAttr)
							if targetValue != "" {
								redirectMap["target_url"] = cty.StringVal(targetValue)
							}
						}

						boolFields := []string{
							"include_subdomains",
							"subpath_matching",
							"preserve_query_string",
							"preserve_path_suffix",
						}

						for _, field := range boolFields {
							if attr := redirectBody.GetAttribute(field); attr != nil {
								value := tfhcl.ExtractStringFromAttribute(attr)
								if value == "enabled" {
									redirectMap[field] = cty.BoolVal(true)
								} else if value == "disabled" {
									redirectMap[field] = cty.BoolVal(false)
								}
							}
						}

						if statusAttr := redirectBody.GetAttribute("status_code"); statusAttr != nil {
							statusTokens := statusAttr.Expr().BuildTokens(nil)
							for _, token := range statusTokens {
								if token.Type == hclsyntax.TokenNumberLit {
									redirectMap["status_code"] = cty.NumberIntVal(parseNumber(string(token.Bytes)))
									break
								}
							}
						}

						if len(redirectMap) > 0 {
							itemMap["redirect"] = cty.ObjectVal(redirectMap)
						}
					}
				}
			}
		}
	}

	if len(itemMap) == 0 {
		return cty.NilVal
	}

	return cty.ObjectVal(itemMap)
}

func transformListWithDynamicBlocks(body *hclwrite.Body, staticBlocks, dynamicBlocks []*hclwrite.Block, kind string) {
	var itemsExprStr string

	// Build for expressions from dynamic blocks
	var forExprs []string
	for _, dynBlock := range dynamicBlocks {
		forExpr := buildForExpressionFromDynamic(dynBlock, kind)
		if forExpr != "" {
			forExprs = append(forExprs, forExpr)
		}
	}

	// Build static items expression
	var staticExpr string
	if len(staticBlocks) > 0 {
		staticExpr = buildStaticItemsExpressionString(staticBlocks, kind)
	}

	// Combine expressions
	if len(forExprs) == 1 && staticExpr == "" {
		// Single dynamic block only
		itemsExprStr = forExprs[0]
	} else if len(forExprs) == 0 && staticExpr != "" {
		// Static items only - use the regular function
		transformStaticItemBlocks(body, staticBlocks, kind)
		for _, block := range dynamicBlocks {
			body.RemoveBlock(block)
		}
		return
	} else if len(forExprs) > 0 || staticExpr != "" {
		// Mixed or multiple - create concat expression
		var allExprs []string
		if staticExpr != "" {
			allExprs = append(allExprs, staticExpr)
		}
		allExprs = append(allExprs, forExprs...)
		itemsExprStr = fmt.Sprintf("concat(%s)", strings.Join(allExprs, ", "))
	}

	// Remove all item-related blocks
	for _, block := range staticBlocks {
		body.RemoveBlock(block)
	}
	for _, block := range dynamicBlocks {
		body.RemoveBlock(block)
	}

	// Set the items attribute
	if itemsExprStr != "" {
		setItemsAttributeFromString(body, itemsExprStr)
	}
}

// buildForExpressionFromDynamic creates a for expression string from a dynamic block
func buildForExpressionFromDynamic(dynBlock *hclwrite.Block, kind string) string {
	dynBody := dynBlock.Body()

	// Get for_each expression
	forEachAttr := dynBody.GetAttribute("for_each")
	if forEachAttr == nil {
		return ""
	}

	forEachStr := strings.TrimSpace(string(forEachAttr.Expr().BuildTokens(nil).Bytes()))

	// Get iterator name (default is the block label)
	iteratorName := dynBlock.Labels()[0]
	if iterAttr := dynBody.GetAttribute("iterator"); iterAttr != nil {
		iteratorName = tfhcl.ExtractStringFromAttribute(iterAttr)
	}

	// Find content block
	var contentBlock *hclwrite.Block
	for _, b := range dynBody.Blocks() {
		if b.Type() == "content" {
			contentBlock = b
			break
		}
	}

	if contentBlock == nil {
		return ""
	}

	// Build object expression from content
	objExpr := buildObjectFromContentBlock(contentBlock, kind, iteratorName)
	if objExpr == "" {
		return ""
	}

	// Create the for expression
	return fmt.Sprintf("[for %s in %s : %s]", iteratorName, forEachStr, objExpr)
}

// buildObjectFromContentBlock creates an object expression string from a content block
func buildObjectFromContentBlock(contentBlock *hclwrite.Block, kind string, iteratorName string) string {
	contentBody := contentBlock.Body()
	var fields []string

	// Process value block first (to get ip, asn, hostname, redirect)
	for _, vBlock := range contentBody.Blocks() {
		if vBlock.Type() == "value" {
			valueFields := extractValueBlockFields(vBlock, kind, iteratorName)
			fields = append(fields, valueFields...)
		}
	}

	// Then add comment attribute
	if commentAttr := contentBody.GetAttribute("comment"); commentAttr != nil {
		commentExpr := strings.TrimSpace(string(commentAttr.Expr().BuildTokens(nil).Bytes()))
		commentExpr = tfhcl.StripIteratorValueSuffix(commentExpr, iteratorName)
		fields = append(fields, fmt.Sprintf("comment = %s", commentExpr))
	}

	if len(fields) == 0 {
		return ""
	}

	return fmt.Sprintf("{\n    %s\n  }", strings.Join(fields, "\n    "))
}

// extractValueBlockFields extracts field expressions from a value block
func extractValueBlockFields(vBlock *hclwrite.Block, kind string, iteratorName string) []string {
	vBody := vBlock.Body()
	var fields []string

	switch kind {
	case "ip":
		if ipAttr := vBody.GetAttribute("ip"); ipAttr != nil {
			ipExpr := strings.TrimSpace(string(ipAttr.Expr().BuildTokens(nil).Bytes()))
			ipExpr = tfhcl.StripIteratorValueSuffix(ipExpr, iteratorName)
			ipExpr = normalizeIPAddressInExpr(ipExpr)
			fields = append(fields, fmt.Sprintf("ip = %s", ipExpr))
		}

	case "asn":
		if asnAttr := vBody.GetAttribute("asn"); asnAttr != nil {
			asnExpr := strings.TrimSpace(string(asnAttr.Expr().BuildTokens(nil).Bytes()))
			asnExpr = tfhcl.StripIteratorValueSuffix(asnExpr, iteratorName)
			fields = append(fields, fmt.Sprintf("asn = %s", asnExpr))
		}

	case "hostname":
		for _, hBlock := range vBody.Blocks() {
			if hBlock.Type() == "hostname" {
				hostnameObj := buildHostnameObjectString(hBlock, iteratorName)
				if hostnameObj != "" {
					fields = append(fields, fmt.Sprintf("hostname = %s", hostnameObj))
				}
			}
		}

	case "redirect":
		for _, rBlock := range vBody.Blocks() {
			if rBlock.Type() == "redirect" {
				redirectObj := buildRedirectObjectString(rBlock, iteratorName)
				if redirectObj != "" {
					fields = append(fields, fmt.Sprintf("redirect = %s", redirectObj))
				}
			}
		}
	}

	return fields
}

// buildHostnameObjectString creates a hostname object expression string
func buildHostnameObjectString(hBlock *hclwrite.Block, iteratorName string) string {
	hBody := hBlock.Body()
	var fields []string

	if urlAttr := hBody.GetAttribute("url_hostname"); urlAttr != nil {
		urlExpr := strings.TrimSpace(string(urlAttr.Expr().BuildTokens(nil).Bytes()))
		urlExpr = tfhcl.StripIteratorValueSuffix(urlExpr, iteratorName)
		fields = append(fields, fmt.Sprintf("url_hostname = %s", urlExpr))
	}

	if len(fields) == 0 {
		return ""
	}

	return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))
}

// buildRedirectObjectString creates a redirect object expression string with boolean conversions
func buildRedirectObjectString(rBlock *hclwrite.Block, iteratorName string) string {
	rBody := rBlock.Body()
	var fields []string

	// Required fields
	if sourceAttr := rBody.GetAttribute("source_url"); sourceAttr != nil {
		sourceExpr := strings.TrimSpace(string(sourceAttr.Expr().BuildTokens(nil).Bytes()))
		sourceExpr = tfhcl.StripIteratorValueSuffix(sourceExpr, iteratorName)
		// For literal strings, ensure path is present
		sourceExpr = ensureSourceURLHasPathInExpr(sourceExpr)
		fields = append(fields, fmt.Sprintf("source_url = %s", sourceExpr))
	}

	if targetAttr := rBody.GetAttribute("target_url"); targetAttr != nil {
		targetExpr := strings.TrimSpace(string(targetAttr.Expr().BuildTokens(nil).Bytes()))
		targetExpr = tfhcl.StripIteratorValueSuffix(targetExpr, iteratorName)
		fields = append(fields, fmt.Sprintf("target_url = %s", targetExpr))
	}

	// Boolean fields that need conversion from "enabled"/"disabled"
	boolFields := []string{
		"include_subdomains",
		"subpath_matching",
		"preserve_query_string",
		"preserve_path_suffix",
	}

	for _, field := range boolFields {
		if attr := rBody.GetAttribute(field); attr != nil {
			value := tfhcl.ExtractStringFromAttribute(attr)
			if value == "enabled" {
				fields = append(fields, fmt.Sprintf("%s = true", field))
			} else if value == "disabled" {
				fields = append(fields, fmt.Sprintf("%s = false", field))
			} else {
				// Keep original expression (might be dynamic)
				exprStr := strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
				exprStr = tfhcl.StripIteratorValueSuffix(exprStr, iteratorName)
				// Convert "enabled"/"disabled" strings in expressions
				exprStr = tfhcl.ConvertEnabledDisabledInExpr(exprStr)
				fields = append(fields, fmt.Sprintf("%s = %s", field, exprStr))
			}
		}
	}

	// Optional status_code
	if statusAttr := rBody.GetAttribute("status_code"); statusAttr != nil {
		statusExpr := strings.TrimSpace(string(statusAttr.Expr().BuildTokens(nil).Bytes()))
		statusExpr = tfhcl.StripIteratorValueSuffix(statusExpr, iteratorName)
		fields = append(fields, fmt.Sprintf("status_code = %s", statusExpr))
	}

	if len(fields) == 0 {
		return ""
	}

	return fmt.Sprintf("{\n      %s\n    }", strings.Join(fields, "\n      "))
}

// buildStaticItemsExpressionString creates a tuple expression string from static item blocks
func buildStaticItemsExpressionString(blocks []*hclwrite.Block, kind string) string {
	var itemStrs []string

	for _, block := range blocks {
		objStr := buildObjectStringFromItemBlock(block, kind)
		if objStr != "" {
			itemStrs = append(itemStrs, objStr)
		}
	}

	if len(itemStrs) == 0 {
		return ""
	}

	return fmt.Sprintf("[%s]", strings.Join(itemStrs, ", "))
}

// buildObjectStringFromItemBlock creates an object expression string from a static item block
func buildObjectStringFromItemBlock(block *hclwrite.Block, kind string) string {
	body := block.Body()
	var fields []string

	// Process value block first
	for _, vBlock := range body.Blocks() {
		if vBlock.Type() == "value" {
			valueFields := extractValueBlockFields(vBlock, kind, "")
			fields = append(fields, valueFields...)
		}
	}

	// Then handle comment
	if commentAttr := body.GetAttribute("comment"); commentAttr != nil {
		commentValue := tfhcl.ExtractStringFromAttribute(commentAttr)
		if commentValue != "" {
			fields = append(fields, fmt.Sprintf(`comment = "%s"`, commentValue))
		}
	}

	if len(fields) == 0 {
		return ""
	}

	return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))
}

// setItemsAttributeFromString sets the items attribute from a string expression.
// Uses the tfhcl.SetAttributeFromExpressionString utility with error handling.
func setItemsAttributeFromString(body *hclwrite.Body, exprStr string) {
	if err := tfhcl.SetAttributeFromExpressionString(body, "items", exprStr); err != nil {
		// Fallback: add as comment with warning
		comment := hclwrite.Tokens{
			&hclwrite.Token{
				Type:  hclsyntax.TokenComment,
				Bytes: []byte(fmt.Sprintf("# MIGRATION WARNING: Could not parse items expression. Manual conversion needed.\n# Attempted: items = %s\n", exprStr)),
			},
		}
		body.AppendUnstructuredTokens(comment)
	}
}

func transformStateItem(item gjson.Result, kind string) map[string]interface{} {
	result := make(map[string]interface{})

	if comment := item.Get("comment"); comment.Exists() && comment.String() != "" {
		result["comment"] = comment.String()
	}

	valueObj := item.Get("value")
	if !valueObj.Exists() {
		return nil
	}

	var value gjson.Result
	if valueObj.IsArray() && len(valueObj.Array()) > 0 {
		value = valueObj.Array()[0]
	} else if valueObj.IsObject() {
		value = valueObj
	} else {
		return nil
	}

	switch kind {
	case "ip":
		if ip := value.Get("ip"); ip.Exists() && ip.String() != "" {
			result["ip"] = normalizeIPAddress(ip.String())
		}

	case "asn":
		if asn := value.Get("asn"); asn.Exists() {
			result["asn"] = state.ConvertToInt64(asn)
		}

	case "hostname":
		if hostname := value.Get("hostname"); hostname.Exists() {
			var hostnameObj map[string]interface{}

			if hostname.IsArray() && len(hostname.Array()) > 0 {
				hostnameData := hostname.Array()[0]
				hostnameObj = make(map[string]interface{})
				if urlHostname := hostnameData.Get("url_hostname"); urlHostname.Exists() {
					hostnameObj["url_hostname"] = urlHostname.String()
				}
			} else if hostname.IsObject() {
				hostnameObj = make(map[string]interface{})
				if urlHostname := hostname.Get("url_hostname"); urlHostname.Exists() {
					hostnameObj["url_hostname"] = urlHostname.String()
				}
			}

			if hostnameObj != nil && len(hostnameObj) > 0 {
				result["hostname"] = hostnameObj
			}
		}

	case "redirect":
		if redirect := value.Get("redirect"); redirect.Exists() {
			var redirectObj map[string]interface{}

			if redirect.IsArray() && len(redirect.Array()) > 0 {
				redirectData := redirect.Array()[0]
				redirectObj = transformRedirectData(redirectData)
			} else if redirect.IsObject() {
				redirectObj = transformRedirectData(redirect)
			}

			if redirectObj != nil && len(redirectObj) > 0 {
				result["redirect"] = redirectObj
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func transformRedirectData(data gjson.Result) map[string]interface{} {
	redirectObj := make(map[string]interface{})

	if sourceURL := data.Get("source_url"); sourceURL.Exists() {
		redirectObj["source_url"] = ensureSourceURLHasPath(sourceURL.String())
	}
	if targetURL := data.Get("target_url"); targetURL.Exists() {
		redirectObj["target_url"] = targetURL.String()
	}
	if statusCode := data.Get("status_code"); statusCode.Exists() {
		redirectObj["status_code"] = state.ConvertToInt64(statusCode)
	}

	// Boolean fields that need "enabled"/"disabled" to true/false conversion
	boolFields := []string{
		"include_subdomains",
		"subpath_matching",
		"preserve_query_string",
		"preserve_path_suffix",
	}

	for _, field := range boolFields {
		if fieldVal := data.Get(field); fieldVal.Exists() {
			// Use the utility function for conversion
			if converted := state.ConvertEnabledDisabledToBool(fieldVal); converted != nil {
				redirectObj[field] = converted
			}
		}
	}

	return redirectObj
}

func parseNumber(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}

// itemBlockHasExpressions checks if an item block contains non-literal expressions
// like each.key, each.value, var.something, etc.
func itemBlockHasExpressions(itemBlock *hclwrite.Block) bool {
	itemBody := itemBlock.Body()

	// Check comment attribute
	if commentAttr := itemBody.GetAttribute("comment"); commentAttr != nil {
		if tfhcl.IsExpressionAttribute(commentAttr) {
			return true
		}
	}

	// Check value block attributes
	for _, valueBlock := range itemBody.Blocks() {
		if valueBlock.Type() == "value" {
			valueBody := valueBlock.Body()

			// Check direct attributes (ip, asn)
			for _, attr := range valueBody.Attributes() {
				if tfhcl.IsExpressionAttribute(attr) {
					return true
				}
			}

			// Check nested blocks (hostname, redirect)
			for _, nestedBlock := range valueBody.Blocks() {
				nestedBody := nestedBlock.Body()
				for _, attr := range nestedBody.Attributes() {
					if tfhcl.IsExpressionAttribute(attr) {
						return true
					}
				}
			}
		}
	}

	return false
}

// buildStaticItemsExpressionStringPreserving creates a tuple expression string
// from static item blocks, preserving expressions like each.key, each.value
func buildStaticItemsExpressionStringPreserving(blocks []*hclwrite.Block, kind string) string {
	var itemStrs []string

	for _, block := range blocks {
		objStr := buildObjectStringFromItemBlockPreserving(block, kind)
		if objStr != "" {
			itemStrs = append(itemStrs, objStr)
		}
	}

	if len(itemStrs) == 0 {
		return ""
	}

	// Format as multi-line array for complex objects
	return fmt.Sprintf("[%s]", strings.Join(itemStrs, ", "))
}

// buildObjectStringFromItemBlockPreserving creates an object expression string
// from a static item block, preserving expressions
func buildObjectStringFromItemBlockPreserving(block *hclwrite.Block, kind string) string {
	body := block.Body()
	var fields []string

	// Handle comment - preserve expression if present
	if commentAttr := body.GetAttribute("comment"); commentAttr != nil {
		commentExpr := strings.TrimSpace(string(commentAttr.Expr().BuildTokens(nil).Bytes()))
		if commentExpr != "" {
			fields = append(fields, fmt.Sprintf("comment = %s", commentExpr))
		}
	}

	// Process value block
	for _, vBlock := range body.Blocks() {
		if vBlock.Type() == "value" {
			valueFields := extractValueBlockFieldsPreserving(vBlock, kind)
			fields = append(fields, valueFields...)
		}
	}

	if len(fields) == 0 {
		return ""
	}

	// Format as multi-line object for proper HCL output
	return fmt.Sprintf("{\n    %s\n  }", strings.Join(fields, "\n    "))
}

// extractValueBlockFieldsPreserving extracts field expressions from a value block,
// preserving expressions like each.value
func extractValueBlockFieldsPreserving(vBlock *hclwrite.Block, kind string) []string {
	vBody := vBlock.Body()
	var fields []string

	switch kind {
	case "ip":
		if ipAttr := vBody.GetAttribute("ip"); ipAttr != nil {
			ipExpr := strings.TrimSpace(string(ipAttr.Expr().BuildTokens(nil).Bytes()))
			ipExpr = normalizeIPAddressInExpr(ipExpr)
			fields = append(fields, fmt.Sprintf("ip = %s", ipExpr))
		}

	case "asn":
		if asnAttr := vBody.GetAttribute("asn"); asnAttr != nil {
			asnExpr := strings.TrimSpace(string(asnAttr.Expr().BuildTokens(nil).Bytes()))
			fields = append(fields, fmt.Sprintf("asn = %s", asnExpr))
		}

	case "hostname":
		for _, hBlock := range vBody.Blocks() {
			if hBlock.Type() == "hostname" {
				hostnameObj := buildHostnameObjectStringPreserving(hBlock)
				if hostnameObj != "" {
					fields = append(fields, fmt.Sprintf("hostname = %s", hostnameObj))
				}
			}
		}

	case "redirect":
		for _, rBlock := range vBody.Blocks() {
			if rBlock.Type() == "redirect" {
				redirectObj := buildRedirectObjectStringPreserving(rBlock)
				if redirectObj != "" {
					fields = append(fields, fmt.Sprintf("redirect = %s", redirectObj))
				}
			}
		}
	}

	return fields
}

// buildHostnameObjectStringPreserving creates a hostname object expression string,
// preserving expressions
func buildHostnameObjectStringPreserving(hBlock *hclwrite.Block) string {
	hBody := hBlock.Body()
	var fields []string

	if urlAttr := hBody.GetAttribute("url_hostname"); urlAttr != nil {
		urlExpr := strings.TrimSpace(string(urlAttr.Expr().BuildTokens(nil).Bytes()))
		fields = append(fields, fmt.Sprintf("url_hostname = %s", urlExpr))
	}

	if len(fields) == 0 {
		return ""
	}

	// Format as multi-line nested object
	return fmt.Sprintf("{\n      %s\n    }", strings.Join(fields, "\n      "))
}

// buildRedirectObjectStringPreserving creates a redirect object expression string
// with boolean conversions, preserving expressions
func buildRedirectObjectStringPreserving(rBlock *hclwrite.Block) string {
	rBody := rBlock.Body()
	var fields []string

	// Required fields
	if sourceAttr := rBody.GetAttribute("source_url"); sourceAttr != nil {
		sourceExpr := strings.TrimSpace(string(sourceAttr.Expr().BuildTokens(nil).Bytes()))
		// For literal strings, ensure path is present
		sourceExpr = ensureSourceURLHasPathInExpr(sourceExpr)
		fields = append(fields, fmt.Sprintf("source_url = %s", sourceExpr))
	}

	if targetAttr := rBody.GetAttribute("target_url"); targetAttr != nil {
		targetExpr := strings.TrimSpace(string(targetAttr.Expr().BuildTokens(nil).Bytes()))
		fields = append(fields, fmt.Sprintf("target_url = %s", targetExpr))
	}

	// Boolean fields that need conversion from "enabled"/"disabled"
	boolFields := []string{
		"include_subdomains",
		"subpath_matching",
		"preserve_query_string",
		"preserve_path_suffix",
	}

	for _, field := range boolFields {
		if attr := rBody.GetAttribute(field); attr != nil {
			value := tfhcl.ExtractStringFromAttribute(attr)
			if value == "enabled" {
				fields = append(fields, fmt.Sprintf("%s = true", field))
			} else if value == "disabled" {
				fields = append(fields, fmt.Sprintf("%s = false", field))
			} else {
				// Keep original expression (might be dynamic)
				exprStr := strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
				// Convert "enabled"/"disabled" strings in expressions
				exprStr = tfhcl.ConvertEnabledDisabledInExpr(exprStr)
				fields = append(fields, fmt.Sprintf("%s = %s", field, exprStr))
			}
		}
	}

	// Optional status_code
	if statusAttr := rBody.GetAttribute("status_code"); statusAttr != nil {
		statusExpr := strings.TrimSpace(string(statusAttr.Expr().BuildTokens(nil).Bytes()))
		fields = append(fields, fmt.Sprintf("status_code = %s", statusExpr))
	}

	if len(fields) == 0 {
		return ""
	}

	return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))
}

// ensureSourceURLHasPath ensures the source_url has a path component.
// The v5 provider requires source_url to have a non-empty path.
// If the URL doesn't have a path, append "/" to it.
func ensureSourceURLHasPath(url string) string {
	if url == "" {
		return url
	}
	// If URL doesn't contain "/" (no path), append "/"
	if !strings.Contains(url, "/") {
		return url + "/"
	}
	return url
}

// ensureSourceURLHasPathInExpr ensures source_url has a path in an HCL expression.
// For literal strings like "test.com", transforms to "test.com/".
// For dynamic expressions, returns unchanged.
func ensureSourceURLHasPathInExpr(expr string) string {
	if expr == "" {
		return expr
	}
	// Check if it's a quoted string literal
	if strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) {
		// Extract the URL value
		url := strings.Trim(expr, `"`)
		// If URL doesn't contain "/" (no path), append "/"
		if !strings.Contains(url, "/") {
			return `"` + url + `/"`
		}
	}
	return expr
}

// normalizeIPAddress normalizes an IP address by removing CIDR notation.
// The v5 provider requires IP addresses to be normalized without CIDR suffix.
// Examples:
//   "10.0.0.0/8" -> "10.0.0.0"
//   "192.168.1.0/24" -> "192.168.1.0"
//   "1.1.1.1" -> "1.1.1.1" (no change)
func normalizeIPAddress(ip string) string {
	if ip == "" {
		return ip
	}
	// Check if IP contains CIDR notation
	if idx := strings.Index(ip, "/"); idx != -1 {
		// Return only the IP address part, removing /prefix
		return ip[:idx]
	}
	return ip
}

// normalizeIPAddressInExpr normalizes IP addresses in HCL expressions.
// For literal strings like "10.0.0.0/8", transforms to "10.0.0.0".
// For dynamic expressions, returns unchanged (cannot normalize at migration time).
func normalizeIPAddressInExpr(expr string) string {
	if expr == "" {
		return expr
	}
	// Check if it's a quoted string literal
	if strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) {
		// Extract the IP value
		ip := strings.Trim(expr, `"`)
		// Normalize and re-quote
		return `"` + normalizeIPAddress(ip) + `"`
	}
	// For non-literal expressions (e.g., variables, each.value), return as-is
	// The user will need to ensure their data source provides normalized IPs
	return expr
}
