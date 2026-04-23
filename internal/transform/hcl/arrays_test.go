package hcl

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// normalizeHCL normalizes HCL output for comparison by standardizing formatting
func normalizeHCL(s string) string {
	// Trim leading/trailing whitespace
	s = strings.TrimSpace(s)
	// Standardize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Remove empty lines (consecutive newlines)
	for strings.Contains(s, "\n\n") {
		s = strings.ReplaceAll(s, "\n\n", "\n")
	}
	// Normalize bracket indentation: both "  }, {" and "    }, {" become "  }, {"
	s = strings.ReplaceAll(s, "    }, {", "  }, {")
	return s
}

// TestParseArrayAttribute tests the ParseArrayAttribute function
func TestParseArrayAttribute(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCount  int
		expectedTypes  []string
		expectedValues []string // for string arrays
	}{
		{
			name:           "Simple string array",
			input:          `items = ["value1", "value2", "value3"]`,
			expectedCount:  3,
			expectedTypes:  []string{"string", "string", "string"},
			expectedValues: []string{"value1", "value2", "value3"},
		},
		{
			name:          "Empty array",
			input:         `items = []`,
			expectedCount: 0,
		},
		{
			name:          "Number array",
			input:         `ports = [80, 443, 8080]`,
			expectedCount: 3,
			expectedTypes: []string{"number", "number", "number"},
		},
		{
			name:          "Boolean array",
			input:         `flags = [true, false, true]`,
			expectedCount: 3,
			expectedTypes: []string{"bool", "bool", "bool"},
		},
		{
			name:          "Object array",
			input:         `items = [{ value = "test1" }, { value = "test2" }]`,
			expectedCount: 2,
			expectedTypes: []string{"object", "object"},
		},
		{
			name:          "Mixed content object array",
			input:         `items = [{ value = "test1", description = "desc1" }, { value = "test2" }]`,
			expectedCount: 2,
			expectedTypes: []string{"object", "object"},
		},
		{
			name:           "Single element array",
			input:          `items = ["single"]`,
			expectedCount:  1,
			expectedTypes:  []string{"string"},
			expectedValues: []string{"single"},
		},
		{
			name:          "Nil attribute",
			input:         "",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input == "" {
				// Test nil case
				elements := ParseArrayAttribute(nil)
				assert.Nil(t, elements)
				return
			}

			// Parse the input HCL
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "Failed to parse test input: %s", diags.Error())

			body := file.Body()
			attr := body.GetAttribute("items")
			if attr == nil {
				attr = body.GetAttribute("ports")
			}
			if attr == nil {
				attr = body.GetAttribute("flags")
			}
			require.NotNil(t, attr, "Failed to get attribute from parsed HCL")

			// Parse the array
			elements := ParseArrayAttribute(attr)

			// Verify count
			assert.Equal(t, tt.expectedCount, len(elements), "Unexpected number of elements")

			// Verify types
			if len(tt.expectedTypes) > 0 {
				for i, expectedType := range tt.expectedTypes {
					if i < len(elements) {
						assert.Equal(t, expectedType, elements[i].Type, "Element %d has unexpected type", i)
					}
				}
			}

			// Verify string values if provided
			if len(tt.expectedValues) > 0 {
				for i, expectedValue := range tt.expectedValues {
					if i < len(elements) && elements[i].Type == "string" {
						actualValue := ExtractStringFromElement(elements[i])
						assert.Equal(t, expectedValue, actualValue, "Element %d has unexpected value", i)
					}
				}
			}
		})
	}
}

