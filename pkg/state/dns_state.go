package state

import (
	"fmt"
	"strconv"

	"github.com/cloudflare/tf-migrate/internal/transform/state"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// DNSRecordSchemaVersion is the schema version for cloudflare_dns_record in v5.
// This must match the Version field in the provider's resource schema.
const DNSRecordSchemaVersion = 4

// TransformDNSRecordState transforms cloudflare_record (v4) state to cloudflare_dns_record (v5).
// It handles two JSON formats:
//   - With "attributes" wrapper: {"attributes": {"zone_id": "...", ...}, "schema_version": ...}
//   - Direct attributes (raw state): {"zone_id": "...", "name": "...", ...}
func TransformDNSRecordState(stateJSON gjson.Result) (string, error) {
	if !stateJSON.Exists() {
		return stateJSON.String(), nil
	}

	// Detect format: check if we have an "attributes" wrapper or direct attributes
	hasAttributesWrapper := stateJSON.Get("attributes").Exists()

	if hasAttributesWrapper {
		// Format: {"attributes": {...}, "schema_version": ...}
		return transformWithAttributesWrapper(stateJSON)
	}

	// Format: direct attributes {"zone_id": "...", "name": "...", ...}
	return transformDirectAttributes(stateJSON)
}

// transformWithAttributesWrapper handles state JSON with "attributes" wrapper
func transformWithAttributesWrapper(stateJSON gjson.Result) (string, error) {
	result := stateJSON.String()
	attrs := stateJSON.Get("attributes")

	if !attrs.Get("name").Exists() || !attrs.Get("type").Exists() || !attrs.Get("zone_id").Exists() {
		// Even for invalid/incomplete instances, we need to set schema_version for v5
		// Delete first to handle duplicate keys in input JSON
		result, _ = sjson.Delete(result, "schema_version")
		result, _ = sjson.Set(result, "schema_version", DNSRecordSchemaVersion)
		return result, nil
	}

	// Transform the single instance
	result = transformSingleDNSInstance(result, stateJSON)

	// Ensure schema_version is set for v5
	// Delete first to handle duplicate keys in input JSON
	result, _ = sjson.Delete(result, "schema_version")
	result, _ = sjson.Set(result, "schema_version", DNSRecordSchemaVersion)

	return result, nil
}

// transformDirectAttributes handles raw state JSON without "attributes" wrapper
func transformDirectAttributes(stateJSON gjson.Result) (string, error) {
	// Check if this looks like a valid DNS record
	if !stateJSON.Get("name").Exists() || !stateJSON.Get("type").Exists() || !stateJSON.Get("zone_id").Exists() {
		return stateJSON.String(), nil
	}

	// Wrap in "attributes" temporarily for transformation
	wrapped := `{"attributes":` + stateJSON.String() + `}`
	wrappedJSON := gjson.Parse(wrapped)

	// Transform using the wrapper path
	result := transformSingleDNSInstance(wrapped, wrappedJSON)

	// Unwrap - extract just the attributes
	unwrapped := gjson.Get(result, "attributes").String()

	return unwrapped, nil
}

func transformSingleDNSInstance(result string, instance gjson.Result) string {
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
		// Check if content has an actual value (not just exists as null)
		hasContent := contentField.Exists() && contentField.Value() != nil
		if valueField.Exists() && !hasContent {
			// Value exists and content is missing or null - use value as content
			result, _ = sjson.Set(result, "attributes.content", valueField.Value())
			result, _ = sjson.Delete(result, "attributes.value")
		} else if valueField.Exists() && hasContent {
			// Both exist with real values - keep content, remove value
			result, _ = sjson.Delete(result, "attributes.value")
		}
	} else {
		// For records that use data field, remove both value and content if they exist
		result, _ = sjson.Delete(result, "attributes.value")
		result, _ = sjson.Delete(result, "attributes.content")
	}

	// Ensure TTL is present
	result = state.EnsureField(result, "attributes", attrs, "ttl", 1.0)

	// Handle meta field transformation
	// meta is a jsontypes.NormalizedType in v5, so it needs to be a JSON string
	//
	// The v4 state can have:
	// 1. A "metadata" field (which we rename to "meta")
	// 2. An existing "meta" field from the API (as a JSON object)
	//
	// In both cases, we need to ensure the final "meta" is a JSON string, not an object
	metadata := attrs.Get("metadata")
	existingMeta := attrs.Get("meta")

	if metadata.Exists() && metadata.Value() != nil {
		// Convert the metadata object to a JSON string
		result, _ = sjson.Set(result, "attributes.meta", metadata.Raw)
	} else if existingMeta.Exists() && existingMeta.Value() != nil {
		// Existing meta field - check if it's an object that needs conversion
		if existingMeta.IsObject() {
			// Convert the object to a JSON string
			result, _ = sjson.Set(result, "attributes.meta", existingMeta.Raw)
		}
		// If it's already a string, leave it as-is
	}

	// Remove deprecated fields
	result = state.RemoveFields(result, "attributes", attrs,
		"hostname", "allow_overwrite", "timeouts", "metadata", "proxiable")

	// Handle data field transformation
	result = transformDataFieldForInstance(result, instance, recordType)

	// Convert priority field to float64 if it exists at root level
	rootPriority := instance.Get("attributes.priority")
	if rootPriority.Exists() && rootPriority.Type == gjson.Number {
		result, _ = sjson.Set(result, "attributes.priority", state.ConvertToFloat64(rootPriority))
	}

	return result
}

