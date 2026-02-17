package access_rule

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("InterfaceMethods", func(t *testing.T) {
		// Cast to concrete type to test all methods
		m := migrator.(*V4ToV5Migrator)

		// Test GetResourceType
		if got := m.GetResourceType(); got != "cloudflare_access_rule" {
			t.Errorf("GetResourceType() = %v, want %v", got, "cloudflare_access_rule")
		}

		// Test GetResourceRename
		oldName, newName := m.GetResourceRename()
		if oldName != "cloudflare_access_rule" || newName != "cloudflare_access_rule" {
			t.Errorf("GetResourceRename() = (%v, %v), want (%v, %v)",
				oldName, newName, "cloudflare_access_rule", "cloudflare_access_rule")
		}

		// Test CanHandle
		if !m.CanHandle("cloudflare_access_rule") {
			t.Error("CanHandle('cloudflare_access_rule') should return true")
		}
		if m.CanHandle("cloudflare_other_resource") {
			t.Error("CanHandle('cloudflare_other_resource') should return false")
		}

		// Test Preprocess (no-op)
		input := "test content"
		if got := m.Preprocess(input); got != input {
			t.Errorf("Preprocess() should return input unchanged, got %v", got)
		}
	})

	t.Run("ConfigTransformation", func(t *testing.T) {
		testConfigTransformations(t, migrator)
	})
}

func testConfigTransformations(t *testing.T, migrator transform.ResourceTransformer) {
	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Basic access rule with configuration block",
			Input: `resource "cloudflare_access_rule" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "block"
  configuration {
    target = "ip"
    value  = "1.2.3.4"
  }
  notes = "Block suspicious IP"
}`,
			Expected: `resource "cloudflare_access_rule" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "block"
  notes      = "Block suspicious IP"
  configuration = {
    target = "ip"
    value  = "1.2.3.4"
  }
}`,
		},
		{
			Name: "Zone-level access rule",
			Input: `resource "cloudflare_access_rule" "zone_rule" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  mode    = "challenge"
  configuration {
    target = "country"
    value  = "CN"
  }
}`,
			Expected: `resource "cloudflare_access_rule" "zone_rule" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  mode    = "challenge"
  configuration = {
    target = "country"
    value  = "CN"
  }
}`,
		},
		{
			Name: "Access rule without notes",
			Input: `resource "cloudflare_access_rule" "no_notes" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "whitelist"
  configuration {
    target = "ip_range"
    value  = "10.0.0.0/8"
  }
}`,
			Expected: `resource "cloudflare_access_rule" "no_notes" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "whitelist"
  configuration = {
    target = "ip_range"
    value  = "10.0.0.0/8"
  }
}`,
		},
		{
			Name: "Multiple access rules",
			Input: `resource "cloudflare_access_rule" "block_ip" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "block"
  configuration {
    target = "ip"
    value  = "192.168.1.1"
  }
  notes = "Block malicious IP"
}

resource "cloudflare_access_rule" "allow_country" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  mode    = "whitelist"
  configuration {
    target = "country"
    value  = "US"
  }
  notes = "Allow US traffic"
}`,
			Expected: `resource "cloudflare_access_rule" "block_ip" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "block"
  notes      = "Block malicious IP"
  configuration = {
    target = "ip"
    value  = "192.168.1.1"
  }
}

resource "cloudflare_access_rule" "allow_country" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  mode    = "whitelist"
  notes   = "Allow US traffic"
  configuration = {
    target = "country"
    value  = "US"
  }
}`,
		},
		{
			Name: "All mode types - js_challenge and managed_challenge",
			Input: `resource "cloudflare_access_rule" "js_challenge" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "js_challenge"
  configuration {
    target = "ip"
    value  = "203.0.113.0"
  }
}

resource "cloudflare_access_rule" "managed_challenge" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "managed_challenge"
  configuration {
    target = "asn"
    value  = "AS13335"
  }
}`,
			Expected: `resource "cloudflare_access_rule" "js_challenge" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "js_challenge"
  configuration = {
    target = "ip"
    value  = "203.0.113.0"
  }
}

resource "cloudflare_access_rule" "managed_challenge" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "managed_challenge"
  configuration = {
    target = "asn"
    value  = "AS13335"
  }
}`,
		},
		{
			Name: "IPv6 address and ASN target",
			Input: `resource "cloudflare_access_rule" "ipv6" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "block"
  configuration {
    target = "ip6"
    value  = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
  }
  notes = "Block IPv6"
}`,
			Expected: `resource "cloudflare_access_rule" "ipv6" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "block"
  notes      = "Block IPv6"
  configuration = {
    target = "ip6"
    value  = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
  }
}`,
		},
		{
			Name: "Special characters in notes",
			Input: `resource "cloudflare_access_rule" "special_chars" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "block"
  configuration {
    target = "ip"
    value  = "1.2.3.4"
  }
  notes = "Block: \"suspicious\" IP with 'quotes' & special chars!"
}`,
			Expected: `resource "cloudflare_access_rule" "special_chars" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "block"
  notes      = "Block: \"suspicious\" IP with 'quotes' & special chars!"
  configuration = {
    target = "ip"
    value  = "1.2.3.4"
  }
}`,
		},
		{
			Name: "Variable references preserved",
			Input: `resource "cloudflare_access_rule" "with_vars" {
  account_id = var.cloudflare_account_id
  mode       = var.access_mode
  configuration {
    target = "ip"
    value  = var.blocked_ip
  }
  notes = "Block ${var.description}"
}`,
			Expected: `resource "cloudflare_access_rule" "with_vars" {
  account_id = var.cloudflare_account_id
  mode       = var.access_mode
  notes      = "Block ${var.description}"
  configuration = {
    target = "ip"
    value  = var.blocked_ip
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

