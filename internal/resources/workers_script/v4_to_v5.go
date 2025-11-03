package workers_script

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with both old and new names
	internal.RegisterMigrator("cloudflare_worker_script", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_workers_script", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_workers_script"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_worker_script" ||
		resourceType == "cloudflare_workers_script"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Rename resource type from cloudflare_worker_script to cloudflare_workers_script
	content = strings.ReplaceAll(content,
		`resource "cloudflare_worker_script"`,
		`resource "cloudflare_workers_script"`)

	return content
}

// v4BindingToV5Type maps v4 binding block names to v5 binding types
var v4BindingToV5Type = map[string]string{
	"plain_text_binding":        "plain_text",
	"kv_namespace_binding":      "kv_namespace",
	"secret_text_binding":       "secret_text",
	"r2_bucket_binding":         "r2_bucket",
	"queue_binding":             "queue",
	"d1_database_binding":       "d1",
	"analytics_engine_binding":  "analytics_engine",
	"service_binding":           "service",
	"hyperdrive_config_binding": "hyperdrive",
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Rename name attribute to script_name
	nameAttr := body.GetAttribute("name")
	if nameAttr != nil {
		tokens := nameAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("script_name", tokens)
		body.RemoveAttribute("name")
	}

	// Transform binding blocks to bindings list
	m.transformBindings(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) transformBindings(body *hclwrite.Body) {
	var bindingObjects []string
	var blocksToRemove []*hclwrite.Block

	// Scan all blocks to find binding blocks
	for _, childBlock := range body.Blocks() {
		blockType := childBlock.Type()

		if bindingType, isBindingBlock := v4BindingToV5Type[blockType]; isBindingBlock {
			// Convert this block to a binding object
			bindingStr := m.convertBindingBlockToObject(childBlock, bindingType)
			if bindingStr != "" {
				bindingObjects = append(bindingObjects, bindingStr)
			}
			blocksToRemove = append(blocksToRemove, childBlock)
		} else if blockType == "webassembly_binding" {
			// webassembly_binding is not supported in v5 - add warning
			body.AppendNewline()
			body.AppendUnstructuredTokens([]*hclwrite.Token{
				{Type: hclsyntax.TokenComment, Bytes: []byte("# MIGRATION WARNING: webassembly_binding is not supported in v5.\n")},
				{Type: hclsyntax.TokenComment, Bytes: []byte("# WebAssembly modules must be bundled into the script content instead.\n")},
			})
			blocksToRemove = append(blocksToRemove, childBlock)
		}
	}

	// Remove the old binding blocks
	for _, blockToRemove := range blocksToRemove {
		body.RemoveBlock(blockToRemove)
	}

	// If we found any bindings, create the bindings attribute
	if len(bindingObjects) > 0 {
		bindingsHCL := fmt.Sprintf("bindings = [%s]", strings.Join(bindingObjects, ",\n  "))

		// Parse and set the new attribute
		file, diags := hclwrite.ParseConfig([]byte(bindingsHCL), "", hcl.InitialPos)
		if !diags.HasErrors() {
			attr := file.Body().GetAttribute("bindings")
			if attr != nil {
				body.SetAttributeRaw("bindings", attr.Expr().BuildTokens(nil))
			}
		}
	}
}

func (m *V4ToV5Migrator) convertBindingBlockToObject(block *hclwrite.Block, bindingType string) string {
	if block == nil {
		return ""
	}

	// Get all attributes from the block
	attrs := block.Body().Attributes()
	if len(attrs) == 0 {
		// Just type field
		return fmt.Sprintf("{\n    type = \"%s\"\n  }", bindingType)
	}

	// Build attribute strings
	var attrStrings []string

	// Add type first
	attrStrings = append(attrStrings, fmt.Sprintf("    type = \"%s\"", bindingType))

	// Get attribute names in sorted order
	var attrNames []string
	for name := range attrs {
		attrNames = append(attrNames, name)
	}
	sort.Strings(attrNames)

	for _, attrName := range attrNames {
		attr := attrs[attrName]
		// Apply attribute renaming for this binding type
		finalAttrName := m.renameBindingAttribute(attrName, block.Type())

		// Get the raw tokens for the expression
		tokens := attr.Expr().BuildTokens(nil)
		var exprStr string
		for _, token := range tokens {
			exprStr += string(token.Bytes)
		}

		attrStrings = append(attrStrings, fmt.Sprintf("    %s = %s", finalAttrName, exprStr))
	}

	return fmt.Sprintf("{\n%s\n  }", strings.Join(attrStrings, "\n"))
}

func (m *V4ToV5Migrator) renameBindingAttribute(attrName, bindingBlockType string) string {
	switch bindingBlockType {
	case "d1_database_binding":
		if attrName == "database_id" {
			return "id"
		}
	case "hyperdrive_config_binding":
		if attrName == "binding" {
			return "name"
		}
	case "queue_binding":
		if attrName == "binding" {
			return "name"
		}
		if attrName == "queue" {
			return "queue_name"
		}
	}
	return attrName
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	// Transform name to script_name
	namePath := "attributes.name"
	scriptNamePath := "attributes.script_name"

	nameValue := stateJSON.Get(namePath)
	if nameValue.Exists() {
		result, _ = sjson.Set(result, scriptNamePath, nameValue.Value())
		result, _ = sjson.Delete(result, namePath)
	}

	// Transform bindings in state
	result = m.transformBindingsInState(result)

	// Remove tags attribute - not supported in v5
	tagsPath := "attributes.tags"
	tagsValue := gjson.Get(result, tagsPath)
	if tagsValue.Exists() {
		result, _ = sjson.Delete(result, tagsPath)
	}

	return result, nil
}

func (m *V4ToV5Migrator) transformBindingsInState(jsonStr string) string {
	result := jsonStr

	var bindings []interface{}

	// Check each v4 binding type and convert to v5 bindings format
	for bindingAttr, bindingType := range v4BindingToV5Type {
		bindingPath := "attributes." + bindingAttr
		bindingValue := gjson.Get(jsonStr, bindingPath)

		if bindingValue.Exists() {
			// Parse the binding data and add type field
			bindingData := bindingValue.Value()
			if bindingArray, ok := bindingData.([]interface{}); ok {
				// Handle array of bindings
				for _, binding := range bindingArray {
					if bindingMap, ok := binding.(map[string]interface{}); ok {
						bindingMap["type"] = bindingType
						m.renameBindingAttributesInState(bindingMap, bindingAttr)
						bindings = append(bindings, bindingMap)
					}
				}
			} else if bindingMap, ok := bindingData.(map[string]interface{}); ok {
				// Handle single binding
				bindingMap["type"] = bindingType
				m.renameBindingAttributesInState(bindingMap, bindingAttr)
				bindings = append(bindings, bindingMap)
			}

			// Remove the old binding attribute
			result, _ = sjson.Delete(result, bindingPath)
		}
	}

	// Handle webassembly_binding separately - remove without migration
	webassemblyPath := "attributes.webassembly_binding"
	webassemblyValue := gjson.Get(jsonStr, webassemblyPath)
	if webassemblyValue.Exists() {
		result, _ = sjson.Delete(result, webassemblyPath)
	}

	// If we found any bindings, add them to the state
	if len(bindings) > 0 {
		bindingsPath := "attributes.bindings"
		result, _ = sjson.Set(result, bindingsPath, bindings)
	}

	return result
}

func (m *V4ToV5Migrator) renameBindingAttributesInState(bindingMap map[string]interface{}, bindingType string) {
	switch bindingType {
	case "d1_database_binding":
		if databaseID, exists := bindingMap["database_id"]; exists {
			bindingMap["id"] = databaseID
			delete(bindingMap, "database_id")
		}
	case "hyperdrive_config_binding":
		if binding, exists := bindingMap["binding"]; exists {
			bindingMap["name"] = binding
			delete(bindingMap, "binding")
		}
	case "queue_binding":
		if binding, exists := bindingMap["binding"]; exists {
			bindingMap["name"] = binding
			delete(bindingMap, "binding")
		}
		if queue, exists := bindingMap["queue"]; exists {
			bindingMap["queue_name"] = queue
			delete(bindingMap, "queue")
		}
	}
}
