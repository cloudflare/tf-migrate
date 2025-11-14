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
  tunnel_secret = base64encode("my-secret-that-is-at-least-32-bytes-long")
}`,
			},
			{
				Name: "tunnel_with_config_src",
				Input: `
resource "cloudflare_tunnel" "with_config" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "config-tunnel"
  secret     = base64encode("another-secret-32-bytes-or-longer-here")
  config_src = "local"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared" "with_config" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "config-tunnel"
  config_src    = "local"
  tunnel_secret = base64encode("another-secret-32-bytes-or-longer-here")
}`,
			},
			{
				Name: "tunnel_with_cloudflare_config",
				Input: `
resource "cloudflare_tunnel" "remote_config" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "remote-tunnel"
  secret     = base64encode("remote-tunnel-secret-32-bytes-minimum")
  config_src = "cloudflare"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared" "remote_config" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "remote-tunnel"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("remote-tunnel-secret-32-bytes-minimum")
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
  config_src = "cloudflare"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared" "tunnel1" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "tunnel-one"
  tunnel_secret = base64encode("first-tunnel-secret-32-bytes-long")
}
resource "cloudflare_zero_trust_tunnel_cloudflared" "tunnel2" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "tunnel-two"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("second-tunnel-secret-32-bytes-long")
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "minimal_state",
				Input: `{
  "type": "cloudflare_tunnel",
  "name": "minimal",
  "attributes": {
    "id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "minimal-tunnel",
    "secret": "dGVzdC1zZWNyZXQtdGhhdC1pcy1hdC1sZWFzdC0zMi1ieXRlcw==",
    "cname": "f70ff02e-f290-4a4e-bd8d-11477cf380d9.cfargotunnel.com",
    "tunnel_token": "eyJhIjoiZjAzN2U1NmU4OTI5M2EwNTc3NDBkZTY4MWFjOWFiYmUiLCJ0IjoiZjcwZmYwMmUtZjI5MC00YTRlLWJkOGQtMTE0NzdjZjM4MGQ5IiwicyI6IllXNW4ifQ=="
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared",
  "name": "minimal",
  "schema_version": 0,
  "attributes": {
    "id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "minimal-tunnel",
    "tunnel_secret": "dGVzdC1zZWNyZXQtdGhhdC1pcy1hdC1sZWFzdC0zMi1ieXRlcw=="
  }
}`,
			},
			{
				Name: "with_config_src",
				Input: `{
  "type": "cloudflare_tunnel",
  "name": "with_config",
  "attributes": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "config-tunnel",
    "secret": "YW5vdGhlci1sb25nLXNlY3JldC10aGF0LW1lZXRzLXRoZS1yZXF1aXJlbWVudHM=",
    "config_src": "local",
    "cname": "a1b2c3d4-e5f6-7890-abcd-ef1234567890.cfargotunnel.com",
    "tunnel_token": "eyJhIjoiZjAzN2U1NmU4OTI5M2EwNTc3NDBkZTY4MWFjOWFiYmUiLCJ0IjoiYTFiMmMzZDQtZTVmNi03ODkwLWFiY2QtZWYxMjM0NTY3ODkwIiwicyI6IllXNW4ifQ=="
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared",
  "name": "with_config",
  "schema_version": 0,
  "attributes": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "config-tunnel",
    "tunnel_secret": "YW5vdGhlci1sb25nLXNlY3JldC10aGF0LW1lZXRzLXRoZS1yZXF1aXJlbWVudHM=",
    "config_src": "local"
  }
}`,
			},
			{
				Name: "without_computed_fields",
				Input: `{
  "type": "cloudflare_tunnel",
  "name": "no_computed",
  "attributes": {
    "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "no-computed-tunnel",
    "secret": "dGVzdC1zZWNyZXQtd2l0aG91dC1jb21wdXRlZC1maWVsZHMtMzItYnl0ZXM="
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared",
  "name": "no_computed",
  "schema_version": 0,
  "attributes": {
    "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "no-computed-tunnel",
    "tunnel_secret": "dGVzdC1zZWNyZXQtd2l0aG91dC1jb21wdXRlZC1maWVsZHMtMzItYnl0ZXM="
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
