package hcl

import (
	"errors"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameResourceType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		oldType  string
		newType  string
		expected bool
		contains string
	}{
		{
			name: "Rename resource type",
			input: `
resource "cloudflare_record" "example" {
  name = "test"
}`,
			oldType:  "cloudflare_record",
			newType:  "cloudflare_dns_record",
			expected: true,
			contains: `resource "cloudflare_dns_record" "example"`,
		},
		{
			name: "Do not rename if type doesn't match",
			input: `
resource "cloudflare_zone" "example" {
  name = "example.com"
}`,
			oldType:  "cloudflare_record",
			newType:  "cloudflare_dns_record",
			expected: false,
			contains: `resource "cloudflare_zone" "example"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			block := file.Body().Blocks()[0]
			result := RenameResourceType(block, tt.oldType, tt.newType)

			assert.Equal(t, tt.expected, result)

			output := string(file.Bytes())
			assert.Contains(t, output, tt.contains)
		})
	}
}

func TestGetResourceType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Get resource type",
			input: `
resource "cloudflare_dns_record" "example" {
  name = "test"
}`,
			expected: "cloudflare_dns_record",
		},
		{
			name: "Return empty for non-resource block",
			input: `
data "cloudflare_zone" "example" {
  name = "example.com"
}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			block := file.Body().Blocks()[0]
			result := GetResourceType(block)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetResourceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Get resource name",
			input: `
resource "cloudflare_dns_record" "my_record" {
  name = "test"
}`,
			expected: "my_record",
		},
		{
			name: "Return empty for non-resource block",
			input: `
data "cloudflare_zone" "my_zone" {
  name = "example.com"
}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			block := file.Body().Blocks()[0]
			result := GetResourceName(block)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertBlocksToAttribute(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		blockType  string
		attrName   string
		preProcess func(*hclwrite.Block)
		contains   string
		notContains string
	}{
		{
			name: "Convert data block to attribute",
			input: `
resource "cloudflare_dns_record" "caa" {
  zone_id = "abc123"
  type    = "CAA"

  data {
    flags = "0"
    tag   = "issue"
  }
}`,
			blockType: "data",
			attrName:  "data",
			contains:  "data =",
			notContains: "data {",
		},
		{
			name: "Convert with preprocessing",
			input: `
resource "cloudflare_dns_record" "caa" {
  zone_id = "abc123"

  data {
    content = "letsencrypt.org"
  }
}`,
			blockType: "data",
			attrName:  "data",
			preProcess: func(block *hclwrite.Block) {
				RenameAttribute(block.Body(), "content", "value")
			},
			contains: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			ConvertBlocksToAttribute(body, tt.blockType, tt.attrName, tt.preProcess)

			output := string(file.Bytes())
			assert.Contains(t, output, tt.contains)
			if tt.notContains != "" {
				assert.NotContains(t, output, tt.notContains)
			}
		})
	}
}

func TestHoistAttributeFromBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		blockType string
		attrName string
		expected bool
		contains string
	}{
		{
			name: "Hoist attribute from block",
			input: `
resource "cloudflare_dns_record" "srv" {
  zone_id = "abc123"
  type    = "SRV"

  data {
    priority = 10
    weight   = 60
  }
}`,
			blockType: "data",
			attrName:  "priority",
			expected:  true,
			contains:  "priority = 10",
		},
		{
			name: "Do not hoist if attribute doesn't exist in block",
			input: `
resource "cloudflare_dns_record" "srv" {
  zone_id = "abc123"

  data {
    weight = 60
  }
}`,
			blockType: "data",
			attrName:  "priority",
			expected:  false,
		},
		{
			name: "Do not hoist if parent already has attribute",
			input: `
resource "cloudflare_dns_record" "srv" {
  zone_id  = "abc123"
  priority = 5

  data {
    priority = 10
    weight   = 60
  }
}`,
			blockType: "data",
			attrName:  "priority",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			result := HoistAttributeFromBlock(body, tt.blockType, tt.attrName)

			assert.Equal(t, tt.expected, result)

			if tt.contains != "" {
				output := string(file.Bytes())
				assert.Contains(t, output, tt.contains)
			}
		})
	}
}

func TestHoistAttributesFromBlock(t *testing.T) {
	input := `
resource "cloudflare_dns_record" "srv" {
  zone_id = "abc123"

  data {
    priority = 10
    weight   = 60
    port     = 5060
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()
	count := HoistAttributesFromBlock(body, "data", "priority", "weight", "missing")

	assert.Equal(t, 2, count, "Should hoist 2 attributes (priority and weight)")

	output := string(file.Bytes())
	assert.Contains(t, output, "priority = 10")
	assert.Contains(t, output, "weight   = 60")
}

func TestFindBlockByType(t *testing.T) {
	input := `
resource "test" "example" {
  name = "test"

  config {
    value = "first"
  }

  settings {
    enabled = true
  }

  config {
    value = "second"
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()

	t.Run("Find existing block", func(t *testing.T) {
		block := FindBlockByType(body, "config")
		assert.NotNil(t, block)
		assert.Equal(t, "config", block.Type())
	})

	t.Run("Return nil for non-existent block", func(t *testing.T) {
		block := FindBlockByType(body, "nonexistent")
		assert.Nil(t, block)
	})
}

func TestFindBlocksByType(t *testing.T) {
	input := `
resource "test" "example" {
  name = "test"

  item {
    value = "first"
  }

  settings {
    enabled = true
  }

  item {
    value = "second"
  }

  item {
    value = "third"
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()

	t.Run("Find multiple blocks", func(t *testing.T) {
		blocks := FindBlocksByType(body, "item")
		assert.Equal(t, 3, len(blocks))
	})

	t.Run("Find single block", func(t *testing.T) {
		blocks := FindBlocksByType(body, "settings")
		assert.Equal(t, 1, len(blocks))
	})

	t.Run("Return empty slice for non-existent block", func(t *testing.T) {
		blocks := FindBlocksByType(body, "nonexistent")
		assert.Equal(t, 0, len(blocks))
	})
}

func TestRemoveBlocksByType(t *testing.T) {
	input := `
resource "test" "example" {
  name = "test"

  deprecated {
    value = "old"
  }

  keep {
    value = "stay"
  }

  deprecated {
    value = "also_old"
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()
	count := RemoveBlocksByType(body, "deprecated")

	assert.Equal(t, 2, count)

	output := string(file.Bytes())
	assert.Contains(t, output, "keep {")
	assert.NotContains(t, output, "deprecated {")
}

func TestProcessBlocksOfType(t *testing.T) {
	input := `
resource "test" "example" {
  name = "test"

  item {
    old_field = "value1"
  }

  settings {
    enabled = true
  }

  item {
    old_field = "value2"
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body().Blocks()[0].Body()

	t.Run("Process all matching blocks", func(t *testing.T) {
		processedCount := 0
		err := ProcessBlocksOfType(body, "item", func(block *hclwrite.Block) error {
			RenameAttribute(block.Body(), "old_field", "new_field")
			processedCount++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, processedCount)

		output := string(file.Bytes())
		assert.Contains(t, output, "new_field")
		assert.NotContains(t, output, "old_field")
	})

	t.Run("Stop on error", func(t *testing.T) {
		testError := errors.New("test error")
		err := ProcessBlocksOfType(body, "item", func(block *hclwrite.Block) error {
			return testError
		})

		assert.Equal(t, testError, err)
	})
}

func TestConvertBlocksToAttributeList(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		blockType string
		expected  string
	}{
		{
			name: "Convert multiple blocks to array attribute",
			input: `
resource "test" "example" {
  name = "test"

  destinations {
    type = "public"
    uri  = "https://app.example.com"
  }

  destinations {
    type = "private"
    cidr = "10.0.0.0/24"
  }
}`,
			blockType: "destinations",
			expected: `
resource "test" "example" {
  name = "test"


  destinations = [
    {
      type = "public"
      uri  = "https://app.example.com"
    },
    {
      type = "private"
      cidr = "10.0.0.0/24"
    }
  ]
}`,
		},
		{
			name: "Convert single block to array attribute",
			input: `
resource "test" "example" {
  name = "test"

  destinations {
    type = "public"
    uri  = "https://app.example.com"
  }
}`,
			blockType: "destinations",
			expected: `
resource "test" "example" {
  name = "test"

  destinations = [
    {
      type = "public"
      uri  = "https://app.example.com"
    }
  ]
}`,
		},
		{
			name: "Return unchanged for non-existent blocks",
			input: `
resource "test" "example" {
  name = "test"
}`,
			blockType: "missing",
			expected: `
resource "test" "example" {
  name = "test"
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			ConvertBlocksToAttributeList(body, tt.blockType, nil)

			output := string(file.Bytes())
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestConvertSingleBlockToAttribute(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		blockType   string
		attrName    string
		expected    bool
		contains    string
		notContains string
	}{
		{
			name: "Convert single block to attribute",
			input: `
resource "test" "example" {
  name = "test"

  config {
    enabled = true
    value   = "test"
  }
}`,
			blockType:   "config",
			attrName:    "config",
			expected:    true,
			contains:    "config =",
			notContains: "config {",
		},
		{
			name: "Return false for non-existent block",
			input: `
resource "test" "example" {
  name = "test"
}`,
			blockType: "missing",
			attrName:  "missing",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body().Blocks()[0].Body()
			result := ConvertSingleBlockToAttribute(body, tt.blockType, tt.attrName)

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

func TestCreateDerivedBlock(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		newType   string
		newName   string
		transform AttributeTransform
		contains  []string
		notContains []string
	}{
		{
			name: "Copy attributes only",
			input: `
resource "cloudflare_argo" "main" {
  zone_id        = "abc123"
  smart_routing  = "on"
  tiered_caching = "on"
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "main",
			transform: AttributeTransform{
				Copy: []string{"zone_id"},
			},
			contains: []string{
				`resource "cloudflare_argo_smart_routing" "main"`,
				"zone_id",
				"abc123",
			},
			notContains: []string{"smart_routing =", "tiered_caching"},
		},
		{
			name: "Rename attributes",
			input: `
resource "cloudflare_argo" "main" {
  zone_id       = "abc123"
  smart_routing = "on"
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "main",
			transform: AttributeTransform{
				Copy:   []string{"zone_id"},
				Rename: map[string]string{"smart_routing": "value"},
			},
			contains: []string{
				`resource "cloudflare_argo_smart_routing" "main"`,
				"zone_id",
				"value",
				`"on"`,
			},
			notContains: []string{"smart_routing ="},
		},
		{
			name: "Set default values",
			input: `
resource "cloudflare_argo" "main" {
  zone_id = "abc123"
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "main",
			transform: AttributeTransform{
				Copy: []string{"zone_id"},
				Set:  map[string]interface{}{"value": "off"},
			},
			contains: []string{
				`resource "cloudflare_argo_smart_routing" "main"`,
				"zone_id",
				"value",
				`"off"`,
			},
		},
		{
			name: "Copy meta-arguments (lifecycle)",
			input: `
resource "cloudflare_argo" "main" {
  zone_id       = "abc123"
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing]
  }
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "main",
			transform: AttributeTransform{
				Copy:              []string{"zone_id"},
				Rename:            map[string]string{"smart_routing": "value"},
				CopyMetaArguments: true,
			},
			contains: []string{
				`resource "cloudflare_argo_smart_routing" "main"`,
				"zone_id",
				"value",
				"lifecycle",
				"ignore_changes",
			},
		},
		{
			name: "Complex transformation with all features",
			input: `
resource "cloudflare_argo" "example" {
  zone_id        = var.zone_id
  smart_routing  = "on"
  tiered_caching = "on"

  lifecycle {
    ignore_changes = [smart_routing]
    create_before_destroy = true
  }
}`,
			newType: "cloudflare_argo_tiered_caching",
			newName: "example_tiered",
			transform: AttributeTransform{
				Copy:              []string{"zone_id"},
				Rename:            map[string]string{"tiered_caching": "value"},
				Set:               map[string]interface{}{"enabled": true},
				CopyMetaArguments: true,
			},
			contains: []string{
				`resource "cloudflare_argo_tiered_caching" "example_tiered"`,
				"zone_id",
				"var.zone_id",
				"value",
				`"on"`,
				"enabled",
				"true",
				"lifecycle",
				"ignore_changes",
				"create_before_destroy",
			},
			// Note: smart_routing appears in the lifecycle block (ignore_changes)
			// This is expected - lifecycle blocks are copied as-is
		},
		{
			name: "Set with different value types",
			input: `
resource "test" "example" {
  zone_id = "abc123"
}`,
			newType: "test_derived",
			newName: "example",
			transform: AttributeTransform{
				Copy: []string{"zone_id"},
				Set: map[string]interface{}{
					"string_val":  "test",
					"int_val":     42,
					"float_val":   3.14,
					"bool_val":    true,
				},
			},
			contains: []string{
				`resource "test_derived" "example"`,
				"zone_id",
				"string_val",
				`"test"`,
				"int_val",
				"42",
				"float_val",
				"3.14",
				"bool_val",
				"true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			originalBlock := file.Body().Blocks()[0]
			newBlock := CreateDerivedBlock(originalBlock, tt.newType, tt.newName, tt.transform)

			// Create a new file with just the new block to get its output
			newFile := hclwrite.NewEmptyFile()
			newFile.Body().AppendBlock(newBlock)
			output := string(newFile.Bytes())

			for _, contains := range tt.contains {
				assert.Contains(t, output, contains, "Output should contain: %s", contains)
			}

			for _, notContains := range tt.notContains {
				assert.NotContains(t, output, notContains, "Output should not contain: %s", notContains)
			}
		})
	}
}

func TestCreateDerivedBlock_ResourceLabels(t *testing.T) {
	// Test that the new block has correct resource type and name labels
	input := `
resource "cloudflare_argo" "main" {
  zone_id = "abc123"
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	originalBlock := file.Body().Blocks()[0]
	newBlock := CreateDerivedBlock(originalBlock, "cloudflare_argo_smart_routing", "smart_main", AttributeTransform{
		Copy: []string{"zone_id"},
	})

	assert.Equal(t, "resource", newBlock.Type())
	labels := newBlock.Labels()
	assert.Equal(t, 2, len(labels))
	assert.Equal(t, "cloudflare_argo_smart_routing", labels[0])
	assert.Equal(t, "smart_main", labels[1])
}

func TestCreateDerivedBlock_NoMetaArguments(t *testing.T) {
	// Test that lifecycle blocks are NOT copied when CopyMetaArguments is false
	input := `
resource "cloudflare_argo" "main" {
  zone_id       = "abc123"
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing]
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	originalBlock := file.Body().Blocks()[0]
	newBlock := CreateDerivedBlock(originalBlock, "cloudflare_argo_smart_routing", "main", AttributeTransform{
		Copy:              []string{"zone_id"},
		Rename:            map[string]string{"smart_routing": "value"},
		CopyMetaArguments: false, // Explicitly false
	})

	file.Body().AppendBlock(newBlock)
	output := string(file.Bytes())

	// The new block should not have lifecycle
	assert.Contains(t, output, `resource "cloudflare_argo_smart_routing" "main"`)
	assert.Contains(t, output, "value")

	// Count occurrences of "lifecycle" - should only be in the original block
	// We'll check that the second resource doesn't have lifecycle by verifying
	// the new block has no nested blocks
	assert.Equal(t, 0, len(newBlock.Body().Blocks()), "New block should have no nested blocks")
}

func TestCreateDerivedBlock_EmptyTransform(t *testing.T) {
	// Test with an empty transform (no attributes copied, renamed, or set)
	input := `
resource "cloudflare_argo" "main" {
  zone_id       = "abc123"
  smart_routing = "on"
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	originalBlock := file.Body().Blocks()[0]
	newBlock := CreateDerivedBlock(originalBlock, "cloudflare_argo_smart_routing", "main", AttributeTransform{})

	file.Body().AppendBlock(newBlock)
	output := string(file.Bytes())

	assert.Contains(t, output, `resource "cloudflare_argo_smart_routing" "main"`)

	// The new block should be essentially empty (just the resource declaration)
	assert.Equal(t, 0, len(newBlock.Body().Attributes()), "New block should have no attributes")
}

func TestApplyAttributeRenamesInLifecycle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		renames  map[string]string
		contains []string
		notContains []string
	}{
		{
			name:  "Rename single identifier in array",
			input: `[smart_routing]`,
			renames: map[string]string{"smart_routing": "value"},
			contains: []string{"value"},
			notContains: []string{"smart_routing"},
		},
		{
			name:  "Rename multiple identifiers in array",
			input: `[smart_routing, tiered_caching]`,
			renames: map[string]string{
				"smart_routing":  "value",
				"tiered_caching": "enabled",
			},
			contains: []string{"value", "enabled"},
			notContains: []string{"smart_routing", "tiered_caching"},
		},
		{
			name:  "Only rename matching identifiers",
			input: `[smart_routing, other_field]`,
			renames: map[string]string{"smart_routing": "value"},
			contains: []string{"value", "other_field"},
			notContains: []string{"smart_routing"},
		},
		{
			name:  "Preserve non-identifier tokens",
			input: `["string_value", 123, true]`,
			renames: map[string]string{"string_value": "renamed"},
			contains: []string{`"string_value"`, "123", "true"},
			// String literals are not renamed, only identifiers
		},
		{
			name:     "No renames when map is empty",
			input:    `[field1, field2]`,
			renames:  map[string]string{},
			contains: []string{"field1", "field2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse as an attribute value to get tokens
			hclInput := "test = " + tt.input
			file, diags := hclwrite.ParseConfig([]byte(hclInput), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			// Get the tokens
			tokens := file.Body().Attributes()["test"].Expr().BuildTokens(nil)

			// Apply renames
			renamedTokens := applyAttributeRenames(tokens, tt.renames)

			// Create output with only the renamed tokens
			outputBody := hclwrite.NewEmptyFile().Body()
			outputBody.SetAttributeRaw("result", renamedTokens)
			output := string(outputBody.BuildTokens(nil).Bytes())

			for _, contains := range tt.contains {
				assert.Contains(t, output, contains, "Output should contain: %s", contains)
			}

			for _, notContains := range tt.notContains {
				assert.NotContains(t, output, notContains, "Output should not contain: %s", notContains)
			}
		})
	}
}

func TestCreateDerivedBlock_LifecycleAttributeRenames(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		newType     string
		newName     string
		transform   AttributeTransform
		contains    []string
		notContains []string
	}{
		{
			name: "Rename attribute in lifecycle ignore_changes",
			input: `
resource "cloudflare_argo" "test" {
  zone_id       = "abc123"
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing]
  }
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "test",
			transform: AttributeTransform{
				Copy:              []string{"zone_id"},
				Rename:            map[string]string{"smart_routing": "value"},
				CopyMetaArguments: true,
			},
			contains: []string{
				"lifecycle",
				"ignore_changes = [value]",
			},
			notContains: []string{
				"ignore_changes = [smart_routing]",
			},
		},
		{
			name: "Rename multiple attributes in lifecycle - filters out non-included attributes",
			input: `
resource "cloudflare_argo" "test" {
  zone_id        = "abc123"
  smart_routing  = "on"
  tiered_caching = "on"

  lifecycle {
    ignore_changes = [smart_routing, tiered_caching]
  }
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "test",
			transform: AttributeTransform{
				Copy:              []string{"zone_id"},
				Rename:            map[string]string{"smart_routing": "value"},
				CopyMetaArguments: true,
			},
			contains: []string{
				"lifecycle",
				"ignore_changes = [value]",  // tiered_caching filtered out (not in Copy or Rename)
			},
			notContains: []string{
				"smart_routing =",      // Check attribute assignment doesn't use old name
				"[smart_routing",       // Check lifecycle arrays don't use old name
				", smart_routing",      // Check lifecycle arrays don't use old name
				"smart_routing,",       // Check lifecycle arrays don't use old name
				"smart_routing]",       // Check lifecycle arrays don't use old name
				"tiered_caching",       // Filtered out because not in valid attributes
			},
		},
		{
			name: "Preserve non-renamed attributes in lifecycle",
			input: `
resource "cloudflare_argo" "test" {
  zone_id       = "abc123"
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing, zone_id]
  }
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "test",
			transform: AttributeTransform{
				Copy:              []string{"zone_id"},
				Rename:            map[string]string{"smart_routing": "value"},
				CopyMetaArguments: true,
			},
			contains: []string{
				"lifecycle",
				"ignore_changes = [value, zone_id]",
			},
		},
		{
			name: "Rename in lifecycle with multiple lifecycle attributes",
			input: `
resource "cloudflare_argo" "test" {
  zone_id       = "abc123"
  smart_routing = "on"

  lifecycle {
    ignore_changes        = [smart_routing]
    create_before_destroy = true
  }
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "test",
			transform: AttributeTransform{
				Copy:              []string{"zone_id"},
				Rename:            map[string]string{"smart_routing": "value"},
				CopyMetaArguments: true,
			},
			contains: []string{
				"lifecycle",
				"ignore_changes",
				"[value]",
				"create_before_destroy = true",
			},
			notContains: []string{
				"smart_routing =",      // Check attribute assignment doesn't use old name
				"[smart_routing",       // Check lifecycle arrays don't use old name
				", smart_routing",      // Check lifecycle arrays don't use old name
				"smart_routing,",       // Check lifecycle arrays don't use old name
				"smart_routing]",       // Check lifecycle arrays don't use old name
			},
		},
		{
			name: "No renames applied when CopyMetaArguments is false",
			input: `
resource "cloudflare_argo" "test" {
  zone_id       = "abc123"
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing]
  }
}`,
			newType: "cloudflare_argo_smart_routing",
			newName: "test",
			transform: AttributeTransform{
				Copy:              []string{"zone_id"},
				Rename:            map[string]string{"smart_routing": "value"},
				CopyMetaArguments: false,
			},
			notContains: []string{
				"lifecycle",
				"ignore_changes",
			},
		},
		{
			name: "Complex lifecycle with multiple renames - filters out non-included attributes",
			input: `
resource "cloudflare_argo" "test" {
  zone_id        = "abc123"
  smart_routing  = "on"
  tiered_caching = "on"
  other_field    = "keep"

  lifecycle {
    ignore_changes = [smart_routing, tiered_caching, other_field]
  }
}`,
			newType: "cloudflare_argo_combined",
			newName: "test",
			transform: AttributeTransform{
				Copy: []string{"zone_id"},
				Rename: map[string]string{
					"smart_routing":  "routing_value",
					"tiered_caching": "caching_value",
				},
				CopyMetaArguments: true,
			},
			contains: []string{
				"lifecycle",
				"ignore_changes = [routing_value, caching_value]",  // other_field filtered out
			},
			notContains: []string{
				"smart_routing =",
				"[smart_routing",
				", smart_routing",
				"smart_routing,",
				"smart_routing]",
				"tiered_caching =",
				"[tiered_caching",
				", tiered_caching",
				"tiered_caching,",
				"tiered_caching]",
				"other_field",  // Filtered out because not in valid attributes
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			originalBlock := file.Body().Blocks()[0]
			newBlock := CreateDerivedBlock(originalBlock, tt.newType, tt.newName, tt.transform)

			// Create a new file with just the new block to get its output
			newFile := hclwrite.NewEmptyFile()
			newFile.Body().AppendBlock(newBlock)
			output := string(newFile.Bytes())

			for _, contains := range tt.contains {
				assert.Contains(t, output, contains, "Output should contain: %s", contains)
			}

			for _, notContains := range tt.notContains {
				assert.NotContains(t, output, notContains, "Output should not contain: %s", notContains)
			}
		})
	}
}

func TestCopyMetaArguments_WithRenames(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		renames     map[string]string
		contains    []string
		notContains []string
	}{
		{
			name: "Apply renames to lifecycle block",
			input: `
resource "original" "test" {
  old_field = "value"

  lifecycle {
    ignore_changes = [old_field]
  }
}`,
			renames: map[string]string{"old_field": "new_field"},
			contains: []string{
				"lifecycle",
				"ignore_changes = [new_field]",
			},
			notContains: []string{
				"old_field",
			},
		},
		{
			name: "No renames when map is nil",
			input: `
resource "original" "test" {
  field = "value"

  lifecycle {
    ignore_changes = [field]
  }
}`,
			renames: nil,
			contains: []string{
				"lifecycle",
				"ignore_changes = [field]",
			},
		},
		{
			name: "No renames when map is empty",
			input: `
resource "original" "test" {
  field = "value"

  lifecycle {
    ignore_changes = [field]
  }
}`,
			renames: map[string]string{},
			contains: []string{
				"lifecycle",
				"ignore_changes = [field]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			originalBlock := file.Body().Blocks()[0]
			newBlock := hclwrite.NewBlock("resource", []string{"new_type", "test"})

			// Pass nil for validAttributes to maintain backward compatibility (no filtering)
			copyMetaArguments(originalBlock, newBlock, tt.renames, nil)

			// Create a new file with just the new block to get its output
			newFile := hclwrite.NewEmptyFile()
			newFile.Body().AppendBlock(newBlock)
			output := string(newFile.Bytes())

			for _, contains := range tt.contains {
				assert.Contains(t, output, contains, "Output should contain: %s", contains)
			}

			for _, notContains := range tt.notContains {
				assert.NotContains(t, output, notContains, "Output should not contain: %s", notContains)
			}
		})
	}
}

func TestCopyMetaArguments_WithValidAttributesFiltering(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		renames         map[string]string
		validAttributes []string
		contains        []string
		notContains     []string
	}{
		{
			name: "Filter out single invalid attribute",
			input: `
resource "original" "test" {
  keep_field   = "value1"
  remove_field = "value2"

  lifecycle {
    ignore_changes = [keep_field, remove_field]
  }
}`,
			renames:         nil,
			validAttributes: []string{"keep_field"},
			contains: []string{
				"lifecycle",
				"ignore_changes = [keep_field]",
			},
			notContains: []string{
				"remove_field",
			},
		},
		{
			name: "Filter out multiple invalid attributes",
			input: `
resource "original" "test" {
  zone_id  = "abc123"
  field_a  = "value_a"
  field_b  = "value_b"
  field_c  = "value_c"

  lifecycle {
    ignore_changes = [zone_id, field_a, field_b, field_c]
  }
}`,
			renames:         nil,
			validAttributes: []string{"zone_id", "field_a"},
			contains: []string{
				"lifecycle",
				"ignore_changes = [zone_id, field_a]",
			},
			notContains: []string{
				"field_b",
				"field_c",
			},
		},
		{
			name: "Filter with renames - keep renamed attribute, remove others",
			input: `
resource "cloudflare_argo" "test" {
  zone_id        = "abc123"
  smart_routing  = "on"
  tiered_caching = "on"

  lifecycle {
    ignore_changes = [smart_routing, tiered_caching]
  }
}`,
			renames:         map[string]string{"smart_routing": "value"},
			validAttributes: []string{"zone_id", "value"},
			contains: []string{
				"lifecycle",
				"ignore_changes = [value]",
			},
			notContains: []string{
				"smart_routing",
				"tiered_caching",
			},
		},
		{
			name: "Filter with multiple renames and mixed attributes",
			input: `
resource "test" "example" {
  id       = "123"
  field_a  = "a"
  field_b  = "b"
  field_c  = "c"
  field_d  = "d"

  lifecycle {
    ignore_changes = [field_a, field_b, field_c, field_d]
  }
}`,
			renames: map[string]string{
				"field_a": "renamed_a",
				"field_b": "renamed_b",
			},
			validAttributes: []string{"id", "renamed_a", "renamed_b", "field_c"},
			contains: []string{
				"lifecycle",
				"ignore_changes = [renamed_a, renamed_b, field_c]",
			},
			notContains: []string{
				"field_a =",
				"field_b =",
				"field_d",
			},
		},
		{
			name: "Empty validAttributes - no filtering applied",
			input: `
resource "original" "test" {
  field_a = "value_a"
  field_b = "value_b"

  lifecycle {
    ignore_changes = [field_a, field_b]
  }
}`,
			renames:         nil,
			validAttributes: []string{},
			contains: []string{
				"lifecycle",
				"ignore_changes = [field_a, field_b]",
			},
		},
		{
			name: "Nil validAttributes - no filtering applied",
			input: `
resource "original" "test" {
  field_a = "value_a"
  field_b = "value_b"

  lifecycle {
    ignore_changes = [field_a, field_b]
  }
}`,
			renames:         nil,
			validAttributes: nil,
			contains: []string{
				"lifecycle",
				"ignore_changes = [field_a, field_b]",
			},
		},
		{
			name: "Filter all attributes - empty lifecycle list",
			input: `
resource "original" "test" {
  field_a = "value_a"
  field_b = "value_b"

  lifecycle {
    ignore_changes = [field_a, field_b]
  }
}`,
			renames:         nil,
			validAttributes: []string{"zone_id"},
			contains: []string{
				"lifecycle",
				"ignore_changes = []",
			},
			notContains: []string{
				"field_a",
				"field_b",
			},
		},
		{
			name: "Preserve other lifecycle attributes while filtering ignore_changes",
			input: `
resource "original" "test" {
  keep_field   = "value1"
  remove_field = "value2"

  lifecycle {
    ignore_changes        = [keep_field, remove_field]
    create_before_destroy = true
    prevent_destroy       = true
  }
}`,
			renames:         nil,
			validAttributes: []string{"keep_field"},
			contains: []string{
				"lifecycle",
				"ignore_changes",
				"[keep_field]",
				"create_before_destroy = true",
				"prevent_destroy       = true",
			},
			notContains: []string{
				"remove_field",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			originalBlock := file.Body().Blocks()[0]
			newBlock := hclwrite.NewBlock("resource", []string{"new_type", "test"})

			copyMetaArguments(originalBlock, newBlock, tt.renames, tt.validAttributes)

			// Create a new file with just the new block to get its output
			newFile := hclwrite.NewEmptyFile()
			newFile.Body().AppendBlock(newBlock)
			output := string(newFile.Bytes())

			for _, contains := range tt.contains {
				assert.Contains(t, output, contains, "Output should contain: %s", contains)
			}

			for _, notContains := range tt.notContains {
				assert.NotContains(t, output, notContains, "Output should not contain: %s", notContains)
			}
		})
	}
}
