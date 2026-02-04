package workers_script

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with both v4 resource names (plural and singular forms)
	internal.RegisterMigrator("cloudflare_workers_script", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_worker_script", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_workers_script"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_workers_script" || resourceType == "cloudflare_worker_script"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Handles both cloudflare_worker_script (singular) and cloudflare_workers_script (plural)
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_worker_script", "cloudflare_workers_script"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Handle resource rename: cloudflare_worker_script → cloudflare_workers_script
	tfhcl.RenameResourceType(block, "cloudflare_worker_script", "cloudflare_workers_script")

	// Rename field: name → script_name
	tfhcl.RenameAttribute(body, "name", "script_name")

	// Remove deprecated fields
	// tags - not supported in v5
	tfhcl.RemoveAttributes(body, "tags")

	// Transform bindings: Convert 10 different binding blocks + dispatch_namespace attr → unified bindings list
	m.transformBindings(body)

	// Transform module boolean → main_module/body_part string
	m.transformModule(body)

	// Transform placement block → object attribute
	m.transformPlacement(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// transformBindings converts v4 binding blocks and dispatch_namespace attribute to v5 unified bindings list
func (m *V4ToV5Migrator) transformBindings(body *hclwrite.Body) {
	var bindingObjects []string
	var dynamicExprs []string

	// Map of v4 block types to v5 binding types
	bindingTypeMap := map[string]string{
		"plain_text_binding":        "plain_text",
		"secret_text_binding":       "secret_text",
		"kv_namespace_binding":      "kv_namespace",
		"webassembly_binding":       "wasm_module",
		"service_binding":           "service",
		"r2_bucket_binding":         "r2_bucket",
		"analytics_engine_binding":  "analytics_engine",
		"queue_binding":             "queue",
		"d1_database_binding":       "d1",
		"hyperdrive_config_binding": "hyperdrive",
	}

	// Process blocks in document order to preserve binding order
	for _, block := range body.Blocks() {
		// Handle dynamic blocks
		if block.Type() == "dynamic" {
			labels := block.Labels()
			if len(labels) > 0 {
				if v5BindingType, ok := bindingTypeMap[labels[0]]; ok {
					dynamicExpr := m.convertDynamicBindingToExpr(block, v5BindingType)
					if dynamicExpr != "" {
						dynamicExprs = append(dynamicExprs, dynamicExpr)
					}
				}
			}
		} else if v5BindingType, ok := bindingTypeMap[block.Type()]; ok {
			bindingObj := m.convertBindingBlockToObject(block, v5BindingType)
			if bindingObj != "" {
				bindingObjects = append(bindingObjects, bindingObj)
			}
		}
	}

	// Handle dispatch_namespace attribute conversion
	if dispatchAttr := body.GetAttribute("dispatch_namespace"); dispatchAttr != nil {
		dispatchValue := tfhcl.ExtractStringFromAttribute(dispatchAttr)
		if dispatchValue != "" {
			bindingObj := "{\n    type = \"dispatch_namespace\"\n    namespace = " + quoteValue(dispatchValue) + "\n  }"
			bindingObjects = append(bindingObjects, bindingObj)
		}
		body.RemoveAttribute("dispatch_namespace")
	}

	// Create unified bindings list if we have any bindings
	if len(bindingObjects) > 0 || len(dynamicExprs) > 0 {
		var bindingsValue string
		if len(dynamicExprs) > 0 {
			// Use concat when there are dynamic expressions
			staticPart := "[\n  " + joinBindings(bindingObjects) + "\n]"
			bindingsValue = "concat(" + staticPart + ", " + dynamicExprs[0]
			for i := 1; i < len(dynamicExprs); i++ {
				bindingsValue += ", " + dynamicExprs[i]
			}
			bindingsValue += ")"
		} else {
			bindingsValue = "[\n  " + joinBindings(bindingObjects) + "\n]"
		}
		// Use SetAttributeFromExpressionString to set the bindings array
		tfhcl.SetAttributeFromExpressionString(body, "bindings", bindingsValue)
	}

	// Remove all v4 binding blocks
	for blockType := range bindingTypeMap {
		removeBlocks(body, blockType)
	}

	// Remove dynamic binding blocks
	removeDynamicBlocks(body, bindingTypeMap)
}

// convertDynamicBindingToExpr converts a dynamic binding block to a conditional expression
// e.g., dynamic "queue_binding" { for_each = cond ? [1] : [] content { ... } }
// becomes: cond ? [{ type = "queue" ... }] : []
func (m *V4ToV5Migrator) convertDynamicBindingToExpr(block *hclwrite.Block, bindingType string) string {
	blockBody := block.Body()

	// Get the for_each attribute
	forEachAttr := blockBody.GetAttribute("for_each")
	if forEachAttr == nil {
		return ""
	}

	forEachExpr := exprToString(forEachAttr.Expr())

	// Find the content block
	var contentBlock *hclwrite.Block
	for _, b := range blockBody.Blocks() {
		if b.Type() == "content" {
			contentBlock = b
			break
		}
	}

	if contentBlock == nil {
		return ""
	}

	// Convert the content block to a binding object
	bindingObj := m.convertBindingBlockToObject(contentBlock, bindingType)
	if bindingObj == "" {
		return ""
	}

	// Extract condition from for_each expression if it's a ternary like "cond ? [1] : []"
	condition := extractCondition(forEachExpr)
	if condition != "" {
		return condition + " ? [" + bindingObj + "] : []"
	}

	// Fallback: wrap in the for_each expression
	return forEachExpr + " != [] ? [" + bindingObj + "] : []"
}

// extractCondition tries to extract the condition from a ternary for_each expression
// e.g., "each.value.use_queue ? [1] : []" → "each.value.use_queue"
func extractCondition(expr string) string {
	// Look for pattern: "condition ? [...] : []"
	if idx := -1; idx < len(expr) {
		for i := 0; i < len(expr); i++ {
			if expr[i] == '?' {
				idx = i
				break
			}
		}
		if idx != -1 {
			condition := expr[:idx]
			// Trim whitespace
			for len(condition) > 0 && (condition[0] == ' ' || condition[0] == '\t') {
				condition = condition[1:]
			}
			for len(condition) > 0 && (condition[len(condition)-1] == ' ' || condition[len(condition)-1] == '\t') {
				condition = condition[:len(condition)-1]
			}
			return condition
		}
	}
	return ""
}

// convertBindingBlockToObject converts a v4 binding block to a v5 binding object string
func (m *V4ToV5Migrator) convertBindingBlockToObject(block *hclwrite.Block, bindingType string) string {
	blockBody := block.Body()
	var attrs []string

	// Always add type first
	attrs = append(attrs, "type = \""+bindingType+"\"")

	// Handle attribute renames based on binding type
	switch bindingType {
	case "wasm_module":
		// webassembly_binding: module → part
		if attr := blockBody.GetAttribute("name"); attr != nil {
			attrs = append(attrs, "name = "+exprToString(attr.Expr()))
		}
		if attr := blockBody.GetAttribute("module"); attr != nil {
			attrs = append(attrs, "part = "+exprToString(attr.Expr()))
		}
	case "queue":
		// queue_binding: binding → name, queue → queue (but update references from .name to .queue_name)
		if attr := blockBody.GetAttribute("binding"); attr != nil {
			attrs = append(attrs, "name = "+exprToString(attr.Expr()))
		}
		if attr := blockBody.GetAttribute("queue"); attr != nil {
			queueExpr := exprToString(attr.Expr())
			// Replace cloudflare_queue.*.name with cloudflare_queue.*.queue_name
			queueExpr = replaceQueueNameReference(queueExpr)
			attrs = append(attrs, "queue = "+queueExpr)
		}
	case "d1":
		// d1_database_binding: database_id → id
		if attr := blockBody.GetAttribute("name"); attr != nil {
			attrs = append(attrs, "name = "+exprToString(attr.Expr()))
		}
		if attr := blockBody.GetAttribute("database_id"); attr != nil {
			attrs = append(attrs, "id = "+exprToString(attr.Expr()))
		}
	case "hyperdrive":
		// hyperdrive_config_binding: binding → name, id stays
		if attr := blockBody.GetAttribute("binding"); attr != nil {
			attrs = append(attrs, "name = "+exprToString(attr.Expr()))
		}
		if attr := blockBody.GetAttribute("id"); attr != nil {
			attrs = append(attrs, "id = "+exprToString(attr.Expr()))
		}
	default:
		// All other binding types: copy attributes in consistent order (name first, then others)
		if attr := blockBody.GetAttribute("name"); attr != nil {
			attrs = append(attrs, "name = "+exprToString(attr.Expr()))
		}
		// Add remaining attributes (except name which we already added)
		for attrName, attr := range blockBody.Attributes() {
			if attrName != "name" {
				attrs = append(attrs, attrName+" = "+exprToString(attr.Expr()))
			}
		}
	}

	return "{\n    " + joinAttributes(attrs) + "\n  }"
}

// transformModule converts module boolean to main_module or body_part string
func (m *V4ToV5Migrator) transformModule(body *hclwrite.Body) {
	moduleAttr := body.GetAttribute("module")
	if moduleAttr == nil {
		return
	}

	// Extract the boolean value
	moduleValue, ok := tfhcl.ExtractBoolFromAttribute(moduleAttr)
	if !ok {
		// If it's a variable reference, we can't transform it statically
		// Leave it as-is and let the user handle it manually
		return
	}

	// Remove the module attribute
	body.RemoveAttribute("module")

	// Add main_module if true, body_part if false
	// Use a default filename since we don't have the actual filename
	if moduleValue {
		body.SetAttributeRaw("main_module", tfhcl.TokensForSimpleValue("worker.js"))
	} else {
		body.SetAttributeRaw("body_part", tfhcl.TokensForSimpleValue("worker.js"))
	}
}

// transformPlacement converts placement block to object attribute
func (m *V4ToV5Migrator) transformPlacement(body *hclwrite.Body) {
	var placementBlock *hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == "placement" {
			placementBlock = block
			break
		}
	}

	if placementBlock == nil {
		return
	}

	placementBody := placementBlock.Body()

	// Extract the mode attribute
	if modeAttr := placementBody.GetAttribute("mode"); modeAttr != nil {
		modeValue := exprToString(modeAttr.Expr())
		// Create object syntax: placement = { mode = "smart" }
		placementObj := "{\n    mode = " + modeValue + "\n  }"
		tfhcl.SetAttributeFromExpressionString(body, "placement", placementObj)
	}

	// Remove the placement block
	removeBlocks(body, "placement")
}

// Helper functions

// replaceQueueNameReference replaces cloudflare_queue.*.name with cloudflare_queue.*.queue_name
func replaceQueueNameReference(expr string) string {
	// Simple string replacement for the common pattern
	// Handles: cloudflare_queue.resource_name.name → cloudflare_queue.resource_name.queue_name
	i := 0
	result := ""
	for i < len(expr) {
		// Look for "cloudflare_queue."
		if i+17 <= len(expr) && expr[i:i+17] == "cloudflare_queue." {
			// Find the next dot or bracket
			j := i + 17
			for j < len(expr) && expr[j] != '.' && expr[j] != '[' {
				j++
			}
			// Check for .name
			if j < len(expr) && expr[j] == '.' && j+5 <= len(expr) && expr[j:j+5] == ".name" {
				// Check it's not part of a longer identifier
				if j+5 >= len(expr) || (expr[j+5] != '_' && !((expr[j+5] >= 'a' && expr[j+5] <= 'z') || (expr[j+5] >= 'A' && expr[j+5] <= 'Z') || (expr[j+5] >= '0' && expr[j+5] <= '9'))) {
					result += expr[i:j] + ".queue_name"
					i = j + 5
					continue
				}
			}
			// Check for [index].name
			if j < len(expr) && expr[j] == '[' {
				k := j + 1
				depth := 1
				for k < len(expr) && depth > 0 {
					if expr[k] == '[' {
						depth++
					} else if expr[k] == ']' {
						depth--
					}
					k++
				}
				if k < len(expr) && expr[k] == '.' && k+5 <= len(expr) && expr[k:k+5] == ".name" {
					if k+5 >= len(expr) || (expr[k+5] != '_' && !((expr[k+5] >= 'a' && expr[k+5] <= 'z') || (expr[k+5] >= 'A' && expr[k+5] <= 'Z') || (expr[k+5] >= '0' && expr[k+5] <= '9'))) {
						result += expr[i:k] + ".queue_name"
						i = k + 5
						continue
					}
				}
			}
		}
		result += string(expr[i])
		i++
	}
	return result
}

// exprToString converts an HCL expression to its string representation
func exprToString(expr *hclwrite.Expression) string {
	if expr == nil {
		return ""
	}
	tokens := expr.BuildTokens(nil)
	var result []byte
	for _, token := range tokens {
		result = append(result, token.Bytes...)
	}
	return string(result)
}

// removeBlocks removes all blocks of a given type from a body
func removeBlocks(body *hclwrite.Body, blockType string) {
	// We need to collect blocks first then remove them
	// to avoid modifying the collection while iterating
	var blocksToRemove []*hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == blockType {
			blocksToRemove = append(blocksToRemove, block)
		}
	}
	for _, block := range blocksToRemove {
		body.RemoveBlock(block)
	}
}

