package zero_trust_device_default_profile

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
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
  exclude                        = []
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
  exclude                        = []
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
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("ConfigTransformation_EdgeCases", func(t *testing.T) {
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
  exclude                        = []
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
  exclude                        = []
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
  exclude                        = []
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
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

resource "cloudflare_zero_trust_device_default_profile" "second" {
  account_id                     = "d138e56e89293a057740de681ac9abbf"
  exclude                        = []
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
  exclude                        = []
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
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
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

	t.Run("StateTransformation_EdgeCases", func(t *testing.T) {
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
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
