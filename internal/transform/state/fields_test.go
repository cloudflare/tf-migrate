package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEnsureField(t *testing.T) {
	tests := []struct {
		name         string
		inputJSON    string
		path         string
		field        string
		defaultValue interface{}
		shouldAdd    bool
	}{
		{
			name: "Add missing field with default value",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"name": "test",
							"type": "A",
							"content": "192.0.2.1"
						}
					}]
				}]
			}`,
			path:         "resources.0.instances.0.attributes",
			field:        "ttl",
			defaultValue: float64(1),
			shouldAdd:    true,
		},
		{
			name: "Do not overwrite existing field",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"name": "test",
							"ttl": 3600
						}
					}]
				}]
			}`,
			path:         "resources.0.instances.0.attributes",
			field:        "ttl",
			defaultValue: 1,
			shouldAdd:    false,
		},
		{
			name: "Add boolean default value",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123"
						}
					}]
				}]
			}`,
			path:         "resources.0.instances.0.attributes",
			field:        "proxied",
			defaultValue: true,
			shouldAdd:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Get(tt.inputJSON, tt.path)
			result := EnsureField(tt.inputJSON, tt.path, instance, tt.field, tt.defaultValue)

			fullPath := tt.path + "." + tt.field
			value := gjson.Get(result, fullPath)

			require.True(t, value.Exists())
			if tt.shouldAdd {
				assert.Equal(t, tt.defaultValue, value.Value())
			}
		})
	}
}

func TestRenameField(t *testing.T) {
	tests := []struct {
		name        string
		inputJSON   string
		path        string
		oldName     string
		newName     string
		expectOld   bool
		expectNew   bool
		expectValue interface{}
	}{
		{
			name: "Rename existing field",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"name": "test",
							"type": "A",
							"value": "192.0.2.1"
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes",
			oldName:     "value",
			newName:     "content",
			expectOld:   false,
			expectNew:   true,
			expectValue: "192.0.2.1",
		},
		{
			name: "Remove old field when both exist",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"value": "192.0.2.1",
							"content": "203.0.113.1"
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes",
			oldName:     "value",
			newName:     "content",
			expectOld:   false,
			expectNew:   true,
			expectValue: "203.0.113.1",
		},
		{
			name: "Do nothing when old field doesn't exist",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"content": "192.0.2.1"
						}
					}]
				}]
			}`,
			path:      "resources.0.instances.0.attributes",
			oldName:   "value",
			newName:   "content",
			expectOld: false,
			expectNew: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Get(tt.inputJSON, tt.path)
			result := RenameField(tt.inputJSON, tt.path, instance, tt.oldName, tt.newName)

			oldPath := tt.path + "." + tt.oldName
			newPath := tt.path + "." + tt.newName

			oldValue := gjson.Get(result, oldPath)
			newValue := gjson.Get(result, newPath)

			assert.Equal(t, tt.expectOld, oldValue.Exists(), "Old field existence mismatch")
			assert.Equal(t, tt.expectNew, newValue.Exists(), "New field existence mismatch")

			if tt.expectValue != nil && newValue.Exists() {
				assert.Equal(t, tt.expectValue, newValue.Value())
			}
		})
	}
}

