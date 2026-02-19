package zero_trust_tunnel_cloudflared

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

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
