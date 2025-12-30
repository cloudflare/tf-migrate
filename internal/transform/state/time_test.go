package state

import (
	"testing"
)

func TestNormalizeDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simplify hours with zero minutes and seconds",
			input:    "24h0m0s",
			expected: "24h",
		},
		{
			name:     "Simplify hours and minutes with zero seconds",
			input:    "1h30m0s",
			expected: "1h30m",
		},
		{
			name:     "Simplify minutes with zero seconds",
			input:    "45m0s",
			expected: "45m",
		},
		{
			name:     "Keep seconds only unchanged",
			input:    "30s",
			expected: "30s",
		},
		{
			name:     "Simplify zero duration",
			input:    "0h0m0s",
			expected: "0h",
		},
		{
			name:     "Cannot simplify hours with seconds",
			input:    "1h0m30s",
			expected: "1h0m30s",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Invalid string unchanged",
			input:    "invalid",
			expected: "invalid",
		},
		{
			name:     "Already simplified hours",
			input:    "2h",
			expected: "2h",
		},
		{
			name:     "Already simplified minutes",
			input:    "15m",
			expected: "15m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeDuration(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeDuration(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeRFC3339(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Date only to RFC3339",
			input:    "2024-01-01",
			expected: "2024-01-01T00:00:00Z",
		},
		{
			name:     "Already RFC3339 with Z",
			input:    "2024-01-01T12:30:45Z",
			expected: "2024-01-01T12:30:45Z",
		},
		{
			name:     "RFC3339 with timezone offset",
			input:    "2024-01-01T12:30:45+02:00",
			expected: "2024-01-01T10:30:45Z",
		},
		{
			name:     "RFC3339Nano with nanoseconds",
			input:    "2024-01-01T12:30:45.123456789Z",
			expected: "2024-01-01T12:30:45Z",
		},
		{
			name:     "Empty string unchanged",
			input:    "",
			expected: "",
		},
		{
			name:     "Invalid string unchanged",
			input:    "invalid",
			expected: "invalid",
		},
		{
			name:     "RFC3339 with negative timezone",
			input:    "2024-06-15T08:00:00-05:00",
			expected: "2024-06-15T13:00:00Z",
		},
		{
			name:     "ISO8601 with Z suffix",
			input:    "2023-12-25T23:59:59Z",
			expected: "2023-12-25T23:59:59Z",
		},
		{
			name:     "Date at start of year",
			input:    "2025-01-01",
			expected: "2025-01-01T00:00:00Z",
		},
		{
			name:     "Date at end of year",
			input:    "2024-12-31",
			expected: "2024-12-31T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeRFC3339(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeRFC3339(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