// removeDynamicBlocks removes dynamic blocks that wrap binding blocks
func removeDynamicBlocks(body *hclwrite.Body, bindingTypeMap map[string]string) {
	var blocksToRemove []*hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == "dynamic" {
			labels := block.Labels()
			if len(labels) > 0 {
				if _, ok := bindingTypeMap[labels[0]]; ok {
					blocksToRemove = append(blocksToRemove, block)
				}
			}
		}
	}
	for _, block := range blocksToRemove {
		body.RemoveBlock(block)
	}
}

func quoteValue(value string) string {
	// If already quoted or is a variable reference, return as-is
	if len(value) == 0 {
		return "\"\""
	}
	if value[0] == '"' && value[len(value)-1] == '"' {
		return value
	}
	if len(value) >= 4 && value[0:4] == "var." {
		return value
	}
	return "\"" + value + "\""
}

func joinBindings(bindings []string) string {
	if len(bindings) == 0 {
		return ""
	}
	result := bindings[0]
	for i := 1; i < len(bindings); i++ {
		result += ", " + bindings[i]
	}
	return result
}

func joinAttributes(attrs []string) string {
	if len(attrs) == 0 {
		return ""
	}
	result := attrs[0]
	for i := 1; i < len(attrs); i++ {
		result += "\n    " + attrs[i]
	}
	return result
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// Rename field: name → script_name
	result = state.RenameField(result, "attributes", attrs, "name", "script_name")

	// Remove deprecated fields
	// tags - not supported in v5
	result = state.RemoveFields(result, "attributes", attrs, "tags")

	// Transform bindings: Convert 10 different binding arrays + dispatch_namespace to unified bindings array
	result = m.transformStateBindings(result, attrs)

	// Transform module boolean → main_module/body_part string
	result = m.transformStateModule(result, attrs)

	// Transform placement if needed
	result = m.transformStatePlacement(result, attrs)

	// Set schema_version to 1 to match provider schema version
	// This prevents the provider from running state upgraders since tf-migrate
	// already produces the final v5 format
	result, _ = sjson.Set(result, "schema_version", 1)

	return result, nil
}

