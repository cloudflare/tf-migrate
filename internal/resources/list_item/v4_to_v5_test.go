package list_item

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		t.Run("ResourceRemoval", testResourceRemoval)
	})
}

func testResourceRemoval(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "List item resource should be removed",
			Input: `resource "cloudflare_list_item" "test_item" {
  account_id = "abc123"
  list_id    = cloudflare_list.test.id
  ip         = "192.0.2.1"
  comment    = "Test IP"
}`,
			// The resource should be completely removed
			Expected: ``,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestProcessCrossResourceMigrations(t *testing.T) {
	// Test the cross-resource state migration function
	t.Run("StateMerging", func(t *testing.T) {
		input := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_list",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "attributes": {
            "account_id": "test-account",
            "id": "list-123",
            "kind": "ip",
            "name": "test-list"
          }
        }
      ]
    },
    {
      "type": "cloudflare_list_item",
      "name": "item1",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "attributes": {
            "account_id": "test-account",
            "list_id": "list-123",
            "ip": "192.0.2.1",
            "comment": "Test IP 1"
          }
        }
      ]
    },
    {
      "type": "cloudflare_list_item",
      "name": "item2",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "attributes": {
            "account_id": "test-account",
            "list_id": "list-123",
            "ip": "192.0.2.2",
            "comment": "Test IP 2"
          }
        }
      ]
    }
  ]
}`

		result := ProcessCrossResourceStateMigrations(input)

		// Verify list_item resources are removed
		if containsString(result, `"type":"cloudflare_list_item"`) {
			t.Error("Expected cloudflare_list_item resources to be removed from state")
		}

		// Verify items are merged into the list
		if !containsString(result, `"items"`) {
			t.Error("Expected items array in cloudflare_list resource")
		}

		// Verify specific item data is present
		if !containsString(result, `"ip":"192.0.2.1"`) {
			t.Error("Expected first IP item in merged state")
		}

		if !containsString(result, `"ip":"192.0.2.2"`) {
			t.Error("Expected second IP item in merged state")
		}

		if !containsString(result, `"comment":"Test IP 1"`) {
			t.Error("Expected first comment in merged state")
		}

		// Verify num_items is set correctly
		if !containsString(result, `"num_items":2`) {
			t.Error("Expected num_items to be set to 2")
		}
	})

	t.Run("HostnameStateMerging", func(t *testing.T) {
		input := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_list",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "attributes": {
            "account_id": "test-account",
            "id": "list-456",
            "kind": "hostname",
            "name": "test-hostname-list"
          }
        }
      ]
    },
    {
      "type": "cloudflare_list_item",
      "name": "hostname_item",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "attributes": {
            "account_id": "test-account",
            "list_id": "list-456",
            "hostname": [
              {
                "url_hostname": "example.com"
              }
            ],
            "comment": "Test hostname"
          }
        }
      ]
    }
  ]
}`

		result := ProcessCrossResourceStateMigrations(input)

		// Verify list_item resources are removed
		if containsString(result, `"type":"cloudflare_list_item"`) {
			t.Error("Expected cloudflare_list_item resources to be removed from state")
		}

		// Verify hostname is transformed from array to object
		if !containsString(result, `"hostname":{"url_hostname":"example.com"}`) {
			t.Error("Expected hostname to be transformed from array to object in merged state")
		}
	})

	t.Run("RedirectStateMerging", func(t *testing.T) {
		input := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_list",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "attributes": {
            "account_id": "test-account",
            "id": "list-789",
            "kind": "redirect",
            "name": "test-redirect-list"
          }
        }
      ]
    },
    {
      "type": "cloudflare_list_item",
      "name": "redirect_item",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "attributes": {
            "account_id": "test-account",
            "list_id": "list-789",
            "redirect": [
              {
                "source_url": "example.com/old",
                "target_url": "example.com/new",
                "status_code": 301,
                "include_subdomains": "enabled",
                "subpath_matching": "disabled",
                "preserve_query_string": "enabled",
                "preserve_path_suffix": "disabled"
              }
            ]
          }
        }
      ]
    }
  ]
}`

		result := ProcessCrossResourceStateMigrations(input)

		// Verify list_item resources are removed
		if containsString(result, `"type":"cloudflare_list_item"`) {
			t.Error("Expected cloudflare_list_item resources to be removed from state")
		}

		// Verify redirect booleans are transformed
		if !containsString(result, `"include_subdomains":true`) {
			t.Error("Expected include_subdomains to be transformed to boolean true")
		}

		if !containsString(result, `"subpath_matching":false`) {
			t.Error("Expected subpath_matching to be transformed to boolean false")
		}

		if !containsString(result, `"preserve_query_string":true`) {
			t.Error("Expected preserve_query_string to be transformed to boolean true")
		}

		if !containsString(result, `"preserve_path_suffix":false`) {
			t.Error("Expected preserve_path_suffix to be transformed to boolean false")
		}
	})
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
