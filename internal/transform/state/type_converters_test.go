package state

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "integer number to float64",
			input:    `42`,
			expected: float64(42),
		},
		{
			name:     "float number to float64",
			input:    `3.14`,
			expected: float64(3.14),
		},
		{
			name:     "numeric string to float64",
			input:    `"123"`,
			expected: float64(123),
		},
		{
			name:     "float string to float64",
			input:    `"45.67"`,
			expected: float64(45.67),
		},
		{
			name:     "non-numeric string unchanged",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "null returns nil",
			input:    `null`,
			expected: nil,
		},
		{
			name:     "boolean returns as-is",
			input:    `true`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Parse(tt.input)
			result := ConvertToFloat64(value)

			if result != tt.expected {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestConvertToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "integer number to int64",
			input:    `42`,
			expected: int64(42),
		},
		{
			name:     "float number to int64 (truncated)",
			input:    `3.14`,
			expected: int64(3),
		},
		{
			name:     "numeric string to int64",
			input:    `"123"`,
			expected: int64(123),
		},
		{
			name:     "non-numeric string unchanged",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "null returns nil",
			input:    `null`,
			expected: nil,
		},
		{
			name:     "boolean returns as-is",
			input:    `false`,
			expected: false,
		},
		{
			name:     "large integer",
			input:    `9223372036854775807`,
			expected: int64(9223372036854775807),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Parse(tt.input)
			result := ConvertToInt64(value)

			if result != tt.expected {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestConvertEnabledDisabledToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "enabled string to true",
			input:    `"enabled"`,
			expected: true,
		},
		{
			name:     "disabled string to false",
			input:    `"disabled"`,
			expected: false,
		},
		{
			name:     "true boolean stays true",
			input:    `true`,
			expected: true,
		},
		{
			name:     "false boolean stays false",
			input:    `false`,
			expected: false,
		},
		{
			name:     "null returns nil",
			input:    `null`,
			expected: nil,
		},
		{
			name:     "other string unchanged",
			input:    `"active"`,
			expected: "active",
		},
		{
			name:     "number returns as-is",
			input:    `1`,
			expected: float64(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Parse(tt.input)
			result := ConvertEnabledDisabledToBool(value)

			if result != tt.expected {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestParseDurationStringToSeconds(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int64
		expectError bool
	}{
		{
			name:     "seconds only",
			input:    "30s",
			expected: 30,
		},
		{
			name:     "minutes only",
			input:    "5m",
			expected: 300,
		},
		{
			name:     "hours only",
			input:    "2h",
			expected: 7200,
		},
		{
			name:     "minutes and seconds",
			input:    "1m30s",
			expected: 90,
		},
		{
			name:     "hours, minutes, and seconds",
			input:    "1h30m45s",
			expected: 5445,
		},
		{
			name:     "large duration",
			input:    "24h",
			expected: 86400,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid format - no unit",
			input:       "30",
			expectError: true,
		},
		{
			name:        "invalid format - unknown unit",
			input:       "30x",
			expectError: true,
		},
		{
			name:        "invalid format - non-numeric",
			input:       "abcs",
			expectError: true,
		},
		{
			name:     "whitespace trimmed",
			input:    "  30s  ",
			expected: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDurationStringToSeconds(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d seconds, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestConvertDurationToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "number stays as number",
			input:    `30`,
			expected: int64(30),
		},
		{
			name:     "string duration converted",
			input:    `"30s"`,
			expected: int64(30),
		},
		{
			name:     "complex duration converted",
			input:    `"1m30s"`,
			expected: int64(90),
		},
		{
			name:     "non-duration string unchanged",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "null returns nil",
			input:    `null`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Parse(tt.input)
			result := ConvertDurationToSeconds(value)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
