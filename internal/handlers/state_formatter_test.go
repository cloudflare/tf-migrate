package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/handlers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestStateFormatterHandler(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkResult func(*testing.T, []byte)
	}{
		{
			name:  "Format valid JSON state",
			input: `{"version":4,"resources":[{"type":"test","instances":[{"attributes":{"id":"123"}}]}]}`,
			checkResult: func(t *testing.T, result []byte) {
				// Should be pretty formatted
				expected := `{
  "resources": [
    {
      "instances": [
        {
          "attributes": {
            "id": "123"
          }
        }
      ],
      "type": "test"
    }
  ],
  "version": 4
}`
				var expectedJSON, resultJSON interface{}
				json.Unmarshal([]byte(expected), &expectedJSON)
				json.Unmarshal(result, &resultJSON)

				expectedBytes, _ := json.MarshalIndent(expectedJSON, "", "  ")
				resultBytes, _ := json.MarshalIndent(resultJSON, "", "  ")

				if string(expectedBytes) != string(resultBytes) {
					t.Errorf("JSON not formatted as expected.\nGot:\n%s\nExpected:\n%s", string(resultBytes), string(expectedBytes))
				}
			},
		},
		{
			name: "Handle already formatted JSON",
			input: `{
  "version": 4,
  "resources": []
}`,
			checkResult: func(t *testing.T, result []byte) {
				// Should remain properly formatted
				if !json.Valid(result) {
					t.Error("Output is not valid JSON")
				}
				// Check it's indented
				if string(result)[0:3] != "{\n " {
					t.Error("JSON should remain formatted with indentation")
				}
			},
		},
		{
			name:  "Leave invalid JSON unchanged",
			input: `{invalid json}`,
			checkResult: func(t *testing.T, result []byte) {
				// Should pass through unchanged
				if string(result) != `{invalid json}` {
					t.Error("Invalid JSON should pass through unchanged")
				}
			},
		},
		{
			name:        "Handle empty content",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlers.NewStateFormatterHandler(log)
			ctx := &transform.Context{
				Content: []byte(tt.input),
			}

			result, err := handler.Handle(ctx)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result.Content)
			}
		})
	}
}

func TestStateFormatterHandlerChaining(t *testing.T) {
	nextHandlerCalled := false

	mockNext := &mockHandler{
		handleFunc: func(ctx *transform.Context) (*transform.Context, error) {
			nextHandlerCalled = true
			// Content should be formatted when next handler is called
			if len(ctx.Content) == 0 {
				t.Error("Content should be set when next handler is called")
			}
			if !json.Valid(ctx.Content) {
				t.Error("Content should be valid JSON")
			}
			return ctx, nil
		},
	}

	handler := handlers.NewStateFormatterHandler(log)
	handler.SetNext(mockNext)

	ctx := &transform.Context{
		Content: []byte(`{"test":true}`),
	}

	_, err := handler.Handle(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !nextHandlerCalled {
		t.Error("Next handler should have been called")
	}
}
