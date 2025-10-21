package processing_test

import (
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/processing"
	"github.com/cloudflare/tf-migrate/internal/core"
	"github.com/cloudflare/tf-migrate/internal/core/transform"
)

func TestProcessConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name: "simple resource passthrough",
			input: `resource "test_resource" "example" {
  name = "test"
}`,
			expected: `resource "test_resource" "example" {
  name = "test"
}`,
			wantErr: false,
		},
		{
			name: "invalid HCL",
			input: `resource "test" {{{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := core.NewRegistry()
			
			result, err := processing.ProcessConfig([]byte(tt.input), "test.tf", reg)
			
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			actual := strings.TrimSpace(string(result))
			expected := strings.TrimSpace(tt.expected)
			if actual != expected {
				t.Errorf("output mismatch:\nExpected:\n%s\nGot:\n%s", expected, actual)
			}
		})
	}
}

func TestProcessConfigWithMockTransformer(t *testing.T) {
	// Test with a mock transformer since no real ones exist yet
	input := `resource "test_resource" "example" {
  name = "test"
}`

	// Create a mock transformer for testing
	reg := core.NewRegistry()
	mockTransformer := &transform.BaseTransformer{
		ResourceType: "test_resource",
		Preprocessor: func(content string) string {
			// Simple string replacement for testing
			return strings.ReplaceAll(content, "test_resource", "transformed_resource")
		},
	}
	reg.Register(mockTransformer)
	
	result, err := processing.ProcessConfig([]byte(input), "test.tf", reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	output := string(result)
	
	// Should transform test_resource to transformed_resource via preprocessor
	if !strings.Contains(output, "transformed_resource") {
		t.Error("expected resource type to be transformed to transformed_resource")
	}
	
	// Should preserve the resource content
	if !strings.Contains(output, `name = "test"`) {
		t.Error("expected resource attributes to be preserved")
	}
}

func TestProcessState(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{
			name: "valid empty state",
			input: `{"version": 4}`,
			wantErr: false,
		},
		{
			name: "state with resources",
			input: `{
  "version": 4,
  "resources": [
    {
      "type": "test_resource",
      "name": "example",
      "instances": [
        {
          "attributes": {
            "id": "123"
          }
        }
      ]
    }
  ]
}`,
			wantErr: false,
		},
		{
			name: "invalid JSON",
			input: `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := core.NewRegistry()
			
			result, err := processing.ProcessState([]byte(tt.input), "terraform.tfstate", reg)
			
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			// Result should be valid JSON
			if len(result) == 0 {
				t.Error("expected non-empty result")
			}
		})
	}
}