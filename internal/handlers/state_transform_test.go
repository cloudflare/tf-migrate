package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/vaishak/tf-migrate/internal/handlers"
	"github.com/vaishak/tf-migrate/internal/interfaces"
	"github.com/vaishak/tf-migrate/internal/registry"
)

func TestStateTransformHandler(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		transformer   *MockResourceTransformer
		expectError   bool
		checkResult   func(*testing.T, *interfaces.TransformContext)
	}{
		{
			name: "Transform simple state resource",
			input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "old_resource",
      "name": "example",
      "instances": [
        {
          "attributes": {
            "id": "123",
            "name": "test"
          }
        }
      ]
    }
  ]
}`,
			transformer: &MockResourceTransformer{
				resourceType: "old_resource",
				stateTransformFunc: func(json gjson.Result, path string) (string, error) {
					// Transform to new format
					return `{
						"attributes": {
							"id": "123",
							"name": "test-transformed"
						}
					}`, nil
				},
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				var result map[string]interface{}
				if err := json.Unmarshal(ctx.Content, &result); err != nil {
					t.Fatalf("Failed to parse result JSON: %v", err)
				}

				resources := result["resources"].([]interface{})
				if len(resources) != 1 {
					t.Errorf("Expected 1 resource, got %d", len(resources))
				}

				resource := resources[0].(map[string]interface{})
				instances := resource["instances"].([]interface{})
				instance := instances[0].(map[string]interface{})
				attrs := instance["attributes"].(map[string]interface{})

				if attrs["name"] != "test-transformed" {
					t.Errorf("Expected transformed name 'test-transformed', got %v", attrs["name"])
				}
			},
		},
		{
			name: "Handle invalid JSON",
			input: `{invalid json`,
			expectError: true,
		},
		{
			name: "Handle empty state",
			input: `{}`,
			expectError: false,
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				var result map[string]interface{}
				if err := json.Unmarshal(ctx.Content, &result); err != nil {
					t.Fatalf("Failed to parse result JSON: %v", err)
				}
				// Should remain empty
				if len(result) != 0 {
					t.Error("Expected empty state to remain empty")
				}
			},
		},
		{
			name: "Skip resources without strategy",
			input: `{
  "resources": [
    {
      "type": "unknown_resource",
      "instances": [{"attributes": {"id": "456"}}]
    }
  ]
}`,
			transformer: &MockResourceTransformer{
				resourceType: "different_resource",
			},
			checkResult: func(t *testing.T, ctx *interfaces.TransformContext) {
				// Should pass through unchanged
				var result map[string]interface{}
				json.Unmarshal(ctx.Content, &result)

				resources := result["resources"].([]interface{})
				resource := resources[0].(map[string]interface{})

				if resource["type"] != "unknown_resource" {
					t.Error("Resource type should remain unchanged")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := registry.NewStrategyRegistry()
			if tt.transformer != nil {
				reg.Register(tt.transformer)
			}

			handler := handlers.NewStateTransformHandler(reg)
			ctx := &interfaces.TransformContext{
				Content:  []byte(tt.input),
				Filename: "terraform.tfstate",
				Metadata: make(map[string]interface{}),
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
				tt.checkResult(t, result)
			}
		})
	}
}
