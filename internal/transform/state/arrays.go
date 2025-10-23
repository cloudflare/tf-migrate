// Package state provides utilities for transforming array structures in Terraform state files
// during provider migrations.
package state

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ArrayToObjectOptions configures how to transform an array to an object
type ArrayToObjectOptions struct {
	SkipFields      []string                                    // Fields to skip when copying
	FieldTransforms map[string]func(gjson.Result) interface{}  // Custom transformations per field
	RenameFields    map[string]string                          // Old field name -> new field name
	DefaultFields   map[string]interface{}                     // Fields to add if not present
}

// TransformArrayToObject transforms the first element of an array to an object.
// This is commonly used when provider schema changes from accepting multiple
// items to only accepting a single configuration object.
//
// Example - SRV record data transformation:
//
// Before (array format):
//   {
//     "data": [
//       {
//         "priority": 10,
//         "weight": 60,
//         "port": 5060,
//         "target": "sipserver.example.com"
//       }
//     ]
//   }
//
// After calling with options.SkipFields = ["priority"]:
//   {
//     "data": {
//       "weight": 60,
//       "port": 5060,
//       "target": "sipserver.example.com"
//     }
//   }
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

// TransformDataFieldArrayToObject handles the common pattern of transforming data field from array to object.
// This function handles both array and object inputs, applying transformations as needed.
//
// Example - CAA record data transformation with field rename and custom transform:
//
// Before state JSON:
//   {
//     "resources": [{
//       "instances": [{
//         "attributes": {
//           "type": "CAA",
//           "data": [
//             {
//               "flags": "0",           // String that should be number
//               "tag": "issue",
//               "content": "letsencrypt.org"  // Should be renamed to "value"
//             }
//           ]
//         }
//       }]
//     }]
//   }
//
// After calling with:
//   options.RenameFields["content"] = "value"
//   options.FieldTransforms["flags"] = convertToNumber
//
// Result:
//   {
//     "resources": [{
//       "instances": [{
//         "attributes": {
//           "type": "CAA",
//           "data": {
//             "flags": {
//               "value": 0,
//               "type": "number"
//             },
//             "tag": "issue",
//             "value": "letsencrypt.org"
//           }
//         }
//       }]
//     }]
//   }
func TransformDataFieldArrayToObject(stateJSON string, path string, instance gjson.Result, recordType string, options ArrayToObjectOptions) string {
	data := instance.Get("data")
	
	if !data.Exists() {
		// Handle case where data doesn't exist but should
		if len(options.DefaultFields) > 0 {
			stateJSON, _ = sjson.Set(stateJSON, path+".data", options.DefaultFields)
		}
		return stateJSON
	}
	
	if data.IsArray() {
		array := data.Array()
		if len(array) == 0 {
			// Empty array - remove the data field
			stateJSON, _ = sjson.Delete(stateJSON, path+".data")
		} else {
			// Transform to object
			obj := TransformArrayToObject(data, options)
			if len(obj) == 0 {
				stateJSON, _ = sjson.Delete(stateJSON, path+".data")
			} else {
				stateJSON, _ = sjson.Set(stateJSON, path+".data", obj)
			}
		}
	} else if data.IsObject() {
		// Already an object - apply any necessary transformations
		obj := make(map[string]interface{})
		
		data.ForEach(func(key, value gjson.Result) bool {
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
			stateJSON, _ = sjson.Delete(stateJSON, path+".data")
		} else {
			stateJSON, _ = sjson.Set(stateJSON, path+".data", obj)
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
//   {
//     "resources": [{
//       "instances": [{
//         "attributes": {
//           "zone_id": "abc123",
//           "priority": [10]  // Single-element array
//         }
//       }]
//     }]
//   }
//
// After calling FlattenArrayField(stateJSON, "resources.0.instances.0.attributes.priority", priorityField):
//   {
//     "resources": [{
//       "instances": [{
//         "attributes": {
//           "zone_id": "abc123",
//           "priority": 10  // Flattened to scalar value
//         }
//       }]
//     }]
//   }
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