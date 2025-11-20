# Integration Test for cloudflare_logpull_retention v4 → v5 Migration
# This file tests ALL Terraform patterns as required by subtask 08

# ============================================================================
# Pattern Group 1: Variables & Locals (Required by Subtask 08)
# ============================================================================

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

locals {
  common_zone     = var.cloudflare_zone_id
  name_prefix     = "test-integration"
  tags            = ["test", "migration", "v4_to_v5"]
  enable_feature  = true
  enable_test     = false
  zone_count      = 3
}

# ============================================================================
# Pattern Group 2: for_each with Maps (3-5 resources)
# ============================================================================

resource "cloudflare_logpull_retention" "map_example" {
  for_each = {
    "zone1" = {
      zone_id = var.cloudflare_zone_id
      enabled = true
    }
    "zone2" = {
      zone_id = var.cloudflare_zone_id
      enabled = false
    }
    "zone3" = {
      zone_id = var.cloudflare_zone_id
      enabled = true
    }
    "zone4" = {
      zone_id = var.cloudflare_zone_id
      enabled = false
    }
  }

  zone_id = each.value.zone_id
  enabled = each.value.enabled
}

# ============================================================================
# Pattern Group 3: for_each with Sets (3-5 items)
# ============================================================================

resource "cloudflare_logpull_retention" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  zone_id = var.cloudflare_zone_id
  enabled = each.key == "alpha" || each.key == "gamma" ? true : false
}

# ============================================================================
# Pattern Group 4: count-based Resources (at least 3)
# ============================================================================

resource "cloudflare_logpull_retention" "counted" {
  count = 3

  zone_id = var.cloudflare_zone_id
  enabled = count.index % 2 == 0 ? true : false  # Alternating true/false
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

resource "cloudflare_logpull_retention" "conditional_enabled" {
  count = local.enable_feature ? 1 : 0

  zone_id = var.cloudflare_zone_id
  enabled = true
}

resource "cloudflare_logpull_retention" "conditional_disabled" {
  count = local.enable_test ? 1 : 0

  zone_id = var.cloudflare_zone_id
  enabled = false  # Won't be created (local.enable_test = false)
}

# ============================================================================
# Pattern Group 6: Terraform Functions
# ============================================================================

resource "cloudflare_logpull_retention" "with_functions" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}

resource "cloudflare_logpull_retention" "with_interpolation" {
  zone_id = "${var.cloudflare_zone_id}"  # String interpolation
  enabled = false
}

# ============================================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# ============================================================================

resource "cloudflare_logpull_retention" "with_lifecycle" {
  zone_id = var.cloudflare_zone_id
  enabled = true

  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_logpull_retention" "with_ignore_changes" {
  zone_id = local.common_zone
  enabled = false

  lifecycle {
    ignore_changes = [enabled]
  }
}

resource "cloudflare_logpull_retention" "with_prevent_destroy" {
  zone_id = var.cloudflare_zone_id
  enabled = true

  lifecycle {
    prevent_destroy = false  # Set to false for testing
  }
}

# ============================================================================
# Pattern Group 8: Edge Cases
# ============================================================================

# Minimal resource (only required fields)
resource "cloudflare_logpull_retention" "minimal" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}

# Using locals reference
resource "cloudflare_logpull_retention" "with_local" {
  zone_id = local.common_zone
  enabled = local.enable_feature
}

# Complex conditional expression
resource "cloudflare_logpull_retention" "complex_conditional" {
  zone_id = var.cloudflare_zone_id
  enabled = local.enable_feature && !local.enable_test ? true : false
}

# ============================================================================
# Pattern Group 9: Additional Real-World Patterns
# ============================================================================

# Testing enabled=true explicitly
resource "cloudflare_logpull_retention" "enabled_explicit" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}

# Testing enabled=false explicitly
resource "cloudflare_logpull_retention" "disabled_explicit" {
  zone_id = var.cloudflare_zone_id
  enabled = false
}

# Using ternary operator
resource "cloudflare_logpull_retention" "ternary_true" {
  zone_id = var.cloudflare_zone_id
  enabled = 1 == 1 ? true : false
}

resource "cloudflare_logpull_retention" "ternary_false" {
  zone_id = var.cloudflare_zone_id
  enabled = 1 == 2 ? true : false
}

# ============================================================================
# Summary:
# - Total resource declarations: 18
# - Total instances created:
#   - map_example: 4 instances
#   - set_example: 4 instances
#   - counted: 3 instances
#   - conditional_enabled: 1 instance
#   - conditional_disabled: 0 instances (count = 0)
#   - Others: 11 instances
#   TOTAL: 4 + 4 + 3 + 1 + 0 + 11 = 23 instances
#
# Patterns covered:
# ✓ Variables (cloudflare_account_id, cloudflare_zone_id)
# ✓ Locals (common_zone, enable_feature, enable_test)
# ✓ for_each with maps (4 resources)
# ✓ for_each with sets using toset() (4 resources)
# ✓ count-based resources (3 resources)
# ✓ Conditional creation with ternary operator
# ✓ String interpolation
# ✓ Lifecycle meta-arguments (create_before_destroy, ignore_changes, prevent_destroy)
# ✓ Terraform functions (toset())
# ✓ Edge cases (minimal config, complex conditionals)
# ✓ Boolean values (both true and false)
# ============================================================================
