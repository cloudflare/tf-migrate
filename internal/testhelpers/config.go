package testhelpers

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// ConfigTestCase represents a test case for configuration transformations
type ConfigTestCase struct {
	Name     string
	Input    string
	Expected string
}

// RunConfigTransformTest runs a single configuration transformation test
func RunConfigTransformTest(t *testing.T, tt ConfigTestCase, migrator transform.ResourceTransformer) {
	t.Helper()

	// Parse the input HCL
	file, diags := hclwrite.ParseConfig([]byte(tt.Input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse input HCL: %v", diags)

	// Create context
	ctx := &transform.Context{
		Content:  []byte(tt.Input),
		Filename: "test.tf",
		AST:      file,
	}

	// Process each resource block
	body := file.Body()
	for _, block := range body.Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 {
			resourceType := block.Labels()[0]
			if migrator.CanHandle(resourceType) {
				result, err := migrator.TransformConfig(ctx, block)
				assert.NoError(t, err, "Failed to transform resource")
				
				// Handle resource splits or removals
				if result != nil && result.RemoveOriginal {
					body.RemoveBlock(block)
					for _, newBlock := range result.Blocks {
						body.AppendBlock(newBlock)
					}
				}
			}
		}
	}

	// Get the transformed output
	output := strings.TrimSpace(string(hclwrite.Format(file.Bytes())))

	// Parse expected for comparison
	expectedFile, diags := hclwrite.ParseConfig([]byte(tt.Expected), "expected.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse expected HCL: %v", diags)
	expectedOutput := strings.TrimSpace(string(hclwrite.Format(expectedFile.Bytes())))

	assert.Equal(t, expectedOutput, output)
}

// RunConfigTransformTests runs multiple configuration transformation tests
func RunConfigTransformTests(t *testing.T, tests []ConfigTestCase, migrator transform.ResourceTransformer) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			RunConfigTransformTest(t, tt, migrator)
		})
	}
}

// RunConfigTransformTestWithFactory runs tests with a migrator factory function
// This is useful when you need a fresh migrator instance for each test
func RunConfigTransformTestWithFactory(t *testing.T, tt ConfigTestCase, factory func() transform.ResourceTransformer) {
	t.Helper()
	migrator := factory()
	RunConfigTransformTest(t, tt, migrator)
}

// RunConfigTransformTestsWithFactory runs multiple tests with a migrator factory function
func RunConfigTransformTestsWithFactory(t *testing.T, tests []ConfigTestCase, factory func() transform.ResourceTransformer) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			RunConfigTransformTestWithFactory(t, tt, factory)
		})
	}
}