// transformStateBindings converts v4 binding arrays and dispatch_namespace to v5 unified bindings array
func (m *V4ToV5Migrator) transformStateBindings(result string, attrs gjson.Result) string {
	var bindings []interface{}

	// Array of binding types in consistent order (same as config)
	bindingTypes := []struct {
		v4ArrayName   string
		v5BindingType string
	}{
		{"plain_text_binding", "plain_text"},
		{"secret_text_binding", "secret_text"},
		{"kv_namespace_binding", "kv_namespace"},
		{"webassembly_binding", "wasm_module"},
		{"service_binding", "service"},
		{"r2_bucket_binding", "r2_bucket"},
		{"analytics_engine_binding", "analytics_engine"},
		{"queue_binding", "queue"},
		{"d1_database_binding", "d1"},
		{"hyperdrive_config_binding", "hyperdrive"},
	}

	// Process each binding array type in order
	for _, bt := range bindingTypes {
		bindingArray := attrs.Get(bt.v4ArrayName)
		if bindingArray.Exists() && bindingArray.IsArray() {
			for _, bindingItem := range bindingArray.Array() {
				binding := m.convertStateBindingToObject(bindingItem, bt.v5BindingType)
				if binding != nil {
					bindings = append(bindings, binding)
				}
			}
		}
	}

	// Handle dispatch_namespace attribute conversion
	if dispatchNS := attrs.Get("dispatch_namespace"); dispatchNS.Exists() {
		if dispatchNS.Type != gjson.Null && dispatchNS.String() != "" {
			binding := map[string]interface{}{
				"type":      "dispatch_namespace",
				"namespace": dispatchNS.String(),
			}
			bindings = append(bindings, binding)
		}
		// Always delete the attribute whether it was null or had a value
		result, _ = sjson.Delete(result, "attributes.dispatch_namespace")
	}

	// Set unified bindings array if we have any bindings
	if len(bindings) > 0 {
		result, _ = sjson.Set(result, "attributes.bindings", bindings)
	}

	// Remove all v4 binding arrays
	for _, bt := range bindingTypes {
		result, _ = sjson.Delete(result, "attributes."+bt.v4ArrayName)
	}

	return result
}