// TestDetermineElementType tests the determineElementType function
func TestDetermineElementType(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType string
	}{
		{
			name:         "String literal",
			input:        `items = ["test"]`,
			expectedType: "string",
		},
		{
			name:         "Number literal",
			input:        `items = [42]`,
			expectedType: "number",
		},
		{
			name:         "Boolean true",
			input:        `items = [true]`,
			expectedType: "bool",
		},
		{
			name:         "Boolean false",
			input:        `items = [false]`,
			expectedType: "bool",
		},
		{
			name:         "Object",
			input:        `items = [{ key = "value" }]`,
			expectedType: "object",
		},
		{
			name:         "Reference",
			input:        `items = [var.test]`,
			expectedType: "reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body()
			attr := body.GetAttribute("items")
			require.NotNil(t, attr)

			elements := ParseArrayAttribute(attr)
			require.Greater(t, len(elements), 0, "Expected at least one element")

			assert.Equal(t, tt.expectedType, elements[0].Type)
		})
	}
}

// TestMergeAttributeAndBlocksToObjectArray tests the main generic function
func TestMergeAttributeAndBlocksToObjectArray(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		arrayAttrName   string
		blockType       string
		outputAttrName  string
		primaryField    string
		optionalFields  []string
		blocksFirst     bool
		expectedOutput  string
		expectModified  bool
	}{
		{
			name: "Merge items array and items_with_description blocks (blocks first)",
			input: `
resource "test" "example" {
  items = ["val1", "val2"]

  items_with_description {
    value       = "val3"
    description = "desc3"
  }
}`,
			arrayAttrName:  "items",
			blockType:      "items_with_description",
			outputAttrName: "items",
			primaryField:   "value",
			optionalFields: []string{"description"},
			blocksFirst:    true,
			expectedOutput: `resource "test" "example" {
  items = [{
    description = "desc3"
    value       = "val3"
  }, {
    description = null
    value       = "val1"
  }, {
    description = null
    value       = "val2"
  }]
}
`,
			expectModified: true,
		},
		{
			name: "Merge with array first",
			input: `
resource "test" "example" {
  tags = ["tag1", "tag2"]

  tag_with_metadata {
    name     = "tag3"
    metadata = "meta3"
  }
}`,
			arrayAttrName:  "tags",
			blockType:      "tag_with_metadata",
			outputAttrName: "tags",
			primaryField:   "name",
			optionalFields: []string{"metadata"},
			blocksFirst:    false,
			expectedOutput: `resource "test" "example" {
  tags = [{
    metadata = null
    name     = "tag1"
  }, {
    metadata = null
    name     = "tag2"
  }, {
    metadata = "meta3"
    name     = "tag3"
  }]
}
`,
			expectModified: true,
		},
		{
			name: "Only array attribute",
			input: `
resource "test" "example" {
  items = ["val1", "val2"]
}`,
			arrayAttrName:  "items",
			blockType:      "items_with_description",
			outputAttrName: "items",
			primaryField:   "value",
			optionalFields: []string{"description"},
			blocksFirst:    true,
			expectedOutput: `resource "test" "example" {
  items = [{
    description = null
    value       = "val1"
  }, {
    description = null
    value       = "val2"
  }]
}
`,
			expectModified: true,
		},
		{
			name: "Only blocks",
			input: `
resource "test" "example" {
  items_with_description {
    value       = "val1"
    description = "desc1"
  }

  items_with_description {
    value       = "val2"
    description = "desc2"
  }
}`,
			arrayAttrName:  "items",
			blockType:      "items_with_description",
			outputAttrName: "items",
			primaryField:   "value",
			optionalFields: []string{"description"},
			blocksFirst:    true,
			expectedOutput: `resource "test" "example" {
  items = [{
    description = "desc1"
    value       = "val1"
  }, {
    description = "desc2"
    value       = "val2"
  }]
}
`,
			expectModified: true,
		},
		{
			name: "Multiple optional fields",
			input: `
resource "test" "example" {
  rules = ["rule1"]

  rule_with_priority {
    expression  = "rule2"
    priority    = "high"
    description = "Important rule"
  }
}`,
			arrayAttrName:  "rules",
			blockType:      "rule_with_priority",
			outputAttrName: "rules",
			primaryField:   "expression",
			optionalFields: []string{"priority", "description"},
			blocksFirst:    true,
			expectedOutput: `resource "test" "example" {
  rules = [{
    description = "Important rule"
    expression  = "rule2"
    priority    = "high"
  }, {
    description = null
    expression  = "rule1"
    priority    = null
  }]
}
`,
			expectModified: true,
		},
		{
			name: "Empty - no array or blocks",
			input: `
resource "test" "example" {
  name = "test"
}`,
			arrayAttrName:  "items",
			blockType:      "items_with_description",
			outputAttrName: "items",
			primaryField:   "value",
			optionalFields: []string{"description"},
			blocksFirst:    true,
			expectedOutput: `resource "test" "example" {
  name = "test"
}
`,
			expectModified: false,
		},
		{
			name: "Blocks with missing optional fields",
			input: `
resource "test" "example" {
  items_with_description {
    value = "val1"
  }

  items_with_description {
    value       = "val2"
    description = "has description"
  }
}`,
			arrayAttrName:  "items",
			blockType:      "items_with_description",
			outputAttrName: "items",
			primaryField:   "value",
			optionalFields: []string{"description"},
			blocksFirst:    true,
			expectedOutput: `resource "test" "example" {
  items = [{
    description = null
    value       = "val1"
  }, {
    description = "has description"
    value       = "val2"
  }]
}
`,
			expectModified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input HCL
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "Failed to parse test input: %s", diags.Error())

			body := file.Body().Blocks()[0].Body()

			// Perform the merge
			modified := MergeAttributeAndBlocksToObjectArray(
				body,
				tt.arrayAttrName,
				tt.blockType,
				tt.outputAttrName,
				tt.primaryField,
				tt.optionalFields,
				tt.blocksFirst,
			)

			// Verify modification flag
			assert.Equal(t, tt.expectModified, modified, "Unexpected modification flag")

			// Verify output
			actual := string(file.Bytes())
			assert.Equal(t, normalizeHCL(tt.expectedOutput), normalizeHCL(actual), "Output doesn't match expected")
		})
	}
}

