package pipeline_test

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/vaishak/tf-migrate/internal/interfaces"
	"github.com/vaishak/tf-migrate/internal/pipeline"
	"github.com/vaishak/tf-migrate/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

type MockHandler struct {
	interfaces.BaseHandler
	name   string
	called bool
}

func (m *MockHandler) Handle(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
	m.called = true
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]interface{})
	}
	ctx.Metadata[m.name] = true
	return m.CallNext(ctx)
}

type MockResourceTransformer struct {
	resourceType        string
	canHandleFunc       func(string) bool
	preprocessFunc      func(string) string
	transformFunc       func(*hclwrite.Block) (*interfaces.TransformResult, error)
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

func (m *MockResourceTransformer) TransformConfig(block *hclwrite.Block) (*interfaces.TransformResult, error) {
	m.transformCalls++
	if m.transformFunc != nil {
		return m.transformFunc(block)
	}
	return &interfaces.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *MockResourceTransformer) TransformState(json gjson.Result, resourcePath string) (string, error) {
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
			reg := registry.NewStrategyRegistry()
			for _, transformer := range tt.transformers {
				reg.Register(transformer)
			}
			p := pipeline.BuildConfigPipeline(reg)

			result, err := p.Transform([]byte(tt.input), "test.tf")

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

func TestPipelineBuilder(t *testing.T) {
	reg := registry.NewStrategyRegistry()

	p := pipeline.NewPipelineBuilder(reg).
		With(pipeline.Preprocess).
		With(pipeline.Parse).
		With(pipeline.TransformResources).
		With(pipeline.Format).
		Build()

	input := `resource "test" "example" { }`
	result, err := p.Transform([]byte(input), "test.tf")

	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("Pipeline returned empty result")
	}
}

func TestPartialPipeline(t *testing.T) {
	p := pipeline.NewPipelineBuilder(nil).
		With(pipeline.Parse).
		With(pipeline.Format).
		Build()

	input := `resource   "test"   "example"   {
  name="test"
}`

	result, err := p.Transform([]byte(input), "test.tf")
	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	// Should be formatted properly
	expected := `resource "test" "example" {
  name = "test"
}`

	if strings.TrimSpace(string(result)) != strings.TrimSpace(expected) {
		t.Errorf("Expected formatted output:\n%s\nGot:\n%s", expected, string(result))
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
		transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
			callLog = append(callLog, "transform")
			return &interfaces.TransformResult{
				Blocks:         []*hclwrite.Block{block},
				RemoveOriginal: false,
			}, nil
		},
	}

	reg := registry.NewStrategyRegistry()
	reg.Register(transformer)

	p := pipeline.BuildConfigPipeline(reg)

	input := `resource "test_resource" "example" { }`
	_, err := p.Transform([]byte(input), "test.tf")

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
		transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
			// Remove the block entirely
			return &interfaces.TransformResult{
				Blocks:         []*hclwrite.Block{},
				RemoveOriginal: true,
			}, nil
		},
	}

	reg := registry.NewStrategyRegistry()
	reg.Register(transformer)

	p := pipeline.BuildConfigPipeline(reg)

	input := `resource "deprecated_resource" "to_remove" {
  name = "remove_me"
}

resource "other_resource" "to_keep" {
  name = "keep_me"
}`

	result, err := p.Transform([]byte(input), "test.tf")
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
		transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
			// Split into two blocks
			block1 := hclwrite.NewBlock("resource", []string{"split_resource_1", "part1"})
			block1.Body().SetAttributeValue("name", cty.StringVal("part1"))

			block2 := hclwrite.NewBlock("resource", []string{"split_resource_2", "part2"})
			block2.Body().SetAttributeValue("name", cty.StringVal("part2"))

			return &interfaces.TransformResult{
				Blocks:         []*hclwrite.Block{block1, block2},
				RemoveOriginal: true,
			}, nil
		},
	}

	reg := registry.NewStrategyRegistry()
	reg.Register(transformer)

	p := pipeline.BuildConfigPipeline(reg)

	input := `resource "combined_resource" "original" {
  name = "combined"
}`

	result, err := p.Transform([]byte(input), "test.tf")
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
	p := pipeline.BuildConfigPipeline(registry.NewStrategyRegistry())

	result, err := p.Transform(nil, "test.tf")
	if err != nil {
		t.Errorf("Unexpected error for nil content: %v", err)
	}
	// Nil content should be treated as empty content and return empty result
	if len(result) != 0 {
		t.Errorf("Expected empty result for nil content, got %d bytes", len(result))
	}

	// Test with invalid HCL
	invalidHCL := `resource "test" {{{`
	_, err = p.Transform([]byte(invalidHCL), "test.tf")
	if err == nil {
		t.Error("Expected error for invalid HCL")
	}
}

