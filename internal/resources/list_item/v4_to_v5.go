package list_item

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of cloudflare_list_item resources from v4 to v5.
// In v5, list items are embedded in the parent cloudflare_list resource.
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_list_item", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return ""
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_list_item"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return &transform.TransformResult{
		Blocks:         nil,
		RemoveOriginal: true,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return "", nil
}

func (m *V4ToV5Migrator) GetCrossResourceDependency() string {
	return "cloudflare_list"
}

// ProcessCrossResourceConfigMigration merges list_item resources into their parent cloudflare_list resources.
func ProcessCrossResourceConfigMigration(file *hclwrite.File) {
	body := file.Body()

	listResources := make(map[string]*hclwrite.Block)
	listItemsByParent := make(map[string][]*hclwrite.Block)

	for _, block := range body.Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 {
			resourceType := block.Labels()[0]
			resourceName := block.Labels()[1]

			if resourceType == "cloudflare_list" {
				listResources[resourceName] = block
			} else if resourceType == "cloudflare_list_item" {
				parentList := extractParentListName(block)
				if parentList != "" {
					listItemsByParent[parentList] = append(listItemsByParent[parentList], block)
				}
			}
		}
	}

	for listName, listBlock := range listResources {
		items := listItemsByParent[listName]
		if len(items) == 0 {
			continue
		}

		kind := tfhcl.ExtractStringFromAttribute(listBlock.Body().GetAttribute("kind"))
		if kind == "" {
			addMigrationWarning(listBlock.Body(), "Cannot determine list kind for merging list_item resources")
			continue
		}

		// Check if list already has items
		existingItems := listBlock.Body().GetAttribute("items")
		if existingItems != nil {
			addMigrationWarning(listBlock.Body(), "List already has items - manual merge may be required")
			for _, itemBlock := range items {
				body.RemoveBlock(itemBlock)
			}
			continue
		}

		// Check if there are dynamic patterns (for_each or count)
		hasDynamic := false
		for _, item := range items {
			if item.Body().GetAttribute("for_each") != nil || item.Body().GetAttribute("count") != nil {
				hasDynamic = true
				break
			}
		}

		if hasDynamic {
			// Use string-based approach for dynamic patterns
			itemsExpr := buildDynamicItemsExpression(items, kind)
			if itemsExpr != "" {
				setItemsAttributeFromString(listBlock.Body(), itemsExpr)
			}
		} else {
			// Use cty.Value approach for static items
			itemsArray := buildStaticItemsFromListItems(items, kind)
			if len(itemsArray) > 0 {
				listBlock.Body().SetAttributeValue("items", cty.TupleVal(itemsArray))
			}
		}

		for _, itemBlock := range items {
			body.RemoveBlock(itemBlock)
		}
	}
}

// buildDynamicItemsExpression builds an items expression for list_items with for_each or count
func buildDynamicItemsExpression(items []*hclwrite.Block, kind string) string {
	if len(items) == 0 {
		return ""
	}

	// Check for single for_each or count pattern
	if len(items) == 1 {
		item := items[0]
		body := item.Body()

		if forEachAttr := body.GetAttribute("for_each"); forEachAttr != nil {
			return buildForEachExpression(item, kind)
		}

		if countAttr := body.GetAttribute("count"); countAttr != nil {
			return buildCountExpression(item, kind)
		}
	}

	// Multiple items or single static item - build array
	var itemStrs []string
	for _, item := range items {
		objStr := buildItemObjectString(item, kind)
		if objStr != "" {
			itemStrs = append(itemStrs, objStr)
		}
	}

	if len(itemStrs) == 0 {
		return ""
	}

	return fmt.Sprintf("[%s]", strings.Join(itemStrs, ", "))
}

// buildForEachExpression creates items = [for k, v in collection : {...}] from for_each pattern
func buildForEachExpression(item *hclwrite.Block, kind string) string {
	body := item.Body()
	forEachAttr := body.GetAttribute("for_each")
	if forEachAttr == nil {
		return ""
	}

	forEachStr := strings.TrimSpace(string(forEachAttr.Expr().BuildTokens(nil).Bytes()))

	// Build object fields
	fields := buildItemFieldStrings(body, kind, "v", "k")

	if len(fields) == 0 {
		return ""
	}

	objExpr := fmt.Sprintf("{\n    %s\n  }", strings.Join(fields, "\n    "))
	return fmt.Sprintf("[for k, v in %s : %s]", forEachStr, objExpr)
}