func TestRemoveFields(t *testing.T) {
	tests := []struct {
		name           string
		inputJSON      string
		path           string
		fieldsToRemove []string
		shouldExist    []string
		shouldNotExist []string
	}{
		{
			name: "Remove multiple fields",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"name": "test",
							"deprecated": "old",
							"obsolete": "remove",
							"keep_this": "value"
						}
					}]
				}]
			}`,
			path:           "resources.0.instances.0.attributes",
			fieldsToRemove: []string{"deprecated", "obsolete"},
			shouldExist:    []string{"zone_id", "name", "keep_this"},
			shouldNotExist: []string{"deprecated", "obsolete"},
		},
		{
			name: "Remove non-existent fields",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"name": "test"
						}
					}]
				}]
			}`,
			path:           "resources.0.instances.0.attributes",
			fieldsToRemove: []string{"missing1", "missing2"},
			shouldExist:    []string{"zone_id", "name"},
			shouldNotExist: []string{"missing1", "missing2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Get(tt.inputJSON, tt.path)
			result := RemoveFields(tt.inputJSON, tt.path, instance, tt.fieldsToRemove...)

			for _, field := range tt.shouldExist {
				fullPath := tt.path + "." + field
				value := gjson.Get(result, fullPath)
				assert.True(t, value.Exists(), "Expected %s to exist", field)
			}

			for _, field := range tt.shouldNotExist {
				fullPath := tt.path + "." + field
				value := gjson.Get(result, fullPath)
				assert.False(t, value.Exists(), "Expected %s to not exist", field)
			}
		})
	}
}

func TestCleanupEmptyField(t *testing.T) {
	tests := []struct {
		name        string
		inputJSON   string
		path        string
		shouldExist bool
	}{
		{
			name: "Remove empty object",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"meta": {}
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.meta",
			shouldExist: false,
		},
		{
			name: "Remove empty array",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"settings": []
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.settings",
			shouldExist: false,
		},
		{
			name: "Remove empty string",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"description": ""
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.description",
			shouldExist: false,
		},
		{
			name: "Keep non-empty object",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"meta": {
								"key": "value"
							}
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.meta",
			shouldExist: true,
		},
		{
			name: "Keep non-empty array",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"tags": ["tag1", "tag2"]
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.tags",
			shouldExist: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := gjson.Get(tt.inputJSON, tt.path)
			result := CleanupEmptyField(tt.inputJSON, tt.path, field)

			value := gjson.Get(result, tt.path)
			assert.Equal(t, tt.shouldExist, value.Exists())
		})
	}
}

