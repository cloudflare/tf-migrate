# Integration test for bot_management migration
# Comprehensive test covering various configuration patterns
# Note: This resource is a zone-level singleton in production - only ONE per zone
# But integration tests create multiple instances with different names for testing patterns

# ============================================================================
# Pattern Group 1: Variables & Locals
# ============================================================================

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

locals {
  common_zone = var.cloudflare_zone_id
  name_prefix = "cftftest"
  test_tags   = ["test", "migration", "bot_management"]
  ai_protection_configs = {
    block    = "block"
    disabled = "disabled"
  }
}

# ============================================================================
# Pattern Group 2: for_each with Maps (5 resources)
# Tests: map iteration, each.value, each.key
# ============================================================================

resource "cloudflare_bot_management" "map_example" {
  for_each = {
    "config1" = {
      enable_js                       = true
      sbfm_definitely_automated       = "block"
      sbfm_likely_automated           = "managed_challenge"
      sbfm_verified_bots              = "allow"
      sbfm_static_resource_protection = true
    }
    "config2" = {
      enable_js                       = false
      sbfm_definitely_automated       = "allow"
      sbfm_likely_automated           = "block"
      sbfm_verified_bots              = "allow"
      sbfm_static_resource_protection = false
    }
    "config3" = {
      enable_js                       = true
      sbfm_definitely_automated       = "managed_challenge"
      sbfm_likely_automated           = "allow"
      sbfm_verified_bots              = "block"
      sbfm_static_resource_protection = true
    }
    "config4" = {
      enable_js                       = false
      sbfm_definitely_automated       = "block"
      sbfm_likely_automated           = "block"
      sbfm_verified_bots              = "allow"
      sbfm_static_resource_protection = false
    }
    "config5" = {
      enable_js                       = true
      sbfm_definitely_automated       = "allow"
      sbfm_likely_automated           = "managed_challenge"
      sbfm_verified_bots              = "allow"
      sbfm_static_resource_protection = true
    }
  }

  zone_id                         = local.common_zone
  enable_js                       = each.value.enable_js
  sbfm_definitely_automated       = each.value.sbfm_definitely_automated
  sbfm_likely_automated           = each.value.sbfm_likely_automated
  sbfm_verified_bots              = each.value.sbfm_verified_bots
  sbfm_static_resource_protection = each.value.sbfm_static_resource_protection
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = "disabled"
}

# ============================================================================
# Pattern Group 3: for_each with Sets (4 resources)
# Tests: toset(), set iteration
# ============================================================================

resource "cloudflare_bot_management" "set_example" {
  for_each = toset(["alpha", "beta", "gamma", "delta"])

  zone_id                         = var.cloudflare_zone_id
  enable_js                       = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = each.key == "alpha" ? true : false
  suppress_session_score          = each.key == "beta" ? true : false
  auto_update_model               = true
  ai_bots_protection              = each.key == "gamma" ? "block" : "disabled"
}

# ============================================================================
# Pattern Group 4: count-based resources (4 resources)
# Tests: count, count.index
# ============================================================================

resource "cloudflare_bot_management" "counted" {
  count = 4

  zone_id                         = local.common_zone
  enable_js                       = count.index % 2 == 0 ? true : false
  sbfm_definitely_automated       = count.index == 0 ? "block" : "managed_challenge"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = count.index < 2 ? true : false
  optimize_wordpress              = false
  suppress_session_score          = count.index == 3 ? true : false
  auto_update_model               = true
  ai_bots_protection              = count.index == 1 ? "block" : "disabled"
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# Tests: count with ternary operator
# ============================================================================

locals {
  enable_fight_mode = true
  enable_test_mode  = false
}

resource "cloudflare_bot_management" "conditional_enabled" {
  count = local.enable_fight_mode ? 1 : 0

  zone_id                         = var.cloudflare_zone_id
  fight_mode                      = true
  enable_js                       = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = "block"
}

resource "cloudflare_bot_management" "conditional_disabled" {
  count = local.enable_test_mode ? 1 : 0

  zone_id                         = var.cloudflare_zone_id
  enable_js                       = false
  sbfm_definitely_automated       = "allow"
  sbfm_likely_automated           = "allow"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = false
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = false
  ai_bots_protection              = "disabled"
}

# ============================================================================
# Pattern Group 6: Terraform Functions
# Tests: String interpolation, local references
# ============================================================================

resource "cloudflare_bot_management" "with_functions" {
  zone_id                         = var.cloudflare_zone_id
  enable_js                       = true
  sbfm_definitely_automated       = local.test_tags[0] == "test" ? "block" : "allow"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = local.test_tags[2] == "bot_management" ? true : false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = local.ai_protection_configs["block"]
}

# ============================================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# Tests: create_before_destroy, ignore_changes
# ============================================================================

resource "cloudflare_bot_management" "with_lifecycle" {
  zone_id                         = local.common_zone
  enable_js                       = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = "disabled"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [optimize_wordpress]
  }
}

resource "cloudflare_bot_management" "with_prevent_destroy" {
  zone_id                         = var.cloudflare_zone_id
  enable_js                       = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = false
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = "disabled"

  lifecycle {
    prevent_destroy = false
  }
}

# ============================================================================
# Pattern Group 8: Edge Cases
# Tests: minimal, maximal, various combinations
# ============================================================================

# Minimal configuration - only required field
resource "cloudflare_bot_management" "minimal" {
  zone_id = var.cloudflare_zone_id
}

# Maximal configuration - all optional fields populated
resource "cloudflare_bot_management" "maximal" {
  zone_id                         = local.common_zone
  enable_js                       = true
  fight_mode                      = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = true
  suppress_session_score          = true
  auto_update_model               = true
  ai_bots_protection              = "block"
}

# All false/allow/disabled values
resource "cloudflare_bot_management" "all_permissive" {
  zone_id                         = var.cloudflare_zone_id
  enable_js                       = false
  fight_mode                      = false
  sbfm_definitely_automated       = "allow"
  sbfm_likely_automated           = "allow"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = false
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = false
  ai_bots_protection              = "disabled"
}

# All restrictive values
resource "cloudflare_bot_management" "all_restrictive" {
  zone_id                         = local.common_zone
  enable_js                       = true
  fight_mode                      = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "block"
  sbfm_verified_bots              = "block"
  sbfm_static_resource_protection = true
  optimize_wordpress              = true
  suppress_session_score          = true
  auto_update_model               = true
  ai_bots_protection              = "block"
}

# Mixed managed_challenge values
resource "cloudflare_bot_management" "managed_challenge_mix" {
  zone_id                         = var.cloudflare_zone_id
  enable_js                       = true
  sbfm_definitely_automated       = "managed_challenge"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = "disabled"
}

# Total resources: 5 (map) + 4 (set) + 4 (count) + 1 (conditional enabled) + 1 (functions) + 2 (lifecycle) + 6 (edge cases) = 23 instances
