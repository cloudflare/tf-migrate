package handlers_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	
	"github.com/cloudflare/tf-migrate/internal/handlers"
	"github.com/cloudflare/tf-migrate/internal/interfaces"
	"github.com/cloudflare/tf-migrate/internal/registry"
)

func TestResourceTransformHandler(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		transformers []*MockResourceTransformer
		expectError  bool
		checkResult  func(*testing.T, *interfaces.TransformContext)
	}{
		{
			name: "Transform single resource",
			input: `resource "old_resource" "example" {
  name = "test"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "old_resource",
					transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
						// Change the resource type
						newBlock := hclwrite.NewBlock("resource", []string{"new_resource", block.Labels()[1]})
						// Copy attributes
						for name, attr := range block.Body().Attributes() {
							newBlock.Body().SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
						}
						return &interfaces.TransformResult{
							Blocks:         []*hclwrite.Block{newBlock},
							RemoveOriginal: true,
						}, nil
					},
				},
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 1 {
					t.Errorf("Expected 1 block after transformation, got %d", len(blocks))
				}
				if blocks[0].Labels()[0] != "new_resource" {
					t.Errorf("Expected resource type to be 'new_resource', got %s", blocks[0].Labels()[0])
				}
			},
		},
		{
			name: "Remove resource entirely",
			input: `resource "deprecated" "to_remove" {
  name = "remove_me"
}
resource "keep_me" "example" {
  name = "keep"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "deprecated",
					transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
						return &interfaces.TransformResult{
							Blocks:         []*hclwrite.Block{},
							RemoveOriginal: true,
						}, nil
					},
				},
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 1 {
					t.Errorf("Expected 1 block after removal, got %d", len(blocks))
				}
				if blocks[0].Labels()[0] != "keep_me" {
					t.Errorf("Wrong block kept, expected 'keep_me', got %s", blocks[0].Labels()[0])
				}
			},
		},
		{
			name: "Split resource into multiple",
			input: `resource "combined" "example" {
  name = "test"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "combined",
					transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
						block1 := hclwrite.NewBlock("resource", []string{"split_a", "part_a"})
						block1.Body().SetAttributeValue("name", cty.StringVal("part_a"))

						block2 := hclwrite.NewBlock("resource", []string{"split_b", "part_b"})
						block2.Body().SetAttributeValue("name", cty.StringVal("part_b"))

						return &interfaces.TransformResult{
							Blocks:         []*hclwrite.Block{block1, block2},
							RemoveOriginal: true,
						}, nil
					},
				},
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 2 {
					t.Errorf("Expected 2 blocks after split, got %d", len(blocks))
				}

				resourceTypes := []string{blocks[0].Labels()[0], blocks[1].Labels()[0]}
				if !contains(resourceTypes, "split_a") || !contains(resourceTypes, "split_b") {
					t.Errorf("Expected split_a and split_b resources, got %v", resourceTypes)
				}
			},
		},
		{
			name: "In-place modification",
			input: `resource "modify_me" "example" {
  old_attribute = "old_value"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "modify_me",
					transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
						// Modify the block in place
						block.Body().RemoveAttribute("old_attribute")
						block.Body().SetAttributeValue("new_attribute", cty.StringVal("new_value"))

						return &interfaces.TransformResult{
							Blocks:         []*hclwrite.Block{block},
							RemoveOriginal: false, // Keep in same position
						}, nil
					},
				},
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 1 {
					t.Errorf("Expected 1 block, got %d", len(blocks))
				}

				attrs := blocks[0].Body().Attributes()
				if _, exists := attrs["old_attribute"]; exists {
					t.Error("old_attribute should have been removed")
				}
				if _, exists := attrs["new_attribute"]; !exists {
					t.Error("new_attribute should have been added")
				}
			},
		},
		{
			name: "No transformer for resource type",
			input: `resource "unknown_type" "example" {
  name = "test"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "different_type",
				},
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				// Should remain unchanged
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 1 {
					t.Errorf("Expected 1 block unchanged, got %d", len(blocks))
				}
				if blocks[0].Labels()[0] != "unknown_type" {
					t.Errorf("Resource type should remain unchanged")
				}
			},
		},
		{
			name: "Transformer returns error",
			input: `resource "error_resource" "example" {
  name = "test"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "error_resource",
					transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
						return nil, fmt.Errorf("transformation failed")
					},
				},
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				// Check that diagnostics were added
				if !ctx.Diagnostics.HasErrors() {
					t.Error("Expected diagnostics to contain error")
				}

				// Original block should remain
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 1 {
					t.Errorf("Expected original block to remain on error, got %d blocks", len(blocks))
				}
			},
		},
		{
			name: "Multiple transformers for different resources",
			input: `resource "type_a" "a" {
  name = "a"
}
resource "type_b" "b" {
  name = "b"
}
resource "type_c" "c" {
  name = "c"
}`,
			transformers: []*MockResourceTransformer{
				{
					resourceType: "type_a",
					transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
						block.Body().SetAttributeValue("transformed", cty.BoolVal(true))
						return &interfaces.TransformResult{
							Blocks:         []*hclwrite.Block{block},
							RemoveOriginal: false,
						}, nil
					},
				},
				{
					resourceType: "type_b",
					transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
						// Remove type_b
						return &interfaces.TransformResult{
							Blocks:         []*hclwrite.Block{},
							RemoveOriginal: true,
						}, nil
					},
				},
				// No transformer for type_c - should remain unchanged
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 2 {
					t.Errorf("Expected 2 blocks (a and c), got %d", len(blocks))
				}

				// Check that type_a was transformed
				for _, block := range blocks {
					if block.Labels()[0] == "type_a" {
						if _, exists := block.Body().Attributes()["transformed"]; !exists {
							t.Error("type_a should have 'transformed' attribute")
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create registry and register transformers
			reg := registry.NewStrategyRegistry()
			for _, transformer := range tt.transformers {
				reg.Register(transformer)
			}

			// Parse the input first to get AST
			parseHandler := handlers.NewParseHandler()
			ctx := &interfaces.TransformContext{
				Content:  []byte(tt.input),
				Filename: "test.tf",
				Metadata: make(map[string]interface{}),
			}

			ctx, err := parseHandler.Handle(ctx)
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Create and run ResourceTransformHandler
			handler := handlers.NewResourceTransformHandler(reg)
			result, err := handler.Handle(ctx)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil && !tt.expectError {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Run custom checks
			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestResourceTransformHandlerRequiresAST(t *testing.T) {
	reg := registry.NewStrategyRegistry()
	handler := handlers.NewResourceTransformHandler(reg)

	ctx := &interfaces.TransformContext{
		Content: []byte("some content"),
	}

	_, err := handler.Handle(ctx)
	if err == nil {
		t.Fatal("Expected error when AST is nil")
	}

	if !strings.Contains(err.Error(), "AST is nil") {
		t.Errorf("Expected error about nil AST, got: %v", err)
	}
}

func TestResourceTransformHandlerMetadata(t *testing.T) {
	transformer := &MockResourceTransformer{
		resourceType: "tracked_resource",
		transformFunc: func(block *hclwrite.Block) (*interfaces.TransformResult, error) {
			return &interfaces.TransformResult{
				Blocks:         []*hclwrite.Block{block},
				RemoveOriginal: false,
			}, nil
		},
	}

	reg := registry.NewStrategyRegistry()
	reg.Register(transformer)

	input := `resource "tracked_resource" "one" {}
resource "tracked_resource" "two" {}
resource "tracked_resource" "three" {}`

	// Parse first
	parseHandler := handlers.NewParseHandler()
	ctx := &interfaces.TransformContext{
		Content:  []byte(input),
		Metadata: make(map[string]interface{}),
	}

	ctx, _ = parseHandler.Handle(ctx)

	handler := handlers.NewResourceTransformHandler(reg)
	result, err := handler.Handle(ctx)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	key := fmt.Sprintf("transformed_%s", transformer.resourceType)
	if count, ok := result.Metadata[key]; ok {
		if count != 3 {
			t.Errorf("Expected transformation count of 3, got %v", count)
		}
	} else {
		t.Error("Expected transformation count in metadata")
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
