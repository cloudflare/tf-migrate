package zero_trust_tunnel_cloudflared_route

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic tunnel route with all fields",
				Input: `
resource "cloudflare_tunnel_route" "example" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  tunnel_id          = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network            = "10.0.0.0/16"
  comment            = "New tunnel route for documentation"
  virtual_network_id = "7f5a1a01-3e68-4e79-9f52-fe5a62b5e0f5"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_route" "example" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  tunnel_id          = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network            = "10.0.0.0/16"
  comment            = "New tunnel route for documentation"
  virtual_network_id = "7f5a1a01-3e68-4e79-9f52-fe5a62b5e0f5"
}
moved {
  from = cloudflare_tunnel_route.example
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.example
}
`,
			},
			{
				Name: "Minimal tunnel route - required fields only",
				Input: `
resource "cloudflare_tunnel_route" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "192.168.1.0/24"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_route" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "192.168.1.0/24"
}
moved {
  from = cloudflare_tunnel_route.minimal
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.minimal
}
`,
			},
			{
				Name: "Tunnel route with IPv6 network",
				Input: `
resource "cloudflare_tunnel_route" "ipv6" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "2001:db8::/32"
  comment    = "IPv6 route"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_route" "ipv6" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "2001:db8::/32"
  comment    = "IPv6 route"
}
moved {
  from = cloudflare_tunnel_route.ipv6
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.ipv6
}
`,
			},
			{
				Name: "Tunnel route with empty comment",
				Input: `
resource "cloudflare_tunnel_route" "empty_comment" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "10.0.0.0/8"
  comment    = ""
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_route" "empty_comment" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "10.0.0.0/8"
  comment    = ""
}
moved {
  from = cloudflare_tunnel_route.empty_comment
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.empty_comment
}
`,
			},
			{
				Name: "Multiple tunnel routes in one file",
				Input: `
resource "cloudflare_tunnel_route" "prod" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "10.0.0.0/16"
  comment    = "Production"
}

resource "cloudflare_tunnel_route" "staging" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  network    = "192.168.0.0/16"
  comment    = "Staging"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_route" "prod" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "10.0.0.0/16"
  comment    = "Production"
}
moved {
  from = cloudflare_tunnel_route.prod
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.prod
}

resource "cloudflare_zero_trust_tunnel_cloudflared_route" "staging" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  network    = "192.168.0.0/16"
  comment    = "Staging"
}
moved {
  from = cloudflare_tunnel_route.staging
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.staging
}
`,
			},
			{
				Name: "Alternative v4 name - cloudflare_zero_trust_tunnel_route with all fields",
				Input: `
resource "cloudflare_zero_trust_tunnel_route" "example" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  tunnel_id          = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network            = "10.0.0.0/16"
  comment            = "New tunnel route for documentation"
  virtual_network_id = "7f5a1a01-3e68-4e79-9f52-fe5a62b5e0f5"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_route" "example" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  tunnel_id          = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network            = "10.0.0.0/16"
  comment            = "New tunnel route for documentation"
  virtual_network_id = "7f5a1a01-3e68-4e79-9f52-fe5a62b5e0f5"
}
moved {
  from = cloudflare_zero_trust_tunnel_route.example
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.example
}
`,
			},
			{
				Name: "Alternative v4 name - cloudflare_zero_trust_tunnel_route minimal",
				Input: `
resource "cloudflare_zero_trust_tunnel_route" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "192.168.1.0/24"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_route" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4a4e-bd8d-11477cf380d9"
  network    = "192.168.1.0/24"
}
moved {
  from = cloudflare_zero_trust_tunnel_route.minimal
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.minimal
}
`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
