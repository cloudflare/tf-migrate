package hcl

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureAttribute(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		attrName     string
		defaultValue interface{}
		expected     string
	}{
		{
			name: "Add missing attribute with string value",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrName:     "ttl",
			defaultValue: 1,
			expected:     "ttl",
		},
		{
			name: "Do not overwrite existing attribute",
			input: `
resource "test" "example" {
  name = "test"
  ttl  = 3600
}`,
			attrName:     "ttl",
			defaultValue: 1,
			expected:     "ttl  = 3600",
		},
		{
			name: "Add attribute with boolean value",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrName:     "proxied",
			defaultValue: true,
			expected:     "proxied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			EnsureAttribute(body, tt.attrName, tt.defaultValue)

			output := string(file.Bytes())
			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestRenameAttribute(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		oldName     string
		newName     string
		expected    bool
		contains    string
		notContains string
	}{
		{
			name: "Rename existing attribute",
			input: `
resource "test" "example" {
  name  = "test"
  value = "192.0.2.1"
}`,
			oldName:     "value",
			newName:     "content",
			expected:    true,
			contains:    "content = \"192.0.2.1\"",
			notContains: "value",
		},
		{
			name: "Return false for non-existent attribute",
			input: `
resource "test" "example" {
  name = "test"
}`,
			oldName:     "missing",
			newName:     "other",
			expected:    false,
			notContains: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			result := RenameAttribute(body, tt.oldName, tt.newName)

			assert.Equal(t, tt.expected, result)

			output := string(file.Bytes())
			if tt.contains != "" {
				assert.Contains(t, output, tt.contains)
			}
			if tt.notContains != "" {
				assert.NotContains(t, output, tt.notContains)
			}
		})
	}
}

func TestRemoveAttributes(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		attrsToRemove    []string
		expectedCount    int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Remove multiple attributes",
			input: `
resource "test" "example" {
  name       = "test"
  deprecated = "old"
  obsolete   = "remove"
  keep_this  = "value"
}`,
			attrsToRemove:    []string{"deprecated", "obsolete"},
			expectedCount:    2,
			shouldContain:    []string{"keep_this"},
			shouldNotContain: []string{"deprecated", "obsolete"},
		},
		{
			name: "Remove non-existent attributes",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrsToRemove: []string{"missing1", "missing2"},
			expectedCount: 0,
		},
		{
			name: "Mixed existing and non-existent",
			input: `
resource "test" "example" {
  name    = "test"
  remove_me = "gone"
}`,
			attrsToRemove:    []string{"remove_me", "nonexistent"},
			expectedCount:    1,
			shouldNotContain: []string{"remove_me"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			count := RemoveAttributes(body, tt.attrsToRemove...)

			assert.Equal(t, tt.expectedCount, count)

			output := string(file.Bytes())
			for _, s := range tt.shouldContain {
				assert.Contains(t, output, s)
			}
			for _, s := range tt.shouldNotContain {
				assert.NotContains(t, output, s)
			}
		})
	}
}

