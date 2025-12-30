package state

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestRemoveFieldsIfExist(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		path         string
		instancePath string
		fields       []string
		expected     string
	}{
		{
			name:         "All fields exist",
			path:         "attributes",
			instancePath: "attributes",
			input: `{
				"attributes": {
					"zone_id": "abc123",
					"field1": "value1",
					"field2": "value2",
					"field3": "value3"
				}
			}`,
			fields: []string{"field1", "field2", "field3"},
			expected: `{
				"zone_id": "abc123"
			}`,
		},
		{
			name:         "Some fields exist",
			path:         "attributes",
			instancePath: "attributes",
			input: `{
				"attributes": {
					"zone_id": "abc123",
					"field1": "value1",
					"field3": "value3"
				}
			}`,
			fields: []string{"field1", "field2", "field3"},
			expected: `{
				"zone_id": "abc123"
			}`,
		},
		{
			name:         "No fields exist",
			path:         "attributes",
			instancePath: "attributes",
			input: `{
				"attributes": {
					"zone_id": "abc123"
				}
			}`,
			fields:   []string{"field1", "field2"},
			expected: `{"zone_id": "abc123"}`,
		},
		{
			name:         "Empty field list",
			path:         "attributes",
			instancePath: "attributes",
			input: `{
				"attributes": {
					"zone_id": "abc123",
					"field1": "value1"
				}
			}`,
			fields:   []string{},
			expected: `{"zone_id": "abc123", "field1": "value1"}`,
		},
		{
			name:         "Nested path",
			path:         "attributes.nested",
			instancePath: "attributes.nested",
			input: `{
				"attributes": {
					"nested": {
						"field1": "value1",
						"field2": "value2"
					}
				}
			}`,
			fields:   []string{"field1"},
			expected: `{"field2": "value2"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := gjson.Parse(tt.input).Get(tt.instancePath)
			result := RemoveFieldsIfExist(tt.input, tt.path, attrs, tt.fields...)

			// Parse and compare the attributes portion
			resultAttrs := gjson.Parse(result).Get(tt.instancePath)
			expectedAttrs := gjson.Parse(`{"result":` + tt.expected + `}`).Get("result")

			// Compare field by field instead of raw JSON
			resultMap := resultAttrs.Map()
			expectedMap := expectedAttrs.Map()

			if len(resultMap) != len(expectedMap) {
				t.Errorf("RemoveFieldsIfExist() field count mismatch: got %d fields, want %d fields", len(resultMap), len(expectedMap))
			}

			for key, expectedVal := range expectedMap {
				if resultVal, ok := resultMap[key]; !ok {
					t.Errorf("RemoveFieldsIfExist() missing field %q", key)
				} else if resultVal.String() != expectedVal.String() {
					t.Errorf("RemoveFieldsIfExist() field %q: got = %v, want %v", key, resultVal.String(), expectedVal.String())
				}
			}
		})
	}
}

func TestRenameFieldsMap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		renames  map[string]string
		expected string
	}{
		{
			name: "Multiple renames",
			input: `{
				"attributes": {
					"old1": "value1",
					"old2": "value2",
					"keep": "value3"
				}
			}`,
			renames: map[string]string{
				"old1": "new1",
				"old2": "new2",
			},
			expected: `{"new1": "value1", "new2": "value2", "keep": "value3"}`,
		},
		{
			name: "Field doesn't exist",
			input: `{
				"attributes": {
					"field1": "value1"
				}
			}`,
			renames: map[string]string{
				"nonexistent": "new_name",
			},
			expected: `{"field1": "value1"}`,
		},
		{
			name: "Field already has new name",
			input: `{
				"attributes": {
					"new_name": "value1"
				}
			}`,
			renames: map[string]string{
				"old_name": "new_name",
			},
			expected: `{"new_name": "value1"}`,
		},
		{
			name: "Empty map",
			input: `{
				"attributes": {
					"field1": "value1"
				}
			}`,
			renames:  map[string]string{},
			expected: `{"field1": "value1"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := gjson.Parse(tt.input).Get("attributes")
			result := RenameFieldsMap(tt.input, "attributes", attrs, tt.renames)

			resultAttrs := gjson.Parse(result).Get("attributes")
			expectedAttrs := gjson.Parse(`{"attributes":` + tt.expected + `}`).Get("attributes")

			// Compare field by field instead of raw JSON
			resultMap := resultAttrs.Map()
			expectedMap := expectedAttrs.Map()

			if len(resultMap) != len(expectedMap) {
				t.Errorf("RenameFieldsMap() field count mismatch: got %d fields, want %d fields", len(resultMap), len(expectedMap))
			}

			for key, expectedVal := range expectedMap {
				if resultVal, ok := resultMap[key]; !ok {
					t.Errorf("RenameFieldsMap() missing field %q", key)
				} else if resultVal.String() != expectedVal.String() {
					t.Errorf("RenameFieldsMap() field %q: got = %v, want %v", key, resultVal.String(), expectedVal.String())
				}
			}
		})
	}
}

func TestSetSchemaVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		version  int
		expected int
	}{
		{
			name:     "Set to 0",
			input:    `{"attributes": {"field": "value"}}`,
			version:  0,
			expected: 0,
		},
		{
			name:     "Set to non-zero",
			input:    `{"attributes": {"field": "value"}}`,
			version:  5,
			expected: 5,
		},
		{
			name:     "Overwrite existing version",
			input:    `{"schema_version": 3, "attributes": {"field": "value"}}`,
			version:  0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SetSchemaVersion(tt.input, tt.version)
			version := gjson.Parse(result).Get("schema_version").Int()

			if int(version) != tt.expected {
				t.Errorf("SetSchemaVersion() got = %v, want %v", version, tt.expected)
			}
		})
	}
}
