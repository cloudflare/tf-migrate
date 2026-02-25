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


# ============================================================================
# Pattern Group 3: for_each with Sets (4 resources)
# Tests: toset(), set iteration
# ============================================================================


# ============================================================================
# Pattern Group 4: count-based resources (4 resources)
# Tests: count, count.index
# ============================================================================


# ============================================================================
# Pattern Group 5: Conditional resource creation (1 resource or 0)
# Tests: conditional count, ternary
# ============================================================================


# ============================================================================
# Pattern Group 6: Cross-resource references (if applicable)
# Tests: resource references
# Note: Device profiles don't typically reference other resources
# ============================================================================

# ============================================================================
# Pattern Group 7: Lifecycle meta-arguments (1 resource)
# Tests: lifecycle, create_before_destroy, ignore_changes
# ============================================================================


# ============================================================================
# Pattern Group 8: Terraform functions (various)
# Tests: base64encode, join, format, etc.
# ============================================================================


# ============================================================================
# Pattern Group 9: Edge Cases
# ============================================================================








# ============================================================================
# Pattern Group 8: Custom Profiles (match + precedence)
# Tests: custom profile routing, precedence transformation, field preservation
# ============================================================================





# TOTAL RESOURCES:
# - for_each maps: 5 instances (default profiles)
# - for_each sets: 4 instances (default profiles)
# - count-based: 4 instances (default profiles)
# - conditional: 1 instance (default profile)
# - lifecycle: 1 instance (default profile)
# - functions: 1 instance (default profile)
# - edge cases: 7 instances (default profiles)
# - custom profiles: 5 instances (3 single + 2 for_each)
# TOTAL: 28 resource instances

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
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.map_example
  to   = cloudflare_zero_trust_device_default_profile.map_example
}

resource "cloudflare_zero_trust_device_default_profile" "set_example" {
  for_each = toset(["alpha", "beta", "gamma", "delta"])

  account_id = var.cloudflare_account_id

  allow_mode_switch              = true
  auto_connect                   = 0
  captive_portal                 = 180
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.set_example
  to   = cloudflare_zero_trust_device_default_profile.set_example
}

resource "cloudflare_zero_trust_device_default_profile" "counted" {
  count = 4

  account_id = local.common_account

  allow_mode_switch              = count.index % 2 == 0
  auto_connect                   = count.index * 15
  captive_portal                 = (count.index + 1) * 100
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.counted
  to   = cloudflare_zero_trust_device_default_profile.counted
}

resource "cloudflare_zero_trust_device_default_profile" "conditional" {
  count = local.name_prefix == "cftftest" ? 1 : 0

  account_id = var.cloudflare_account_id

  allow_mode_switch              = true
  auto_connect                   = 0
  captive_portal                 = 300
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.conditional
  to   = cloudflare_zero_trust_device_default_profile.conditional
}

resource "cloudflare_zero_trust_device_default_profile" "with_lifecycle" {
  account_id = var.cloudflare_account_id

  allow_mode_switch = true
  auto_connect      = 0
  captive_portal    = 180

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [captive_portal]
  }
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.with_lifecycle
  to   = cloudflare_zero_trust_device_default_profile.with_lifecycle
}

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
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.with_functions
  to   = cloudflare_zero_trust_device_default_profile.with_functions
}

# Edge Case 1: Minimal config (only required fields + default)
resource "cloudflare_zero_trust_device_default_profile" "minimal" {
  account_id                     = var.cloudflare_account_id
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.minimal
  to   = cloudflare_zero_trust_device_default_profile.minimal
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
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.maximal
  to   = cloudflare_zero_trust_device_default_profile.maximal
}

# Edge Case 3: service_mode_v2 with both mode and port
resource "cloudflare_zero_trust_device_default_profile" "service_mode_both" {
  account_id = var.cloudflare_account_id

  service_mode_v2 = {
    mode = "proxy"
    port = 443
  }
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.service_mode_both
  to   = cloudflare_zero_trust_device_default_profile.service_mode_both
}

