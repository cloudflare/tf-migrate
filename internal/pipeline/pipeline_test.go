package pipeline_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/tidwall/gjson"

	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal/pipeline"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type MockHandler struct {
	transform.BaseHandler
	name   string
	called bool
}

func (m *MockHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
	m.called = true
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]interface{})
	}
	ctx.Metadata[m.name] = true
	return m.Next(ctx)
}

type MockResourceTransformer struct {
	resourceType        string
	canHandleFunc       func(string) bool
	preprocessFunc      func(string) string
	transformFunc       func(*transform.Context, *hclwrite.Block) (*transform.TransformResult, error)
	transformStateCalls int
	preprocessCalls     int
	transformCalls      int
}

func (m *MockResourceTransformer) CanHandle(resourceType string) bool {
	if m.canHandleFunc != nil {
		return m.canHandleFunc(resourceType)
	}
	return resourceType == m.resourceType
}

func (m *MockResourceTransformer) GetResourceType() string {
	return m.resourceType
}

func (m *MockResourceTransformer) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	m.transformCalls++
	if m.transformFunc != nil {
		return m.transformFunc(ctx, block)
	}
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *MockResourceTransformer) TransformState(ctx *transform.Context, json gjson.Result, resourcePath string) (string, error) {
	m.transformStateCalls++
	return "", nil
}

func (m *MockResourceTransformer) Preprocess(content string) string {
	m.preprocessCalls++
	if m.preprocessFunc != nil {
		return m.preprocessFunc(content)
	}
	return content
}

type MockProvider struct {
	mockProviders map[string]transform.ResourceTransformer
}

func (mp *MockProvider) GetMigrator(resourceType string, sourceVersion int, targetVersion int) transform.ResourceTransformer {
	m, ok := mp.mockProviders[fmt.Sprintf("%s:%d:%d", resourceType, sourceVersion, targetVersion)]
	if ok {
		return m
	}
	return nil
}

func (mp *MockProvider) GetAllMigrators(sourceVersion int, targetVersion int, resources ...string) []transform.ResourceTransformer {
	transformers := make([]transform.ResourceTransformer, 0)
	for _, m := range mp.mockProviders {
		transformers = append(transformers, m)
	}
	return transformers
}

var (
	log           = hclog.New(&hclog.LoggerOptions{})
	sourceVersion = 100
	targetVersion = 200
)

func setupTestMigrators(t *testing.T, transformers ...transform.ResourceTransformer) *MockProvider {
	t.Helper()
	provider := &MockProvider{mockProviders: make(map[string]transform.ResourceTransformer)}
	for _, resourceTransformer := range transformers {
		rt := resourceTransformer
		resourceType := rt.GetResourceType()
		provider.mockProviders[fmt.Sprintf("%s:%d:%d", resourceType, sourceVersion, targetVersion)] = rt
	}
	return provider
}