// convertStateBindingToObject converts a v4 binding object to v5 format with attribute renames
func (m *V4ToV5Migrator) convertStateBindingToObject(bindingItem gjson.Result, bindingType string) map[string]interface{} {
	binding := map[string]interface{}{
		"type": bindingType,
	}

	// Handle attribute renames based on binding type
	switch bindingType {
	case "wasm_module":
		// webassembly_binding: module → part
		if name := bindingItem.Get("name"); name.Exists() {
			binding["name"] = name.Value()
		}
		if module := bindingItem.Get("module"); module.Exists() {
			binding["part"] = module.Value()
		}
	case "queue":
		// queue_binding: binding → name, queue stays as queue
		if bindingName := bindingItem.Get("binding"); bindingName.Exists() {
			binding["name"] = bindingName.Value()
		}
		if queueName := bindingItem.Get("queue"); queueName.Exists() {
			binding["queue"] = queueName.Value()
		}
	case "d1":
		// d1_database_binding: database_id → id
		if name := bindingItem.Get("name"); name.Exists() {
			binding["name"] = name.Value()
		}
		if databaseID := bindingItem.Get("database_id"); databaseID.Exists() {
			binding["id"] = databaseID.Value()
		}
	case "hyperdrive":
		// hyperdrive_config_binding: binding → name
		if bindingName := bindingItem.Get("binding"); bindingName.Exists() {
			binding["name"] = bindingName.Value()
		}
		if id := bindingItem.Get("id"); id.Exists() {
			binding["id"] = id.Value()
		}
	default:
		// All other binding types: copy fields as-is
		bindingItem.ForEach(func(key, value gjson.Result) bool {
			binding[key.String()] = value.Value()
			return true
		})
	}

	return binding
}

