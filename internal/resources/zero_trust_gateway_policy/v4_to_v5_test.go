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
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Minimal state with numeric conversions",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_teams_rule",
    "name": "test",
    "instances": [{
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "Test Policy",
        "action": "block",
        "precedence": 100,
        "version": 1
      },
      "schema_version": 0
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_gateway_policy",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "Test Policy",
        "action": "block",
        "precedence": 100.0,
        "version": 1.0
      }
    }]
  }]
}`,
			},
			{
				Name: "State with rule_settings array to object",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_teams_rule",
    "name": "test",
    "instances": [{
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "Test",
        "action": "block",
        "rule_settings": [{
          "block_page_enabled": true,
          "block_page_reason": "Access denied"
        }]
      },
      "schema_version": 0
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_gateway_policy",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "Test",
        "action": "block",
        "rule_settings": {
          "block_page_enabled": true,
          "block_reason": "Access denied"
        }
      }
    }]
  }]
}`,
			},
			{
				Name: "State with nested blocks and type conversions",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_teams_rule",
    "name": "test",
    "instances": [{
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "Test",
        "action": "allow",
        "rule_settings": [{
          "notification_settings": [{
            "enabled": true,
            "message": "Custom message"
          }],
          "l4override": [{
            "ip": "192.168.1.1",
            "port": 8080
          }]
        }]
      },
      "schema_version": 0
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_gateway_policy",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "Test",
        "action": "allow",
        "rule_settings": {
          "notification_settings": {
            "enabled": true,
            "msg": "Custom message"
          },
          "l4override": {
            "ip": "192.168.1.1",
            "port": 8080.0
          }
        }
      }
    }]
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
