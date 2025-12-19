// Package state provides utilities for transforming array structures in Terraform state files
// during provider migrations.
package state

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ArrayToObjectOptions configures how to transform an array to an object
type ArrayToObjectOptions struct {
	SkipFields         []string                                  // Fields to skip when copying
	FieldTransforms    map[string]func(gjson.Result) interface{} // Custom transformations per field
	RenameFields       map[string]string                         // Old field name -> new field name
	DefaultFields      map[string]interface{}                    // Fields to add if not present
	EnsureObjectExists bool                                      // If true, create empty object even when field is missing/null/empty array
}

// TransformArrayToObject transforms the first element of an array to an object.
// This is commonly used when provider schema changes from accepting multiple
// items to only accepting a single configuration object.
//
// Example - SRV record data transformation:
//
// Before (array format):
//
//	{
//	  "data": [
//	    {
//	      "priority": 10,
//	      "weight": 60,
//	      "port": 5060,
//	      "target": "sipserver.example.com"
//	    }
//	  ]
//	}
//
// After calling with options.SkipFields = ["priority"]:
//
//	{
//	  "data": {
//	    "weight": 60,
//	    "port": 5060,
//	    "target": "sipserver.example.com"
//	  }
//	}
func TransformArrayToObject(data gjson.Result, options ArrayToObjectOptions) map[string]interface{} {
	obj := make(map[string]interface{})

	if !data.IsArray() {
		return obj
	}

	array := data.Array()
	if len(array) == 0 {
		return obj
	}

	firstElem := array[0]
	firstElem.ForEach(func(key, value gjson.Result) bool {
		k := key.String()

		// Skip unwanted fields
		for _, skip := range options.SkipFields {
			if k == skip {
				return true // continue
			}
		}

		// Apply field rename if configured
		if newName, exists := options.RenameFields[k]; exists {
			k = newName
		}

		// Apply custom transform if exists
		if transform, exists := options.FieldTransforms[k]; exists {
			obj[k] = transform(value)
		} else {
			obj[k] = ConvertGjsonValue(value)
		}
		return true
	})

	// Add default fields if they don't exist
	for field, defaultValue := range options.DefaultFields {
		if _, exists := obj[field]; !exists {
			obj[field] = defaultValue
		}
	}

	return obj
}

// TransformFieldArrayToObject transforms any field from array to object.
// This is a generalized helper that works with any field name.
// It handles both array and object inputs, applying transformations as needed.
//
// Example - CAA record data transformation with field rename and custom transform:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "type": "CAA",
//	        "data": [
//	          {
//	            "flags": "0",           // String that should be number
//	            "tag": "issue",
//	            "content": "letsencrypt.org"  // Should be renamed to "value"
//	          }
//	        ]
//	      }
//	    }]
//	  }]
//	}
//
// After calling with:
//
//	options.RenameFields["content"] = "value"
//	options.FieldTransforms["flags"] = convertToNumber
//	result = TransformFieldArrayToObject(result, "attributes", attrs, "data", options)
//
// Result:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "type": "CAA",
//	        "data": {
//	          "flags": 0,
//	          "tag": "issue",
//	          "value": "letsencrypt.org"
//	        }
//	      }
//	    }]
//	  }]
//	}
//
// Example - Transforming config field in zero_trust_access_identity_provider:
//
// Before state JSON:
//
//	{
//	  "attributes": {
//	    "type": "azureAD",
//	    "config": [
//	      {
//	        "client_id": "test-id",
//	        "api_token": "deprecated"
//	      }
//	    ]
//	  }
//	}
//
// After calling with:
//
//	options.SkipFields = []string{"api_token"}
//	result = TransformFieldArrayToObject(result, "attributes", attrs, "config", options)
//
// Result:
//
//	{
//	  "attributes": {
//	    "type": "azureAD",
//	    "config": {
//	      "client_id": "test-id"
//	    }
//	  }
//	}
func TransformFieldArrayToObject(
	stateJSON string,
	path string,
	instance gjson.Result,
	fieldName string,
	options ArrayToObjectOptions,
) string {
	field := instance.Get(fieldName)

	if !field.Exists() || field.Type == gjson.Null {
		// Handle case where field doesn't exist or is null
		if options.EnsureObjectExists {
			// Create empty object with default fields
			obj := make(map[string]interface{})
			for field, defaultValue := range options.DefaultFields {
				obj[field] = defaultValue
			}
			stateJSON, _ = sjson.Set(stateJSON, path+"."+fieldName, obj)
		} else if len(options.DefaultFields) > 0 {
			stateJSON, _ = sjson.Set(stateJSON, path+"."+fieldName, options.DefaultFields)
		}
		return stateJSON
	}

	if field.IsArray() {
		array := field.Array()
		if len(array) == 0 {
			// Empty array
			if options.EnsureObjectExists {
				// Create empty object with default fields instead of deleting
				obj := make(map[string]interface{})
				for field, defaultValue := range options.DefaultFields {
					obj[field] = defaultValue
				}
				stateJSON, _ = sjson.Set(stateJSON, path+"."+fieldName, obj)
			} else {
				// Remove the field
				stateJSON, _ = sjson.Delete(stateJSON, path+"."+fieldName)
			}
		} else {
			// Transform to object
			obj := TransformArrayToObject(field, options)
			if len(obj) == 0 && !options.EnsureObjectExists {
				stateJSON, _ = sjson.Delete(stateJSON, path+"."+fieldName)
			} else {
				stateJSON, _ = sjson.Set(stateJSON, path+"."+fieldName, obj)
			}
		}
	} else if field.IsObject() {
		// Already an object - apply any necessary transformations
		obj := make(map[string]interface{})

		field.ForEach(func(key, value gjson.Result) bool {
			k := key.String()

			// Skip unwanted fields
			for _, skip := range options.SkipFields {
				if k == skip {
					return true
				}
			}

			// Apply field rename if configured
			if newName, exists := options.RenameFields[k]; exists {
				k = newName
			}

			// Apply custom transform if exists
			if transform, exists := options.FieldTransforms[k]; exists {
				obj[k] = transform(value)
			} else {
				obj[k] = ConvertGjsonValue(value)
			}
			return true
		})

		// Add default fields if they don't exist
		for field, defaultValue := range options.DefaultFields {
			if _, exists := obj[field]; !exists {
				obj[field] = defaultValue
			}
		}

		if len(obj) == 0 {
			stateJSON, _ = sjson.Delete(stateJSON, path+"."+fieldName)
		} else {
			stateJSON, _ = sjson.Set(stateJSON, path+"."+fieldName, obj)
		}
	}

	return stateJSON
}

// FlattenArrayField flattens an array field with a single element to just the element.
// This is useful when a field changes from supporting multiple values to a single value.
//
// Example - Flattening single-element priority array:
//
// Before state JSON:
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "priority": [10]  // Single-element array
//	      }
//	    }]
//	  }]
//	}
//
// After calling FlattenArrayField(stateJSON, "resources.0.instances.0.attributes.priority", priorityField):
//
//	{
//	  "resources": [{
//	    "instances": [{
//	      "attributes": {
//	        "zone_id": "abc123",
//	        "priority": 10  // Flattened to scalar value
//	      }
//	    }]
//	  }]
//	}
func FlattenArrayField(stateJSON string, path string, field gjson.Result) string {
	if field.IsArray() {
		array := field.Array()
		if len(array) == 1 {
			// Single element array - flatten it
			stateJSON, _ = sjson.Set(stateJSON, path, array[0].Value())
		}
	}
	return stateJSON
}