// buildCountExpression creates items = [for i in range(count) : {...}] from count pattern
func buildCountExpression(item *hclwrite.Block, kind string) string {
	body := item.Body()
	countAttr := body.GetAttribute("count")
	if countAttr == nil {
		return ""
	}

	countStr := strings.TrimSpace(string(countAttr.Expr().BuildTokens(nil).Bytes()))

	// Build object fields replacing count.index with i
	fields := buildItemFieldStringsForCount(body, kind)

	if len(fields) == 0 {
		return ""
	}

	objExpr := fmt.Sprintf("{\n    %s\n  }", strings.Join(fields, "\n    "))
	return fmt.Sprintf("[for i in range(%s) : %s]", countStr, objExpr)
}

// buildItemFieldStrings extracts field strings from a list_item block
func buildItemFieldStrings(body *hclwrite.Body, kind string, valVar, keyVar string) []string {
	var fields []string

	// Add kind-specific field
	switch kind {
	case "ip":
		if ipAttr := body.GetAttribute("ip"); ipAttr != nil {
			ipExpr := strings.TrimSpace(string(ipAttr.Expr().BuildTokens(nil).Bytes()))
			ipExpr = replaceEachReferences(ipExpr, valVar, keyVar)
			fields = append(fields, fmt.Sprintf("ip = %s", ipExpr))
		}

	case "asn":
		if asnAttr := body.GetAttribute("asn"); asnAttr != nil {
			asnExpr := strings.TrimSpace(string(asnAttr.Expr().BuildTokens(nil).Bytes()))
			asnExpr = replaceEachReferences(asnExpr, valVar, keyVar)
			fields = append(fields, fmt.Sprintf("asn = %s", asnExpr))
		}

	case "hostname":
		if hostnameAttr := body.GetAttribute("hostname"); hostnameAttr != nil {
			hostnameExpr := extractHostnameExpression(hostnameAttr)
			hostnameExpr = replaceEachReferences(hostnameExpr, valVar, keyVar)
			fields = append(fields, fmt.Sprintf("hostname = %s", hostnameExpr))
		}

	case "redirect":
		if redirectAttr := body.GetAttribute("redirect"); redirectAttr != nil {
			redirectExpr := extractRedirectExpression(redirectAttr)
			redirectExpr = replaceEachReferences(redirectExpr, valVar, keyVar)
			fields = append(fields, fmt.Sprintf("redirect = %s", redirectExpr))
		}
	}

	// Add comment if present
	if commentAttr := body.GetAttribute("comment"); commentAttr != nil {
		commentExpr := strings.TrimSpace(string(commentAttr.Expr().BuildTokens(nil).Bytes()))
		commentExpr = replaceEachReferences(commentExpr, valVar, keyVar)
		fields = append(fields, fmt.Sprintf("comment = %s", commentExpr))
	}

	return fields
}