func transformDataFieldForInstance(result string, instance gjson.Result, recordType string) string {
	// Check if data field exists and is an array
	data := instance.Get("attributes.data")
	isDataArray := data.IsArray()

	// Simple record types that don't use data field
	// But MX records with data arrays should be processed as complex types
	if isSimpleRecordType(recordType) && (!isDataArray || recordType != "MX") {
		if data.Exists() {
			result, _ = sjson.Delete(result, "attributes.data")
		}
		return result
	}

	// Setup transformation options for complex record types
	options := state.ArrayToObjectOptions{
		SkipFields: []string{"name", "proto"},
		FieldTransforms: map[string]func(gjson.Result) interface{}{
			"flags":          transformFlagsValue,
			"algorithm":      transformNumericValue,
			"key_tag":        transformNumericValue,
			"type":           transformNumericValue,
			"usage":          transformNumericValue,
			"selector":       transformNumericValue,
			"matching_type":  transformNumericValue,
			"weight":         transformNumericValue,
			"priority":       transformNumericValue,
			"port":           transformNumericValue,
			"protocol":       transformNumericValue,
			"digest_type":    transformNumericValue,
			"order":          transformNumericValue,
			"preference":     transformNumericValue,
			"altitude":       transformNumericValue,
			"lat_degrees":    transformNumericValue,
			"lat_minutes":    transformNumericValue,
			"lat_seconds":    transformNumericValue,
			"long_degrees":   transformNumericValue,
			"long_minutes":   transformNumericValue,
			"long_seconds":   transformNumericValue,
			"precision_horz": transformNumericValue,
			"precision_vert": transformNumericValue,
			"size":           transformNumericValue,
		},
		RenameFields:  map[string]string{},
		DefaultFields: map[string]interface{}{},
	}

	// CAA-specific transformations - rename content to value
	if recordType == "CAA" {
		options.RenameFields["content"] = "value"
		options.DefaultFields["flags"] = nil
	} else {
		// For non-CAA records, skip the content field (it's not in v5 data schema)
		options.SkipFields = append(options.SkipFields, "content")
	}

	// For MX and URI, skip priority field in data as it will be hoisted
	// SRV keeps priority in the data field
	if recordType == "MX" || recordType == "URI" {
		options.SkipFields = append(options.SkipFields, "priority")
	}

	// Transform the data field
	result = state.TransformFieldArrayToObject(result, "attributes", instance.Get("attributes"), "data", options)

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
func transformNumericValue(value gjson.Result) interface{} {
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

// transformFlagsValue transforms the flags value to the correct DynamicAttribute format
// In v5, flags is a NormalizedDynamicType which requires {"type": "...", "value": ...} structure
func transformFlagsValue(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.Number:
		// Preserve numeric type - use Float() to get actual number value
		return map[string]interface{}{
			"type":  "number",
			"value": value.Float(),
		}
	case gjson.String:
		strVal := value.String()
		if strVal == "" {
			return nil
		}

		// Try to parse numeric strings as numbers (flags is typically 0 or 128)
		// This ensures the type matches what the API returns (number, not string)
		if f, err := strconv.ParseFloat(strVal, 64); err == nil {
			return map[string]interface{}{
				"type":  "number",
				"value": f,
			}
		}

		// Keep non-numeric strings as strings
		return map[string]interface{}{
			"type":  "string",
			"value": strVal,
		}
	case gjson.Null:
		return nil
	default:
		return nil
	}
}

// isSimpleRecordType checks if a record type is simple (doesn't use data field)
func isSimpleRecordType(recordType string) bool {
	simpleTypes := map[string]bool{
		"A": true, "AAAA": true, "CNAME": true, "MX": true,
		"NS": true, "PTR": true, "TXT": true, "OPENPGPKEY": true,
	}
	return simpleTypes[recordType]
}