func TestRemoveObjectIfAllNull(t *testing.T) {
	tests := []struct {
		name        string
		inputJSON   string
		path        string
		fields      []string
		shouldExist bool
	}{
		{
			name: "Remove object with all null fields",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"settings": {
								"flatten_cname": null,
								"ipv4_only": null,
								"ipv6_only": null
							}
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.settings",
			fields:      []string{"flatten_cname", "ipv4_only", "ipv6_only"},
			shouldExist: false,
		},
		{
			name: "Keep object with at least one non-null field",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"settings": {
								"flatten_cname": null,
								"ipv4_only": true,
								"ipv6_only": null
							}
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.settings",
			fields:      []string{"flatten_cname", "ipv4_only", "ipv6_only"},
			shouldExist: true,
		},
		{
			name: "Handle non-existent object",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123"
						}
					}]
				}]
			}`,
			path:        "resources.0.instances.0.attributes.settings",
			fields:      []string{"field1", "field2"},
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := gjson.Get(tt.inputJSON, tt.path)
			result := RemoveObjectIfAllNull(tt.inputJSON, tt.path, obj, tt.fields)

			value := gjson.Get(result, tt.path)
			assert.Equal(t, tt.shouldExist, value.Exists())
		})
	}
}

func TestEnsureTimestamps(t *testing.T) {
	tests := []struct {
		name              string
		inputJSON         string
		path              string
		defaultTime       string
		expectCreated     string
		expectModified    string
		createdShouldAdd  bool
		modifiedShouldAdd bool
	}{
		{
			name: "Add both timestamps when missing",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"name": "test"
						}
					}]
				}]
			}`,
			path:              "resources.0.instances.0.attributes",
			defaultTime:       "2024-01-01T00:00:00Z",
			expectCreated:     "2024-01-01T00:00:00Z",
			expectModified:    "2024-01-01T00:00:00Z",
			createdShouldAdd:  true,
			modifiedShouldAdd: true,
		},
		{
			name: "Add modified_on matching created_on",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"name": "test",
							"created_on": "2023-12-01T10:00:00Z"
						}
					}]
				}]
			}`,
			path:              "resources.0.instances.0.attributes",
			defaultTime:       "2024-01-01T00:00:00Z",
			expectCreated:     "2023-12-01T10:00:00Z",
			expectModified:    "2023-12-01T10:00:00Z",
			createdShouldAdd:  false,
			modifiedShouldAdd: true,
		},
		{
			name: "Do not overwrite existing timestamps",
			inputJSON: `{
				"resources": [{
					"instances": [{
						"attributes": {
							"zone_id": "abc123",
							"name": "test",
							"created_on": "2023-12-01T10:00:00Z",
							"modified_on": "2023-12-15T15:30:00Z"
						}
					}]
				}]
			}`,
			path:              "resources.0.instances.0.attributes",
			defaultTime:       "2024-01-01T00:00:00Z",
			expectCreated:     "2023-12-01T10:00:00Z",
			expectModified:    "2023-12-15T15:30:00Z",
			createdShouldAdd:  false,
			modifiedShouldAdd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Get(tt.inputJSON, tt.path)
			result := EnsureTimestamps(tt.inputJSON, tt.path, instance, tt.defaultTime)

			createdPath := tt.path + ".created_on"
			modifiedPath := tt.path + ".modified_on"

			createdValue := gjson.Get(result, createdPath)
			modifiedValue := gjson.Get(result, modifiedPath)

			require.True(t, createdValue.Exists())
			require.True(t, modifiedValue.Exists())

			assert.Equal(t, tt.expectCreated, createdValue.String())
			assert.Equal(t, tt.expectModified, modifiedValue.String())
		})
	}
}

func TestConvertGjsonValue(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		expected interface{}
	}{
		{
			name:     "Convert integer",
			json:     `{"value": 42}`,
			path:     "value",
			expected: int64(42),
		},
		{
			name:     "Convert float",
			json:     `{"value": 3.14}`,
			path:     "value",
			expected: 3.14,
		},
		{
			name:     "Convert string",
			json:     `{"value": "test"}`,
			path:     "value",
			expected: "test",
		},
		{
			name:     "Convert true",
			json:     `{"value": true}`,
			path:     "value",
			expected: true,
		},
		{
			name:     "Convert false",
			json:     `{"value": false}`,
			path:     "value",
			expected: false,
		},
		{
			name:     "Convert null",
			json:     `{"value": null}`,
			path:     "value",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Get(tt.json, tt.path)
			result := ConvertGjsonValue(value)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertGjsonToJSON(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		path         string
		expectedType string
		expectNumber bool
	}{
		{
			name:         "Convert number preserving type",
			json:         `{"value": 42}`,
			path:         "value",
			expectedType: "json.Number",
			expectNumber: true,
		},
		{
			name:         "Convert float preserving type",
			json:         `{"value": 3.14159}`,
			path:         "value",
			expectedType: "json.Number",
			expectNumber: true,
		},
		{
			name:         "Convert string",
			json:         `{"value": "test"}`,
			path:         "value",
			expectedType: "string",
			expectNumber: false,
		},
		{
			name:         "Convert boolean",
			json:         `{"value": true}`,
			path:         "value",
			expectedType: "bool",
			expectNumber: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := gjson.Get(tt.json, tt.path)
			result := ConvertGjsonToJSON(value)

			if tt.expectNumber {
				_, ok := result.(json.Number)
				assert.True(t, ok, "Expected result to be json.Number")
			} else {
				switch tt.expectedType {
				case "string":
					_, ok := result.(string)
					assert.True(t, ok, "Expected result to be string")
				case "bool":
					_, ok := result.(bool)
					assert.True(t, ok, "Expected result to be bool")
				}
			}
		})
	}
}

func TestGetResourceAttribute(t *testing.T) {
	tests := []struct {
		name          string
		stateJSON     string
		resourceType  string
		resourceName  string
		attributeName string
		expected      string
	}{
		{
			name: "Get string attribute from existing resource",
			stateJSON: `{
				"resources": [
					{
						"type": "cloudflare_tiered_cache",
						"name": "example",
						"instances": [
							{
								"attributes": {
									"zone_id": "test-zone-id",
									"cache_type": "smart",
									"id": "test-id"
								}
							}
						]
					}
				]
			}`,
			resourceType:  "cloudflare_tiered_cache",
			resourceName:  "example",
			attributeName: "cache_type",
			expected:      "smart",
		},
		{
			name: "Get attribute from resource with multiple instances",
			stateJSON: `{
				"resources": [
					{
						"type": "cloudflare_dns_record",
						"name": "test",
						"instances": [
							{
								"attributes": {
									"zone_id": "zone1",
									"name": "test",
									"type": "A",
									"content": "192.0.2.1"
								}
							},
							{
								"attributes": {
									"zone_id": "zone2",
									"name": "test2",
									"type": "AAAA"
								}
							}
						]
					}
				]
			}`,
			resourceType:  "cloudflare_dns_record",
			resourceName:  "test",
			attributeName: "type",
			expected:      "A", // Should get first instance
		},
		{
			name: "Resource does not exist",
			stateJSON: `{
				"resources": [
					{
						"type": "cloudflare_tiered_cache",
						"name": "example",
						"instances": [
							{
								"attributes": {
									"zone_id": "test-zone-id",
									"cache_type": "smart"
								}
							}
						]
					}
				]
			}`,
			resourceType:  "cloudflare_tiered_cache",
			resourceName:  "nonexistent",
			attributeName: "cache_type",
			expected:      "",
		},
		{
			name: "Attribute does not exist",
			stateJSON: `{
				"resources": [
					{
						"type": "cloudflare_tiered_cache",
						"name": "example",
						"instances": [
							{
								"attributes": {
									"zone_id": "test-zone-id",
									"cache_type": "smart"
								}
							}
						]
					}
				]
			}`,
			resourceType:  "cloudflare_tiered_cache",
			resourceName:  "example",
			attributeName: "nonexistent_attr",
			expected:      "",
		},
		{
			name: "Empty state JSON",
			stateJSON: `{
				"resources": []
			}`,
			resourceType:  "cloudflare_tiered_cache",
			resourceName:  "example",
			attributeName: "cache_type",
			expected:      "",
		},
		{
			name:          "Empty string as state",
			stateJSON:     "",
			resourceType:  "cloudflare_tiered_cache",
			resourceName:  "example",
			attributeName: "cache_type",
			expected:      "",
		},
		{
			name: "Multiple resources, find correct one",
			stateJSON: `{
				"resources": [
					{
						"type": "cloudflare_tiered_cache",
						"name": "first",
						"instances": [
							{
								"attributes": {
									"cache_type": "off"
								}
							}
						]
					},
					{
						"type": "cloudflare_tiered_cache",
						"name": "second",
						"instances": [
							{
								"attributes": {
									"cache_type": "smart"
								}
							}
						]
					},
					{
						"type": "cloudflare_tiered_cache",
						"name": "third",
						"instances": [
							{
								"attributes": {
									"cache_type": "generic"
								}
							}
						]
					}
				]
			}`,
			resourceType:  "cloudflare_tiered_cache",
			resourceName:  "second",
			attributeName: "cache_type",
			expected:      "smart",
		},
		{
			name: "Get nested attribute path",
			stateJSON: `{
				"resources": [
					{
						"type": "cloudflare_load_balancer",
						"name": "example",
						"instances": [
							{
								"attributes": {
									"zone_id": "test-zone-id",
									"id": "test-id"
								}
							}
						]
					}
				]
			}`,
			resourceType:  "cloudflare_load_balancer",
			resourceName:  "example",
			attributeName: "zone_id",
			expected:      "test-zone-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetResourceAttribute(tt.stateJSON, tt.resourceType, tt.resourceName, tt.attributeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