// buildItemFieldStringsForCount extracts field strings, replacing count.index with i
func buildItemFieldStringsForCount(body *hclwrite.Body, kind string) []string {
	var fields []string

	switch kind {
	case "ip":
		if ipAttr := body.GetAttribute("ip"); ipAttr != nil {
			ipExpr := strings.TrimSpace(string(ipAttr.Expr().BuildTokens(nil).Bytes()))
			ipExpr = strings.ReplaceAll(ipExpr, "count.index", "i")
			fields = append(fields, fmt.Sprintf("ip = %s", ipExpr))
		}

	case "asn":
		if asnAttr := body.GetAttribute("asn"); asnAttr != nil {
			asnExpr := strings.TrimSpace(string(asnAttr.Expr().BuildTokens(nil).Bytes()))
			asnExpr = strings.ReplaceAll(asnExpr, "count.index", "i")
			fields = append(fields, fmt.Sprintf("asn = %s", asnExpr))
		}

	case "hostname":
		if hostnameAttr := body.GetAttribute("hostname"); hostnameAttr != nil {
			hostnameExpr := extractHostnameExpression(hostnameAttr)
			hostnameExpr = strings.ReplaceAll(hostnameExpr, "count.index", "i")
			fields = append(fields, fmt.Sprintf("hostname = %s", hostnameExpr))
		}

	case "redirect":
		if redirectAttr := body.GetAttribute("redirect"); redirectAttr != nil {
			redirectExpr := extractRedirectExpression(redirectAttr)
			redirectExpr = strings.ReplaceAll(redirectExpr, "count.index", "i")
			fields = append(fields, fmt.Sprintf("redirect = %s", redirectExpr))
		}
	}

	if commentAttr := body.GetAttribute("comment"); commentAttr != nil {
		commentExpr := strings.TrimSpace(string(commentAttr.Expr().BuildTokens(nil).Bytes()))
		commentExpr = strings.ReplaceAll(commentExpr, "count.index", "i")
		fields = append(fields, fmt.Sprintf("comment = %s", commentExpr))
	}

	return fields
}

// replaceEachReferences replaces each.value with v and each.key with k
func replaceEachReferences(expr string, valVar, keyVar string) string {
	expr = strings.ReplaceAll(expr, "each.value", valVar)
	expr = strings.ReplaceAll(expr, "each.key", keyVar)
	return expr
}

// buildItemObjectString builds a simple object string from a static list_item
func buildItemObjectString(item *hclwrite.Block, kind string) string {
	body := item.Body()
	var fields []string

	switch kind {
	case "ip":
		if ipAttr := body.GetAttribute("ip"); ipAttr != nil {
			ipValue := tfhcl.ExtractStringFromAttribute(ipAttr)
			if ipValue != "" {
				fields = append(fields, fmt.Sprintf(`ip = "%s"`, ipValue))
			}
		}

	case "asn":
		if asnAttr := body.GetAttribute("asn"); asnAttr != nil {
			asnExpr := strings.TrimSpace(string(asnAttr.Expr().BuildTokens(nil).Bytes()))
			fields = append(fields, fmt.Sprintf("asn = %s", asnExpr))
		}

	case "hostname":
		if hostnameAttr := body.GetAttribute("hostname"); hostnameAttr != nil {
			hostnameExpr := extractHostnameExpression(hostnameAttr)
			fields = append(fields, fmt.Sprintf("hostname = %s", hostnameExpr))
		}

	case "redirect":
		if redirectAttr := body.GetAttribute("redirect"); redirectAttr != nil {
			redirectExpr := extractRedirectExpression(redirectAttr)
			fields = append(fields, fmt.Sprintf("redirect = %s", redirectExpr))
		}
	}

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

// extractHostnameExpression extracts the hostname attribute and converts it to proper format
func extractHostnameExpression(attr *hclwrite.Attribute) string {
	tokens := attr.Expr().BuildTokens(nil)
	return strings.TrimSpace(string(tokens.Bytes()))
}

// extractRedirectExpression extracts redirect attribute with boolean conversion
func extractRedirectExpression(attr *hclwrite.Attribute) string {
	expr := strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
	// Convert "enabled"/"disabled" strings to booleans
	expr = strings.ReplaceAll(expr, `"enabled"`, "true")
	expr = strings.ReplaceAll(expr, `"disabled"`, "false")
	return expr
}

// setItemsAttributeFromString sets the items attribute from a string expression
func setItemsAttributeFromString(body *hclwrite.Body, exprStr string) {
	attrHCL := fmt.Sprintf("items = %s", exprStr)
	file, diags := hclwrite.ParseConfig([]byte(attrHCL), "items", hcl.InitialPos)
	if diags.HasErrors() {
		addMigrationWarning(body, fmt.Sprintf("Could not parse items expression: %s", attrHCL))
		return
	}

	if itemsAttr := file.Body().GetAttribute("items"); itemsAttr != nil {
		body.SetAttributeRaw("items", itemsAttr.Expr().BuildTokens(nil))
	}
}

// ProcessCrossResourceStateMigration merges list_item state into parent list state.
func ProcessCrossResourceStateMigration(stateJSON string) string {
	result := stateJSON

	lists := make(map[string]int)
	var listItems []listItemStateInfo

	resources := gjson.Get(stateJSON, "resources")
	if resources.Exists() && resources.IsArray() {
		for i, resource := range resources.Array() {
			resourceType := resource.Get("type").String()

			if resourceType == "cloudflare_list" {
				listID := resource.Get("instances.0.attributes.id").String()
				if listID != "" {
					lists[listID] = i
				}
			} else if resourceType == "cloudflare_list_item" {
				item := extractListItemStateInfo(resource, i)
				if item != nil {
					listItems = append(listItems, *item)
				}
			}
		}
	}

	for listID, listIndex := range lists {
		var itemsForList []listItemStateInfo
		for _, item := range listItems {
			if item.listID == listID {
				itemsForList = append(itemsForList, item)
			}
		}

		if len(itemsForList) > 0 {
			result = mergeItemsIntoListState(result, listIndex, itemsForList)
		}
	}

	result = removeListItemResourcesFromState(result)

	return result
}

type listItemStateInfo struct {
	listID    string
	accountID string
	itemData  map[string]interface{}
}

func extractParentListName(itemBlock *hclwrite.Block) string {
	body := itemBlock.Body()
	listIdAttr := body.GetAttribute("list_id")
	if listIdAttr == nil {
		return ""
	}

	tokens := listIdAttr.Expr().BuildTokens(nil)
	tokenStr := string(tokens.Bytes())

	if strings.Contains(tokenStr, "cloudflare_list.") {
		parts := strings.Split(tokenStr, ".")
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[1])
		}
	} else if strings.Contains(tokenStr, `cloudflare_list["`) {
		start := strings.Index(tokenStr, `["`) + 2
		end := strings.Index(tokenStr[start:], `"`)
		if end > 0 {
			return tokenStr[start : start+end]
		}
	}

	return ""
}

