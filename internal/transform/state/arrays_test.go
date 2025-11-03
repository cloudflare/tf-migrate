package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestTransformArrayToObject(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		options  ArrayToObjectOptions
		expected map[string]interface{}
	}{
		{
			name: "Transform basic array to object",
			json: `{
				"data": [
					{
						"priority": 10,
						"weight": 60,
						"port": 5060,
						"target": "sipserver.example.com"
					}
				]
			}`,
			path: "data",
			options: ArrayToObjectOptions{},
			expected: map[string]interface{}{
				"priority": int64(10),
				"weight":   int64(60),
				"port":     int64(5060),
				"target":   "sipserver.example.com",
			},
		},
		{
			name: "Transform with skip fields",
			json: `{
				"data": [
					{
						"priority": 10,
						"weight": 60,
						"port": 5060
					}
				]
			}`,
			path: "data",
			options: ArrayToObjectOptions{
				SkipFields: []string{"priority"},
			},
			expected: map[string]interface{}{
				"weight": int64(60),
				"port":   int64(5060),
			},
		},
		{
			name: "Transform with rename fields",
			json: `{
				"data": [
					{
						"content": "letsencrypt.org",
						"tag": "issue"
					}
				]
			}`,
			path: "data",
			options: ArrayToObjectOptions{
				RenameFields: map[string]string{
					"content": "value",
				},
			},
			expected: map[string]interface{}{
				"value": "letsencrypt.org",
				"tag":   "issue",
			},
		},
		{
			name: "Transform with default fields",
			json: `{
				"data": [
					{
						"value": "test"
					}
				]
			}`,
			path: "data",
			options: ArrayToObjectOptions{
				DefaultFields: map[string]interface{}{
					"description": nil,
				},
			},
			expected: map[string]interface{}{
				"value":       "test",
				"description": nil,
			},
		},
		{
			name: "Transform with custom field transform",
			json: `{
				"data": [
					{
						"flags": "0",
						"tag": "issue"
					}
				]
			}`,
			path: "data",
			options: ArrayToObjectOptions{
				FieldTransforms: map[string]func(gjson.Result) interface{}{
					"flags": func(value gjson.Result) interface{} {
						// Convert string "0" to int 0
						return int64(0)
					},
				},
			},
			expected: map[string]interface{}{
				"flags": int64(0),
				"tag":   "issue",
			},
		},
		{
			name: "Return empty for empty array",
			json: `{
				"data": []
			}`,
			path:     "data",
			options:  ArrayToObjectOptions{},
			expected: map[string]interface{}{},
		},
		{
			name: "Return empty for non-array",
			json: `{
				"data": "not an array"
			}`,
			path:     "data",
			options:  ArrayToObjectOptions{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := gjson.Get(tt.json, tt.path)
			result := TransformArrayToObject(data, tt.options)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformDataFieldArrayToObject(t *testing.T) {
	tests := []struct {
		name        string
		inputJSON   string
		path        string
		recordType  string
		options     ArrayToObjectOptions
		expectedKey string
		expectValue bool
	}{
		{
			name: "Transform array data field to object",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"type": "SRV",
							"data": [
								{
									"priority": 10,
									"weight": 60,
									"port": 5060,
									"target": "sipserver.example.com"
								}
							]
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes",
			recordType:  "SRV",
			options:     ArrayToObjectOptions{},
			expectedKey: "data.priority",
			expectValue: true,
		},
		{
			name: "Remove empty array data field",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"type": "A",
							"data": []
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes",
			recordType:  "A",
			options:     ArrayToObjectOptions{},
			expectedKey: "data",
			expectValue: false,
		},
		{
			name: "Handle existing object data field",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"type": "CAA",
							"data": {
								"flags": 0,
								"tag": "issue",
								"value": "letsencrypt.org"
							}
						}
					}]
				}]
			}`,
			path:       "resources.0.instances.0.attributes",
			recordType: "CAA",
			options: ArrayToObjectOptions{
				SkipFields: []string{"flags"},
			},
			expectedKey: "data.tag",
			expectValue: true,
		},
		{
			name: "Add default fields when data doesn't exist",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"type": "A"
						}
					}]
				}]
			}`,
			path:       "resources.0.instances.0.attributes",
			recordType: "A",
			options: ArrayToObjectOptions{
				DefaultFields: map[string]interface{}{
					"value": "192.0.2.1",
				},
			},
			expectedKey: "data.value",
			expectValue: true,
		},
		{
			name: "Delete data field when all fields are skipped",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"type": "A",
							"data": [
								{
									"priority": 10
								}
							]
						}
					}]
				}]
			}`,
			path:       "resources.0.instances.0.attributes",
			recordType: "A",
			options: ArrayToObjectOptions{
				SkipFields: []string{"priority"},
			},
			expectedKey: "data",
			expectValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Get(tt.inputJSON, tt.path)
			result := TransformDataFieldArrayToObject(tt.inputJSON, tt.path, instance, tt.recordType, tt.options)

			fullPath := tt.path + "." + tt.expectedKey
			value := gjson.Get(result, fullPath)

			if tt.expectValue {
				assert.True(t, value.Exists(), "Expected %s to exist", fullPath)
			} else {
				assert.False(t, value.Exists(), "Expected %s to not exist", fullPath)
			}
		})
	}
}

func TestFlattenArrayField(t *testing.T) {
	tests := []struct {
		name          string
		inputJSON     string
		path          string
		expectedValue interface{}
		expectArray   bool
	}{
		{
			name: "Flatten single-element array",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"priority": [10]
						}
					}]
				}]
			}`,
			path:          "resources.0.instances.0.attributes.priority",
			expectedValue: float64(10),
			expectArray:   false,
		},
		{
			name: "Do not flatten multi-element array",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"tags": ["tag1", "tag2", "tag3"]
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.tags",
			expectArray: true,
		},
		{
			name: "Do not flatten empty array",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"items": []
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.items",
			expectArray: true,
		},
		{
			name: "Handle non-array field",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"priority": 10
						}
					}]
				}]
			}`,
			path:          "resources.0.instances.0.attributes.priority",
			expectedValue: float64(10),
			expectArray:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := gjson.Get(tt.inputJSON, tt.path)
			result := FlattenArrayField(tt.inputJSON, tt.path, field)

			value := gjson.Get(result, tt.path)
			require.True(t, value.Exists())

			if tt.expectArray {
				assert.True(t, value.IsArray(), "Expected value to be an array")
			} else {
				assert.False(t, value.IsArray(), "Expected value to not be an array")
				if tt.expectedValue != nil {
					assert.Equal(t, tt.expectedValue, value.Value())
				}
			}
		})
	}
}