// TestExtractStringFieldFromBlock tests the private helper function
func TestExtractStringFieldFromBlock(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		fieldName     string
		expectedValue string
		expectedOk    bool
	}{
		{
			name: "Extract existing field",
			input: `
block "test" {
  value = "test_value"
}`,
			fieldName:     "value",
			expectedValue: "test_value",
			expectedOk:    true,
		},
		{
			name: "Extract non-existent field",
			input: `
block "test" {
  other = "value"
}`,
			fieldName:     "missing",
			expectedValue: "",
			expectedOk:    false,
		},
		{
			name: "Extract from block with multiple fields",
			input: `
block "test" {
  field1 = "value1"
  field2 = "value2"
  field3 = "value3"
}`,
			fieldName:     "field2",
			expectedValue: "value2",
			expectedOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			blocks := file.Body().Blocks()
			require.Greater(t, len(blocks), 0)

			block := blocks[0]
			value, ok := extractStringFieldFromBlock(block, tt.fieldName)

			assert.Equal(t, tt.expectedOk, ok, "Unexpected ok value")
			assert.Equal(t, tt.expectedValue, value, "Unexpected extracted value")
		})
	}
}

// TestExtractStringFieldFromArrayElement tests the private helper function
func TestExtractStringFieldFromArrayElement(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedOk    bool
	}{
		{
			name:          "Extract from string element",
			input:         `items = ["test_value"]`,
			expectedValue: "test_value",
			expectedOk:    true,
		},
		{
			name:          "Extract from multiple elements",
			input:         `items = ["val1", "val2", "val3"]`,
			expectedValue: "val1",
			expectedOk:    true,
		},
		{
			name:          "Non-string element",
			input:         `items = [42]`,
			expectedValue: "",
			expectedOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			attr := file.Body().GetAttribute("items")
			require.NotNil(t, attr)

			elements := ParseArrayAttribute(attr)
			require.Greater(t, len(elements), 0)

			value, ok := extractStringFieldFromArrayElement(elements[0])

			assert.Equal(t, tt.expectedOk, ok)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

// TestEdgeCases tests edge cases and error handling
func TestEdgeCases(t *testing.T) {
	t.Run("ParseArrayAttribute with nil", func(t *testing.T) {
		result := ParseArrayAttribute(nil)
		assert.Nil(t, result)
	})

	t.Run("MergeAttributeAndBlocksToObjectArray with empty body", func(t *testing.T) {
		file := hclwrite.NewEmptyFile()
		body := file.Body()

		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		assert.False(t, modified)
	})

	t.Run("Special characters in values", func(t *testing.T) {
		input := `
resource "test" "example" {
  items = ["val-with-dash", "val_with_underscore", "val.with.dots"]
}`

		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		assert.True(t, modified)

		// Verify all values are preserved
		attr := body.GetAttribute("items")
		require.NotNil(t, attr)

		// The attribute should exist and be a tuple
		val := attr.Expr().BuildTokens(nil)
		assert.NotEmpty(t, val)
	})

	t.Run("Empty string values", func(t *testing.T) {
		input := `
resource "test" "example" {
  items = ["", "non-empty", ""]
}`

		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		assert.True(t, modified)
	})

	t.Run("Comments with blank lines between sections produce no phantom elements", func(t *testing.T) {
		// Regression test: blank lines between comment-separated sections in a string
		// array generated phantom `{ value =\n description = null }` entries.
		// The blank line TokenNewline at the top level was collected into currentElement
		// and saved as a phantom element on the next comma. Real-world pattern from
		// contractor_allow_list_macos in zero-trust-global/teams_lists.tf:
		//   items = [
		//     ### Source header
		//     ## Section one
		//     "captive.apple.com",
		//                          ← blank line here became phantom element
		//     ## Section two
		//     "push.apple.com"
		//   ]
		input := `
resource "test" "example" {
  items = [
    ### Source: https://support.apple.com
    ### Date: 2024-08-20

    ## Device setup
    # comment about device setup
    "captive.apple.com",
    "gs.apple.com",
    "humb.apple.com",

    ## Device management
    # comment about MDM
    "push.apple.com",
    "deviceenrollment.apple.com"
  ]
}`

		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		assert.True(t, modified)
		output := string(file.Bytes())

		// Output must be valid HCL — no bare `value =` with nothing after it
		_, parseDiags := hclwrite.ParseConfig([]byte(output), "", hcl.InitialPos)
		assert.False(t, parseDiags.HasErrors(), "output must be valid HCL:\n%s", output)

		// Must contain all 5 real string values
		for _, v := range []string{"captive.apple.com", "gs.apple.com", "humb.apple.com", "push.apple.com", "deviceenrollment.apple.com"} {
			assert.Contains(t, output, `"`+v+`"`, "missing value %q in output", v)
		}

		// Bare `value =\n` is the failure signature — must not appear
		assert.NotContains(t, output, "value =\n", "bare 'value =' found — phantom element from blank line")
	})

	t.Run("Resource references as array elements are preserved on one line", func(t *testing.T) {
		// Regression test: items = [resource.name.attr, resource.name.attr]
		// Reference-type elements must produce `value = resource.name.attr` (inline),
		// not `value =\n  resource.name.attr` split across lines.
		input := `
resource "test" "example" {
  items = [
    cloudflare_zero_trust_tunnel_cloudflared_route.athens_ipv4.network,
    cloudflare_zero_trust_tunnel_cloudflared_route.athens_ipv6.network
  ]
}`

		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		assert.True(t, modified)
		output := string(file.Bytes())

		// Output must be valid HCL
		_, parseDiags := hclwrite.ParseConfig([]byte(output), "", hcl.InitialPos)
		assert.False(t, parseDiags.HasErrors(), "output must be valid HCL:\n%s", output)

		// References must not be split to next line
		assert.NotContains(t, output, "value =\n", "bare 'value =' found — reference split across lines")
		assert.Contains(t, output, "cloudflare_zero_trust_tunnel_cloudflared_route.athens_ipv4.network")
		assert.Contains(t, output, "cloudflare_zero_trust_tunnel_cloudflared_route.athens_ipv6.network")
	})
}

// Helper function to extract string value from an ArrayElement (for testing)
func ExtractStringFromElement(elem ArrayElement) string {
	val, _ := extractStringFieldFromArrayElement(elem)
	return val
}

// TestForExpressionHandling verifies that for expressions in array attributes
// are left completely untouched by MergeAttributeAndBlocksToObjectArray.
// Splitting [for k, v in map : v] at the comma would produce invalid HCL.
func TestForExpressionHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "for expression with two iteration variables is preserved",
			input: `
resource "test" "example" {
  items = [for cidr, _ in local.vault_cidrs : cidr]
}`,
		},
		{
			name: "for expression with single iteration variable is preserved",
			input: `
resource "test" "example" {
  items = [for cidr in local.cidrs : cidr]
}`,
		},
		{
			name: "for expression with function call is preserved",
			input: `
resource "test" "example" {
  items = [for cidr, _ in merge(local.pdx_cidrs, local.ams_cidrs) : cidr]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "input must be valid HCL")

			body := file.Body().Blocks()[0].Body()
			original := string(file.Bytes())

			modified := MergeAttributeAndBlocksToObjectArray(
				body,
				"items",
				"items_with_description",
				"items",
				"value",
				[]string{"description"},
				true,
			)

			// Must not modify anything — the for expression is opaque
			assert.False(t, modified, "should not report modification for for expressions")
			assert.Equal(t, original, string(file.Bytes()), "for expression must be preserved verbatim")

			// Output must still be valid HCL
			_, parseDiags := hclwrite.ParseConfig(file.Bytes(), "", hcl.InitialPos)
			assert.False(t, parseDiags.HasErrors(), "output must be valid HCL: %s", parseDiags.Error())
		})
	}
}

// TestParseArrayAttributeForExpression verifies ParseArrayAttribute returns nil for for expressions
func TestParseArrayAttributeForExpression(t *testing.T) {
	inputs := []string{
		`items = [for cidr, _ in local.vault_cidrs : cidr]`,
		`items = [for cidr in local.cidrs : cidr]`,
		`items = [for k, v in merge(local.a, local.b) : v]`,
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			attr := file.Body().GetAttribute("items")
			require.NotNil(t, attr)

			elements := ParseArrayAttribute(attr)
			assert.Nil(t, elements, "ParseArrayAttribute must return nil for for expressions")
		})
	}
}

// TestOrderPreservation verifies that TupleVal preserves insertion order
func TestOrderPreservation(t *testing.T) {
	t.Run("Blocks first ordering", func(t *testing.T) {
		input := `
resource "test" "example" {
  items = ["array_first", "array_second", "array_third"]

  items_with_description {
    value       = "block_first"
    description = "desc1"
  }

  items_with_description {
    value       = "block_second"
    description = "desc2"
  }
}`

		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()

		// Test with blocks first
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true, // blocks first
		)

		assert.True(t, modified)

		// Parse the output to verify general ordering
		// All block items should come before all array items
		output := string(file.Bytes())

		// Find positions of all values
		blockFirstIdx := indexOfSubstring(output, "block_first")
		blockSecondIdx := indexOfSubstring(output, "block_second")
		arrayFirstIdx := indexOfSubstring(output, "array_first")
		arraySecondIdx := indexOfSubstring(output, "array_second")
		arrayThirdIdx := indexOfSubstring(output, "array_third")

		// At least verify that blocks come before array items
		assert.Less(t, blockFirstIdx, arrayFirstIdx, "block items should come before array items")
		assert.Less(t, blockSecondIdx, arrayFirstIdx, "block items should come before array items")

		// Verify array items are in order
		assert.Less(t, arrayFirstIdx, arraySecondIdx, "array_first should come before array_second")
		assert.Less(t, arraySecondIdx, arrayThirdIdx, "array_second should come before array_third")
	})

	t.Run("Array first ordering", func(t *testing.T) {
		input := `
resource "test" "example" {
  items = ["array_first", "array_second"]

  items_with_description {
    value       = "block_first"
    description = "desc1"
  }
}`

		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()

		// Test with array first
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			false, // array first
		)

		assert.True(t, modified)

		output := string(file.Bytes())

		arrayFirstIdx := indexOfSubstring(output, "array_first")
		blockFirstIdx := indexOfSubstring(output, "block_first")

		// Verify array items come before block items
		assert.Less(t, arrayFirstIdx, blockFirstIdx, "array items should come before block items when arraysFirst=false")
	})
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestMergeWithNoOptionalFields tests merging when there are no optional fields
func TestMergeWithNoOptionalFields(t *testing.T) {
	input := `
resource "test" "example" {
  names = ["alice", "bob"]

  name_block {
    name = "charlie"
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()

	modified := MergeAttributeAndBlocksToObjectArray(
		body,
		"names",
		"name_block",
		"names",
		"name",
		[]string{}, // no optional fields
		false,      // array first
	)

	assert.True(t, modified)

	// Verify the output structure (less strict about exact formatting)
	actual := string(file.Bytes())

	// Verify all expected values are present in correct order
	assert.Contains(t, actual, "alice")
	assert.Contains(t, actual, "bob")
	assert.Contains(t, actual, "charlie")

	// Verify alice comes before bob, and bob comes before charlie
	aliceIdx := strings.Index(actual, "alice")
	bobIdx := strings.Index(actual, "bob")
	charlieIdx := strings.Index(actual, "charlie")

	assert.Less(t, aliceIdx, bobIdx, "alice should come before bob")
	assert.Less(t, bobIdx, charlieIdx, "bob should come before charlie")

	// Verify the names attribute exists
	assert.Contains(t, actual, "names =")
}

// TestMergeAttributeAndBlocksToObjectArray_AttributeSyntax tests Bug #001:
// items_with_description written as an attribute (not a block) is not migrated.
// These tests should FAIL before the fix is applied.
func TestMergeAttributeAndBlocksToObjectArray_AttributeSyntax(t *testing.T) {
	// Case A: blockType attr is an opaque expression (local reference).
	// Expected behaviour: rename items_with_description → items verbatim.
	t.Run("blockType_as_opaque_local_reference", func(t *testing.T) {
		input := `
resource "cloudflare_zero_trust_list" "do_not_inspect_tunnels" {
  account_id             = var.account_id
  name                   = "Do Not Inspect Tunnels"
  type                   = "IP"
  items_with_description = local.do_not_inspect_tunnels
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		assert.True(t, modified, "should be modified when items_with_description attr is present")

		actual := string(file.Bytes())
		// items_with_description must be gone
		assert.NotContains(t, actual, "items_with_description", "items_with_description attribute must be removed")
		// items must now reference the local
		assert.Contains(t, actual, "items", "items attribute must be present")
		assert.Contains(t, actual, "local.do_not_inspect_tunnels", "local reference must be preserved")
	})

	// Case B: blockType attr is an inline object list with resource references.
	// Expected behaviour: merge with existing items array, preserve references.
	t.Run("blockType_as_inline_object_list_with_references", func(t *testing.T) {
		input := `
resource "cloudflare_zero_trust_list" "do_not_inspect_IPs_employees" {
  account_id  = var.account_id
  name        = "IP addresses to never inspect - Cloudflare Employees"
  type        = "IP"
  items_with_description = [
    {
      value       = cloudflare_zero_trust_tunnel_cloudflared_route.athens_staging_tunnel_ipv4.network
      description = "Athens Staging IPv4"
    },
    {
      value       = cloudflare_zero_trust_tunnel_cloudflared_route.athens_tunnel_ipv4.network
      description = "Athens IPv4"
    }
  ]
  items = [{
    description = null
    value       = "8.14.199.1"
    }, {
    description = null
    value       = "8.14.199.2"
  }]
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true, // blocksFirst: items_with_description entries come first
		)

		assert.True(t, modified)

		actual := string(file.Bytes())
		assert.NotContains(t, actual, "items_with_description", "items_with_description must be removed")
		// Reference items (from items_with_description) must appear before static items
		athensIdx := strings.Index(actual, "athens_staging_tunnel_ipv4")
		staticIdx := strings.Index(actual, "8.14.199.1")
		assert.Greater(t, athensIdx, 0, "Athens reference must be present")
		assert.Greater(t, staticIdx, 0, "static IP must be present")
		assert.Less(t, athensIdx, staticIdx, "items_with_description entries (blocksFirst=true) must precede items entries")
		// Both descriptions must be preserved
		assert.Contains(t, actual, `"Athens Staging IPv4"`)
		assert.Contains(t, actual, `"Athens IPv4"`)
		// Static items null descriptions must be preserved
		assert.Contains(t, actual, "8.14.199.2")
	})

	// Case B2: blockType attr is an inline object list with STATIC strings only.
	// Expected behaviour: fully merge into items, all values present.
	t.Run("blockType_as_inline_object_list_static_strings", func(t *testing.T) {
		input := `
resource "cloudflare_zero_trust_list" "example" {
  account_id = "abc123"
  type       = "IP"
  items_with_description = [
    {
      value       = "192.168.1.1"
      description = "Gateway"
    },
    {
      value       = "10.0.0.1"
      description = "Internal"
    }
  ]
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		assert.True(t, modified)
		actual := string(file.Bytes())
		assert.NotContains(t, actual, "items_with_description")
		assert.Contains(t, actual, `"192.168.1.1"`)
		assert.Contains(t, actual, `"Gateway"`)
		assert.Contains(t, actual, `"10.0.0.1"`)
		assert.Contains(t, actual, `"Internal"`)
	})

	// Case C: items_with_description attr present AND items_with_description blocks present.
	// This is a pathological case — both syntaxes coexist. Both must be merged.
	t.Run("blockType_as_attribute_and_blocks_coexist", func(t *testing.T) {
		input := `
resource "cloudflare_zero_trust_list" "example" {
  account_id = "abc123"
  type       = "IP"
  items_with_description = [
    {
      value       = "192.168.1.1"
      description = "From attr"
    }
  ]
  items_with_description {
    value       = "10.0.0.1"
    description = "From block"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		modified := MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		assert.True(t, modified)
		actual := string(file.Bytes())
		assert.NotContains(t, actual, "items_with_description")
		assert.Contains(t, actual, `"192.168.1.1"`)
		assert.Contains(t, actual, `"10.0.0.1"`)
		assert.Contains(t, actual, `"From attr"`)
		assert.Contains(t, actual, `"From block"`)
	})

	// Case D: No items_with_description attribute, no blocks — nothing to do.
	t.Run("no_blockType_attribute_or_block_is_noop", func(t *testing.T) {
		input := `
resource "cloudflare_zero_trust_list" "example" {
  account_id = "abc123"
  type       = "IP"
  items = [{
    description = null
    value       = "1.2.3.4"
  }]
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := file.Body().Blocks()[0].Body()
		// Already-migrated resource — items_with_description is absent as both attr and block.
		// The items attr is already an object list; ParseArrayAttribute returns [] for object arrays.
		// This call should not corrupt the existing items attribute.
		MergeAttributeAndBlocksToObjectArray(
			body,
			"items",
			"items_with_description",
			"items",
			"value",
			[]string{"description"},
			true,
		)

		actual := string(file.Bytes())
		assert.NotContains(t, actual, "items_with_description")
		assert.Contains(t, actual, "1.2.3.4")
	})
}
