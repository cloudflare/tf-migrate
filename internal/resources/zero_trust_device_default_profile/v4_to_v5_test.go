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

resource "cloudflare_zero_trust_device_default_profile" "second" {
  account_id                     = "d138e56e89293a057740de681ac9abbf"
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
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
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		migrator := NewV4ToV5Migrator()
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic default profile state",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "name": "Default Profile",
            "description": "Default device settings",
            "default": true
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415"
          }
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "state with service_mode_v2 transformation",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "default": true,
            "service_mode_v2_mode": "warp",
            "service_mode_v2_port": 8080
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "service_mode_v2": {
              "mode": "warp",
              "port": 8080
            }
          }
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "state with numeric type conversions",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "default": true,
            "auto_connect": 300,
            "captive_portal": 180
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "auto_connect": 300.0,
            "captive_portal": 180.0
          }
        }
      ]
    }
  ]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})

	// Note: No StateTransformation_CustomProfile tests
	// State transformation tests for conditional routing (custom vs default) are not feasible
	// because GetResourceType() is called before TransformState() has access to instance data.
	// The config transformation tests above already validate the routing logic.

	t.Run("StateTransformation_EdgeCases", func(t *testing.T) {
		migrator := NewV4ToV5Migrator()
		tests := []testhelpers.StateTestCase{
			{
				Name: "old resource name - cloudflare_device_settings_policy",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_device_settings_policy",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "default": true
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415"
          }
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "only service_mode_v2_mode field",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "service_mode_v2_mode": "warp"
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415"
          }
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "only service_mode_v2_port field",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "service_mode_v2_port": 9000
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "service_mode_v2": {
              "port": 9000
            }
          }
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "all fields with type conversions",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "default": true,
            "name": "Default Profile",
            "description": "Default settings",
            "match": "identity.email == \"test@example.com\"",
            "precedence": 10,
            "enabled": true,
            "auto_connect": 0,
            "captive_portal": 300,
            "allow_mode_switch": true,
            "allow_updates": false,
            "allowed_to_leave": true,
            "disable_auto_fallback": false,
            "exclude_office_ips": true,
            "support_url": "https://support.example.com",
            "switch_locked": false,
            "tunnel_protocol": "wireguard",
            "service_mode_v2_mode": "warp",
            "service_mode_v2_port": 8080
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "auto_connect": 0,
            "captive_portal": 300,
            "allow_mode_switch": true,
            "allow_updates": false,
            "allowed_to_leave": true,
            "disable_auto_fallback": false,
            "exclude_office_ips": true,
            "support_url": "https://support.example.com",
            "switch_locked": false,
            "tunnel_protocol": "wireguard",
            "service_mode_v2": {
              "mode": "warp",
              "port": 8080
            }
          }
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "minimal state - no optional fields",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "default": true
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415"
          }
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "state with zero values for numeric fields",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "auto_connect": 0,
            "captive_portal": 0
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "auto_connect": 0,
            "captive_portal": 0
          }
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "empty instance attributes",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {}
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {}
        }
      ]
    }
  ]
}`,
			},
			{
				Name: "state with fallback_domains and empty exclude array - both should be removed",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "cloudflare_zero_trust_device_profiles",
      "name": "example",
      "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "default": true,
            "fallback_domains": [
              {
                "suffix": "corp.example.com",
                "description": "Corporate network",
                "dns_server": ["10.0.0.1", "10.0.0.2"]
              },
              {
                "suffix": "internal.example.com",
                "description": "Internal services",
                "dns_server": ["10.1.0.1"]
              }
            ],
            "exclude": [],
            "tunnel_protocol": "wireguard"
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_zero_trust_device_default_profile",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "account_id": "f037e56e89293a057740de681ac9abbe",
            "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
            "tunnel_protocol": "wireguard"
          }
        }
      ]
    }
  ]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
