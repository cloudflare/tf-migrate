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

// runConfigTransformTest runs a single configuration transformation test
// It automatically handles preprocessing when needed (mimics production pipeline)
func runConfigTransformTest(t *testing.T, tt ConfigTestCase, migrator transform.ResourceTransformer) {
	t.Helper()

	// Step 1: Preprocess (string-level transformations)
	processedContent := migrator.Preprocess(tt.Input)

	// Step 2: Parse the preprocessed HCL
	file, diags := hclwrite.ParseConfig([]byte(processedContent), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse preprocessed HCL: %v", diags)

	// Step 3: Create context with preprocessed content
	ctx := &transform.Context{
		Content:  []byte(processedContent),
		Filename: "test.tf",
		AST:      file,
	}

	// Step 4: Transform using HCL AST
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

	// Step 5: Format and get output
	output := string(hclwrite.Format(file.Bytes()))
	// Normalize whitespace for comparison
	output = NormalizeHCLWhitespace(output)
	output = strings.TrimSpace(output)

	// Parse expected for comparison
	expectedFile, diags := hclwrite.ParseConfig([]byte(tt.Expected), "expected.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse expected HCL: %v", diags)
	expectedOutput := string(hclwrite.Format(expectedFile.Bytes()))
	// Normalize whitespace for comparison
	expectedOutput = NormalizeHCLWhitespace(expectedOutput)
	expectedOutput = strings.TrimSpace(expectedOutput)

	assert.Equal(t, expectedOutput, output)
}

// RunConfigTransformTests runs multiple configuration transformation tests
func RunConfigTransformTests(t *testing.T, tests []ConfigTestCase, migrator transform.ResourceTransformer) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			runConfigTransformTest(t, tt, migrator)
		})
	}
}
