package workers_script

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
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
	resourceName := tfhcl.GetResourceName(block)

	// Check if this is the singular form (needs rename + moved block)
	wasSingular := block.Type() == "resource" && len(block.Labels()) > 0 && block.Labels()[0] == "cloudflare_worker_script"

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

	// Generate moved block if the resource was renamed (singular → plural)
	if wasSingular {
		oldType, newType := m.GetResourceRename()
		from := oldType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block, movedBlock},
			RemoveOriginal: true,
		}, nil
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// transformBindings converts v4 binding blocks and dispatch_namespace attribute to v5 unified bindings list
func (m *V4ToV5Migrator) transformBindings(body *hclwrite.Body) {
	var bindingObjects []string

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
		if v5BindingType, ok := bindingTypeMap[block.Type()]; ok {
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
	if len(bindingObjects) > 0 {
		bindingsValue := "[\n  " + joinBindings(bindingObjects) + "\n]"
		// Use SetAttributeFromExpressionString to set the bindings array
		tfhcl.SetAttributeFromExpressionString(body, "bindings", bindingsValue)
	}

	// Remove all v4 binding blocks
	for blockType := range bindingTypeMap {
		removeBlocks(body, blockType)
	}
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
		// queue_binding: binding → name, queue → queue_name
		if attr := blockBody.GetAttribute("binding"); attr != nil {
			attrs = append(attrs, "name = "+exprToString(attr.Expr()))
		}
		if attr := blockBody.GetAttribute("queue"); attr != nil {
			attrs = append(attrs, "queue_name = "+exprToString(attr.Expr()))
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

// TransformState is a no-op for workers_script migration.
// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState).
// The moved block generated in TransformConfig triggers the provider's migration logic.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
