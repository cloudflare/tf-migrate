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
}`,
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
}`,
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
}`,
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
}`,
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
resource "cloudflare_zero_trust_tunnel_cloudflared_route" "staging" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  network    = "192.168.0.0/16"
  comment    = "Staging"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Complete state with all fields",
				Input: `{
  "type": "cloudflare_tunnel_route",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "10.0.0.0/16",
    "comment": "New tunnel route for documentation",
    "virtual_network_id": "7f5a1a01-3e68-4e79-9f52-fe5a62b5e0f5"
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_route",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "10.0.0.0/16",
    "comment": "New tunnel route for documentation",
    "virtual_network_id": "7f5a1a01-3e68-4e79-9f52-fe5a62b5e0f5"
  }
}`,
			},
			{
				Name: "Minimal state - required fields only",
				Input: `{
  "type": "cloudflare_tunnel_route",
  "name": "minimal",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "192.168.1.0/24"
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_route",
  "name": "minimal",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "192.168.1.0/24"
  }
}`,
			},
			{
				Name: "State with empty comment",
				Input: `{
  "type": "cloudflare_tunnel_route",
  "name": "empty_comment",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "10.0.0.0/8",
    "comment": ""
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_route",
  "name": "empty_comment",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "10.0.0.0/8",
    "comment": ""
  }
}`,
			},
			{
				Name: "State with missing optional fields (null values)",
				Input: `{
  "type": "cloudflare_tunnel_route",
  "name": "no_optionals",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "172.16.0.0/12",
    "comment": null,
    "virtual_network_id": null
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_route",
  "name": "no_optionals",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "172.16.0.0/12",
    "comment": null,
    "virtual_network_id": null
  }
}`,
			},
			{
				Name: "IPv6 network state",
				Input: `{
  "type": "cloudflare_tunnel_route",
  "name": "ipv6",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "2001:db8::/32",
    "comment": "IPv6 route"
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_route",
  "name": "ipv6",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "2001:db8::/32",
    "comment": "IPv6 route"
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformationWithMockAPI", func(t *testing.T) {
		// Create a mock API server
		mockServer := testhelpers.NewMockAPIServer()
		defer mockServer.Close()

		// Mock the tunnel routes list endpoint
		// The v6 SDK uses the path: /accounts/{account_id}/teamnet/routes
		accountID := "f037e56e89293a057740de681ac9abbe"
		mockServer.AddTunnelRoutesListHandler(accountID, []map[string]interface{}{
			{
				"id":        "550e8400-e29b-41d4-a716-446655440000",
				"network":   "10.0.0.0/16",
				"tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
				"comment":   "Test route",
			},
		})

		tests := []testhelpers.StateTestCase{
			{
				Name: "State transformation with API lookup for UUID",
				Input: `{
  "type": "cloudflare_tunnel_route",
  "name": "example",
  "attributes": {
    "id": "10.0.0.0/16",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "10.0.0.0/16",
    "comment": "Test route"
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_route",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4a4e-bd8d-11477cf380d9",
    "network": "10.0.0.0/16",
    "comment": "Test route"
  }
}`,
				APIClient: mockServer.Client,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
