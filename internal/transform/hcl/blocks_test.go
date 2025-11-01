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

func TestConvertSingleBlockToAttribute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		blockType string
		attrName string
		expected bool
		contains string
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
			blockType: "config",
			attrName:  "config",
			expected:  true,
			contains:  "config =",
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
