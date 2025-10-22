package handlers_test

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/handlers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestPreprocessHandler(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		resources      []*MockResourceTransformer
		expectedOutput string
	}{
		{
			name:           "No registered resources",
			input:          `resource "cloudflare_record" "test" {}`,
			resources:      []*MockResourceTransformer{},
			expectedOutput: `resource "cloudflare_record" "test" {}`,
		},
		{
			name:  "Single resource preprocessor",
			input: `resource "old_resource" "test" {}`,
			resources: []*MockResourceTransformer{
				{
					resourceType: "old_resource",
					preprocessFunc: func(content string) string {
						return "resource \"new_resource\" \"test\" {}"
					},
				},
			},
			expectedOutput: `resource "new_resource" "test" {}`,
		},
		{
			name:  "Multiple resource preprocessors applied in order",
			input: `resource "resource_a" "test" {} resource "resource_b" "test2" {}`,
			resources: []*MockResourceTransformer{
				{
					resourceType: "resource_a",
					preprocessFunc: func(content string) string {
						// First preprocessor changes resource_a to resource_x
						return "resource \"resource_x\" \"test\" {} resource \"resource_b\" \"test2\" {}"
					},
				},
				{
					resourceType: "resource_b",
					preprocessFunc: func(content string) string {
						// Second preprocessor changes resource_b to resource_y
						return "resource \"resource_x\" \"test\" {} resource \"resource_y\" \"test2\" {}"
					},
				},
			},
			expectedOutput: `resource "resource_x" "test" {} resource "resource_y" "test2" {}`,
		},
		{
			name:  "Preprocessor that returns content unchanged",
			input: `resource "some_resource" "test" {}`,
			resources: []*MockResourceTransformer{
				{
					resourceType: "other_resource",
					preprocessFunc: func(content string) string {
						// This preprocessor doesn't find anything to change
						return content
					},
				},
			},
			expectedOutput: `resource "some_resource" "test" {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMockMigratorProvider(tt.resources)
			handler := handlers.NewPreprocessHandler(provider)

			ctx := &transform.Context{
				Content: []byte(tt.input),
			}

			result, err := handler.Handle(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if string(result.Content) != tt.expectedOutput {
				t.Errorf("Expected output:\n%s\nGot:\n%s", tt.expectedOutput, string(result.Content))
			}

			for i, resource := range tt.resources {
				if resource.preprocessCalls != 1 {
					t.Errorf("Resource %d preprocessor called %d times, expected 1", i, resource.preprocessCalls)
				}
			}
		})
	}
}

func TestPreprocessHandlerCallsAllRegisteredPreprocessors(t *testing.T) {
	// This test verifies that ALL registered preprocessors are called,
	// not just those whose resource types are found in the content

	callOrder := []string{}

	resource1 := &MockResourceTransformer{
		resourceType: "resource_type_1",
		preprocessFunc: func(content string) string {
			callOrder = append(callOrder, "resource_1")
			return content
		},
	}

	resource2 := &MockResourceTransformer{
		resourceType: "resource_type_2",
		preprocessFunc: func(content string) string {
			callOrder = append(callOrder, "resource_2")
			return content
		},
	}

	resource3 := &MockResourceTransformer{
		resourceType: "resource_type_3",
		preprocessFunc: func(content string) string {
			callOrder = append(callOrder, "resource_3")
			return content
		},
	}

	provider := NewMockMigratorProvider([]*MockResourceTransformer{resource1, resource2, resource3})
	handler := handlers.NewPreprocessHandler(provider)

	ctx := &transform.Context{
		Content: []byte(`resource "completely_different" "test" {}`),
	}

	_, err := handler.Handle(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedOrder := []string{"resource_1", "resource_2", "resource_3"}
	if len(callOrder) != len(expectedOrder) {
		t.Errorf("Expected %d preprocessor calls, got %d", len(expectedOrder), len(callOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(callOrder) || callOrder[i] != expected {
			t.Errorf("Expected call order[%d] to be %s, got %v", i, expected, callOrder)
		}
	}
}
