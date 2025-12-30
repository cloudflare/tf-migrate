package state

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestIsEmptyStructure(t *testing.T) {
	tests := []struct {
		name          string
		actual        string
		emptyTemplate string
		expected      bool
	}{
		{
			name:          "Exact match - simple null fields",
			actual:        `{"id": null, "version": null}`,
			emptyTemplate: `{"id": null, "version": null}`,
			expected:      true,
		},
		{
			name:          "Has value in one field",
			actual:        `{"id": "abc123", "version": null}`,
			emptyTemplate: `{"id": null, "version": null}`,
			expected:      false,
		},
		{
			name:          "Has value in multiple fields",
			actual:        `{"id": "abc123", "version": "1.0"}`,
			emptyTemplate: `{"id": null, "version": null}`,
			expected:      false,
		},
		{
			name:          "Extra field in actual",
			actual:        `{"id": null, "version": null, "extra": "field"}`,
			emptyTemplate: `{"id": null, "version": null}`,
			expected:      false,
		},
		{
			name:          "Missing field in actual",
			actual:        `{"id": null}`,
			emptyTemplate: `{"id": null, "version": null}`,
			expected:      false,
		},
		{
			name:          "Nested objects match",
			actual:        `{"outer": {"inner": null}}`,
			emptyTemplate: `{"outer": {"inner": null}}`,
			expected:      true,
		},
		{
			name:          "Nested objects don't match",
			actual:        `{"outer": {"inner": "value"}}`,
			emptyTemplate: `{"outer": {"inner": null}}`,
			expected:      false,
		},
		{
			name:          "Non-existent actual",
			actual:        ``,
			emptyTemplate: `{"id": null}`,
			expected:      true,
		},
		{
			name:          "Empty string template - invalid JSON",
			actual:        `{"id": null}`,
			emptyTemplate: ``,
			expected:      false,
		},
		{
			name:          "Invalid JSON in template",
			actual:        `{"id": null}`,
			emptyTemplate: `{invalid}`,
			expected:      false,
		},
		{
			name:          "Array fields match",
			actual:        `{"items": []}`,
			emptyTemplate: `{"items": []}`,
			expected:      true,
		},
		{
			name:          "Array fields don't match",
			actual:        `{"items": ["value"]}`,
			emptyTemplate: `{"items": []}`,
			expected:      false,
		},
		{
			name:          "Mixed types - number vs null",
			actual:        `{"count": 0}`,
			emptyTemplate: `{"count": null}`,
			expected:      false,
		},
		{
			name:          "Mixed types - empty string vs null",
			actual:        `{"name": ""}`,
			emptyTemplate: `{"name": null}`,
			expected:      false,
		},
		{
			name:          "Complex nested structure all null",
			actual:        `{"input": {"id": null, "version": null}, "output": {"status": null}}`,
			emptyTemplate: `{"input": {"id": null, "version": null}, "output": {"status": null}}`,
			expected:      true,
		},
		{
			name:          "Complex nested structure with value",
			actual:        `{"input": {"id": null, "version": null}, "output": {"status": "active"}}`,
			emptyTemplate: `{"input": {"id": null, "version": null}, "output": {"status": null}}`,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual gjson.Result
			if tt.actual != "" {
				actual = gjson.Parse(tt.actual)
			} else {
				// Create a non-existent result
				actual = gjson.Parse(`{}`).Get("nonexistent")
			}

			result := IsEmptyStructure(actual, tt.emptyTemplate)
			if result != tt.expected {
				t.Errorf("IsEmptyStructure() = %v, want %v", result, tt.expected)
			}
		})
	}
}
