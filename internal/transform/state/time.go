// Package state provides utilities for transforming Terraform state files
package state

import (
	"strings"
	"time"
)

// NormalizeDuration normalizes Go duration strings by removing unnecessary zero components.
// This is useful when state contains verbose duration strings like "24h0m0s" that should be "24h".
//
// Example transformations:
//   "24h0m0s" -> "24h"
//   "1h30m0s" -> "1h30m"
//   "45m0s"   -> "45m"
//   "30s"     -> "30s" (unchanged)
//
// Usage in migrations:
//   duration := attrs.Get("timeout").String()
//   normalized := NormalizeDuration(duration)
//   result, _ = sjson.Set(result, "attributes.timeout", normalized)
func NormalizeDuration(duration string) string {
	// Remove unnecessary zero components from right to left
	duration = strings.ReplaceAll(duration, "h0m0s", "h")
	duration = strings.ReplaceAll(duration, "m0s", "m")
	// Don't modify if it's just seconds
	return duration
}

// NormalizeRFC3339 normalizes date strings to RFC3339 format.
// Attempts to parse various common date formats and converts them to RFC3339.
// Returns the original string if it cannot be parsed.
//
// Supported input formats:
//   - RFC3339: "2006-01-02T15:04:05Z07:00"
//   - ISO8601 with Z: "2006-01-02T15:04:05Z"
//   - Date only: "2006-01-02"
//   - RFC3339Nano: "2006-01-02T15:04:05.999999999Z07:00"
//
// Example:
//   "2024-01-01" -> "2024-01-01T00:00:00Z"
//   "2024-01-01T12:30:45Z" -> "2024-01-01T12:30:45Z"
//   "invalid" -> "invalid" (unchanged)
//
// Usage in migrations:
//   createdOn := attrs.Get("created_on").String()
//   normalized := NormalizeRFC3339(createdOn)
//   result, _ = sjson.Set(result, "attributes.created_on", normalized)
func NormalizeRFC3339(dateStr string) string {
	// List of formats to try parsing
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}

	// If we can't parse it, return as-is
	return dateStr
}
