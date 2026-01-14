package state

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// ConvertToFloat64 converts a value to float64, handling various input types.
// Returns nil for null/invalid values.
//
// Before → After transformations:
//
//	42 (json.Number)     → 42.0 (float64)
//	"123" (string)       → 123.0 (float64)
//	"hello" (string)     → "hello" (unchanged, not numeric)
//	null                 → nil
//
// Essential for v4→v5 migrations where TypeInt becomes Float64Attribute.
func ConvertToFloat64(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.Number:
		return value.Float()
	case gjson.String:
		// Try to parse string as number
		if f, err := strconv.ParseFloat(value.String(), 64); err == nil {
			return f
		}
		// Return as-is if not a number
		return value.String()
	case gjson.Null:
		return nil
	default:
		return value.Value()
	}
}

// ConvertToInt64 converts a value to int64, handling various input types.
// Returns nil for null/invalid values.
//
// Before → After transformations:
//
//	42 (json.Number)     → 42 (int64)
//	"123" (string)       → 123 (int64)
//	"hello" (string)     → "hello" (unchanged, not numeric)
//	null                 → nil
//
// Essential for v4→v5 migrations where TypeString becomes Int64Attribute.
func ConvertToInt64(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.Number:
		return value.Int()
	case gjson.String:
		// Try to parse string as integer
		if i, err := strconv.ParseInt(value.String(), 10, 64); err == nil {
			return i
		}
		// Return as-is if not a number
		return value.String()
	case gjson.Null:
		return nil
	default:
		return value.Value()
	}
}

// ConvertEnabledDisabledToBool converts "enabled"/"disabled" string values to booleans.
// Returns nil for null values. Returns the original value if not "enabled" or "disabled".
//
// Before → After transformations:
//
//	"enabled"  → true
//	"disabled" → false
//	true       → true (already boolean)
//	false      → false (already boolean)
//	null       → nil
//
// This is a common pattern in Cloudflare v4 resources where boolean fields
// were represented as "enabled"/"disabled" strings.
func ConvertEnabledDisabledToBool(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.String:
		switch value.String() {
		case "enabled":
			return true
		case "disabled":
			return false
		default:
			return value.String()
		}
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

// ConvertDurationToSeconds converts duration values to int64 seconds.
// Handles both numeric values (assumed to be seconds) and Go duration strings.
// Returns nil for null/invalid values.
//
// Before → After transformations:
//
//	30 (json.Number)     → 30 (int64, seconds)
//	"30s" (string)       → 30 (int64, seconds)
//	"1m30s" (string)     → 90 (int64, seconds)
//	"2h" (string)        → 7200 (int64, seconds)
//	"hello" (string)     → "hello" (unchanged, not a duration)
//	null                 → nil
//
// This is common in Cloudflare v4 resources where duration fields were strings,
// but v5 expects int64 seconds.
func ConvertDurationToSeconds(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.Number:
		// Already a number, assume it's in seconds
		return value.Int()
	case gjson.String:
		// Try to parse as duration string (e.g., "30s", "1m30s")
		if seconds, err := ParseDurationStringToSeconds(value.String()); err == nil {
			return seconds
		}
		// Return as-is if not a valid duration
		return value.String()
	case gjson.Null:
		return nil
	default:
		return value.Value()
	}
}

// ParseDurationStringToSeconds parses Go duration format strings to seconds.
// Supports: s (seconds), m (minutes), h (hours)
// Examples: "30s" → 30, "1m30s" → 90, "2h" → 7200
//
// This is exported so it can be used in both state and config transformations.
func ParseDurationStringToSeconds(durationStr string) (int64, error) {
	durationStr = strings.TrimSpace(durationStr)
	if durationStr == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	var totalSeconds int64 = 0
	remaining := durationStr

	// Parse components like "1m30s"
	for len(remaining) > 0 {
		// Find the next unit (s, m, h)
		var num string
		var unit byte
		var foundUnit bool

		for i := 0; i < len(remaining); i++ {
			c := remaining[i]
			if c == 's' || c == 'm' || c == 'h' {
				num = remaining[:i]
				unit = c
				remaining = remaining[i+1:]
				foundUnit = true
				break
			}
		}

		if !foundUnit {
			return 0, fmt.Errorf("invalid duration format: %s", durationStr)
		}

		// Parse the number
		value, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number in duration: %s", num)
		}

		// Convert to seconds based on unit
		switch unit {
		case 's':
			totalSeconds += value
		case 'm':
			totalSeconds += value * 60
		case 'h':
			totalSeconds += value * 3600
		default:
			return 0, fmt.Errorf("unknown unit: %c", unit)
		}
	}

	return totalSeconds, nil
}
