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

func TestTransformFieldArrayToObject_EnsureObjectExists(t *testing.T) {
	tests := []struct {
		name            string
		inputJSON       string
		path            string
		fieldName       string
		options         ArrayToObjectOptions
		expectedExists  bool
		expectedIsEmpty bool
		expectedValue   map[string]interface{}
	}{
		{
			name: "EnsureObjectExists creates empty object when field missing",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"name": "Test"
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: true,
			},
			expectedExists:  true,
			expectedIsEmpty: true,
			expectedValue:   map[string]interface{}{},
		},
		{
			name: "EnsureObjectExists creates empty object when field is null",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"name": "Test",
					"config": null
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: true,
			},
			expectedExists:  true,
			expectedIsEmpty: true,
			expectedValue:   map[string]interface{}{},
		},
		{
			name: "EnsureObjectExists creates empty object when field is empty array",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"name": "Test",
					"config": []
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: true,
			},
			expectedExists:  true,
			expectedIsEmpty: true,
			expectedValue:   map[string]interface{}{},
		},
		{
			name: "EnsureObjectExists with DefaultFields creates object with defaults when field missing",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"name": "Test"
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: true,
				DefaultFields: map[string]interface{}{
					"enabled": false,
					"timeout": 30,
				},
			},
			expectedExists:  true,
			expectedIsEmpty: false,
			expectedValue: map[string]interface{}{
				"enabled": false,
				"timeout": int64(30),
			},
		},
		{
			name: "Without EnsureObjectExists, missing field stays missing",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"name": "Test"
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: false,
			},
			expectedExists: false,
		},
		{
			name: "Without EnsureObjectExists, empty array is deleted",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"name": "Test",
					"config": []
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: false,
			},
			expectedExists: false,
		},
		{
			name: "EnsureObjectExists transforms array with SkipFields and RenameFields",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"config": [
						{
							"client_id": "test-client",
							"api_token": "deprecated",
							"idp_public_cert": "CERT123"
						}
					]
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: true,
				SkipFields:         []string{"api_token"},
				RenameFields: map[string]string{
					"idp_public_cert": "idp_public_certs",
				},
			},
			expectedExists:  true,
			expectedIsEmpty: false,
			expectedValue: map[string]interface{}{
				"client_id":       "test-client",
				"idp_public_certs": "CERT123",
			},
		},
		{
			name: "EnsureObjectExists with FieldTransforms",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"config": [
						{
							"cert": "CERT123"
						}
					]
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: true,
				RenameFields: map[string]string{
					"cert": "certs",
				},
				FieldTransforms: map[string]func(gjson.Result) interface{}{
					"certs": func(value gjson.Result) interface{} {
						// Wrap string in array
						return []string{value.String()}
					},
				},
			},
			expectedExists:  true,
			expectedIsEmpty: false,
			expectedValue: map[string]interface{}{
				"certs": []interface{}{"CERT123"},
			},
		},
		{
			name: "EnsureObjectExists preserves existing object and applies transformations",
			inputJSON: `{
				"attributes": {
					"id": "test-id",
					"config": {
						"client_id": "test-client",
						"api_token": "deprecated",
						"enabled": true
					}
				}
			}`,
			path:      "attributes",
			fieldName: "config",
			options: ArrayToObjectOptions{
				EnsureObjectExists: true,
				SkipFields:         []string{"api_token"},
			},
			expectedExists:  true,
			expectedIsEmpty: false,
			expectedValue: map[string]interface{}{
				"client_id": "test-client",
				"enabled":   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Parse(tt.inputJSON).Get(tt.path)
			result := TransformFieldArrayToObject(tt.inputJSON, tt.path, instance, tt.fieldName, tt.options)

			// Check if field exists
			fieldPath := tt.path + "." + tt.fieldName
			fieldValue := gjson.Get(result, fieldPath)

			if tt.expectedExists {
				assert.True(t, fieldValue.Exists(), "Expected %s to exist", fieldPath)
				assert.True(t, fieldValue.IsObject(), "Expected %s to be an object", fieldPath)

				if tt.expectedIsEmpty {
					assert.Equal(t, 0, len(fieldValue.Map()), "Expected %s to be an empty object", fieldPath)
				} else {
					// Check expected value
					actualMap := make(map[string]interface{})
					fieldValue.ForEach(func(key, value gjson.Result) bool {
						actualMap[key.String()] = ConvertGjsonValue(value)
						return true
					})
					assert.Equal(t, tt.expectedValue, actualMap, "Field values don't match")
				}
			} else {
				assert.False(t, fieldValue.Exists(), "Expected %s to not exist", fieldPath)
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