func TestPipelineEndToEnd(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		transformers   []*MockResourceTransformer
		expectedOutput string
		expectError    bool
		errorContains  string
	}{
		{
			name: "Simple resource transformation",
			input: `resource "cloudflare_record" "example" {
  name    = "example"
  zone_id = "12345"
  type    = "A"
  value   = "192.0.2.1"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "cloudflare_record",
					preprocessFunc: func(content string) string {
						return strings.ReplaceAll(content, `"cloudflare_record"`, `"cloudflare_dns_record"`)
					},
				},
			},
			expectedOutput: `resource "cloudflare_dns_record" "example" {
  name    = "example"
  zone_id = "12345"
  type    = "A"
  value   = "192.0.2.1"
}`,
		},
		{
			name: "Multiple resource transformation",
			input: `resource "cloudflare_record" "example1" {
  name = "example1"
}

resource "cloudflare_load_balancer" "example2" {
  name = "example2"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "cloudflare_record",
					preprocessFunc: func(content string) string {
						return strings.ReplaceAll(content, `"cloudflare_record"`, `"cloudflare_dns_record"`)
					},
				},
				{
					resourceType: "cloudflare_load_balancer",
					preprocessFunc: func(content string) string {
						return strings.ReplaceAll(content, `"cloudflare_load_balancer"`, `"cloudflare_lb"`)
					},
				},
			},
			expectedOutput: `resource "cloudflare_dns_record" "example1" {
  name = "example1"
}

resource "cloudflare_lb" "example2" {
  name = "example2"
}`,
		},
		{
			name: "No transformation needed",
			input: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}`,
			transformers: []*MockResourceTransformer{},
			expectedOutput: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}`,
		},
		{
			name: "Invalid HCL syntax",
			input: `resource "test" "example" {
  invalid syntax here
}`,
			transformers:  []*MockResourceTransformer{},
			expectError:   true,
			errorContains: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert slice to interface slice for variadic function
			var transformers []transform.ResourceTransformer
			for _, tr := range tt.transformers {
				transformers = append(transformers, tr)
			}
			providers := setupTestMigrators(t, transformers...)
			p := pipeline.BuildConfigPipeline(log, providers)

			ctx := &transform.Context{
				Content:       []byte(tt.input),
				Filename:      "test.tf",
				SourceVersion: sourceVersion,
				TargetVersion: targetVersion,
			}
			result, err := p.Transform(ctx)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error containing '%s', got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			actualOutput := strings.TrimSpace(string(result))
			expectedOutput := strings.TrimSpace(tt.expectedOutput)

			if actualOutput != expectedOutput {
				t.Errorf("Output mismatch:\nExpected:\n%s\n\nGot:\n%s", expectedOutput, actualOutput)
			}
		})
	}
}

func TestTransformerCallOrder(t *testing.T) {
	callLog := []string{}

	transformer := &MockResourceTransformer{
		resourceType: "test_resource",
		preprocessFunc: func(content string) string {
			callLog = append(callLog, "preprocess")
			return content
		},
		transformFunc: func(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
			callLog = append(callLog, "transform")
			return &transform.TransformResult{
				Blocks:         []*hclwrite.Block{block},
				RemoveOriginal: false,
			}, nil
		},
	}

	providers := setupTestMigrators(t, transformer)
	p := pipeline.BuildConfigPipeline(log, providers)

	input := `resource "test_resource" "example" { }`
	ctx := &transform.Context{
		Content:       []byte(input),
		Filename:      "test.tf",
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
		Metadata:      make(map[string]interface{}),
	}
	_, err := p.Transform(ctx)

	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	// Verify call order
	expectedOrder := []string{"preprocess", "transform"}
	if len(callLog) != len(expectedOrder) {
		t.Errorf("Expected %d calls, got %d", len(expectedOrder), len(callLog))
	}

	for i, expected := range expectedOrder {
		if i >= len(callLog) || callLog[i] != expected {
			t.Errorf("Expected call %d to be '%s', got '%s'", i, expected, callLog[i])
		}
	}
}

func TestResourceTransformationWithRemoval(t *testing.T) {
	transformer := &MockResourceTransformer{
		resourceType: "deprecated_resource",
		transformFunc: func(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
			// Remove the block entirely
			return &transform.TransformResult{
				Blocks:         []*hclwrite.Block{},
				RemoveOriginal: true,
			}, nil
		},
	}

	providers := setupTestMigrators(t, transformer)
	p := pipeline.BuildConfigPipeline(log, providers)

	input := `resource "deprecated_resource" "to_remove" {
  name = "remove_me"
}

resource "other_resource" "to_keep" {
  name = "keep_me"
}`

	ctx := &transform.Context{
		Content:       []byte(input),
		Filename:      "test.tf",
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
		Metadata:      make(map[string]interface{}),
	}
	result, err := p.Transform(ctx)
	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	// The deprecated resource should be removed
	output := string(result)
	if strings.Contains(output, "deprecated_resource") {
		t.Error("Deprecated resource should have been removed")
	}
	if !strings.Contains(output, "other_resource") {
		t.Error("Other resource should have been kept")
	}
}