// buildStaticItemsFromListItems creates cty.Value items from static list_item resources (no for_each/count)
func buildStaticItemsFromListItems(items []*hclwrite.Block, kind string) []cty.Value {
	var itemObjects []cty.Value

	for _, itemBlock := range items {
		body := itemBlock.Body()
		itemMap := make(map[string]cty.Value)

		// Add kind-specific field first
		switch kind {
		case "ip":
			if ipAttr := body.GetAttribute("ip"); ipAttr != nil {
				ipValue := tfhcl.ExtractStringFromAttribute(ipAttr)
				if ipValue != "" {
					itemMap["ip"] = cty.StringVal(ipValue)
				}
			}

		case "asn":
			if asnAttr := body.GetAttribute("asn"); asnAttr != nil {
				asnTokens := asnAttr.Expr().BuildTokens(nil)
				for _, token := range asnTokens {
					if token.Type == hclsyntax.TokenNumberLit {
						itemMap["asn"] = cty.NumberIntVal(parseNumber(string(token.Bytes)))
						break
					}
				}
			}

		case "hostname":
			if hostnameAttr := body.GetAttribute("hostname"); hostnameAttr != nil {
				// Try to extract hostname object structure
				hostnameObj := extractHostnameObject(hostnameAttr)
				if hostnameObj != cty.NilVal {
					itemMap["hostname"] = hostnameObj
				}
			}

		case "redirect":
			if redirectAttr := body.GetAttribute("redirect"); redirectAttr != nil {
				// Try to extract redirect object structure with boolean conversion
				redirectObj := extractRedirectObject(redirectAttr)
				if redirectObj != cty.NilVal {
					itemMap["redirect"] = redirectObj
				}
			}
		}

		// Add comment
		if commentAttr := body.GetAttribute("comment"); commentAttr != nil {
			commentValue := tfhcl.ExtractStringFromAttribute(commentAttr)
			if commentValue != "" {
				itemMap["comment"] = cty.StringVal(commentValue)
			}
		}

		if len(itemMap) > 0 {
			itemObjects = append(itemObjects, cty.ObjectVal(itemMap))
		}
	}

	return itemObjects
}