func TestGenericPipelineBuilder(t *testing.T) {
	t.Run("Custom handler factory", func(t *testing.T) {
		reg := registry.NewStrategyRegistry()

		handler1 := &MockHandler{name: "handler1"}
		handler2 := &MockHandler{name: "handler2"}

		customFactory1 := func(_ *registry.StrategyRegistry) interfaces.TransformationHandler {
			return handler1
		}
		customFactory2 := func(_ *registry.StrategyRegistry) interfaces.TransformationHandler {
			return handler2
		}

		p := pipeline.NewPipelineBuilder(reg).
			With(customFactory1).
			With(customFactory2).
			With(pipeline.Parse).
			With(pipeline.Format).
			Build()

		// Test the pipeline
		result, err := p.Transform([]byte(`resource "test" "example" {}`), "test.tf")
		if err != nil {
			t.Fatalf("Pipeline failed: %v", err)
		}

		// Verify handlers were called
		if !handler1.called {
			t.Error("Handler1 was not called")
		}
		if !handler2.called {
			t.Error("Handler2 was not called")
		}

		if len(result) == 0 {
			t.Error("Pipeline returned empty result")
		}
	})

	t.Run("WithHandler for pre-created instances", func(t *testing.T) {
		reg := registry.NewStrategyRegistry()

		handler := &MockHandler{name: "custom"}

		p := pipeline.NewPipelineBuilder(reg).
			WithHandler(handler).
			With(pipeline.Parse).
			With(pipeline.Format).
			Build()

		_, err := p.Transform([]byte(`resource "test" "example" {}`), "test.tf")
		if err != nil {
			t.Fatalf("Pipeline failed: %v", err)
		}

		if !handler.called {
			t.Error("Custom handler was not called")
		}
	})

	t.Run("WithHandlers for multiple instances", func(t *testing.T) {
		reg := registry.NewStrategyRegistry()

		handlers := []interfaces.TransformationHandler{
			&MockHandler{name: "h1"},
			&MockHandler{name: "h2"},
			&MockHandler{name: "h3"},
		}

		p := pipeline.NewPipelineBuilder(reg).
			WithHandlers(handlers...).
			With(pipeline.Parse).
			With(pipeline.Format).
			Build()

		// Test the pipeline
		_, err := p.Transform([]byte(`resource "test" "example" {}`), "test.tf")
		if err != nil {
			t.Fatalf("Pipeline failed: %v", err)
		}

		// Check all were called
		for i, h := range handlers {
			if mock, ok := h.(*MockHandler); ok {
				if !mock.called {
					t.Errorf("Handler %d was not called", i)
				}
			}
		}
	})

	t.Run("Dynamic pipeline construction", func(t *testing.T) {
		reg := registry.NewStrategyRegistry()

		builder := pipeline.NewPipelineBuilder(reg)

		// Dynamically add handlers based on conditions
		needPreprocess := true
		needTransform := true

		if needPreprocess {
			builder = builder.With(pipeline.Preprocess)
		}

		builder = builder.With(pipeline.Parse)

		if needTransform {
			builder = builder.With(pipeline.TransformResources)
		}

		builder = builder.With(pipeline.Format)

		p := builder.Build()

		// Test the pipeline
		_, err := p.Transform([]byte(`resource "test" "example" {}`), "test.tf")
		if err != nil {
			t.Fatalf("Pipeline failed: %v", err)
		}
	})
}

func TestPredefinedPipelines(t *testing.T) {
	reg := registry.NewStrategyRegistry()

	t.Run("BuildConfigPipeline uses correct handlers", func(t *testing.T) {
		p := pipeline.BuildConfigPipeline(reg)
		if p == nil {
			t.Fatal("BuildConfigPipeline returned nil")
		}

		// Should work on HCL
		_, err := p.Transform([]byte(`resource "test" "example" {}`), "test.tf")
		if err != nil {
			t.Errorf("Config pipeline failed on valid HCL: %v", err)
		}
	})

	t.Run("BuildStatePipeline uses correct handlers", func(t *testing.T) {
		p := pipeline.BuildStatePipeline(reg)
		if p == nil {
			t.Fatal("BuildStatePipeline returned nil")
		}

		// Should work on JSON
		_, err := p.Transform([]byte(`{"version":4}`), "terraform.tfstate")
		if err != nil {
			t.Errorf("State pipeline failed on valid JSON: %v", err)
		}
	})
}
