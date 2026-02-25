package zero_trust_device_default_profile

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		migrator := NewV4ToV5Migrator()
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic default profile with minimal fields",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Default Profile"
  description = "Default device settings"
  default    = true
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default profile with service_mode_v2 fields",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "Default Profile"
  description          = "Default device settings"
  default              = true
  service_mode_v2_mode = "warp"
  service_mode_v2_port = 8080
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  service_mode_v2 = {
    mode = "warp"
    port = 8080
  }
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default profile with all settings",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id              = "f037e56e89293a057740de681ac9abbe"
  name                    = "Default Profile"
  description             = "Default device settings"
  default                 = true
  allow_mode_switch       = false
  allow_updates           = true
  allowed_to_leave        = false
  auto_connect            = 300
  captive_portal          = 180
  disable_auto_fallback   = false
  exclude_office_ips      = true
  support_url             = "https://support.example.com"
  switch_locked           = false
  tunnel_protocol         = "wireguard"
  service_mode_v2_mode    = "warp"
  service_mode_v2_port    = 8080
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id            = "f037e56e89293a057740de681ac9abbe"
  allow_mode_switch     = false
  allow_updates         = true
  allowed_to_leave      = false
  auto_connect          = 300
  captive_portal        = 180
  disable_auto_fallback = false
  exclude_office_ips    = true
  support_url           = "https://support.example.com"
  switch_locked         = false
  tunnel_protocol       = "wireguard"
  service_mode_v2 = {
    mode = "warp"
    port = 8080
  }
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("ConfigTransformation_EdgeCases", func(t *testing.T) {
		migrator := NewV4ToV5Migrator()
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "old resource name - cloudflare_device_settings_policy",
				Input: `
resource "cloudflare_device_settings_policy" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Default"
  description = "Default policy"
  default     = true
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_device_settings_policy.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "only service_mode_v2_mode without port",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  default              = true
  service_mode_v2_mode = "warp"
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "only service_mode_v2_port without mode",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  default              = true
  service_mode_v2_port = 8080
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  service_mode_v2 = {
    port = 8080
  }
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "multiple resources in one file",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "first" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "First"
  default    = true
}

resource "cloudflare_zero_trust_device_profiles" "second" {
  account_id  = "d138e56e89293a057740de681ac9abbf"
  name        = "Second"
  description = "Second profile"
  default     = true
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "first" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.first
  to   = cloudflare_zero_trust_device_default_profile.first
}

resource "cloudflare_zero_trust_device_default_profile" "second" {
  account_id                     = "d138e56e89293a057740de681ac9abbf"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.second
  to   = cloudflare_zero_trust_device_default_profile.second
}`,
			},
			{
				Name: "resource with variables",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id = var.cloudflare_account_id
  name       = "Default"
  default    = true
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = var.cloudflare_account_id
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "all removed fields present with various values",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  default     = true
  name        = "Default Profile Name"
  description = "This is a very long description that should be removed"
  match       = "identity.email == \"user@example.com\""
  precedence  = 100
  enabled     = false
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("ConfigTransformation_CustomProfile", func(t *testing.T) {
		migrator := NewV4ToV5Migrator()
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "custom profile with match and precedence (no default field)",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Custom Profile"
  description = "Custom device settings"
  match       = "identity.email == \"user@example.com\""
  precedence  = 100
  enabled     = true
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Custom Profile"
  description = "Custom device settings"
  match       = "identity.email == \"user@example.com\""
  precedence  = 1000
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_custom_profile.example
}`,
			},
			{
				Name: "custom profile with default=false",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Custom Profile"
  description = "Custom device settings"
  default     = false
  match       = "identity.email == \"user@example.com\""
  precedence  = 100
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Custom Profile"
  description = "Custom device settings"
  match       = "identity.email == \"user@example.com\""
  precedence  = 1000
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_custom_profile.example
}`,
			},
			{
				Name: "custom profile with service_mode_v2",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "Custom Profile"
  match                = "identity.email == \"user@example.com\""
  precedence           = 200
  service_mode_v2_mode = "proxy"
  service_mode_v2_port = 8080
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Custom Profile"
  match      = "identity.email == \"user@example.com\""
  precedence = 1100
  service_mode_v2 = {
    mode = "proxy"
    port = 8080
  }
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_custom_profile.example
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("ConfigTransformation_RoutingLogic", func(t *testing.T) {
		migrator := NewV4ToV5Migrator()
		tests := []testhelpers.ConfigTestCase{
			// Test all permutations of default, match, and precedence
			// Priority: default field > presence of match+precedence

			// --- default=true cases (always routes to default profile) ---
			{
				Name: "default=true, no match, no precedence → default profile",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  default     = true
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default=true, has match, no precedence → default profile (invalid config but default wins)",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  default     = true
  match       = "identity.email == \"user@example.com\""
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default=true, no match, has precedence → default profile (invalid config but default wins)",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  default     = true
  precedence  = 100
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default=true, has match, has precedence → default profile (default takes priority)",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  default     = true
  match       = "identity.email == \"user@example.com\""
  precedence  = 100
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},

			// --- default=false cases ---
			{
				Name: "default=false, no match, no precedence → default profile",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  default     = false
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default=false, has match, no precedence → default profile (missing precedence)",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  default     = false
  match       = "identity.email == \"user@example.com\""
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default=false, no match, has precedence → default profile (missing match)",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  default     = false
  precedence  = 100
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default=false, has match, has precedence → custom profile",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Custom Profile"
  description = "Custom settings"
  default     = false
  match       = "identity.email == \"user@example.com\""
  precedence  = 100
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Custom Profile"
  description = "Custom settings"
  match       = "identity.email == \"user@example.com\""
  precedence  = 1000
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_custom_profile.example
}`,
			},

			// --- no default field cases ---
			{
				Name: "no default, no match, no precedence → default profile",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "no default, has match, no precedence → default profile (missing precedence)",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  match       = "identity.email == \"user@example.com\""
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "no default, no match, has precedence → default profile (missing match)",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  precedence  = 100
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id                     = "f037e56e89293a057740de681ac9abbe"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "no default, has match, has precedence → custom profile",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Custom Profile"
  description = "Custom settings"
  match       = "identity.email == \"user@example.com\""
  precedence  = 150
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Custom Profile"
  description = "Custom settings"
  match       = "identity.email == \"user@example.com\""
  precedence  = 1050
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_custom_profile.example
}`,
			},

			// --- Edge cases with different precedence values ---
			{
				Name: "custom profile with precedence=1 → becomes 901",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  match       = "identity.email == \"test@example.com\""
  precedence  = 1
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  match       = "identity.email == \"test@example.com\""
  precedence  = 901
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_custom_profile.example
}`,
			},
			{
				Name: "custom profile with precedence=999 → becomes 1899",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  match       = "identity.email == \"test@example.com\""
  precedence  = 999
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Profile"
  match       = "identity.email == \"test@example.com\""
  precedence  = 1899
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_custom_profile.example
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})


	t.Run("StateTransformation_Removed", func(t *testing.T) {
		t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders (MoveState + UpgradeState)")
	})

	t.Run("ConfigTransformation_SplitTunnelEmbedding", func(t *testing.T) {
		migrator := NewV4ToV5Migrator()
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "default profile with single exclude split tunnel",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Default Profile"
  default    = true
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "exclude"
  tunnels {
    address     = "192.168.1.0/24"
    description = "Local network"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  exclude = [{
    address     = "192.168.1.0/24"
    description = "Local network"
  }]
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default profile with single include split tunnel",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Default Profile"
  default    = true
}

resource "cloudflare_split_tunnel" "include_tunnel" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "include"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Corporate network"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  include = [{
    address     = "10.0.0.0/8"
    description = "Corporate network"
  }]
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default profile with multiple exclude split tunnels",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Default Profile"
  default    = true
}

resource "cloudflare_split_tunnel" "exclude_tunnel1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "exclude"
  tunnels {
    address     = "192.168.1.0/24"
    description = "Network 1"
  }
}

resource "cloudflare_split_tunnel" "exclude_tunnel2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Network 2"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  exclude = [{
    address     = "192.168.1.0/24"
    description = "Network 1"
  }, {
    address     = "10.0.0.0/8"
    description = "Network 2"
  }]
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default profile with both exclude and include split tunnels",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Default Profile"
  default    = true
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "exclude"
  tunnels {
    address     = "192.168.1.0/24"
    description = "Exclude network"
  }
}

resource "cloudflare_split_tunnel" "include_tunnel" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "include"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Include network"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  exclude = [{
    address     = "192.168.1.0/24"
    description = "Exclude network"
  }]
  include = [{
    address     = "10.0.0.0/8"
    description = "Include network"
  }]
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "default profile with split tunnels and other settings",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "example" {
  account_id            = "f037e56e89293a057740de681ac9abbe"
  name                  = "Default Profile"
  default               = true
  allow_mode_switch     = false
  allow_updates         = true
  tunnel_protocol       = "wireguard"
  service_mode_v2_mode  = "warp"
  service_mode_v2_port  = 8080
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "exclude"
  tunnels {
    address     = "192.168.0.0/16"
    description = "Private network"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "example" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  allow_mode_switch = false
  allow_updates     = true
  tunnel_protocol   = "wireguard"
  service_mode_v2 = {
    mode = "warp"
    port = 8080
  }
  exclude = [{
    address     = "192.168.0.0/16"
    description = "Private network"
  }]
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.example
  to   = cloudflare_zero_trust_device_default_profile.example
}`,
			},
			{
				Name: "v4 custom profile with split tunnel referencing it",
				Input: `
resource "cloudflare_zero_trust_device_profiles" "employees" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Employee Profile"
  match      = "identity.groups == \"employees\""
  precedence = 100
}

resource "cloudflare_split_tunnel" "employee_tunnel" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = cloudflare_zero_trust_device_profiles.employees.id
  mode       = "include"
  tunnels {
    address     = "10.100.0.0/16"
    description = "Employee resources"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Employee Profile"
  match      = "identity.groups == \"employees\""
  precedence = 1000
  include = [{
    address     = "10.100.0.0/16"
    description = "Employee resources"
  }]
}

moved {
  from = cloudflare_zero_trust_device_profiles.employees
  to   = cloudflare_zero_trust_device_custom_profile.employees
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

} 