// extractHostnameObject extracts hostname object from attribute
func extractHostnameObject(attr *hclwrite.Attribute) cty.Value {
	// For simple cases, try to parse as object
	tokens := attr.Expr().BuildTokens(nil)
	tokensStr := strings.TrimSpace(string(tokens.Bytes()))

	// If it looks like an object { url_hostname = "..." }
	if strings.HasPrefix(tokensStr, "{") {
		// Parse to extract url_hostname
		if strings.Contains(tokensStr, "url_hostname") {
			// Simple extraction - find the quoted value after url_hostname =
			start := strings.Index(tokensStr, `"`)
			if start != -1 {
				end := strings.Index(tokensStr[start+1:], `"`)
				if end != -1 {
					urlHostname := tokensStr[start+1 : start+1+end]
					return cty.ObjectVal(map[string]cty.Value{
						"url_hostname": cty.StringVal(urlHostname),
					})
				}
			}
		}
	}

	return cty.NilVal
}

// extractRedirectObject extracts redirect object from attribute with boolean conversion
func extractRedirectObject(attr *hclwrite.Attribute) cty.Value {
	tokens := attr.Expr().BuildTokens(nil)
	tokensStr := strings.TrimSpace(string(tokens.Bytes()))

	if !strings.HasPrefix(tokensStr, "{") {
		return cty.NilVal
	}

	redirectMap := make(map[string]cty.Value)

	// Extract string fields
	stringFields := []string{"source_url", "target_url"}
	for _, field := range stringFields {
		value := extractQuotedValue(tokensStr, field)
		if value != "" {
			redirectMap[field] = cty.StringVal(value)
		}
	}

	// Extract and convert boolean fields
	boolFields := []string{"include_subdomains", "subpath_matching", "preserve_query_string", "preserve_path_suffix"}
	for _, field := range boolFields {
		value := extractQuotedValue(tokensStr, field)
		if value == "enabled" {
			redirectMap[field] = cty.BoolVal(true)
		} else if value == "disabled" {
			redirectMap[field] = cty.BoolVal(false)
		}
	}

	// Extract status_code as number
	statusCode := extractNumberValue(tokensStr, "status_code")
	if statusCode > 0 {
		redirectMap["status_code"] = cty.NumberIntVal(statusCode)
	}

	if len(redirectMap) == 0 {
		return cty.NilVal
	}

	return cty.ObjectVal(redirectMap)
}

// extractQuotedValue extracts a quoted string value for a field from HCL object string
func extractQuotedValue(hclStr, fieldName string) string {
	// Look for field = "value" pattern
	pattern := fieldName + " "
	idx := strings.Index(hclStr, pattern)
	if idx == -1 {
		pattern = fieldName + "="
		idx = strings.Index(hclStr, pattern)
	}
	if idx == -1 {
		return ""
	}

	// Find the first quote after the field name
	rest := hclStr[idx+len(fieldName):]
	quoteStart := strings.Index(rest, `"`)
	if quoteStart == -1 {
		return ""
	}

	rest = rest[quoteStart+1:]
	quoteEnd := strings.Index(rest, `"`)
	if quoteEnd == -1 {
		return ""
	}

	return rest[:quoteEnd]
}

// extractNumberValue extracts a number value for a field from HCL object string
func extractNumberValue(hclStr, fieldName string) int64 {
	pattern := fieldName + " "
	idx := strings.Index(hclStr, pattern)
	if idx == -1 {
		pattern = fieldName + "="
		idx = strings.Index(hclStr, pattern)
	}
	if idx == -1 {
		return 0
	}

	rest := hclStr[idx+len(fieldName):]
	// Skip whitespace and =
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '=') {
		rest = rest[1:]
	}

	// Parse number
	var n int64
	for len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
		n = n*10 + int64(rest[0]-'0')
		rest = rest[1:]
	}

	return n
}

