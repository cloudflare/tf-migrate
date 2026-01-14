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

resource "cloudflare_zero_trust_tunnel_cloudflared" "tunnel2" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "tunnel-two"
  tunnel_secret = base64encode("second-tunnel-secret-32-bytes-long")
  config_src = "local"
}`,
			},
			{
				Name: "alternative_v4_name_minimal",
				Input: `
resource "cloudflare_zero_trust_tunnel" "alt_minimal" {
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
				Name: "alternative_v4_name_with_config",
				Input: `
resource "cloudflare_zero_trust_tunnel" "alt_config" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "alt-config-tunnel"
  secret     = base64encode("alternative-config-secret-32-bytes-ok")
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared" "alt_config" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "alt-config-tunnel"
  config_src    = "local"
  tunnel_secret = base64encode("alternative-config-secret-32-bytes-ok")
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
    "tunnel_secret": "dGVzdC1zZWNyZXQtdGhhdC1pcy1hdC1sZWFzdC0zMi1ieXRlcw==",
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
    "tunnel_secret": "dGVzdC1zZWNyZXQtd2l0aG91dC1jb21wdXRlZC1maWVsZHMtMzItYnl0ZXM=",
    "config_src": "local"
  }
}`,
			},
			{
				Name: "alternative_v4_name_minimal_state",
				Input: `{
  "type": "cloudflare_zero_trust_tunnel",
  "name": "alt_minimal",
  "attributes": {
    "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "alt-minimal-tunnel",
    "secret": "YWx0ZXJuYXRpdmUtbmFtZS1zZWNyZXQtMzItYnl0ZXMtbG9uZw==",
    "cname": "c3d4e5f6-a7b8-9012-cdef-123456789012.cfargotunnel.com",
    "tunnel_token": "eyJhIjoiZjAzN2U1NmU4OTI5M2EwNTc3NDBkZTY4MWFjOWFiYmUiLCJ0IjoiYzNkNGU1ZjYtYTdiOC05MDEyLWNkZWYtMTIzNDU2Nzg5MDEyIiwicyI6IllXNW4ifQ=="
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared",
  "name": "alt_minimal",
  "schema_version": 0,
  "attributes": {
    "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "alt-minimal-tunnel",
    "tunnel_secret": "YWx0ZXJuYXRpdmUtbmFtZS1zZWNyZXQtMzItYnl0ZXMtbG9uZw==",
    "config_src": "local"
  }
}`,
			},
			{
				Name: "alternative_v4_name_with_config_state",
				Input: `{
  "type": "cloudflare_zero_trust_tunnel",
  "name": "alt_config",
  "attributes": {
    "id": "d4e5f6a7-b8c9-0123-def0-234567890123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "alt-config-tunnel",
    "secret": "YWx0ZXJuYXRpdmUtY29uZmlnLXNlY3JldC0zMi1ieXRlcy1vaw==",
    "cname": "d4e5f6a7-b8c9-0123-def0-234567890123.cfargotunnel.com",
    "tunnel_token": "eyJhIjoiZjAzN2U1NmU4OTI5M2EwNTc3NDBkZTY4MWFjOWFiYmUiLCJ0IjoiZDRlNWY2YTctYjhjOS0wMTIzLWRlZjAtMjM0NTY3ODkwMTIzIiwicyI6IllXNW4ifQ=="
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared",
  "name": "alt_config",
  "schema_version": 0,
  "attributes": {
    "id": "d4e5f6a7-b8c9-0123-def0-234567890123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "alt-config-tunnel",
    "tunnel_secret": "YWx0ZXJuYXRpdmUtY29uZmlnLXNlY3JldC0zMi1ieXRlcy1vaw==",
    "config_src": "local"
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
