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
		name     string
		input    string
		oldName  string
		newName  string
		expected bool
		contains string
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
		name           string
		input          string
		attrsToRemove  []string
		expectedCount  int
		shouldContain  []string
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
