package snippet_rules

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic snippet_rules migration",
				Input: `resource "cloudflare_snippet_rules" "test" {
  zone_id = "abc123"
  rules {
    expression   = "true"
    snippet_name = "test_snippet_0"
  }
  rules {
    enabled      = true
    expression   = "http.request.uri.path contains \"/api\""
    description  = "API route handler"
    snippet_name = "test_snippet_1"
  }
}`,
				Expected: `resource "cloudflare_snippet_rules" "test" {
  zone_id = "abc123"
  rules = [
    {
      expression   = "true"
      snippet_name = "test_snippet_0"
    },
    {
      enabled      = true
      expression   = "http.request.uri.path contains \"/api\""
      description  = "API route handler"
      snippet_name = "test_snippet_1"
    }
  ]
}`,
			},
			{
				Name: "snippet_rules with all attributes",
				Input: `resource "cloudflare_snippet_rules" "test" {
  zone_id = "abc123"
  rules {
    enabled      = false
    expression   = "http.host eq \"example.com\""
    snippet_name = "snippet_a"
    description  = "First rule"
  }
  rules {
    enabled      = true
    expression   = "http.request.method eq \"POST\""
    snippet_name = "snippet_b"
    description  = "Second rule"
  }
  rules {
    expression   = "ip.src in {10.0.0.0/8}"
    snippet_name = "snippet_c"
  }
}`,
				Expected: `resource "cloudflare_snippet_rules" "test" {
  zone_id = "abc123"
  rules = [
    {
      enabled      = false
      expression   = "http.host eq \"example.com\""
      description  = "First rule"
      snippet_name = "snippet_a"
    },
    {
      enabled      = true
      expression   = "http.request.method eq \"POST\""
      description  = "Second rule"
      snippet_name = "snippet_b"
    },
    {
      expression   = "ip.src in {10.0.0.0/8}"
      snippet_name = "snippet_c"
    }
  ]
}`,
			},
			{
				Name: "snippet_rules with cloudflare_snippet references",
				Input: `resource "cloudflare_snippet_rules" "test" {
  zone_id = "abc123"
  rules {
    expression   = "http.host eq \"example.com\""
    snippet_name = cloudflare_snippet.redirect_snippet.name
  }
  rules {
    enabled      = true
    expression   = "http.request.uri.path contains \"/api\""
    snippet_name = cloudflare_snippet.api_snippet.name
    description  = "API handler"
  }
}`,
				Expected: `resource "cloudflare_snippet_rules" "test" {
  zone_id = "abc123"
  rules = [
    {
      expression   = "http.host eq \"example.com\""
      snippet_name = cloudflare_snippet.redirect_snippet.snippet_name
    },
    {
      enabled      = true
      expression   = "http.request.uri.path contains \"/api\""
      description  = "API handler"
      snippet_name = cloudflare_snippet.api_snippet.snippet_name
    }
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "v4 indexed format with basic rules",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "abc123",
    "rules.#": "2",
    "rules.0.expression": "true",
    "rules.0.snippet_name": "test_snippet_0",
    "rules.1.enabled": true,
    "rules.1.expression": "http.request.uri.path contains \"/api\"",
    "rules.1.snippet_name": "test_snippet_1",
    "rules.1.description": "API route handler"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "rules": [
      {
        "enabled": true,
        "expression": "true",
        "snippet_name": "test_snippet_0",
        "description": ""
      },
      {
        "enabled": true,
        "expression": "http.request.uri.path contains \"/api\"",
        "snippet_name": "test_snippet_1",
        "description": "API route handler"
      }
    ]
  }
}`,
			},
			{
				Name: "v4 format with explicit enabled=false",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "abc123",
    "rules.#": "1",
    "rules.0.enabled": false,
    "rules.0.expression": "http.host eq \"example.com\"",
    "rules.0.snippet_name": "snippet_a",
    "rules.0.description": "Test rule"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "rules": [
      {
        "enabled": false,
        "expression": "http.host eq \"example.com\"",
        "snippet_name": "snippet_a",
        "description": "Test rule"
      }
    ]
  }
}`,
			},
			{
				Name: "v4 format with empty rules",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "abc123",
    "rules.#": "0"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "rules": []
  }
}`,
			},
			{
				Name: "v5 format passthrough",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "rules": [
      {
        "id": "rule-1",
        "enabled": false,
        "expression": "true",
        "snippet_name": "snippet_1",
        "description": "First rule",
        "last_updated": "2024-01-01T00:00:00Z"
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "rules": [
      {
        "id": "rule-1",
        "enabled": false,
        "expression": "true",
        "snippet_name": "snippet_1",
        "description": "First rule",
        "last_updated": "2024-01-01T00:00:00Z"
      }
    ]
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
