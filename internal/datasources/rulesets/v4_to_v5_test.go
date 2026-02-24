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

}

// TestV4ToV5TransformationState_Removed is a marker indicating state transformation tests were removed.
// State migration is now handled by the provider's StateUpgraders.
func TestV4ToV5TransformationState_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}