# Edge Case 5: Zero values for numeric fields
resource "cloudflare_zero_trust_device_default_profile" "zero_values" {
  account_id = var.cloudflare_account_id

  auto_connect                   = 0
  captive_portal                 = 0
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.zero_values
  to   = cloudflare_zero_trust_device_default_profile.zero_values
}

# Edge Case 6: Variable references throughout
resource "cloudflare_zero_trust_device_default_profile" "with_variables" {
  account_id = var.cloudflare_account_id

  allow_mode_switch              = true
  auto_connect                   = 15
  captive_portal                 = 180
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.with_variables
  to   = cloudflare_zero_trust_device_default_profile.with_variables
}

# Edge Case 7: Old resource name (cloudflare_device_settings_policy)
# This tests that both v4 resource names are handled
resource "cloudflare_zero_trust_device_default_profile" "old_name" {
  account_id = var.cloudflare_account_id

  allow_mode_switch              = false
  auto_connect                   = 0
  captive_portal                 = 600
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_device_settings_policy.old_name
  to   = cloudflare_zero_trust_device_default_profile.old_name
}

# Edge Case 8: String interpolation
resource "cloudflare_zero_trust_device_default_profile" "interpolation" {
  account_id = var.cloudflare_account_id

  support_url                    = "https://${local.name_prefix}-support.cf-tf-test.com"
  auto_connect                   = 0
  captive_portal                 = 180
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.interpolation
  to   = cloudflare_zero_trust_device_default_profile.interpolation
}

# Basic custom profile
resource "cloudflare_zero_trust_device_custom_profile" "custom_employees" {
  account_id  = var.cloudflare_account_id
  name        = "Employee Profile"
  description = "Custom profile for employees"
  match       = "identity.groups == \"employees\""
  precedence  = 1000
}

moved {
  from = cloudflare_zero_trust_device_profiles.custom_employees
  to   = cloudflare_zero_trust_device_custom_profile.custom_employees
}

# Custom profile with service_mode_v2
resource "cloudflare_zero_trust_device_custom_profile" "custom_contractors" {
  account_id  = var.cloudflare_account_id
  name        = "Contractor Profile"
  description = "Custom profile for contractors"
  match       = "identity.groups == \"contractors\""
  precedence  = 1100
  service_mode_v2 = {
    mode = "proxy"
    port = 8080
  }
}

moved {
  from = cloudflare_zero_trust_device_profiles.custom_contractors
  to   = cloudflare_zero_trust_device_custom_profile.custom_contractors
}

# Custom profile with many optional fields
resource "cloudflare_zero_trust_device_custom_profile" "custom_admins" {
  account_id            = var.cloudflare_account_id
  name                  = "Admin Profile"
  description           = "Custom profile for administrators"
  match                 = "identity.groups == \"admins\""
  precedence            = 950
  allow_mode_switch     = false
  allow_updates         = true
  allowed_to_leave      = true
  auto_connect          = 30
  captive_portal        = 300
  disable_auto_fallback = true
  exclude_office_ips    = true
  support_url           = "https://admin-support.cf-tf-test.com"
  switch_locked         = true
  tunnel_protocol       = "wireguard"
}

moved {
  from = cloudflare_zero_trust_device_profiles.custom_admins
  to   = cloudflare_zero_trust_device_custom_profile.custom_admins
}

# Multiple custom profiles with for_each
resource "cloudflare_zero_trust_device_custom_profile" "custom_teams" {
  for_each = {
    "engineering" = {
      precedence = 150
      match      = "identity.groups == \"engineering\""
    }
    "sales" = {
      precedence = 160
      match      = "identity.groups == \"sales\""
    }
  }

  account_id   = var.cloudflare_account_id
  name         = "${title(each.key)} Team Profile"
  description  = "Custom profile for ${each.key} team"
  match        = each.value.match
  precedence   = 1000
  auto_connect = 15
}

moved {
  from = cloudflare_zero_trust_device_profiles.custom_teams
  to   = cloudflare_zero_trust_device_custom_profile.custom_teams
}
