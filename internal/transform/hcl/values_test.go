package hcl

import (
	"testing"

	"github.com/tidwall/gjson"
)

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

func TestIsEmptyValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "null is empty",
			input:    `null`,
			expected: true,
		},
		{
			name:     "false is empty",
			input:    `false`,
			expected: true,
		},
		{
			name:     "zero number is empty",
			input:    `0`,
			expected: true,
		},
		{
			name:     "empty string is empty",
			input:    `""`,
			expected: true,
		},
		{
			name:     "empty array is empty",
			input:    `[]`,
			expected: true,
		},
		{
			name:     "empty object is empty",
			input:    `{}`,
			expected: true,
		},
		{
			name:     "true is not empty",
			input:    `true`,
			expected: false,
		},
		{
			name:     "non-zero number is not empty",
			input:    `42`,
			expected: false,
		},
		{
			name:     "non-empty string is not empty",
			input:    `"hello"`,
			expected: false,
		},
		{
			name:     "non-empty array is not empty",
			input:    `[1]`,
			expected: false,
		},
		{
			name:     "non-empty object is not empty",
			input:    `{"key": "value"}`,
			expected: false,
		},
		{
			name:     "object with all empty values is empty",
			input:    `{"a": "", "b": 0, "c": null}`,
			expected: true,
		},
		{
			name:     "object with one non-empty value is not empty",
			input:    `{"a": "", "b": 1}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Parse(tt.input)
			result := IsEmptyValue(value)
			if result != tt.expected {
				t.Errorf("IsEmptyValue(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
