package spectrum_application

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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

// TransformState transforms the JSON state from v4 to v5.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Get attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		// Even for invalid instances, set schema_version
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// 1. Convert dns array to object
	result = m.convertArrayToObject(result, attrs, "dns")

	// 2. Convert origin_dns array to object
	result = m.convertArrayToObject(result, attrs, "origin_dns")

	// 3. Convert edge_ips array to object
	result = m.convertArrayToObject(result, attrs, "edge_ips")

	// 4. Convert origin_port_range to origin_port string (wrapped in DynamicAttribute format)
	result = m.convertOriginPortRangeState(result, attrs)

	// 5. Wrap any existing origin_port integer in DynamicAttribute format
	result = m.wrapOriginPortDynamic(result, attrs)

	// 6. Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// convertArrayToObject converts a MaxItems:1 array field to an object
func (m *V4ToV5Migrator) convertArrayToObject(result string, attrs gjson.Result, fieldName string) string {
	field := attrs.Get(fieldName)
	if field.Exists() && field.IsArray() {
		array := field.Array()
		if len(array) > 0 {
			// Take first element and set as object
			result, _ = sjson.Set(result, "attributes."+fieldName, array[0].Value())
		} else {
			// Empty array - delete it
			result, _ = sjson.Delete(result, "attributes."+fieldName)
		}
	}
	return result
}

// convertOriginPortRangeState converts origin_port_range array to origin_port string
func (m *V4ToV5Migrator) convertOriginPortRangeState(result string, attrs gjson.Result) string {
	portRangeField := attrs.Get("origin_port_range")
	if portRangeField.Exists() && portRangeField.IsArray() {
		array := portRangeField.Array()
		if len(array) > 0 {
			start := array[0].Get("start").Int()
			end := array[0].Get("end").Int()
			// Create string "start-end"
			portRange := fmt.Sprintf("%d-%d", start, end)

			// For DynamicAttribute, wrap value with type information
			// Format: {"type": "string", "value": "start-end"}
			dynamicValue := map[string]interface{}{
				"type":  "string",
				"value": portRange,
			}
			result, _ = sjson.Set(result, "attributes.origin_port", dynamicValue)
		}
		// Remove origin_port_range
		result, _ = sjson.Delete(result, "attributes.origin_port_range")
	}
	return result
}

// wrapOriginPortDynamic wraps any existing origin_port value in DynamicAttribute format
// This handles cases where origin_port was already an integer in v4
func (m *V4ToV5Migrator) wrapOriginPortDynamic(result string, attrs gjson.Result) string {
	originPortField := attrs.Get("origin_port")

	// Only wrap if origin_port exists and is not already wrapped
	// (convertOriginPortRangeState may have already set it)
	if !originPortField.Exists() {
		return result
	}

	// Check if it's already wrapped (has "type" field)
	if originPortField.IsObject() && originPortField.Get("type").Exists() {
		// Already wrapped, skip
		return result
	}

	// Wrap the value based on its type
	var dynamicValue map[string]interface{}

	if originPortField.Type == gjson.Number {
		// Integer port - wrap as number type
		dynamicValue = map[string]interface{}{
			"type":  "number",
			"value": originPortField.Int(),
		}
	} else if originPortField.Type == gjson.String {
		// String port (shouldn't happen in v4, but handle it)
		dynamicValue = map[string]interface{}{
			"type":  "string",
			"value": originPortField.String(),
		}
	} else {
		// Unexpected type, leave as-is
		return result
	}

	result, _ = sjson.Set(result, "attributes.origin_port", dynamicValue)
	return result
}
