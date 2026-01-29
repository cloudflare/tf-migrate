# Comprehensive integration test for cloudflare_leaked_credential_check v4→v5 migration
# This file tests all Terraform patterns and edge cases

# Variables for testing
variable "zone_ids" {
  type = list(string)
  default = [
    "0da42c8d2132a9ddaf714f9e7c920711",
    "1ea53b5d3134b8eebf825e9f8d921712",
    "2fa64c6e4245c9ffcg936faf9e032823"
  ]
}

variable "enable_checks" {
  type    = bool
  default = true
}

variable "environment" {
  type    = string
  default = "production"
}

# Locals for DRY
locals {
  name_prefix = "cftftest"

  zone_map = {
    prod    = "0da42c8d2132a9ddaf714f9e7c920711"
    staging = "1ea53b5d3134b8eebf825e9f8d921712"
    dev     = "2fa64c6e4245c9ffcg936faf9e032823"
  }

  zones_to_protect = toset([
    "0da42c8d2132a9ddaf714f9e7c920711",
    "1ea53b5d3134b8eebf825e9f8d921712",
    "2fa64c6e4245c9ffcg936faf9e032823",
    "3gb75d7f5356daggdh047gbg0f143934"
  ])
}

# ============================================================================
# Basic Resources (instances 1-4)
# ============================================================================

# 1. Basic resource with enabled=true
resource "cloudflare_leaked_credential_check" "basic_enabled" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}

# 2. Basic resource with enabled=false
resource "cloudflare_leaked_credential_check" "basic_disabled" {
  zone_id = "1ea53b5d3134b8eebf825e9f8d921712"
  enabled = false
}

# 3. Resource with variable reference for zone_id
resource "cloudflare_leaked_credential_check" "with_var_zone" {
  zone_id = var.zone_ids[0]
  enabled = true
}

# 4. Resource with variable reference for enabled
resource "cloudflare_leaked_credential_check" "with_var_enabled" {
  zone_id = "2fa64c6e4245c9ffcg936faf9e032823"
  enabled = var.enable_checks
}

# ============================================================================
# count-based Resources (instances 5-7)
# ============================================================================

# 5-7. Multiple instances using count
resource "cloudflare_leaked_credential_check" "count_based" {
  count   = 3
  zone_id = var.zone_ids[count.index]
  enabled = count.index < 2 ? true : false
}

# ============================================================================
# for_each with Map (instances 8-10)
# ============================================================================

# 8-10. Resources using for_each with map
resource "cloudflare_leaked_credential_check" "for_each_map" {
  for_each = local.zone_map

  zone_id = each.value
  enabled = each.key == "prod" ? true : false
}

# ============================================================================
# for_each with Set (instances 11-14)
# ============================================================================

# 11-14. Resources using for_each with set
resource "cloudflare_leaked_credential_check" "for_each_set" {
  for_each = local.zones_to_protect

  zone_id = each.value
  enabled = true
}

# ============================================================================
# Conditional Resources (instances 15-16)
# ============================================================================

# 15. Conditionally created resource (production only)
resource "cloudflare_leaked_credential_check" "conditional_prod" {
  count   = var.environment == "production" ? 1 : 0
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}

# 16. Conditionally created resource (non-production)
resource "cloudflare_leaked_credential_check" "conditional_nonprod" {
  count   = var.environment != "production" ? 1 : 0
  zone_id = "1ea53b5d3134b8eebf825e9f8d921712"
  enabled = false
}

# ============================================================================
# Resources with Lifecycle Meta-Arguments (instances 17-18)
# ============================================================================

# 17. Resource with create_before_destroy
resource "cloudflare_leaked_credential_check" "with_lifecycle_cbd" {
  zone_id = "3gb75d7f5356daggdh047gbg0f143934"
  enabled = true

  lifecycle {
    create_before_destroy = true
  }
}

# 18. Resource with prevent_destroy and ignore_changes
resource "cloudflare_leaked_credential_check" "with_lifecycle_complex" {
  zone_id = "4hc86e8g6467ebhhe158hch1g254045"
  enabled = true

  lifecycle {
    prevent_destroy = false
    ignore_changes  = [enabled]
  }
}

# ============================================================================
# Resources with depends_on (instances 19-20)
# ============================================================================

# 19. Resource with depends_on referencing another check
resource "cloudflare_leaked_credential_check" "with_depends" {
  zone_id = "5id97f9h7578fciif269idi2h365156"
  enabled = true

  depends_on = [
    cloudflare_leaked_credential_check.basic_enabled
  ]
}

# 20. Resource with multiple dependencies
resource "cloudflare_leaked_credential_check" "with_multi_depends" {
  zone_id = "6je08g0i8689gdjjg370jej3i476267"
  enabled = false

  depends_on = [
    cloudflare_leaked_credential_check.basic_enabled,
    cloudflare_leaked_credential_check.basic_disabled
  ]
}

# ============================================================================
# Complex Expression Resources (instances 21-23)
# ============================================================================

# 21. Resource with conditional expression for enabled
resource "cloudflare_leaked_credential_check" "conditional_expr" {
  zone_id = "7kf19h1j9790hekkg481kfk4j587378"
  enabled = var.environment == "production" ? true : false
}

# 22. Resource with local reference
resource "cloudflare_leaked_credential_check" "with_local" {
  zone_id = local.zone_map["prod"]
  enabled = true
}

# 23. Resource with string interpolation
resource "cloudflare_leaked_credential_check" "with_interpolation" {
  zone_id = var.zone_ids[0]
  enabled = var.enable_checks

  # Comment showing interpolation pattern
  # name = "${local.name_prefix}-check"  # (no name field, just for pattern demo)
}

# ============================================================================
# Edge Cases (instances 24-25)
# ============================================================================

# 24. Resource with comments
# This resource enables credential checking for the main production zone
resource "cloudflare_leaked_credential_check" "with_comments" {
  # Zone identifier for production
  zone_id = "8lg20i2k0801iflh592lgl5k698489"
  enabled = true # Enable credential leak detection
}

# 25. Minimal resource (all required fields only)
resource "cloudflare_leaked_credential_check" "minimal" {
  zone_id = "9mh31j3l1912jgmm603mhm6l709590"
  enabled = true
}

# ============================================================================
# Total Instances: 25
# - Basic: 4
# - count-based: 3
# - for_each (map): 3
# - for_each (set): 4
# - Conditional: 2
# - Lifecycle: 2
# - depends_on: 2
# - Complex expressions: 3
# - Edge cases: 2
# ============================================================================
