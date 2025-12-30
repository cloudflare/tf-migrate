// Package state provides utilities for transforming Terraform state files
package state

import (
	"encoding/json"
	"reflect"

	"github.com/tidwall/gjson"
)

// IsEmptyStructure checks if a gjson.Result matches an empty template structure.
// This is useful for detecting when a complex nested object has only null/empty values.
//
// The comparison is deep equality - both structure and values must match.
//
// Example - Detecting empty input structure:
//
// Template:
//   {"id": null, "version": null}
//
// Actual value 1:
//   {"id": null, "version": null}
//   -> Returns true (matches template)
//
// Actual value 2:
//   {"id": "abc123", "version": null}
//   -> Returns false (id has a value)
//
// Actual value 3:
//   {"id": null, "version": null, "extra": "field"}
//   -> Returns false (has extra field)
//
// Usage in migrations:
//   emptyTemplate := `{"id":null,"version":null}`
//   if state.IsEmptyStructure(input, emptyTemplate) {
//       // This input is effectively empty, clean it up
//       result, _ = sjson.Delete(result, "attributes.input")
//   }
func IsEmptyStructure(actual gjson.Result, emptyTemplate string) bool {
	if !actual.Exists() {
		return true
	}

	var actualMap map[string]interface{}
	var templateMap map[string]interface{}

	// Parse actual value
	if err := json.Unmarshal([]byte(actual.Raw), &actualMap); err != nil {
		return false
	}

	// Parse template
	if err := json.Unmarshal([]byte(emptyTemplate), &templateMap); err != nil {
		return false
	}

	// Deep equality check
	return reflect.DeepEqual(actualMap, templateMap)
}
