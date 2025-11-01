package dns_record

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"

	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	"github.com/cloudflare/tf-migrate/internal/transform/structural"
)

// V4ToV5Migrator handles migration of DNS record resources from v4 to v5
type V4ToV5Migrator struct {
	typeUpdater *structural.ResourceTypeUpdater
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		typeUpdater: &structural.ResourceTypeUpdater{
			OldType: "cloudflare_record",
			NewType: "cloudflare_dns_record",
		},
	}
	internal.RegisterMigrator("cloudflare_record", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_dns_record"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_record"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for DNS records
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename cloudflare_record to cloudflare_dns_record
	tfhcl.RenameResourceType(block, "cloudflare_record", "cloudflare_dns_record")

	body := block.Body()

	// Ensure TTL is present for v5 (required field)
	tfhcl.EnsureAttribute(body, "ttl", 1)

	// Get the record type
	typeAttr := body.GetAttribute("type")
	recordType := ""
	if typeAttr != nil {
		// Extract the record type value
		recordType = tfhcl.ExtractStringFromAttribute(typeAttr)
	}

	// Handle simple record types or records without type
	// When type is missing, we still need to rename value to content
	if recordType == "" || m.isSimpleRecordType(recordType) {
		// Rename value to content for simple record types
		tfhcl.RenameAttribute(body, "value", "content")
	}

	// Remove deprecated attributes
	tfhcl.RemoveAttributes(body, "allow_overwrite", "hostname")

	// Process data blocks
	m.processDataBlocks(block, recordType)

	// Process data attribute for CAA records
	m.processDataAttribute(block, recordType)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// processDataBlocks converts data blocks to attribute format
func (m *V4ToV5Migrator) processDataBlocks(block *hclwrite.Block, recordType string) {
	body := block.Body()

	// For SRV, MX, and URI records, hoist priority from data block to root
	// Note: SRV will keep priority in BOTH places (root and data)
	if recordType == "SRV" || recordType == "MX" || recordType == "URI" {
		tfhcl.HoistAttributeFromBlock(body, "data", "priority")
	}

	// Convert data blocks to attribute, with preprocessing for CAA records
	tfhcl.ConvertBlocksToAttribute(body, "data", "data", func(dataBlock *hclwrite.Block) {
		if recordType == "CAA" {
			// Rename content to value in CAA data blocks
			tfhcl.RenameAttribute(dataBlock.Body(), "content", "value")
			// In v5, flags format is preserved as-is (string stays string, number stays number)
		}
		// Remove priority from data block for MX/URI since it's hoisted to root only
		// SRV keeps priority in BOTH the data block AND root
		if recordType == "MX" || recordType == "URI" {
			dataBlock.Body().RemoveAttribute("priority")
		}
		// Note: For SRV, we do NOT remove priority from data block
	})
}

// processDataAttribute handles data as an attribute (not a block) for CAA records
func (m *V4ToV5Migrator) processDataAttribute(block *hclwrite.Block, recordType string) {
	dataAttr := block.Body().GetAttribute("data")
	if dataAttr != nil && recordType == "CAA" {
		expr := dataAttr.Expr()
		tokens := expr.BuildTokens(nil)

		newTokens := make(hclwrite.Tokens, 0, len(tokens))
		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			// Check if this is "content" identifier inside data - rename to "value"
			if token.Type == hclsyntax.TokenIdent && string(token.Bytes) == "content" {
				if i+1 < len(tokens) && (tokens[i+1].Type == hclsyntax.TokenEqual ||
					(string(tokens[i+1].Bytes) == " " && i+2 < len(tokens) && tokens[i+2].Type == hclsyntax.TokenEqual)) {
					valueToken := &hclwrite.Token{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte("value"),
					}
					newTokens = append(newTokens, valueToken)
				} else {
					newTokens = append(newTokens, token)
				}
			} else {
				newTokens = append(newTokens, token)
			}
		}

		block.Body().SetAttributeRaw("data", newTokens)
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// This function can receive either:
	// 1. A full state document (in unit tests)
	// 2. A single resource instance (in actual migration framework)
	// We need to handle both cases

	result := stateJSON.String()

	// Check if this is a full state document (has "resources" key) or a single instance
	if stateJSON.Get("resources").Exists() {
		// Full state document - transform all resources
		return m.transformFullState(result, stateJSON)
	}

	// Single instance - check if it's a valid DNS record instance
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")
	if !attrs.Get("name").Exists() || !attrs.Get("type").Exists() || !attrs.Get("zone_id").Exists() {
		// Even for invalid/incomplete instances, we need to set schema_version for v5
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Transform the single instance
	result = m.transformSingleDNSInstance(result, stateJSON)
	
	// Ensure schema_version is set to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// transformFullState handles transformation of a full state document
func (m *V4ToV5Migrator) transformFullState(result string, stateJSON gjson.Result) (string, error) {
	// Process all resources in the state
	resources := stateJSON.Get("resources")
	if !resources.Exists() {
		return result, nil
	}

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()

		// Check if this is a DNS record resource we need to migrate
		if !m.CanHandle(resourceType) {
			return true // continue
		}

		// Rename cloudflare_record to cloudflare_dns_record
		if resourceType == "cloudflare_record" {
			resourcePath := "resources." + key.String() + ".type"
			result, _ = sjson.Set(result, resourcePath, "cloudflare_dns_record")
		}

		// Process each instance
		instances := resource.Get("instances")
		instances.ForEach(func(instKey, instance gjson.Result) bool {
			instPath := "resources." + key.String() + ".instances." + instKey.String()

			// Transform the instance attributes in place
			attrs := instance.Get("attributes")
			if attrs.Exists() && attrs.Get("name").Exists() &&
				attrs.Get("type").Exists() && attrs.Get("zone_id").Exists() {
				// Get the instance JSON string
				instJSON := instance.String()
				// Transform it
				transformedInst := m.transformSingleDNSInstance(instJSON, instance)
				
				// Ensure schema_version is set to 0 for v5
				transformedInst, _ = sjson.Set(transformedInst, "schema_version", 0)
				
				// Update the result with the transformed instance
				// Using the raw JSON string directly to preserve all fields including schema_version
				result, _ = sjson.SetRaw(result, instPath, transformedInst)
			} else {
				// Even if attributes don't exist or are incomplete, update schema_version
				schemaPath := instPath + ".schema_version"
				result, _ = sjson.Set(result, schemaPath, 0)
			}
			return true
		})

		return true
	})

	return result, nil
}

// transformSingleDNSInstance transforms a single DNS record instance
func (m *V4ToV5Migrator) transformSingleDNSInstance(result string, instance gjson.Result) string {
	attrs := instance.Get("attributes")

	// Clean up meta field - remove if empty or invalid
	result = state.CleanupEmptyField(result, "attributes.meta", instance.Get("attributes.meta"))

	// Clean up settings field - remove if all values are null
	result = state.RemoveObjectIfAllNull(result, "attributes.settings",
		instance.Get("attributes.settings"),
		[]string{"flatten_cname", "ipv4_only", "ipv6_only"})

	// Ensure timestamp fields exist
	result = state.EnsureTimestamps(result, "attributes", attrs, "2024-01-01T00:00:00Z")

	// Handle field renames: value -> content
	// But only for record types that use content (not data)
	recordType := instance.Get("attributes.type").String()
	valueField := attrs.Get("value")
	contentField := attrs.Get("content")
	
	// Records that use data field don't have content
	usesDataField := recordType == "SRV" || recordType == "CAA" || 
		recordType == "CERT" || recordType == "DNSKEY" || recordType == "DS" ||
		recordType == "LOC" || recordType == "NAPTR" || recordType == "SMIMEA" ||
		recordType == "SSHFP" || recordType == "SVCB" || recordType == "HTTPS" ||
		recordType == "TLSA" || recordType == "URI"

	if !usesDataField {
		if valueField.Exists() && !contentField.Exists() {
			// Only value exists - rename it to content
			result, _ = sjson.Set(result, "attributes.content", valueField.Value())
			result, _ = sjson.Delete(result, "attributes.value")
		} else if valueField.Exists() && contentField.Exists() {
			// Both exist - keep content, remove value
			result, _ = sjson.Delete(result, "attributes.value")
		}
	} else {
		// For records that use data field, remove both value and content if they exist
		result, _ = sjson.Delete(result, "attributes.value")
		result, _ = sjson.Delete(result, "attributes.content")
	}

	// Ensure TTL is present
	result = state.EnsureField(result, "attributes", attrs, "ttl", 1.0)

	// Remove deprecated fields
	result = state.RemoveFields(result, "attributes", attrs,
		"hostname", "allow_overwrite", "timeouts")

	// Handle data field transformation
	result = m.transformDataFieldForInstance(result, instance, recordType)

	// Convert priority field to float64 if it exists at root level
	rootPriority := instance.Get("attributes.priority")
	if rootPriority.Exists() && rootPriority.Type == gjson.Number {
		result, _ = sjson.Set(result, "attributes.priority", state.ConvertToFloat64(rootPriority))
	}

	return result
}

// transformDataFieldForInstance handles the transformation of the data field for a single instance
func (m *V4ToV5Migrator) transformDataFieldForInstance(result string, instance gjson.Result, recordType string) string {
	// Check if data field exists and is an array
	data := instance.Get("attributes.data")
	isDataArray := data.IsArray()

	// Simple record types that don't use data field
	// But MX records with data arrays should be processed as complex types
	if m.isSimpleRecordType(recordType) && (!isDataArray || recordType != "MX") {
		if data.Exists() {
			result, _ = sjson.Delete(result, "attributes.data")
		}
		return result
	}

	// Setup transformation options for complex record types
	options := state.ArrayToObjectOptions{
		SkipFields: []string{"name", "proto"},
		FieldTransforms: map[string]func(gjson.Result) interface{}{
			"flags":          m.transformFlagsValue,
			"algorithm":      m.transformNumericValue,
			"key_tag":        m.transformNumericValue,
			"type":           m.transformNumericValue,
			"usage":          m.transformNumericValue,
			"selector":       m.transformNumericValue,
			"matching_type":  m.transformNumericValue,
			"weight":         m.transformNumericValue,
			"priority":       m.transformNumericValue,
			"port":           m.transformNumericValue,
			"protocol":       m.transformNumericValue,
			"digest_type":    m.transformNumericValue,
			"order":          m.transformNumericValue,
			"preference":     m.transformNumericValue,
			"altitude":       m.transformNumericValue,
			"lat_degrees":    m.transformNumericValue,
			"lat_minutes":    m.transformNumericValue,
			"lat_seconds":    m.transformNumericValue,
			"long_degrees":   m.transformNumericValue,
			"long_minutes":   m.transformNumericValue,
			"long_seconds":   m.transformNumericValue,
			"precision_horz": m.transformNumericValue,
			"precision_vert": m.transformNumericValue,
			"size":           m.transformNumericValue,
		},
		RenameFields:  map[string]string{},
		DefaultFields: map[string]interface{}{},
	}

	// CAA-specific transformations
	if recordType == "CAA" {
		options.RenameFields["content"] = "value"
		options.DefaultFields["flags"] = nil
	}

	// For MX and URI, skip priority field in data as it will be hoisted
	// SRV keeps priority in the data field
	if recordType == "MX" || recordType == "URI" {
		options.SkipFields = append(options.SkipFields, "priority")
	}

	// Transform the data field
	result = state.TransformDataFieldArrayToObject(result, "attributes", instance.Get("attributes"), recordType, options)

	// Generate content field for CAA records
	if recordType == "CAA" {
		dataArray := instance.Get("attributes.data")
		if dataArray.IsArray() {
			array := dataArray.Array()
			if len(array) > 0 {
				flags := array[0].Get("flags")
				tag := array[0].Get("tag")
				value := array[0].Get("content")

				// Format the content field
				flagsStr := "0"
				if flags.Exists() {
					switch flags.Type {
					case gjson.Number:
						flagsStr = flags.Raw
					case gjson.String:
						if flags.String() != "" {
							flagsStr = flags.String()
						}
					}
				}

				if tag.Exists() && value.Exists() {
					content := fmt.Sprintf("%s %s %s", flagsStr, tag.String(), value.String())
					result, _ = sjson.Set(result, "attributes.content", content)
				}
			}
		}
	}

	// For SRV, MX and URI records, ensure priority is at root level
	if recordType == "SRV" || recordType == "MX" || recordType == "URI" {
		// Check original instance for priority (before transformation)
		originalPriority := instance.Get("attributes.priority")
		
		if originalPriority.Exists() {
			// Preserve the original priority at root level
			result, _ = sjson.Set(result, "attributes.priority", originalPriority.Float())
		} else {
			// If not at root in original, check data array
			dataArray := instance.Get("attributes.data")
			if dataArray.IsArray() {
				array := dataArray.Array()
				if len(array) > 0 {
					priority := array[0].Get("priority")
					if priority.Exists() {
						// Set priority at root level for v5 compatibility
						result, _ = sjson.Set(result, "attributes.priority", priority.Float())
					}
				}
			}
		}
		
		// Generate content field for MX and URI records (not SRV)
		if recordType == "MX" || recordType == "URI" {
			dataArray := instance.Get("attributes.data")
			if dataArray.IsArray() {
				array := dataArray.Array()
				if len(array) > 0 {
					priority := array[0].Get("priority")
					
					if recordType == "MX" {
						target := array[0].Get("target")
						if priority.Exists() && target.Exists() {
							content := fmt.Sprintf("%v %s", priority.Value(), target.String())
							result, _ = sjson.Set(result, "attributes.content", content)
						}
					} else if recordType == "URI" {
						weight := array[0].Get("weight")
						target := array[0].Get("target")
						if priority.Exists() && weight.Exists() && target.Exists() {
							content := fmt.Sprintf("%v %v %s", priority.Value(), weight.Value(), target.String())
							result, _ = sjson.Set(result, "attributes.content", content)
						}
					}
				}
			}
		}
	}

	return result
}

// transformNumericValue converts integer values to float64 for v5 compatibility
func (m *V4ToV5Migrator) transformNumericValue(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.Number:
		// Convert to float64
		return value.Float()
	case gjson.String:
		// Try to parse as number
		if f, err := strconv.ParseFloat(value.String(), 64); err == nil {
			return f
		}
		return value.String()
	case gjson.Null:
		return nil
	default:
		return value.Value()
	}
}

// transformFlagsValue transforms the flags value to the correct format
func (m *V4ToV5Migrator) transformFlagsValue(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.Number:
		return map[string]interface{}{
			"value": json.Number(value.Raw),
			"type":  "number",
		}
	case gjson.String:
		if _, err := strconv.ParseFloat(value.String(), 64); err == nil {
			return map[string]interface{}{
				"value": json.Number(value.String()),
				"type":  "number",
			}
		} else if value.String() == "" {
			return nil
		} else {
			return map[string]interface{}{
				"value": value.String(),
				"type":  "string",
			}
		}
	case gjson.Null:
		return nil
	default:
		return nil
	}
}

// isSimpleRecordType checks if a record type is simple (doesn't use data field)
func (m *V4ToV5Migrator) isSimpleRecordType(recordType string) bool {
	simpleTypes := map[string]bool{
		"A": true, "AAAA": true, "CNAME": true, "MX": true,
		"NS": true, "PTR": true, "TXT": true, "OPENPGPKEY": true,
	}
	return simpleTypes[recordType]
}
