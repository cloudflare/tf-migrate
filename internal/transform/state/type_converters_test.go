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
			name:     "integer number",
			input:    `42`,
			expected: 42.0,
		},
		{
			name:     "float number",
			input:    `42.5`,
			expected: 42.5,
		},
		{
			name:     "string number",
			input:    `"123"`,
			expected: 123.0,
		},
		{
			name:     "non-numeric string",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "null value",
			input:    `null`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Parse(tt.input)
			result := ConvertToFloat64(value)

			if result != tt.expected {
				t.Errorf("ConvertToFloat64() = %v, want %v", result, tt.expected)
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
			name:     "integer number",
			input:    `42`,
			expected: int64(42),
		},
		{
			name:     "large integer number",
			input:    `3600`,
			expected: int64(3600),
		},
		{
			name:     "string number",
			input:    `"123"`,
			expected: int64(123),
		},
		{
			name:     "string large number",
			input:    `"31536000"`,
			expected: int64(31536000),
		},
		{
			name:     "non-numeric string",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "null value",
			input:    `null`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Parse(tt.input)
			result := ConvertToInt64(value)

			if result != tt.expected {
				t.Errorf("ConvertToInt64() = %v (type %T), want %v (type %T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}