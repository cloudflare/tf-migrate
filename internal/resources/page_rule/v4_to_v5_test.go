package page_rule

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions {
    cache_level = "bypass"
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  status  = "active"
  actions = {
    cache_level = "bypass"
  }
}`,
			},
			{
				Name: "With forwarding_url nested block",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/old/*"
  actions {
    forwarding_url {
      url         = "https://example.com/new/"
      status_code = 301
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/old/*"
  status  = "active"
  actions = {
    forwarding_url = {
      url         = "https://example.com/new/"
      status_code = 301
    }
  }
}`,
			},
			{
				Name: "With cache_key_fields deeply nested",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions {
    cache_key_fields {
      cookie {
        check_presence = ["sessionid"]
      }
      host {
        resolved = true
      }
      user {
        device_type = true
        geo         = false
      }
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  status  = "active"
  actions = {
    cache_key_fields = {
      cookie = {
        check_presence = ["sessionid"]
      }
      host = {
        resolved = true
      }
      user = {
        device_type = true
        geo         = false
      }
    }
  }
}`,
			},
			{
				Name: "With cache_ttl_by_status blocks to map",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions {
    cache_ttl_by_status {
      codes = "200"
      ttl   = 3600
    }
    cache_ttl_by_status {
      codes = "404"
      ttl   = 300
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  status  = "active"
  actions = {
    cache_ttl_by_status = {
      "200" = "3600"
      "404" = "300"
    }
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Minimal state with actions",
				Input: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "bypass"
    }]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "bypass"
    }
  }
}`,
			},
			{
				Name: "With cache_ttl_by_status array to map",
				Input: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_ttl_by_status": [
        {"codes": "200", "ttl": 3600},
        {"codes": "404", "ttl": 300}
      ]
    }]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_ttl_by_status": {
        "200": "3600",
        "404": "300"
      }
    }
  }
}`,
			},
			{
				Name: "With forwarding_url nested array",
				Input: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "forwarding_url": [{
        "url": "https://example.com/new/",
        "status_code": 301
      }]
    }]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "forwarding_url": {
        "url": "https://example.com/new/",
        "status_code": 301
      }
    }
  }
}`,
			},
			{
				Name: "With deprecated fields removed",
				Input: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "bypass",
      "minify": [{"css": "on", "html": "off", "js": "on"}],
      "disable_railgun": false
    }]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "bypass"
    }
  }
}`,
			},
			{
				Name: "With cache_key_fields deeply nested",
				Input: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_key_fields": [{
        "cookie": [{"check_presence": ["sessionid"], "include": []}],
        "header": [],
        "host": [{"resolved": true}],
        "query_string": [{"exclude": ["utm_*"], "include": []}],
        "user": [{"device_type": true, "geo": false}]
      }]
    }]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_key_fields": {
        "cookie": {"check_presence": ["sessionid"], "include": null},
        "header": null,
        "host": {"resolved": true},
        "query_string": {"exclude": ["utm_*"], "include": null},
        "user": {"device_type": true, "geo": false}
      }
    }
  }
}`,
			},
			{
				Name: "With empty and false actions converted to null (no config provided)",
				Input: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "bypass",
      "always_use_https": false,
      "disable_security": false,
      "forwarding_url": [],
      "cache_key_fields": {}
    }]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "bypass",
      "always_use_https": null,
      "disable_security": null,
      "forwarding_url": null,
      "cache_key_fields": null
    }
  }
}`,
			},
			{
				Name: "With browser_cache_ttl string to int64 conversion",
				Input: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "cache_everything",
      "edge_cache_ttl": 7200,
      "browser_cache_ttl": "3600"
    }]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "example",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "cache_everything",
      "edge_cache_ttl": 7200.0,
      "browser_cache_ttl": 3600
    }
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})

	t.Run("ActionFieldTransformationWithConfig", func(t *testing.T) {
		// These tests verify that TransformEmptyValuesToNull respects user-configured values
		tests := []struct {
			Name          string
			InputState    string
			InputConfig   string
			ExpectedState string
			Description   string
		}{
			{
				Name: "Empty actions with no config - all converted to null",
				InputState: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "bypass",
      "always_use_https": false,
      "disable_security": false,
      "ssl": "",
      "edge_cache_ttl": 0
    }]
  }
}`,
				InputConfig: `resource "cloudflare_page_rule" "test" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "bypass"
  }
}`,
				ExpectedState: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "bypass",
      "always_use_https": null,
      "disable_security": null,
      "ssl": null,
      "edge_cache_ttl": null
    }
  }
}`,
				Description: "When actions are not in config, empty values should be converted to null",
			},
			{
				Name: "Explicitly set false values in config - preserved",
				InputState: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "bypass",
      "always_use_https": false,
      "disable_security": false
    }]
  }
}`,
				InputConfig: `resource "cloudflare_page_rule" "test" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "bypass"
    always_use_https = false
    disable_security = false
  }
}`,
				ExpectedState: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "bypass",
      "always_use_https": false,
      "disable_security": false
    }
  }
}`,
				Description: "Explicitly configured false values should be preserved",
			},
			{
				Name: "Mix of configured and unconfigured empty values",
				InputState: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "bypass",
      "always_use_https": false,
      "disable_security": false,
      "browser_check": false,
      "ssl": ""
    }]
  }
}`,
				InputConfig: `resource "cloudflare_page_rule" "test" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "bypass"
    always_use_https = false
  }
}`,
				ExpectedState: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "bypass",
      "always_use_https": false,
      "disable_security": null,
      "browser_check": null,
      "ssl": null
    }
  }
}`,
				Description: "Only configured fields are preserved, others converted to null",
			},
			{
				Name: "Empty nested objects without config",
				InputState: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "bypass",
      "forwarding_url": [],
      "cache_key_fields": {}
    }]
  }
}`,
				InputConfig: `resource "cloudflare_page_rule" "test" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "bypass"
  }
}`,
				ExpectedState: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "bypass",
      "forwarding_url": null,
      "cache_key_fields": null
    }
  }
}`,
				Description: "Empty arrays and objects should be converted to null when not in config",
			},
			{
				Name: "Configured nested objects preserved",
				InputState: `{
  "schema_version": 1,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1,
    "status": "active",
    "actions": [{
      "cache_level": "bypass",
      "forwarding_url": [{
        "url": "https://example.com/new/",
        "status_code": 301
      }]
    }]
  }
}`,
				InputConfig: `resource "cloudflare_page_rule" "test" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "bypass"
    forwarding_url = {
      url = "https://example.com/new/"
      status_code = 301
    }
  }
}`,
				ExpectedState: `{
  "schema_version": 0,
  "type": "cloudflare_page_rule",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "target": "example.com/*",
    "priority": 1.0,
    "status": "active",
    "actions": {
      "cache_level": "bypass",
      "forwarding_url": {
        "url": "https://example.com/new/",
        "status_code": 301
      }
    }
  }
}`,
				Description: "Configured nested objects should be preserved and transformed",
			},
		}

		for _, tt := range tests {
			t.Run(tt.Name, func(t *testing.T) {
				// Parse config using hclwrite
				configFile, diags := hclwrite.ParseConfig([]byte(tt.InputConfig), "test.tf", hcl.InitialPos)
				require.False(t, diags.HasErrors(), "Failed to parse config: %v", diags)

				// Parse state
				inputResult := gjson.Parse(tt.InputState)

				// Create context with config files
				ctx := &transform.Context{
					StateJSON: tt.InputState,
					CFGFiles: map[string]*hclwrite.File{
						"test.tf": configFile,
					},
				}

				// Transform state
				result, err := migrator.TransformState(ctx, inputResult, "cloudflare_page_rule.test", "test")
				require.NoError(t, err, "TransformState failed")

				// Compare results using JSONEq
				assert.JSONEq(t, tt.ExpectedState, result, "State transformation mismatch for: %s", tt.Description)
			})
		}
	})
}
