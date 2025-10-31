package state

import (
	"strconv"

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
