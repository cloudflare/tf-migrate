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
}
