package hcl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// ParseDurationStringToSeconds parses Go duration format strings to seconds.
// Supports: s (seconds), m (minutes), h (hours)
// Examples: "30s" → 30, "1m30s" → 90, "2h" → 7200
//
// This is used in both state and config transformations (e.g. zero_trust_tunnel_cloudflared_config
// duration attributes that changed from string to int64 seconds between v4 and v5).
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

// IsEmptyValue checks if a gjson.Result value is considered "empty" (default/zero).
// Used when transforming state fields where v4 sets empty strings/zeros and v5 uses null.
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
