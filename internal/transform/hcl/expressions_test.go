package hcl

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveFunctionWrapper(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		attrName    string
		funcName    string
		expected    string
		notContains string
	}{
		{
			name: "Remove toset from list attribute",
			input: `
resource "test" "example" {
  allowed_idps = toset(["abc-123", "def-456"])
}`,
			attrName:    "allowed_idps",
			funcName:    "toset",
			expected:    `allowed_idps = ["abc-123", "def-456"]`,
			notContains: "toset",
		},
		{
			name: "Remove toset from empty list",
			input: `
resource "test" "example" {
  custom_pages = toset([])
}`,
			attrName:    "custom_pages",
			funcName:    "toset",
			expected:    `custom_pages = []`,
			notContains: "toset",
		},
		{
			name: "Remove toset from single element list",
			input: `
resource "test" "example" {
  scopes = toset(["openid"])
}`,
			attrName:    "scopes",
			funcName:    "toset",
			expected:    `scopes = ["openid"]`,
			notContains: "toset",
		},
		{
			name: "Remove toset with multi-line formatting",
			input: `
resource "test" "example" {
  allowed_idps = toset([
    "abc-123",
    "def-456",
    "ghi-789"
  ])
}`,
			attrName:    "allowed_idps",
			funcName:    "toset",
			expected:    `allowed_idps = [
    "abc-123",
    "def-456",
    "ghi-789"
  ]`,
			notContains: "toset",
		},
		{
			name: "No-op when attribute doesn't exist",
			input: `
resource "test" "example" {
  name = "test"
}`,
			attrName:    "missing_attr",
			funcName:    "toset",
			expected:    `name = "test"`,
			notContains: "",
		},
		{
			name: "No-op when function doesn't match",
			input: `
resource "test" "example" {
  value = tonumber("123")
}`,
			attrName:    "value",
			funcName:    "toset",
			expected:    `value = tonumber("123")`,
			notContains: "",
		},
		{
			name: "Remove toset with complex list elements",
			input: `
resource "test" "example" {
  domains = toset(["example.com", "test.example.com", "api.example.com"])
}`,
			attrName:    "domains",
			funcName:    "toset",
			expected:    `domains = ["example.com", "test.example.com", "api.example.com"]`,
			notContains: "toset",
		},
		{
			name: "Preserve other attributes unchanged",
			input: `
resource "test" "example" {
  name         = "test"
  allowed_idps = toset(["abc-123"])
  type         = "self_hosted"
}`,
			attrName:    "allowed_idps",
			funcName:    "toset",
			expected:    `allowed_idps = ["abc-123"]`,
			notContains: "toset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "Input HCL should parse without errors")

			body := file.Body().Blocks()[0].Body()
			RemoveFunctionWrapper(body, tt.attrName, tt.funcName)

			output := string(file.Bytes())
			assert.Contains(t, output, tt.expected, "Output should contain expected attribute")

			if tt.notContains != "" {
				assert.NotContains(t, output, tt.notContains, "Output should not contain unwrapped function")
			}
		})
	}
}

func TestRemoveFunctionWrapper_MultipleAttributes(t *testing.T) {
	input := `
resource "test" "example" {
  allowed_idps        = toset(["abc-123", "def-456"])
  custom_pages        = toset(["page-1", "page-2"])
  self_hosted_domains = toset(["app.example.com"])
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()

	// Remove toset from all three attributes
	RemoveFunctionWrapper(body, "allowed_idps", "toset")
	RemoveFunctionWrapper(body, "custom_pages", "toset")
	RemoveFunctionWrapper(body, "self_hosted_domains", "toset")

	output := string(file.Bytes())

	// Verify all three attributes were transformed
	assert.Contains(t, output, `allowed_idps        = ["abc-123", "def-456"]`)
	assert.Contains(t, output, `custom_pages        = ["page-1", "page-2"]`)
	assert.Contains(t, output, `self_hosted_domains = ["app.example.com"]`)

	// Verify no toset remains
	assert.NotContains(t, output, "toset")
}

func TestRemoveFunctionWrapper_NestedBlock(t *testing.T) {
	input := `
resource "test" "example" {
  name = "test"

  saas_app {
    scopes = toset(["openid", "email", "profile"])
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	// Get the nested block body
	resourceBody := file.Body().Blocks()[0].Body()
	saasAppBlock := resourceBody.Blocks()[0]
	saasAppBody := saasAppBlock.Body()

	RemoveFunctionWrapper(saasAppBody, "scopes", "toset")

	output := string(file.Bytes())

	assert.Contains(t, output, `scopes = ["openid", "email", "profile"]`)
	assert.NotContains(t, output, "toset")
}
