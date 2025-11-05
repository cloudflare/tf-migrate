package handlers_test

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal/handlers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestFormatterHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupAST       func() *hclwrite.File
		expectedOutput string
		expectError    bool
	}{
		{
			name: "Format simple resource",
			setupAST: func() *hclwrite.File {
				f := hclwrite.NewEmptyFile()
				block := f.Body().AppendNewBlock("resource", []string{"test_resource", "example"})
				block.Body().SetAttributeValue("name", cty.StringVal("test"))
				block.Body().SetAttributeValue("count", cty.NumberIntVal(5))
				return f
			},
			expectedOutput: `resource "test_resource" "example" {
  name  = "test"
  count = 5
}`,
		},
		{
			name: "Format multiple resources",
			setupAST: func() *hclwrite.File {
				f := hclwrite.NewEmptyFile()

				block1 := f.Body().AppendNewBlock("resource", []string{"resource_a", "a"})
				block1.Body().SetAttributeValue("name", cty.StringVal("a"))

				f.Body().AppendNewline()

				block2 := f.Body().AppendNewBlock("resource", []string{"resource_b", "b"})
				block2.Body().SetAttributeValue("name", cty.StringVal("b"))

				return f
			},
			expectedOutput: `resource "resource_a" "a" {
  name = "a"
}

resource "resource_b" "b" {
  name = "b"
}`,
		},
		{
			name: "Format nested blocks",
			setupAST: func() *hclwrite.File {
				f := hclwrite.NewEmptyFile()
				resource := f.Body().AppendNewBlock("resource", []string{"complex", "example"})
				resource.Body().SetAttributeValue("name", cty.StringVal("test"))

				nested := resource.Body().AppendNewBlock("nested_block", nil)
				nested.Body().SetAttributeValue("key", cty.StringVal("value"))

				deepNested := nested.Body().AppendNewBlock("deep", nil)
				deepNested.Body().SetAttributeValue("id", cty.NumberIntVal(123))

				return f
			},
			expectedOutput: `resource "complex" "example" {
  name = "test"
  nested_block {
    key = "value"
    deep {
      id = 123
    }
  }
}`,
		},
		{
			name: "Format empty file",
			setupAST: func() *hclwrite.File {
				return hclwrite.NewEmptyFile()
			},
			expectedOutput: ``,
		},
		{
			name: "Format with comments preserved",
			setupAST: func() *hclwrite.File {
				f := hclwrite.NewEmptyFile()

				// Add a resource with attributes
				block := f.Body().AppendNewBlock("resource", []string{"test", "example"})
				block.Body().SetAttributeValue("name", cty.StringVal("test"))

				// Note: hclwrite doesn't preserve comments in the CFGFile,
				// so this test mainly ensures formatting doesn't break
				return f
			},
			expectedOutput: `resource "test" "example" {
  name = "test"
}`,
		},
		{
			name: "Format data source",
			setupAST: func() *hclwrite.File {
				f := hclwrite.NewEmptyFile()
				block := f.Body().AppendNewBlock("data", []string{"external", "example"})
				block.Body().SetAttributeValue("program", cty.ListVal([]cty.Value{
					cty.StringVal("python"),
					cty.StringVal("script.py"),
				}))
				return f
			},
			expectedOutput: `data "external" "example" {
  program = ["python", "script.py"]
}`,
		},
		{
			name: "Format with various attribute types",
			setupAST: func() *hclwrite.File {
				f := hclwrite.NewEmptyFile()
				block := f.Body().AppendNewBlock("resource", []string{"test", "example"})

				block.Body().SetAttributeValue("string_attr", cty.StringVal("value"))
				block.Body().SetAttributeValue("number_attr", cty.NumberIntVal(42))
				block.Body().SetAttributeValue("bool_attr", cty.BoolVal(true))
				block.Body().SetAttributeValue("list_attr", cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}))

				return f
			},
			expectedOutput: `resource "test" "example" {
  string_attr = "value"
  number_attr = 42
  bool_attr   = true
  list_attr   = ["a", "b", "c"]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler
			handler := handlers.NewFormatterHandler(log)

			// Create context with CFGFile
			ctx := &transform.Context{
				CFGFile:  tt.setupAST(),
				Filename: "test.tf",
			}

			// Process
			result, err := handler.Handle(ctx)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Compare output (normalize whitespace)
			actualOutput := strings.TrimSpace(string(result.Content))
			expectedOutput := strings.TrimSpace(tt.expectedOutput)

			if actualOutput != expectedOutput {
				t.Errorf("Output mismatch:\nExpected:\n%s\n\nGot:\n%s", expectedOutput, actualOutput)
			}
		})
	}
}

func TestFormatterHandlerRequiresAST(t *testing.T) {
	handler := handlers.NewFormatterHandler(log)

	ctx := &transform.Context{
		Content: []byte("some content"),
	}

	_, err := handler.Handle(ctx)
	if err == nil {
		t.Fatal("Expected error when CFGFile is nil")
	}

	if !strings.Contains(err.Error(), "CFGFile is nil") {
		t.Errorf("Expected error about nil CFGFile, got: %v", err)
	}
}

func TestFormatterHandlerChaining(t *testing.T) {
	nextHandlerCalled := false

	mockNext := &mockHandler{
		handleFunc: func(ctx *transform.Context) (*transform.Context, error) {
			nextHandlerCalled = true
			if len(ctx.Content) == 0 {
				t.Error("Content should be set when next handler is called")
			}
			return ctx, nil
		},
	}

	handler := handlers.NewFormatterHandler(log)
	handler.SetNext(mockNext)

	f := hclwrite.NewEmptyFile()
	f.Body().AppendNewBlock("resource", []string{"test", "example"})

	ctx := &transform.Context{
		CFGFile: f,
	}

	_, err := handler.Handle(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !nextHandlerCalled {
		t.Error("Next handler should have been called")
	}
}

func TestFormatterPreservesAST(t *testing.T) {
	handler := handlers.NewFormatterHandler(log)

	f := hclwrite.NewEmptyFile()
	originalBlock := f.Body().AppendNewBlock("resource", []string{"test", "example"})
	originalBlock.Body().SetAttributeValue("name", cty.StringVal("test"))

	ctx := &transform.Context{
		CFGFile: f,
	}

	originalBlockCount := len(f.Body().Blocks())

	result, err := handler.Handle(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.CFGFile != f {
		t.Error("CFGFile reference should be unchanged")
	}

	if len(result.CFGFile.Body().Blocks()) != originalBlockCount {
		t.Error("CFGFile structure should be unchanged after formatting")
	}
}

func TestFormatterSpecialCharacters(t *testing.T) {
	handler := handlers.NewFormatterHandler(log)

	f := hclwrite.NewEmptyFile()
	block := f.Body().AppendNewBlock("resource", []string{"test", "example"})

	// Test various special characters and escaping
	block.Body().SetAttributeValue("escaped_quotes", cty.StringVal(`He said "Hello"`))
	block.Body().SetAttributeValue("newlines", cty.StringVal("line1\nline2"))
	block.Body().SetAttributeValue("tabs", cty.StringVal("col1\tcol2"))

	ctx := &transform.Context{
		CFGFile: f,
	}

	result, err := handler.Handle(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := string(result.Content)

	// Check that special characters are properly escaped
	if !strings.Contains(output, `\"`) {
		t.Error("Quotes should be escaped in output")
	}
	if !strings.Contains(output, `\n`) {
		t.Error("Newlines should be escaped in output")
	}
	if !strings.Contains(output, `\t`) {
		t.Error("Tabs should be escaped in output")
	}
}
