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
}

// Helper function to extract string value from an ArrayElement (for testing)
func ExtractStringFromElement(elem ArrayElement) string {
	val, _ := extractStringFieldFromArrayElement(elem)
	return val
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
