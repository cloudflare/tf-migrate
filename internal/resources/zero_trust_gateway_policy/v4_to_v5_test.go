package zero_trust_gateway_policy

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal gateway policy",
				Input: `resource "cloudflare_teams_rule" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Test Policy"
  description = "Block policy"
  precedence  = 100
  action      = "block"
}`,
				Expected: `resource "cloudflare_zero_trust_gateway_policy" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Test Policy"
  description = "Block policy"
  precedence  = 100
  action      = "block"
}

moved {
  from = cloudflare_teams_rule.example
  to   = cloudflare_zero_trust_gateway_policy.example
}`,
			},
			{
				Name: "Gateway policy with rule_settings",
				Input: `resource "cloudflare_teams_rule" "with_settings" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Block Policy"
  action      = "block"
  precedence  = 200

  rule_settings {
    block_page_enabled = true
    block_page_reason  = "Access denied"
    override_ips       = ["1.1.1.1"]
  }
}`,
				Expected: `resource "cloudflare_zero_trust_gateway_policy" "with_settings" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Block Policy"
  action      = "block"
  precedence  = 200

  rule_settings = {
    block_page_enabled = true
    override_ips       = ["1.1.1.1"]
    block_reason       = "Access denied"
  }
}

moved {
  from = cloudflare_teams_rule.with_settings
  to   = cloudflare_zero_trust_gateway_policy.with_settings
}`,
			},
			{
				Name: "Gateway policy with nested blocks",
				Input: `resource "cloudflare_teams_rule" "with_nested" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test"
  action     = "allow"

  rule_settings {
    notification_settings {
      enabled = true
      message = "Custom notification"
    }
    l4override {
      ip   = "192.168.1.1"
      port = 8080
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_gateway_policy" "with_nested" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test"
  action     = "allow"

  rule_settings = {
    l4override = {
      ip   = "192.168.1.1"
      port = 8080
    }
    notification_settings = {
      enabled = true
      msg     = "Custom notification"
    }
  }
}

moved {
  from = cloudflare_teams_rule.with_nested
  to   = cloudflare_zero_trust_gateway_policy.with_nested
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}

// TestV4ToV5Transformation_AlreadyV5Named tests the scenario from BUGS-2007:
// The user has already run tf-migrate once (or manually renamed resources),
// so the resource type is already "cloudflare_zero_trust_gateway_policy" (v5 name),
// but rule_settings and notification_settings are still in v4 block syntax.
// tf-migrate must still convert the blocks even when the resource name is already v5.
func TestV4ToV5Transformation_AlreadyV5Named(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "v5-named resource with rule_settings still in block syntax",
			Input: `resource "cloudflare_zero_trust_gateway_policy" "terraform_managed_resource_f65deb9d" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block Policy"
  action     = "block"
  precedence = 10

  rule_settings {
    block_page_enabled = true
    block_page_reason  = "Access denied"
  }
}`,
			Expected: `resource "cloudflare_zero_trust_gateway_policy" "terraform_managed_resource_f65deb9d" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block Policy"
  action     = "block"
  precedence = 10

  rule_settings = {
    block_page_enabled = true
    block_reason       = "Access denied"
  }
}`,
		},
		{
			Name: "v5-named resource with nested notification_settings block",
			Input: `resource "cloudflare_zero_trust_gateway_policy" "with_notification" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Notify Policy"
  action     = "block"
  precedence = 20

  rule_settings {
    notification_settings {
      enabled = true
      message = "You have been blocked"
    }
  }
}`,
			Expected: `resource "cloudflare_zero_trust_gateway_policy" "with_notification" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Notify Policy"
  action     = "block"
  precedence = 20

  rule_settings = {
    notification_settings = {
      enabled = true
      msg     = "You have been blocked"
    }
  }
}`,
		},
		{
			Name: "v5-named resource with l4override block",
			Input: `resource "cloudflare_zero_trust_gateway_policy" "l4_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "L4 Override"
  action     = "l4_override"
  precedence = 30

  rule_settings {
    l4override {
      ip   = "10.0.0.1"
      port = 8080
    }
  }
}`,
			Expected: `resource "cloudflare_zero_trust_gateway_policy" "l4_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "L4 Override"
  action     = "l4_override"
  precedence = 30

  rule_settings = {
    l4override = {
      ip   = "10.0.0.1"
      port = 8080
    }
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
