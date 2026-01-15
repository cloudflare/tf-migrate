# Integration test for url_normalization_settings migration
# Comprehensive test covering various configuration patterns
# Note: This is a SINGLETON resource in production - only ONE per zone
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
  test_tags   = ["test", "migration", "url_normalization"]
  type_configs = {
    cloudflare = "cloudflare"
    rfc3986    = "rfc3986"
  }
  scope_configs = {
    none     = "none"
    incoming = "incoming"
    both     = "both"
  }
}

# ============================================================================
# Pattern Group 2: for_each with Maps (6 resources)
# Tests: map iteration, each.value, each.key, all combinations
# ============================================================================

resource "cloudflare_url_normalization_settings" "map_example" {
  for_each = {
    "cloudflare_incoming" = {
      type  = "cloudflare"
      scope = "incoming"
    }
    "cloudflare_both" = {
      type  = "cloudflare"
      scope = "both"
    }
    "cloudflare_none" = {
      type  = "cloudflare"
      scope = "none"
    }
    "rfc3986_incoming" = {
      type  = "rfc3986"
      scope = "incoming"
    }
    "rfc3986_both" = {
      type  = "rfc3986"
      scope = "both"
    }
    "rfc3986_none" = {
      type  = "rfc3986"
      scope = "none"
    }
  }

  zone_id = local.common_zone
  type    = each.value.type
  scope   = each.value.scope
}

# ============================================================================
# Pattern Group 3: for_each with Sets (4 resources)
# Tests: toset(), set iteration
# ============================================================================

resource "cloudflare_url_normalization_settings" "set_example" {
  for_each = toset(["alpha", "beta", "gamma", "delta"])

  zone_id = var.cloudflare_zone_id
  type    = each.key == "alpha" || each.key == "beta" ? "cloudflare" : "rfc3986"
  scope   = each.key == "alpha" || each.key == "gamma" ? "incoming" : "both"
}

# ============================================================================
# Pattern Group 4: count-based resources (3 resources)
# Tests: count, count.index
# ============================================================================

resource "cloudflare_url_normalization_settings" "counted" {
  count = 3

  zone_id = local.common_zone
  type    = count.index == 0 ? "cloudflare" : "rfc3986"
  scope   = count.index == 0 ? "incoming" : (count.index == 1 ? "both" : "none")
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# Tests: count with ternary operator
# ============================================================================

locals {
  enable_cloudflare_norm = true
  enable_test_norm       = false
}

resource "cloudflare_url_normalization_settings" "conditional_enabled" {
  count = local.enable_cloudflare_norm ? 1 : 0

  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "conditional_disabled" {
  count = local.enable_test_norm ? 1 : 0

  zone_id = var.cloudflare_zone_id
  type    = "rfc3986"
  scope   = "none"
}

# ============================================================================
# Pattern Group 6: Terraform Functions
# Tests: String interpolation, local references, map lookups
# ============================================================================

resource "cloudflare_url_normalization_settings" "with_functions" {
  zone_id = var.cloudflare_zone_id
  type    = local.test_tags[0] == "test" ? local.type_configs["cloudflare"] : local.type_configs["rfc3986"]
  scope   = local.scope_configs["incoming"]
}

resource "cloudflare_url_normalization_settings" "with_map_lookup" {
  zone_id = local.common_zone
  type    = local.type_configs["rfc3986"]
  scope   = local.scope_configs["both"]
}

# ============================================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# Tests: create_before_destroy, ignore_changes
# ============================================================================

resource "cloudflare_url_normalization_settings" "with_lifecycle" {
  zone_id = local.common_zone
  type    = "cloudflare"
  scope   = "incoming"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [scope]
  }
}

resource "cloudflare_url_normalization_settings" "with_prevent_destroy" {
  zone_id = var.cloudflare_zone_id
  type    = "rfc3986"
  scope   = "both"

  lifecycle {
    prevent_destroy = false
  }
}

# ============================================================================
# Pattern Group 8: Edge Cases
# Tests: all type/scope combinations, various patterns
# ============================================================================

# Cloudflare type with all scopes
resource "cloudflare_url_normalization_settings" "cloudflare_incoming" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "cloudflare_both" {
  zone_id = local.common_zone
  type    = "cloudflare"
  scope   = "both"
}

resource "cloudflare_url_normalization_settings" "cloudflare_none" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "none"
}

# RFC3986 type with all scopes
resource "cloudflare_url_normalization_settings" "rfc3986_incoming" {
  zone_id = local.common_zone
  type    = "rfc3986"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "rfc3986_both" {
  zone_id = var.cloudflare_zone_id
  type    = "rfc3986"
  scope   = "both"
}

resource "cloudflare_url_normalization_settings" "rfc3986_none" {
  zone_id = local.common_zone
  type    = "rfc3986"
  scope   = "none"
}

# Total resources: 6 (map) + 4 (set) + 3 (count) + 1 (conditional enabled) + 2 (functions) + 2 (lifecycle) + 6 (edge cases) = 24 instances
