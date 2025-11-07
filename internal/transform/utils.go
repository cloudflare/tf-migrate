package transform

import (
	"time"

	"github.com/tidwall/sjson"
)

// ConvertDateToRFC3339 converts a date string from RFC1123Z format to RFC3339 format.
// This is useful for migrating date fields between provider versions that use different formats.
//
// Example:
//
//	Input:  "Tue, 04 Nov 2025 21:52:44 +0000" (RFC1123Z)
//	Output: "2025-11-04T21:52:44Z" (RFC3339)
//
// Parameters:
//   - jsonString: The JSON string containing the date field
//   - jsonPath: The path to the date field in the JSON (e.g., "attributes.modified_on")
//   - dateValue: The current date value in RFC1123Z format
//
// Returns the modified JSON string with the date converted to RFC3339, or the original
// JSON string if the date cannot be parsed.
func ConvertDateToRFC3339(jsonString string, jsonPath string, dateValue string) string {
	// Try to parse the RFC1123Z format (with timezone)
	t, err := time.Parse(time.RFC1123Z, dateValue)
	if err != nil {
		// If parsing fails, try RFC1123 without timezone
		t, err = time.Parse(time.RFC1123, dateValue)
		if err != nil {
			// If still fails, return original JSON unchanged
			return jsonString
		}
	}

	// Convert to RFC3339 format
	rfc3339 := t.Format(time.RFC3339)
	result, _ := sjson.Set(jsonString, jsonPath, rfc3339)

	return result
}
