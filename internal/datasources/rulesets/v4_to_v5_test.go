package rulesets

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic datasource with account_id only",
				Input: `
data "cloudflare_rulesets" "test" {
  account_id = "abc123"
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  account_id = "abc123"
}`,
			},
			{
				Name: "Datasource with zone_id",
				Input: `
data "cloudflare_rulesets" "test" {
  zone_id = "def456"
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  zone_id = "def456"
}`,
			},
			{
				Name: "Remove filter block",
				Input: `
data "cloudflare_rulesets" "test" {
  account_id = "abc123"
  filter {
    name = "my-ruleset"
  }
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  account_id = "abc123"
}`,
			},
			{
				Name: "Remove include_rules field",
				Input: `
data "cloudflare_rulesets" "test" {
  account_id    = "abc123"
  include_rules = true
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  account_id = "abc123"
}`,
			},
			{
				Name: "Remove both filter and include_rules",
				Input: `
data "cloudflare_rulesets" "test" {
  account_id    = "abc123"
  include_rules = true
  filter {
    name    = "test"
    kind    = "custom"
    phase   = "http_request_firewall_managed"
  }
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  account_id = "abc123"
}`,
			},
			{
				Name: "Variable references preserved",
				Input: `
data "cloudflare_rulesets" "test" {
  account_id = var.account_id
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  account_id = var.account_id
}`,
			},
			{
				Name: "Multiple datasources in one file",
				Input: `
data "cloudflare_rulesets" "account_rulesets" {
  account_id = var.account_id
}

data "cloudflare_rulesets" "zone_rulesets" {
  zone_id = var.zone_id
  filter {
    kind = "custom"
  }
}`,
				Expected: `data "cloudflare_rulesets" "account_rulesets" {
  account_id = var.account_id
}

data "cloudflare_rulesets" "zone_rulesets" {
  zone_id = var.zone_id
}`,
			},
			{
				Name: "Empty filter block",
				Input: `
data "cloudflare_rulesets" "test" {
  account_id = "abc123"
  filter {}
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  account_id = "abc123"
}`,
			},
			{
				Name: "Complex filter with all fields",
				Input: `
data "cloudflare_rulesets" "test" {
  account_id = "abc123"
  filter {
    id      = "ruleset-123"
    name    = "my-ruleset"
    kind    = "custom"
    phase   = "http_request_firewall_managed"
    version = "1"
  }
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  account_id = "abc123"
}`,
			},
			{
				Name: "Filter with partial fields",
				Input: `
data "cloudflare_rulesets" "test" {
  zone_id = "zone-456"
  filter {
    phase = "http_request_firewall_custom"
  }
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  zone_id = "zone-456"
}`,
			},
			{
				Name: "Include rules with false value",
				Input: `
data "cloudflare_rulesets" "test" {
  account_id    = "abc123"
  include_rules = false
}`,
				Expected: `data "cloudflare_rulesets" "test" {
  account_id = "abc123"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Minimal state with account_id",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "rulesets": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "rulesets": []
  }
}`,
			},
			{
				Name: "State with filter - filter removed",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "filter": {
      "name": "test"
    },
    "rulesets": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "rulesets": []
  }
}`,
			},
			{
				Name: "State with include_rules - field removed",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "include_rules": true,
    "rulesets": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "rulesets": []
  }
}`,
			},
			{
				Name: "State with both filter and include_rules",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "filter": {
      "name": "test",
      "kind": "custom"
    },
    "include_rules": true,
    "rulesets": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "rulesets": []
  }
}`,
			},
			{
				Name: "State with populated rulesets (computed field untouched)",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "rulesets": [
      {
        "id": "ruleset-1",
        "name": "My Ruleset",
        "description": "Test ruleset",
        "kind": "custom",
        "phase": "http_request_firewall_managed",
        "version": "1"
      },
      {
        "id": "ruleset-2",
        "name": "Another Ruleset",
        "kind": "managed",
        "phase": "http_request_firewall_custom",
        "version": "2"
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "rulesets": [
      {
        "id": "ruleset-1",
        "name": "My Ruleset",
        "description": "Test ruleset",
        "kind": "custom",
        "phase": "http_request_firewall_managed",
        "version": "1"
      },
      {
        "id": "ruleset-2",
        "name": "Another Ruleset",
        "kind": "managed",
        "phase": "http_request_firewall_custom",
        "version": "2"
      }
    ]
  }
}`,
			},
			{
				Name: "State with complex filter object",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "zone-456",
    "filter": {
      "id": "ruleset-123",
      "name": "test",
      "kind": "custom",
      "phase": "http_request_firewall_managed",
      "version": "1"
    },
    "rulesets": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "zone-456",
    "rulesets": []
  }
}`,
			},
			{
				Name: "Empty state - sets schema_version",
				Input: `{
  "schema_version": 0
}`,
				Expected: `{
  "schema_version": 0
}`,
			},
			{
				Name: "State without attributes - sets schema_version",
				Input: `{
  "schema_version": 1
}`,
				Expected: `{
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
