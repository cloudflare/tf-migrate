package zero_trust_tunnel_cloudflared_virtual_network

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "old_v4_name_minimal",
				Input: `
resource "cloudflare_tunnel_virtual_network" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "minimal-network"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "minimal-network"
}`,
			},
			{
				Name: "new_v4_name_minimal",
				Input: `
resource "cloudflare_zero_trust_tunnel_virtual_network" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "minimal-network"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "minimal-network"
}`,
			},
			{
				Name: "old_v4_name_complete",
				Input: `
resource "cloudflare_tunnel_virtual_network" "complete" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name              = "complete-network"
  is_default_network = true
  comment           = "Production virtual network"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "complete" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name              = "complete-network"
  is_default_network = true
  comment           = "Production virtual network"
}`,
			},
			{
				Name: "new_v4_name_complete",
				Input: `
resource "cloudflare_zero_trust_tunnel_virtual_network" "complete" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name              = "complete-network"
  is_default_network = true
  comment           = "Production virtual network"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "complete" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name              = "complete-network"
  is_default_network = true
  comment           = "Production virtual network"
}`,
			},
			{
				Name: "default_network",
				Input: `
resource "cloudflare_tunnel_virtual_network" "default" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name              = "default-network"
  is_default_network = true
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "default" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name              = "default-network"
  is_default_network = true
}`,
			},
			{
				Name: "with_comment_only",
				Input: `
resource "cloudflare_tunnel_virtual_network" "documented" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name      = "documented-network"
  comment   = "This network is used for production traffic routing"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "documented" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name      = "documented-network"
  comment   = "This network is used for production traffic routing"
}`,
			},
			{
				Name: "is_default_network_false",
				Input: `
resource "cloudflare_tunnel_virtual_network" "not_default" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name              = "not-default-network"
  is_default_network = false
  comment           = "Test network"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "not_default" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name              = "not-default-network"
  is_default_network = false
  comment           = "Test network"
}`,
			},
			{
				Name: "multiple_networks_both_v4_names",
				Input: `
resource "cloudflare_tunnel_virtual_network" "old_style" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name      = "old-style-network"
}

resource "cloudflare_zero_trust_tunnel_virtual_network" "new_style" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name      = "new-style-network"
  comment   = "New style resource"
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "old_style" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name      = "old-style-network"
}

resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "new_style" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name      = "new-style-network"
  comment   = "New style resource"
}`,
			},
			{
				Name: "empty_comment",
				Input: `
resource "cloudflare_tunnel_virtual_network" "empty_comment" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name      = "network-with-empty-comment"
  comment   = ""
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "empty_comment" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name      = "network-with-empty-comment"
  comment   = ""
}`,
			},
		}
		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "complete_state",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "my-network",
						"is_default_network": true,
						"comment": "Production network"
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "my-network",
						"is_default_network": true,
						"comment": "Production network"
					}
				}`,
			},
			{
				Name: "minimal_state_adds_defaults",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "minimal-network"
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "minimal-network",
						"comment": "",
						"is_default_network": false
					}
				}`,
			},
			{
				Name: "null_comment_gets_default",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "network-null-comment",
						"comment": null,
						"is_default_network": true
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "network-null-comment",
						"comment": "",
						"is_default_network": true
					}
				}`,
			},
			{
				Name: "null_is_default_network_gets_default",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "network-null-default",
						"is_default_network": null,
						"comment": "Test"
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "network-null-default",
						"comment": "Test",
						"is_default_network": false
					}
				}`,
			},
			{
				Name: "both_fields_null_get_defaults",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "network-both-null",
						"comment": null,
						"is_default_network": null
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "network-both-null",
						"comment": "",
						"is_default_network": false
					}
				}`,
			},
			{
				Name: "is_default_network_false_preserved",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "not-default-network",
						"is_default_network": false,
						"comment": "Test network"
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "not-default-network",
						"comment": "Test network",
						"is_default_network": false
					}
				}`,
			},
			{
				Name: "empty_comment_preserved",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "empty-comment-network",
						"comment": ""
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "empty-comment-network",
						"comment": "",
						"is_default_network": false
					}
				}`,
			},
		}
		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
