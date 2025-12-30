// Package state provides utilities for transforming Terraform state files
// during provider migrations. These utilities handle JSON manipulation
// for state file migrations between provider versions.
package state

import (
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// EnsureField ensures a field exists in the state with a default value.
// If the field doesn't exist, it's added with the defaultValue.
//
// Example - Adding required TTL field to DNS record state:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "type": "A",
//	        "content": "192.0.2.1"
//	      }
//	    }]
//	  }]
//	}
//
// After calling EnsureField(stateJSON, "resources.0.instances.0.attributes", instance, "ttl", 1):
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "type": "A",
//	        "content": "192.0.2.1",
//	        "ttl": 1
//	      }
//	    }]
//	  }]
//	}
func EnsureField(stateJSON string, path string, instance gjson.Result, field string, defaultValue interface{}) string {
	if !instance.Get(field).Exists() {
		result, _ := sjson.Set(stateJSON, path+"."+field, defaultValue)
		return result
	}
	return stateJSON
}

// RenameField renames a field in the state.
// If both old and new fields exist, the old field is removed (new takes precedence).
//
// Example - Renaming 'value' to 'content' in DNS record state:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "type": "A",
//	        "value": "192.0.2.1"  // Old field name
//	      }
//	    }]
//	  }]
//	}
//
// After calling RenameField(stateJSON, "resources.0.instances.0.attributes", instance, "value", "content"):
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "type": "A",
//	        "content": "192.0.2.1"  // New field name
//	      }
//	    }]
//	  }]
//	}
func RenameField(stateJSON string, path string, instance gjson.Result, oldName, newName string) string {
	oldField := instance.Get(oldName)
	newField := instance.Get(newName)

	if oldField.Exists() && !newField.Exists() {
		// Copy old to new
		stateJSON, _ = sjson.Set(stateJSON, path+"."+newName, oldField.Value())
		// Delete old
		stateJSON, _ = sjson.Delete(stateJSON, path+"."+oldName)
	} else if oldField.Exists() && newField.Exists() {
		// If both exist, just remove the old one (new takes precedence)
		stateJSON, _ = sjson.Delete(stateJSON, path+"."+oldName)
	}

	return stateJSON
}

// RemoveFields removes multiple fields from the state.
//
// Example - Removing deprecated fields from DNS record state:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "type": "A",
//	        "content": "192.0.2.1",
//	        "hostname": "test.example.com",  // Deprecated
//	        "allow_overwrite": true,         // Deprecated
//	        "timeouts": {}                    // Deprecated
//	      }
//	    }]
//	  }]
//	}
//
// After calling RemoveFields(stateJSON, path, instance, "hostname", "allow_overwrite", "timeouts"):
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "type": "A",
//	        "content": "192.0.2.1"
//	      }
//	    }]
//	  }]
//	}
func RemoveFields(stateJSON string, path string, instance gjson.Result, fields ...string) string {
	for _, field := range fields {
		if instance.Get(field).Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, path+"."+field)
		}
	}
	return stateJSON
}

// CleanupEmptyField removes a field if it's empty or invalid.
//
// Example - Cleaning up empty meta field:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "meta": {},  // Empty object
//	        "settings": []  // Empty array
//	      }
//	    }]
//	  }]
//	}
//
// After calling CleanupEmptyField(stateJSON, "resources.0.instances.0.attributes.meta", metaField):
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "settings": []  // meta removed, settings still present
//	      }
//	    }]
//	  }]
//	}
func CleanupEmptyField(stateJSON string, path string, field gjson.Result) string {
	if field.Exists() {
		// Check various empty conditions
		if field.String() == "{}" ||
			field.String() == "[]" ||
			field.String() == "" ||
			(field.IsObject() && len(field.Map()) == 0) ||
			(field.IsArray() && len(field.Array()) == 0) {
			stateJSON, _ = sjson.Delete(stateJSON, path)
		}
	}
	return stateJSON
}

// RemoveObjectIfAllNull removes an object if all specified fields are null.
//
// Example - Removing settings object with all null values:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "settings": {
//	          "flatten_cname": null,
//	          "ipv4_only": null,
//	          "ipv6_only": null
//	        }
//	      }
//	    }]
//	  }]
//	}
//
// After calling RemoveObjectIfAllNull(stateJSON, path+".settings", settingsObj, []string{"flatten_cname", "ipv4_only", "ipv6_only"}):
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test"
//	        // settings removed because all fields were null
//	      }
//	    }]
//	  }]
//	}
func RemoveObjectIfAllNull(stateJSON string, path string, obj gjson.Result, fields []string) string {
	if !obj.Exists() {
		return stateJSON
	}

	allNull := true
	for _, field := range fields {
		val := obj.Get(field)
		if val.Exists() && val.Type != gjson.Null && val.Value() != nil {
			allNull = false
			break
		}
	}

	if allNull {
		stateJSON, _ = sjson.Delete(stateJSON, path)
	}
	return stateJSON
}

