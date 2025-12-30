// Package state provides utilities for transforming Terraform state files
package state

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// RemoveFieldsIfExist removes multiple fields from state at the given path if they exist.
// This is a batch operation that prevents repeated existence checks and deletions.
//
// Example - Removing multiple deprecated fields:
//
// Before state JSON:
//
//	{
//	  "attributes": {
//	    "zone_id": "abc123",
//	    "deprecated1": "value1",
//	    "deprecated2": "value2",
//	    "deprecated3": "value3"
//	  }
//	}
//
// After calling RemoveFieldsIfExist(result, "attributes", attrs, "deprecated1", "deprecated2", "deprecated3"):
//
//	{
//	  "attributes": {
//	    "zone_id": "abc123"
//	  }
//	}
func RemoveFieldsIfExist(stateJSON string, path string, instance gjson.Result, fields ...string) string {
	for _, field := range fields {
		if instance.Get(field).Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, path+"."+field)
		}
	}
	return stateJSON
}

// RenameFieldsMap renames multiple fields according to the provided mapping.
// Keys are old names, values are new names.
//
// Example - Batch renaming fields:
//
// Before:
//
//	{
//	  "old_name1": "value1",
//	  "old_name2": "value2"
//	}
//
// After calling RenameFieldsMap(result, path, instance, map[string]string{"old_name1": "new_name1", "old_name2": "new_name2"}):
//
//	{
//	  "new_name1": "value1",
//	  "new_name2": "value2"
//	}
func RenameFieldsMap(stateJSON string, path string, instance gjson.Result, renames map[string]string) string {
	for oldName, newName := range renames {
		stateJSON = RenameField(stateJSON, path, instance, oldName, newName)
	}
	return stateJSON
}

// SetSchemaVersion is a convenience helper to set the schema version in state.
// This eliminates the repeated pattern of manually setting schema_version.
//
// Example:
//
// Before:
//
//	result, _ = sjson.Set(result, "schema_version", 0)
//
// After:
//
//	result = state.SetSchemaVersion(result, 0)
func SetSchemaVersion(stateJSON string, version int) string {
	result, _ := sjson.Set(stateJSON, "schema_version", version)
	return result
}