// Test resource transformation with block splitting
func TestResourceTransformationWithSplitting(t *testing.T) {
	transformer := &MockResourceTransformer{
		resourceType: "combined_resource",
		transformFunc: func(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
			// Split into two blocks
			block1 := hclwrite.NewBlock("resource", []string{"split_resource_1", "part1"})
			block1.Body().SetAttributeValue("name", cty.StringVal("part1"))

			block2 := hclwrite.NewBlock("resource", []string{"split_resource_2", "part2"})
			block2.Body().SetAttributeValue("name", cty.StringVal("part2"))

			return &transform.TransformResult{
				Blocks:         []*hclwrite.Block{block1, block2},
				RemoveOriginal: true,
			}, nil
		},
	}

	providers := setupTestMigrators(t, transformer)
	p := pipeline.BuildConfigPipeline(log, providers)

	input := `resource "combined_resource" "original" {
  name = "combined"
}`

	ctx := &transform.Context{
		Content:       []byte(input),
		Filename:      "test.tf",
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
		Metadata:      make(map[string]interface{}),
	}
	result, err := p.Transform(ctx)
	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	output := string(result)
	if strings.Contains(output, "combined_resource") {
		t.Error("Original combined resource should have been removed")
	}
	if !strings.Contains(output, "split_resource_1") {
		t.Error("First split resource should be present")
	}
	if !strings.Contains(output, "split_resource_2") {
		t.Error("Second split resource should be present")
	}
}

// Test error propagation through pipeline
func TestPipelineErrorPropagation(t *testing.T) {
	// Test with nil content - should handle gracefully as empty content
	providers := setupTestMigrators(t)
	p := pipeline.BuildConfigPipeline(log, providers)
	ctx := &transform.Context{
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
		Metadata:      make(map[string]interface{}),
	}
	result, err := p.Transform(ctx)
	if err != nil {
		t.Errorf("Unexpected error for nil content: %v", err)
	}
	// Nil content should be treated as empty content and return empty result
	if len(result) != 0 {
		t.Errorf("Expected empty result for nil content, got %d bytes", len(result))
	}

	// Test with invalid HCL
	invalidHCL := `resource "test" {{{`
	ctx = &transform.Context{
		Content:       []byte(invalidHCL),
		Filename:      "test.tf",
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
		Metadata:      make(map[string]interface{}),
	}
	_, err = p.Transform(ctx)
	if err == nil {
		t.Error("Expected error for invalid HCL")
	}
}

func TestStandardPipelines(t *testing.T) {
	providers := setupTestMigrators(t)
	t.Run("BuildConfigPipeline uses correct handlers", func(t *testing.T) {
		p := pipeline.BuildConfigPipeline(log, providers)
		if p == nil {
			t.Fatal("BuildConfigPipeline returned nil")
		}

		// Should work on HCL
		ctx := &transform.Context{
			Content:       []byte(`resource "test" "example" {}`),
			Filename:      "test.tf",
			SourceVersion: sourceVersion,
			TargetVersion: targetVersion,
		}
		_, err := p.Transform(ctx)
		if err != nil {
			t.Errorf("Config pipeline failed on valid HCL: %v", err)
		}
	})

	t.Run("BuildStatePipeline uses correct handlers", func(t *testing.T) {
		p := pipeline.BuildStatePipeline(log, providers)
		if p == nil {
			t.Fatal("BuildStatePipeline returned nil")
		}

		// Should work on JSON
		ctx := &transform.Context{
			StateJSON:     `{"version":4}`,
			Filename:      "terraform.tfstate",
			SourceVersion: sourceVersion,
			TargetVersion: targetVersion,
		}
		_, err := p.Transform(ctx)
		if err != nil {
			t.Errorf("State pipeline failed on valid JSON: %v", err)
		}
	})
}
