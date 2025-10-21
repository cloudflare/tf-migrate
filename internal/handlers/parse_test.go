package handlers_test

import (
	"strings"
	"testing"

	"github.com/vaishak/tf-migrate/internal/handlers"
	"github.com/vaishak/tf-migrate/internal/interfaces"
)

func TestParseHandler(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectError   bool
		errorContains string
		checkAST      func(*testing.T, *interfaces.TransformContext)
	}{
		{
			name: "Valid HCL parsing",
			input: `resource "test_resource" "example" {
  name = "test"
  count = 5
}`,
			expectError: false,
			checkAST: func(t *testing.T, ctx *interfaces.TransformContext) {
				if ctx.AST == nil {
					t.Fatal("AST should not be nil")
				}

				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 1 {
					t.Errorf("Expected 1 block, got %d", len(blocks))
				}

				if blocks[0].Type() != "resource" {
					t.Errorf("Expected resource block, got %s", blocks[0].Type())
				}

				labels := blocks[0].Labels()
				if len(labels) != 2 || labels[0] != "test_resource" || labels[1] != "example" {
					t.Errorf("Unexpected labels: %v", labels)
				}
			},
		},
		{
			name: "Multiple resources",
			input: `resource "resource_a" "a" {
  name = "a"
}

resource "resource_b" "b" {
  name = "b"
}

data "data_source" "example" {
  id = "123"
}`,
			expectError: false,
			checkAST: func(t *testing.T, ctx *interfaces.TransformContext) {
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 3 {
					t.Errorf("Expected 3 blocks, got %d", len(blocks))
				}

				// Check block types
				expectedTypes := []string{"resource", "resource", "data"}
				for i, block := range blocks {
					if block.Type() != expectedTypes[i] {
						t.Errorf("Block %d: expected type %s, got %s", i, expectedTypes[i], block.Type())
					}
				}
			},
		},
		{
			name:          "Invalid HCL syntax - missing closing brace",
			input:         `resource "test" "example" {`,
			expectError:   true,
			errorContains: "failed to parse",
		},
		{
			name:          "Invalid HCL syntax - malformed attribute",
			input:         `resource "test" "example" { name = }`,
			expectError:   true,
			errorContains: "failed to parse",
		},
		{
			name: "Empty file",
			input: "",
			expectError: false,
			checkAST: func(t *testing.T, ctx *interfaces.TransformContext) {
				if ctx.AST == nil {
					t.Fatal("AST should not be nil even for empty file")
				}
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 0 {
					t.Errorf("Expected 0 blocks for empty file, got %d", len(blocks))
				}
			},
		},
		{
			name: "File with only comments",
			input: `# This is a comment
/* Block comment */`,
			expectError: false,
			checkAST: func(t *testing.T, ctx *interfaces.TransformContext) {
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 0 {
					t.Errorf("Expected 0 blocks for file with only comments, got %d", len(blocks))
				}
			},
		},
		{
			name: "Complex nested structure",
			input: `resource "complex_resource" "example" {
  name = "test"

  nested_block {
    key = "value"

    deeply_nested {
      id = 123
    }
  }

  another_block {
    enabled = true
  }
}`,
			expectError: false,
			checkAST: func(t *testing.T, ctx *interfaces.TransformContext) {
				blocks := ctx.AST.Body().Blocks()
				if len(blocks) != 1 {
					t.Errorf("Expected 1 block, got %d", len(blocks))
				}

				nestedBlocks := blocks[0].Body().Blocks()
				if len(nestedBlocks) != 2 {
					t.Errorf("Expected 2 nested blocks, got %d", len(nestedBlocks))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlers.NewParseHandler()

			ctx := &interfaces.TransformContext{
				Content:  []byte(tt.input),
				Filename: "test.tf",
			}

			result, err := handler.Handle(ctx)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error containing '%s', got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
				if !result.Diagnostics.HasErrors() {
					t.Error("Expected diagnostics to contain errors")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkAST != nil {
				tt.checkAST(t, result)
			}
		})
	}
}

func TestParseHandlerPreservesContent(t *testing.T) {
	input := `resource "test" "example" {
  name = "test"
}`

	handler := handlers.NewParseHandler()
	ctx := &interfaces.TransformContext{
		Content: []byte(input),
	}

	result, err := handler.Handle(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(result.Content) != input {
		t.Error("ParseHandler should not modify content")
	}
}

func TestParseHandlerChaining(t *testing.T) {
	nextHandlerCalled := false

	mockNext := &mockHandler{
		handleFunc: func(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
			nextHandlerCalled = true
			if ctx.AST == nil {
				t.Error("AST should be set when next handler is called")
			}
			return ctx, nil
		},
	}

	handler := handlers.NewParseHandler()
	handler.SetNext(mockNext)

	ctx := &interfaces.TransformContext{
		Content: []byte(`resource "test" "example" {}`),
	}

	_, err := handler.Handle(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !nextHandlerCalled {
		t.Error("Next handler should have been called")
	}
}

type mockHandler struct {
	interfaces.BaseHandler
	handleFunc func(*interfaces.TransformContext) (*interfaces.TransformContext, error)
}

func (m *mockHandler) Handle(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx)
	}
	return m.CallNext(ctx)
}