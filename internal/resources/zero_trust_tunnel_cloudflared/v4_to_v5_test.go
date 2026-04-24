package zero_trust_tunnel_cloudflared

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// TestComputedAttributeMappings verifies that GetComputedAttributeMappings returns
	// entries for BOTH v4 type names so cross-file .secret references are rewritten
	// regardless of which name was used in the original config.
	t.Run("ComputedAttributeMappings", func(t *testing.T) {
		mapper, ok := migrator.(transform.ComputedAttributeMapper)
		if !ok {
			t.Fatal("migrator does not implement ComputedAttributeMapper")
		}
		mappings := mapper.GetComputedAttributeMappings()

		// Build a lookup by OldResourceType for easy assertions.
		byOldType := make(map[string]transform.ComputedAttributeMapping, len(mappings))
		for _, m := range mappings {
			byOldType[m.OldResourceType] = m
		}

		// Case 1: deprecated cloudflare_tunnel name (pre-existing behaviour)
		old, ok := byOldType["cloudflare_tunnel"]
		if !ok {
			t.Errorf("missing mapping for OldResourceType=cloudflare_tunnel")
		} else {
			if old.OldAttribute != "secret" {
				t.Errorf("cloudflare_tunnel mapping: want OldAttribute=secret, got %q", old.OldAttribute)
			}
			if old.NewResourceType != "cloudflare_zero_trust_tunnel_cloudflared" {
				t.Errorf("cloudflare_tunnel mapping: want NewResourceType=cloudflare_zero_trust_tunnel_cloudflared, got %q", old.NewResourceType)
			}
			if old.NewAttribute != "tunnel_secret" {
				t.Errorf("cloudflare_tunnel mapping: want NewAttribute=tunnel_secret, got %q", old.NewAttribute)
			}
		}

		// Case 2: preferred v4 name cloudflare_zero_trust_tunnel_cloudflared (the bug — this was missing)
		preferred, ok := byOldType["cloudflare_zero_trust_tunnel_cloudflared"]
		if !ok {
			t.Errorf("missing mapping for OldResourceType=cloudflare_zero_trust_tunnel_cloudflared; " +
				"cross-file .secret references on resources that already use the v5 type name will not be rewritten")
		} else {
			if preferred.OldAttribute != "secret" {
				t.Errorf("cloudflare_zero_trust_tunnel_cloudflared mapping: want OldAttribute=secret, got %q", preferred.OldAttribute)
			}
			if preferred.NewResourceType != "cloudflare_zero_trust_tunnel_cloudflared" {
				t.Errorf("cloudflare_zero_trust_tunnel_cloudflared mapping: want NewResourceType=cloudflare_zero_trust_tunnel_cloudflared, got %q", preferred.NewResourceType)
			}
			if preferred.NewAttribute != "tunnel_secret" {
				t.Errorf("cloudflare_zero_trust_tunnel_cloudflared mapping: want NewAttribute=tunnel_secret, got %q", preferred.NewAttribute)
			}
		}
	})

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "minimal_tunnel",
				Input: `
resource "cloudflare_tunnel" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "minimal-tunnel"
  secret     = base64encode("my-secret-that-is-at-least-32-bytes-long")
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared" "minimal" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "minimal-tunnel"
  config_src    = "local"
  tunnel_secret = base64encode("my-secret-that-is-at-least-32-bytes-long")
}

moved {
  from = cloudflare_tunnel.minimal
  to   = cloudflare_zero_trust_tunnel_cloudflared.minimal
}`,
			},
			{
				Name: "multiple_tunnels",
				Input: `
resource "cloudflare_tunnel" "tunnel1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "tunnel-one"
  secret     = base64encode("first-tunnel-secret-32-bytes-long")
}

resource "cloudflare_tunnel" "tunnel2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "tunnel-two"
  secret     = base64encode("second-tunnel-secret-32-bytes-long")
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared" "tunnel1" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "tunnel-one"
  tunnel_secret = base64encode("first-tunnel-secret-32-bytes-long")
  config_src = "local"
}

moved {
  from = cloudflare_tunnel.tunnel1
  to   = cloudflare_zero_trust_tunnel_cloudflared.tunnel1
}

resource "cloudflare_zero_trust_tunnel_cloudflared" "tunnel2" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "tunnel-two"
  tunnel_secret = base64encode("second-tunnel-secret-32-bytes-long")
  config_src = "local"
}

moved {
  from = cloudflare_tunnel.tunnel2
  to   = cloudflare_zero_trust_tunnel_cloudflared.tunnel2
}`,
			},
			{
				Name: "preferred_v4_name_minimal",
				Input: `
resource "cloudflare_zero_trust_tunnel_cloudflared" "alt_minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "alt-minimal-tunnel"
  secret     = base64encode("alternative-name-secret-32-bytes-long")
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared" "alt_minimal" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "alt-minimal-tunnel"
  tunnel_secret = base64encode("alternative-name-secret-32-bytes-long")
  config_src    = "local"
}`,
			},
			{
				Name: "preferred_v4_name_with_existing_config_src",
				Input: `
resource "cloudflare_zero_trust_tunnel_cloudflared" "alt_config" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "alt-config-tunnel"
  secret     = base64encode("alternative-config-secret-32-bytes-ok")
  config_src = "cloudflare"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared" "alt_config" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "alt-config-tunnel"
  tunnel_secret = base64encode("alternative-config-secret-32-bytes-ok")
  config_src    = "cloudflare"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