func TestExtractStringFromAttribute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		attrName string
		expected string
	}{
		{
			name: "Extract quoted string",
			input: `
resource "test" "example" {
  type = "A"
}`,
			attrName: "type",
			expected: "A",
		},
		{
			name: "Extract simple identifier",
			input: `
resource "test" "example" {
  enabled = var.enabled
}`,
			attrName: "enabled",
			expected: "var",
		},
		{
			name: "Return empty for non-existent attribute",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrName: "missing",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			attr := body.GetAttribute(tt.attrName)

			result := ExtractStringFromAttribute(attr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractStringFromAttribute_Nil(t *testing.T) {
	result := ExtractStringFromAttribute(nil)
	assert.Equal(t, "", result)
}

func TestHasAttribute(t *testing.T) {
	input := `
resource "test" "example" {
  name = "test"
  type = "A"
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()

	assert.True(t, HasAttribute(body, "name"))
	assert.True(t, HasAttribute(body, "type"))
	assert.False(t, HasAttribute(body, "missing"))
}

func TestCopyAndRenameAttribute(t *testing.T) {
	tests := []struct {
		name     string
		fromHCL  string
		toHCL    string
		oldName  string
		newName  string
		expected bool
	}{
		{
			name: "Copy and rename existing attribute",
			fromHCL: `
resource "source" "example" {
  value = "test"
}`,
			toHCL: `
resource "target" "example" {
  name = "target"
}`,
			oldName:  "value",
			newName:  "content",
			expected: true,
		},
		{
			name: "Return false for non-existent attribute",
			fromHCL: `
resource "source" "example" {
  name = "source"
}`,
			toHCL: `
resource "target" "example" {
  name = "target"
}`,
			oldName:  "missing",
			newName:  "content",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fromFile, diags := hclwrite.ParseConfig([]byte(tt.fromHCL), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())
			fromBody := fromFile.Body().Blocks()[0].Body()

			toFile, diags := hclwrite.ParseConfig([]byte(tt.toHCL), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())
			toBody := toFile.Body().Blocks()[0].Body()

			result := CopyAndRenameAttribute(fromBody, toBody, tt.oldName, tt.newName)
			assert.Equal(t, tt.expected, result)

			if tt.expected {
				assert.NotNil(t, toBody.GetAttribute(tt.newName))
			}
		})
	}
}

func TestApplyAttributeRenames(t *testing.T) {
	input := `
resource "test" "example" {
  old_name1 = "value1"
  old_name2 = "value2"
  keep_this = "value3"
  old_name3 = "value4"
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()

	renames := AttributeRenameMap{
		"old_name1": "new_name1",
		"old_name2": "new_name2",
		"old_name3": "new_name3",
		"missing":   "also_missing",
	}

	count := ApplyAttributeRenames(body, renames)
	assert.Equal(t, 3, count, "Should rename 3 existing attributes")

	output := string(file.Bytes())
	assert.Contains(t, output, "new_name1")
	assert.Contains(t, output, "new_name2")
	assert.Contains(t, output, "new_name3")
	assert.Contains(t, output, "keep_this")
	assert.NotContains(t, output, "old_name1")
	assert.NotContains(t, output, "old_name2")
	assert.NotContains(t, output, "old_name3")
}

func TestConditionalRenameAttribute(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		oldName   string
		newName   string
		condition func(*hclwrite.Attribute) bool
		expected  bool
		contains  string
	}{
		{
			name: "Rename when condition is true",
			input: `
resource "test" "example" {
  priority = 10
}`,
			oldName: "priority",
			newName: "weight",
			condition: func(attr *hclwrite.Attribute) bool {
				return true
			},
			expected: true,
			contains: "weight",
		},
		{
			name: "Do not rename when condition is false",
			input: `
resource "test" "example" {
  priority = 10
}`,
			oldName: "priority",
			newName: "weight",
			condition: func(attr *hclwrite.Attribute) bool {
				return false
			},
			expected: false,
			contains: "priority",
		},
		{
			name: "Return false for non-existent attribute",
			input: `
resource "test" "example" {
  name = "test"
}`,
			oldName: "missing",
			newName: "new",
			condition: func(attr *hclwrite.Attribute) bool {
				return true
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			result := ConditionalRenameAttribute(body, tt.oldName, tt.newName, tt.condition)

			assert.Equal(t, tt.expected, result)

			output := string(file.Bytes())
			assert.Contains(t, output, tt.contains)
		})
	}
}

func TestAttributeValueContainsKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		attrName string
		key      string
		expected bool
	}{
		{
			name: "Returns false for nil attribute",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrName: "nonexistent",
			key:      "some_key",
			expected: false,
		},
		{
			name: "Returns false when value is simple identifier",
			input: `
resource "test" "example" {
  value = my_identifier
}`,
			attrName: "value",
			key:      "my_identifier",
			expected: false,
		},
		{
			name: "Returns false when key does not exist in non-object",
			input: `
resource "test" "example" {
  value = some_other_value
}`,
			attrName: "value",
			key:      "my_identifier",
			expected: false,
		},
		{
			name: "Returns false for quoted string",
			input: `
resource "test" "example" {
  value = "my_identifier"
}`,
			attrName: "value",
			key:      "my_identifier",
			expected: false,
		},
		{
			name: "Returns false for dotted notation (not an object)",
			input: `
resource "test" "example" {
  value = var.my_key
}`,
			attrName: "value",
			key:      "var",
			expected: false,
		},
		{
			name: "Returns false for list values",
			input: `
resource "test" "example" {
  values = [first_item, second_item]
}`,
			attrName: "values",
			key:      "second_item",
			expected: false,
		},
		{
			name: "Returns true when key exists as object key",
			input: `
resource "test" "example" {
  config = {
    key = some_value
  }
}`,
			attrName: "config",
			key:      "key",
			expected: true,
		},
		{
			name: "Returns false when checking for value identifier in object",
			input: `
resource "test" "example" {
  config = {
    key = target_identifier
  }
}`,
			attrName: "config",
			key:      "target_identifier",
			expected: false,
		},
		{
			name: "Returns true when multiple keys exist in object",
			input: `
resource "test" "example" {
  config = {
    first_key = "value1"
    second_key = "value2"
    third_key = "value3"
  }
}`,
			attrName: "config",
			key:      "second_key",
			expected: true,
		},
		{
			name: "Returns false when key does not exist in object",
			input: `
resource "test" "example" {
  config = {
    first_key = "value1"
    second_key = "value2"
  }
}`,
			attrName: "config",
			key:      "missing_key",
			expected: false,
		},
		{
			name: "Returns false for numeric value",
			input: `
resource "test" "example" {
  count = 5
}`,
			attrName: "count",
			key:      "5",
			expected: false,
		},
		{
			name: "Returns false for boolean value",
			input: `
resource "test" "example" {
  enabled = true
}`,
			attrName: "enabled",
			key:      "true",
			expected: false,
		},
		{
			name: "Returns false for function call",
			input: `
resource "test" "example" {
  value = lookup(my_map, "key")
}`,
			attrName: "value",
			key:      "lookup",
			expected: false,
		},
		{
			name: "Returns true for nested object with matching key",
			input: `
resource "test" "example" {
  config = {
    nested = {
      inner_key = "value"
    }
  }
}`,
			attrName: "config",
			key:      "nested",
			expected: true,
		},
		{
			name: "Returns false for key in deeply nested object",
			input: `
resource "test" "example" {
  config = {
    nested = {
      inner_key = "value"
    }
  }
}`,
			attrName: "config",
			key:      "inner_key",
			expected: false,
		},
		{
			name: "Returns false for empty object",
			input: `
resource "test" "example" {
  config = {}
}`,
			attrName: "config",
			key:      "any_key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			attr := body.GetAttribute(tt.attrName)

			result := AttributeValueContainsKey(attr, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateNestedAttributeFromFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		attrName string
		fields   map[string]string // fieldName -> value (as string for easy testing)
		expected []string          // strings that should appear in output
	}{
		{
			name: "Create nested attribute from simple fields",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrName: "http_config",
			fields: map[string]string{
				"port":   "80",
				"path":   `"/health"`,
				"method": `"GET"`,
			},
			expected: []string{
				"http_config = {",
				`method = "GET"`,
				`path   = "/health"`,
				"port   = 80",
			},
		},
		{
			name: "Create nested attribute with complex values",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrName: "config",
			fields: map[string]string{
				"enabled": "true",
				"codes":   `["200", "201"]`,
				"timeout": "30",
			},
			expected: []string{
				"config = {",
				`codes   = ["200", "201"]`,
				"enabled = true",
				"timeout = 30",
			},
		},
		{
			name: "Do nothing with empty fields map",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrName: "empty_config",
			fields:   map[string]string{},
			expected: []string{
				"name = \"test\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()

			// Convert string values to tokens for the fields map
			tokensMap := make(map[string]hclwrite.Tokens)
			for fieldName, value := range tt.fields {
				// Parse the value string to tokens
				valueFile, _ := hclwrite.ParseConfig([]byte("dummy = "+value), "", hcl.InitialPos)
				if attr := valueFile.Body().GetAttribute("dummy"); attr != nil {
					tokensMap[fieldName] = attr.Expr().BuildTokens(nil)
				}
			}

			CreateNestedAttributeFromFields(body, tt.attrName, tokensMap)

			output := string(file.Bytes())
			for _, expected := range tt.expected {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestMoveAttributesToNestedObject(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		nestedAttrName   string
		fieldNames       []string
		expectedMoved    int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Move HTTP fields into http_config",
			input: `
resource "cloudflare_healthcheck" "example" {
  zone_id = "abc123"
  type    = "HTTP"
  port    = 80
  path    = "/health"
  method  = "GET"
}`,
			nestedAttrName: "http_config",
			fieldNames:     []string{"port", "path", "method"},
			expectedMoved:  3,
			shouldContain: []string{
				"http_config = {",
				`method = "GET"`,
				`path   = "/health"`,
				"port   = 80",
				"zone_id = \"abc123\"",
				`type    = "HTTP"`,
			},
			shouldNotContain: []string{
				"port    = 80\n  path",    // Should not be at root level
				"method  = \"GET\"\n  zone", // Should not be at root level
			},
		},
		{
			name: "Move TCP fields into tcp_config",
			input: `
resource "cloudflare_healthcheck" "example" {
  zone_id = "abc123"
  type    = "TCP"
  port    = 8080
  method  = "connection_established"
}`,
			nestedAttrName: "tcp_config",
			fieldNames:     []string{"port", "method"},
			expectedMoved:  2,
			shouldContain: []string{
				"tcp_config = {",
				`method = "connection_established"`,
				"port   = 8080",
			},
			shouldNotContain: []string{
				"port    = 8080\n  method",
			},
		},
		{
			name: "Move subset of fields",
			input: `
resource "test" "example" {
  name   = "test"
  field1 = "value1"
  field2 = "value2"
  field3 = "value3"
}`,
			nestedAttrName: "config",
			fieldNames:     []string{"field1", "field3"},
			expectedMoved:  2,
			shouldContain: []string{
				"config = {",
				`field1 = "value1"`,
				`field3 = "value3"`,
				`field2 = "value2"`, // Should remain at root level
			},
		},
		{
			name: "Move no fields when none exist",
			input: `
resource "test" "example" {
  name = "test"
}`,
			nestedAttrName: "config",
			fieldNames:     []string{"missing1", "missing2"},
			expectedMoved:  0,
			shouldContain: []string{
				`name = "test"`,
			},
			shouldNotContain: []string{
				"config = {",
			},
		},
		{
			name: "Move only existing fields",
			input: `
resource "test" "example" {
  name    = "test"
  present = "value"
}`,
			nestedAttrName: "config",
			fieldNames:     []string{"present", "missing"},
			expectedMoved:  1,
			shouldContain: []string{
				"config = {",
				`present = "value"`,
			},
			shouldNotContain: []string{
				"missing",
				"present = \"value\"\n  name", // Should not be at root level
			},
		},
		{
			name: "Move fields with complex values",
			input: `
resource "test" "example" {
  name           = "test"
  expected_codes = ["200", "201", "204"]
  header_map     = {
    "Host" = ["example.com"]
  }
}`,
			nestedAttrName: "http_config",
			fieldNames:     []string{"expected_codes", "header_map"},
			expectedMoved:  2,
			shouldContain: []string{
				"http_config = {",
				`expected_codes = ["200", "201", "204"]`,
				`header_map = {`, // Whitespace alignment may vary
				`"Host" = ["example.com"]`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			count := MoveAttributesToNestedObject(body, tt.nestedAttrName, tt.fieldNames)

			assert.Equal(t, tt.expectedMoved, count, "Should move %d attributes", tt.expectedMoved)

			output := string(file.Bytes())
			for _, expected := range tt.shouldContain {
				assert.Contains(t, output, expected)
			}
			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, output, notExpected)
			}

			// Verify moved fields are removed from root level
			if tt.expectedMoved > 0 {
				for _, fieldName := range tt.fieldNames {
					if body.GetAttribute(fieldName) != nil {
						t.Errorf("Field %s should have been removed from root level", fieldName)
					}
				}
			}
		})
	}
}
