# Terraform Provider v4 to v5 Migration Test
# Resource: cloudflare_regional_tiered_cache
# Test Coverage: All Terraform patterns for simple zone-level settings

# ============================================================================
# Standard Variables (Auto-provided by test infrastructure)
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

# Locals for common values
locals {
  test_zones = {
    zone1 = var.cloudflare_zone_id
    zone2 = var.cloudflare_zone_id
    zone3 = var.cloudflare_zone_id
  }
  name_prefix        = "cftftest"
  enable_cache       = true
  enable_experimental = false
}

# Pattern 1: Basic resource with value="on"
resource "cloudflare_regional_tiered_cache" "basic_on" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}

# Pattern 2: Basic resource with value="off"
resource "cloudflare_regional_tiered_cache" "basic_off" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

# Pattern 3: for_each with map (3 instances)
resource "cloudflare_regional_tiered_cache" "for_each_map" {
  for_each = {
    "enabled" = {
      zone_id = var.cloudflare_zone_id
      value   = "on"
    }
    "disabled" = {
      zone_id = var.cloudflare_zone_id
      value   = "off"
    }
    "testing" = {
      zone_id = var.cloudflare_zone_id
      value   = "on"
    }
  }

  zone_id = each.value.zone_id
  value   = each.value.value
}

# Pattern 4: for_each with set using toset() (4 instances)
resource "cloudflare_regional_tiered_cache" "for_each_set" {
  for_each = toset(["alpha", "beta", "gamma", "delta"])

  zone_id = var.cloudflare_zone_id
  value   = "on"
}

# Pattern 5: count-based resources (3 instances)
resource "cloudflare_regional_tiered_cache" "counted" {
  count = 3

  zone_id = var.cloudflare_zone_id
  value   = count.index == 0 ? "on" : "off"
}

# Pattern 6: Conditional creation - enabled (1 instance created)
resource "cloudflare_regional_tiered_cache" "conditional_enabled" {
  count = local.enable_cache ? 1 : 0

  zone_id = var.cloudflare_zone_id
  value   = "on"
}

# Pattern 7: Conditional creation - disabled (0 instances created)
resource "cloudflare_regional_tiered_cache" "conditional_disabled" {
  count = local.enable_experimental ? 1 : 0

  zone_id = var.cloudflare_zone_id
  value   = "off"
}

# Pattern 8: Using terraform functions
resource "cloudflare_regional_tiered_cache" "with_functions" {
  zone_id = var.cloudflare_zone_id
  value   = join("", ["o", "n"]) # Results in "on"
}

# Pattern 9: String interpolation
resource "cloudflare_regional_tiered_cache" "with_interpolation" {
  zone_id = "${var.cloudflare_zone_id}"
  value   = "on"
}

# Pattern 10: Lifecycle meta-arguments
resource "cloudflare_regional_tiered_cache" "with_lifecycle" {
  zone_id = var.cloudflare_zone_id
  value   = "on"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [value]
  }
}

# Pattern 11: Prevent destroy lifecycle
resource "cloudflare_regional_tiered_cache" "with_prevent_destroy" {
  zone_id = var.cloudflare_zone_id
  value   = "off"

  lifecycle {
    prevent_destroy = false # Set to false for testing
  }
}

# Pattern 12: Using local reference
resource "cloudflare_regional_tiered_cache" "from_local" {
  zone_id = local.test_zones["zone1"]
  value   = "on"
}

# Pattern 13: Mixed patterns - for_each with conditional value
resource "cloudflare_regional_tiered_cache" "mixed_pattern" {
  for_each = local.test_zones

  zone_id = each.value
  value   = each.key == "zone1" ? "on" : "off"
}

# Total resource instances in this test:
# basic_on: 1
# basic_off: 1
# for_each_map: 3
# for_each_set: 4
# counted: 3
# conditional_enabled: 1
# conditional_disabled: 0
# with_functions: 1
# with_interpolation: 1
# with_lifecycle: 1
# with_prevent_destroy: 1
# from_local: 1
# mixed_pattern: 3
# TOTAL: 21 instances