// transformStateModule converts module boolean to main_module or body_part string
func (m *V4ToV5Migrator) transformStateModule(result string, attrs gjson.Result) string {
	moduleAttr := attrs.Get("module")
	if !moduleAttr.Exists() {
		return result
	}

	// Only process if not null
	if moduleAttr.Type != gjson.Null {
		// Get the boolean value
		moduleBool := moduleAttr.Bool()

		// Add main_module if true, body_part if false
		// Use a default filename since we don't have the actual filename
		if moduleBool {
			result, _ = sjson.Set(result, "attributes.main_module", "worker.js")
		} else {
			result, _ = sjson.Set(result, "attributes.body_part", "worker.js")
		}
	}

	// Always remove the module attribute whether it was null or had a value
	result, _ = sjson.Delete(result, "attributes.module")

	return result
}

// transformStatePlacement converts placement array to object if needed
func (m *V4ToV5Migrator) transformStatePlacement(result string, attrs gjson.Result) string {
	// Use helper to transform placement from array to object
	// Empty arrays will be removed, non-empty arrays will be unwrapped to objects
	result = state.TransformFieldArrayToObject(
		result,
		"attributes",
		attrs,
		"placement",
		state.ArrayToObjectOptions{
			EnsureObjectExists: false, // Remove empty arrays instead of creating empty objects
		},
	)

	return result
}