func extractListItemStateInfo(resource gjson.Result, index int) *listItemStateInfo {
	attrs := resource.Get("instances.0.attributes")
	if !attrs.Exists() {
		return nil
	}

	listID := attrs.Get("list_id").String()
	if listID == "" {
		return nil
	}

	info := &listItemStateInfo{
		listID:    listID,
		accountID: attrs.Get("account_id").String(),
		itemData:  make(map[string]interface{}),
	}

	if ip := attrs.Get("ip"); ip.Exists() && ip.String() != "" {
		info.itemData["ip"] = ip.String()
	}

	if asn := attrs.Get("asn"); asn.Exists() && asn.Type == gjson.Number {
		info.itemData["asn"] = asn.Int()
	}

	if hostname := attrs.Get("hostname"); hostname.Exists() {
		if hostname.IsArray() && len(hostname.Array()) > 0 {
			info.itemData["hostname"] = hostname.Array()[0].Value()
		} else if hostname.IsObject() {
			info.itemData["hostname"] = hostname.Value()
		}
	}

	if redirect := attrs.Get("redirect"); redirect.Exists() {
		var redirectObj map[string]interface{}

		if redirect.IsArray() && len(redirect.Array()) > 0 {
			redirectData := redirect.Array()[0]
			redirectObj = transformRedirectData(redirectData)
		} else if redirect.IsObject() {
			redirectObj = transformRedirectData(redirect)
		}

		if redirectObj != nil {
			info.itemData["redirect"] = redirectObj
		}
	}

	if comment := attrs.Get("comment"); comment.Exists() && comment.String() != "" {
		info.itemData["comment"] = comment.String()
	}

	return info
}

func mergeItemsIntoListState(jsonStr string, listIndex int, items []listItemStateInfo) string {
	result := jsonStr

	itemsPath := fmt.Sprintf("resources.%d.instances.0.attributes.items", listIndex)
	existingItems := gjson.Get(jsonStr, itemsPath)

	var allItems []interface{}

	if existingItems.Exists() && existingItems.IsArray() {
		for _, item := range existingItems.Array() {
			allItems = append(allItems, item.Value())
		}
	}

	for _, item := range items {
		allItems = append(allItems, item.itemData)
	}

	if len(allItems) > 0 {
		result, _ = sjson.Set(result, itemsPath, allItems)
		numItemsPath := fmt.Sprintf("resources.%d.instances.0.attributes.num_items", listIndex)
		result, _ = sjson.Set(result, numItemsPath, float64(len(allItems)))
	}

	return result
}

func removeListItemResourcesFromState(jsonStr string) string {
	result := jsonStr

	resources := gjson.Get(jsonStr, "resources")
	if !resources.Exists() || !resources.IsArray() {
		return result
	}

	var indicesToRemove []int
	for i, resource := range resources.Array() {
		resourceType := resource.Get("type").String()
		if resourceType == "cloudflare_list_item" {
			indicesToRemove = append(indicesToRemove, i)
		}
	}

	// Remove in reverse order to preserve indices
	for i := len(indicesToRemove) - 1; i >= 0; i-- {
		path := fmt.Sprintf("resources.%d", indicesToRemove[i])
		result, _ = sjson.Delete(result, path)
	}

	return result
}

func transformRedirectData(data gjson.Result) map[string]interface{} {
	redirectObj := make(map[string]interface{})

	if sourceURL := data.Get("source_url"); sourceURL.Exists() {
		redirectObj["source_url"] = sourceURL.String()
	}
	if targetURL := data.Get("target_url"); targetURL.Exists() {
		redirectObj["target_url"] = targetURL.String()
	}
	if statusCode := data.Get("status_code"); statusCode.Exists() {
		redirectObj["status_code"] = state.ConvertToInt64(statusCode)
	}

	boolFields := []string{
		"include_subdomains",
		"subpath_matching",
		"preserve_query_string",
		"preserve_path_suffix",
	}

	for _, field := range boolFields {
		if fieldVal := data.Get(field); fieldVal.Exists() {
			if fieldVal.String() == "enabled" {
				redirectObj[field] = true
			} else if fieldVal.String() == "disabled" {
				redirectObj[field] = false
			} else if fieldVal.Type == gjson.True || fieldVal.Type == gjson.False {
				redirectObj[field] = fieldVal.Bool()
			}
		}
	}

	return redirectObj
}

func addMigrationWarning(body *hclwrite.Body, message string) {
	comment := hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenComment,
			Bytes: []byte("# MIGRATION WARNING: " + message + "\n"),
		},
	}
	body.AppendUnstructuredTokens(comment)
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
