# Integration test for zero_trust_device_default_profile migration
# Covers v4 cloudflare_zero_trust_device_profiles (default=true) → v5 cloudflare_zero_trust_device_default_profile

# ============================================================================
# Pattern Group 1: Variables & Locals
# ============================================================================

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (not used by this account-scoped resource)"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (not used by this account-scoped resource)"
  type        = string
}

locals {
  common_account = var.cloudflare_account_id
  name_prefix    = "cftftest"
  test_tags      = ["test", "migration", "device_profile"]
}

# ============================================================================
# Pattern Group 2: for_each with Maps (5 resources)
# Tests: map iteration, each.value, each.key
# ============================================================================

resource "cloudflare_zero_trust_device_default_profile" "map_example" {
  for_each = {
    "profile1" = {
      allow_mode_switch = true
      auto_connect      = 0
      captive_portal    = 180
    }
    "profile2" = {
      allow_mode_switch = false
      auto_connect      = 15
      captive_portal    = 300
    }
    "profile3" = {
      allow_mode_switch = true
      auto_connect      = 0
      captive_portal    = 0
    }
    "profile4" = {
      allow_mode_switch = false
      auto_connect      = 30
      captive_portal    = 600
    }
    "profile5" = {
      allow_mode_switch = true
      auto_connect      = 60
      captive_portal    = 900
    }
  }

  account_id = local.common_account

  allow_mode_switch              = each.value.allow_mode_switch
  auto_connect                   = each.value.auto_connect
  captive_portal                 = each.value.captive_portal
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# ============================================================================
# Pattern Group 3: for_each with Sets (4 resources)
# Tests: toset(), set iteration
# ============================================================================

resource "cloudflare_zero_trust_device_default_profile" "set_example" {
  for_each = toset(["alpha", "beta", "gamma", "delta"])

  account_id = var.cloudflare_account_id

  allow_mode_switch              = true
  auto_connect                   = 0
  captive_portal                 = 180
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# ============================================================================
# Pattern Group 4: count-based resources (4 resources)
# Tests: count, count.index
# ============================================================================

resource "cloudflare_zero_trust_device_default_profile" "counted" {
  count = 4

  account_id = local.common_account

  allow_mode_switch              = count.index % 2 == 0
  auto_connect                   = count.index * 15
  captive_portal                 = (count.index + 1) * 100
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# ============================================================================
# Pattern Group 5: Conditional resource creation (1 resource or 0)
# Tests: conditional count, ternary
# ============================================================================

resource "cloudflare_zero_trust_device_default_profile" "conditional" {
  count = local.name_prefix == "cftftest" ? 1 : 0

  account_id = var.cloudflare_account_id

  allow_mode_switch              = true
  auto_connect                   = 0
  captive_portal                 = 300
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# ============================================================================
# Pattern Group 6: Cross-resource references (if applicable)
# Tests: resource references
# Note: Device profiles don't typically reference other resources
# ============================================================================

# ============================================================================
# Pattern Group 7: Lifecycle meta-arguments (1 resource)
# Tests: lifecycle, create_before_destroy, ignore_changes
# ============================================================================

resource "cloudflare_zero_trust_device_default_profile" "with_lifecycle" {
  account_id = var.cloudflare_account_id

  allow_mode_switch = true
  auto_connect      = 0
  captive_portal    = 180

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [captive_portal]
  }
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# ============================================================================
# Pattern Group 8: Terraform functions (various)
# Tests: base64encode, join, format, etc.
# ============================================================================

resource "cloudflare_zero_trust_device_default_profile" "with_functions" {
  account_id = var.cloudflare_account_id

  # Using terraform functions
  allow_mode_switch              = length(local.test_tags) > 0
  auto_connect                   = min(30, 15 * 2)
  captive_portal                 = max(180, 100)
  switch_locked                  = contains(local.test_tags, "migration")
  support_url                    = format("https://support.%s.cf-tf-test.com", local.name_prefix)
  tunnel_protocol                = "wireguard"
  disable_auto_fallback          = true
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# ============================================================================
# Pattern Group 9: Edge Cases
# ============================================================================

# Edge Case 1: Minimal config (only required fields + default)
resource "cloudflare_zero_trust_device_default_profile" "minimal" {
  account_id                     = var.cloudflare_account_id
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# Edge Case 2: Maximal config (all optional fields populated)
resource "cloudflare_zero_trust_device_default_profile" "maximal" {
  account_id = var.cloudflare_account_id

  # All optional fields
  allow_mode_switch     = true
  allow_updates         = true
  allowed_to_leave      = true
  auto_connect          = 30
  captive_portal        = 300
  disable_auto_fallback = true
  exclude_office_ips    = true
  switch_locked         = false
  support_url           = "https://support.cf-tf-test.com"
  tunnel_protocol       = "wireguard"

  service_mode_v2 = {
    mode = "proxy"
    port = 8080
  }
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# Edge Case 3: service_mode_v2 with both mode and port
resource "cloudflare_zero_trust_device_default_profile" "service_mode_both" {
  account_id = var.cloudflare_account_id

  service_mode_v2 = {
    mode = "proxy"
    port = 443
  }
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# Edge Case 5: Zero values for numeric fields
resource "cloudflare_zero_trust_device_default_profile" "zero_values" {
  account_id = var.cloudflare_account_id

  auto_connect                   = 0
  captive_portal                 = 0
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# Edge Case 6: Variable references throughout
resource "cloudflare_zero_trust_device_default_profile" "with_variables" {
  account_id = var.cloudflare_account_id

  allow_mode_switch              = true
  auto_connect                   = 15
  captive_portal                 = 180
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# Edge Case 7: Old resource name (cloudflare_device_settings_policy)
# This tests that both v4 resource names are handled
resource "cloudflare_zero_trust_device_default_profile" "old_name" {
  account_id = var.cloudflare_account_id

  allow_mode_switch              = false
  auto_connect                   = 0
  captive_portal                 = 600
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# Edge Case 8: String interpolation
resource "cloudflare_zero_trust_device_default_profile" "interpolation" {
  account_id = var.cloudflare_account_id

  support_url                    = "https://${local.name_prefix}-support.cf-tf-test.com"
  auto_connect                   = 0
  captive_portal                 = 180
  exclude                        = []
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

# TOTAL RESOURCES:
# - for_each maps: 5 instances
# - for_each sets: 4 instances
# - count-based: 4 instances
# - conditional: 1 instance
# - lifecycle: 1 instance
# - functions: 1 instance
# - edge cases: 7 instances
# TOTAL: 23 resource instances