// EnsureTimestamps ensures created_on and modified_on fields exist with defaults.
// If created_on exists, modified_on defaults to the same value.
//
// Example - Adding timestamps to DNS record:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "type": "A",
//	        "content": "192.0.2.1"
//	      }
//	    }]
//	  }]
//	}
//
// After calling EnsureTimestamps(stateJSON, path, instance, "2024-01-01T00:00:00Z"):
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "name": "test",
//	        "type": "A",
//	        "content": "192.0.2.1",
//	        "created_on": "2024-01-01T00:00:00Z",
//	        "modified_on": "2024-01-01T00:00:00Z"
//	      }
//	    }]
//	  }]
//	}
func EnsureTimestamps(stateJSON string, path string, instance gjson.Result, defaultTime string) string {
	createdOn := instance.Get("created_on")
	if !createdOn.Exists() {
		stateJSON, _ = sjson.Set(stateJSON, path+".created_on", defaultTime)
	}

	modifiedOn := instance.Get("modified_on")
	if !modifiedOn.Exists() {
		if createdOn.Exists() {
			stateJSON, _ = sjson.Set(stateJSON, path+".modified_on", createdOn.String())
		} else {
			stateJSON, _ = sjson.Set(stateJSON, path+".modified_on", defaultTime)
		}
	}

	return stateJSON
}

// ConvertMaxItemsOneArrayToObject converts a MaxItems:1 array field to an object.
// This is commonly used when a TypeList with MaxItems:1 in v4 becomes a SingleNestedAttribute in v5.
// Empty arrays are deleted from the state.
//
// Example - Converting dns field from array to object:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "protocol": "tcp/22",
//	        "dns": [
//	          {
//	            "type": "CNAME",
//	            "name": "test.example.com"
//	          }
//	        ]
//	      }
//	    }]
//	  }]
//	}
//
// After calling ConvertMaxItemsOneArrayToObject(stateJSON, path, instance, "dns"):
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "protocol": "tcp/22",
//	        "dns": {
//	          "type": "CNAME",
//	          "name": "test.example.com"
//	        }
//	      }
//	    }]
//	  }]
//	}
//
// If the array is empty, the field is deleted:
//
//	"dns": []  →  field removed entirely
func ConvertMaxItemsOneArrayToObject(stateJSON string, path string, instance gjson.Result, fieldName string) string {
	field := instance.Get(fieldName)
	if field.Exists() && field.IsArray() {
		array := field.Array()
		if len(array) > 0 {
			// Take first element and set as object
			stateJSON, _ = sjson.Set(stateJSON, path+"."+fieldName, array[0].Value())
		} else {
			// Empty array - delete it
			stateJSON, _ = sjson.Delete(stateJSON, path+"."+fieldName)
		}
	}
	return stateJSON
}

// ConvertGjsonValue converts a gjson value to appropriate Go type
func ConvertGjsonValue(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.Number:
		// Check if it's an integer or float
		if value.Float() == float64(int64(value.Float())) {
			return int64(value.Float())
		}
		return value.Float()
	case gjson.String:
		return value.String()
	case gjson.True:
		return true
	case gjson.False:
		return false
	case gjson.Null:
		return nil
	default:
		// For arrays and objects, return the raw value
		return value.Value()
	}
}

// ConvertGjsonToJSON converts gjson value preserving number types
func ConvertGjsonToJSON(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.Number:
		// Preserve as json.Number to maintain exact numeric representation
		return json.Number(value.Raw)
	case gjson.String:
		return value.String()
	case gjson.True:
		return true
	case gjson.False:
		return false
	case gjson.Null:
		return nil
	default:
		return value.Value()
	}
}

// IsEmptyValue checks if a gjson.Result value is considered "empty" (default/zero)
func IsEmptyValue(value gjson.Result) bool {
	if !value.Exists() {
		return true
	}

	switch value.Type {
	case gjson.Null:
		return true
	case gjson.False:
		return true
	case gjson.Number:
		return value.Num == 0
	case gjson.String:
		return value.Str == ""
	case gjson.JSON:
		// Check if it's an empty array or object
		if value.IsArray() {
			return len(value.Array()) == 0
		}
		if value.IsObject() {
			// Empty object or object with all empty values
			isEmpty := true
			value.ForEach(func(_, v gjson.Result) bool {
				if !IsEmptyValue(v) {
					isEmpty = false
					return false
				}
				return true
			})
			return isEmpty
		}
		return false
	default:
		return false
	}
}

// GetResourceAttribute retrieves an attribute value from the state for a given resource.
// This is useful when config transformation needs to resolve variable values by looking
// up the actual value that was applied in the state.
//
// Example - Getting the actual cache_type value from state when config uses a variable:
//
// Config:
//
//	resource "cloudflare_tiered_cache" "example" {
//	  zone_id    = "test-zone-id"
//	  cache_type = var.cache_type_value  // Variable reference
//	}
//
// State JSON:
//
//	{
//	  "resources": [{
//	    "type": "cloudflare_tiered_cache",
//	    "name": "example",
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "test-zone-id",
//	        "cache_type": "smart"  // Actual resolved value
//	      }
//	    }]
//	  }]
//	}
//
// Usage:
//
//	value := GetResourceAttribute(stateJSON, "cloudflare_tiered_cache", "example", "cache_type")
//	// Returns: "smart"
//
// If the resource or attribute doesn't exist, returns an empty string.
func GetResourceAttribute(stateJSON, resourceType, resourceName, attributeName string) string {
	if stateJSON == "" {
		return ""
	}

	state := gjson.Parse(stateJSON)
	resources := state.Get("resources")
	if !resources.Exists() {
		return ""
	}

	var result string
	resources.ForEach(func(_, resource gjson.Result) bool {
		if resource.Get("type").String() == resourceType &&
			resource.Get("name").String() == resourceName {
			instances := resource.Get("instances")
			if instances.Exists() && len(instances.Array()) > 0 {
				attrValue := instances.Get("0.attributes." + attributeName)
				if attrValue.Exists() {
					result = attrValue.String()
					return false // Stop iteration
				}
			}
		}
		return true // Continue iteration
	})

	return result
}
