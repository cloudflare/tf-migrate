# Comprehensive Zone Migration Integration Test
# Covers all Terraform patterns and zone-specific edge cases

# ========================================
# Pattern Group 1: Variables & Locals
# ========================================

locals {
  common_account = var.cloudflare_account_id
  name_prefix    = "integration-test"
  zone_types     = ["full", "partial", "secondary"]
  test_domains   = ["example.com", "test.com", "demo.com"]
}

# ========================================
# Pattern Group 2: Basic Zone Configurations
# ========================================

# Test Case 1: Minimal zone (only required fields)
resource "cloudflare_zone" "minimal" {
  account_id = var.cloudflare_account_id
  zone       = "minimal.example.com"
}

# Test Case 2: Full zone with all v4 attributes
resource "cloudflare_zone" "maximal" {
  account_id          = var.cloudflare_account_id
  zone                = "maximal.example.com"
  paused              = false
  type                = "full"
  jump_start          = true
  plan                = "enterprise"
  vanity_name_servers = ["ns1.custom.example.com", "ns2.custom.example.com"]
}

# Test Case 3: Zone with paused = true
resource "cloudflare_zone" "paused_zone" {
  account_id = var.cloudflare_account_id
  zone       = "paused.example.com"
  paused     = true
  type       = "full"
}

# Test Case 4: Partial zone type
resource "cloudflare_zone" "partial_zone" {
  account_id = var.cloudflare_account_id
  zone       = "partial.example.com"
  type       = "partial"
  paused     = false
}

# Test Case 5: Secondary zone type
resource "cloudflare_zone" "secondary_zone" {
  account_id = var.cloudflare_account_id
  zone       = "secondary.example.com"
  type       = "secondary"
}

# Test Case 6: Zone with vanity name servers
resource "cloudflare_zone" "with_vanity_ns" {
  account_id          = var.cloudflare_account_id
  zone                = "vanity.example.com"
  type                = "full"
  vanity_name_servers = ["ns1.vanity.example.com", "ns2.vanity.example.com", "ns3.vanity.example.com"]
}

# ========================================
# Pattern Group 3: for_each with Maps
# ========================================

resource "cloudflare_zone" "map_zones" {
  for_each = {
    "prod" = {
      domain = "prod.example.com"
      type   = "full"
      paused = false
    }
    "staging" = {
      domain = "staging.example.com"
      type   = "full"
      paused = false
    }
    "dev" = {
      domain = "dev.example.com"
      type   = "partial"
      paused = true
    }
  }

  account_id = var.cloudflare_account_id
  zone       = each.value.domain
  type       = each.value.type
  paused     = each.value.paused
}

# ========================================
# Pattern Group 4: for_each with Sets
# ========================================

resource "cloudflare_zone" "set_zones" {
  for_each = toset([
    "alpha.example.com",
    "beta.example.com",
    "gamma.example.com",
    "delta.example.com"
  ])

  account_id = var.cloudflare_account_id
  zone       = each.value
  type       = "full"
}

# ========================================
# Pattern Group 5: count-based Resources
# ========================================

resource "cloudflare_zone" "counted_zones" {
  count = 3

  account_id = var.cloudflare_account_id
  zone       = "zone-${count.index}.example.com"
  type       = "full"
  paused     = count.index == 1 ? true : false
}

# ========================================
# Pattern Group 6: Conditional Creation
# ========================================

locals {
  enable_test_zone    = true
  enable_feature_zone = false
}

resource "cloudflare_zone" "conditional_enabled" {
  count = local.enable_test_zone ? 1 : 0

  account_id = var.cloudflare_account_id
  zone       = "conditional-enabled.example.com"
  type       = "full"
}

resource "cloudflare_zone" "conditional_disabled" {
  count = local.enable_feature_zone ? 1 : 0

  account_id = var.cloudflare_account_id
  zone       = "conditional-disabled.example.com"
  type       = "full"
}

# ========================================
# Pattern Group 7: Terraform Functions
# ========================================

resource "cloudflare_zone" "with_functions" {
  account_id = var.cloudflare_account_id
  zone       = join("-", ["function", "test", "example.com"])
  type       = "full"
}

resource "cloudflare_zone" "with_interpolation" {
  account_id = var.cloudflare_account_id
  zone       = "${local.name_prefix}-interpolated.example.com"
  type       = "full"
  paused     = false
}

resource "cloudflare_zone" "with_locals" {
  account_id = local.common_account
  zone       = "with-locals.example.com"
  type       = local.zone_types[0]
}

# ========================================
# Pattern Group 8: Lifecycle Meta-Arguments
# ========================================

resource "cloudflare_zone" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  zone       = "lifecycle-test.example.com"
  type       = "full"
  paused     = false

  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_zone" "with_ignore_changes" {
  account_id = var.cloudflare_account_id
  zone       = "ignore-changes.example.com"
  type       = "full"

  lifecycle {
    ignore_changes = [paused]
  }
}

resource "cloudflare_zone" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  zone       = "prevent-destroy.example.com"
  type       = "full"

  lifecycle {
    prevent_destroy = false
  }
}

# ========================================
# Pattern Group 9: Edge Cases
# ========================================

# Test Case: Unicode domain name
resource "cloudflare_zone" "unicode_domain" {
  account_id = var.cloudflare_account_id
  zone       = "例え.テスト"
  type       = "full"
}

# Test Case: Hyphenated domain
resource "cloudflare_zone" "hyphenated_domain" {
  account_id = var.cloudflare_account_id
  zone       = "test-hyphen-domain.example.com"
  type       = "full"
}

# Test Case: Subdomain with many levels
resource "cloudflare_zone" "deep_subdomain" {
  account_id = var.cloudflare_account_id
  zone       = "level4.level3.level2.level1.example.com"
  type       = "full"
}

# Test Case: Zone with removed attributes (jump_start and plan)
resource "cloudflare_zone" "with_removed_attrs" {
  account_id = var.cloudflare_account_id
  zone       = "removed-attrs.example.com"
  type       = "full"
  jump_start = false
  plan       = "free"
}

# Test Case: Zone with all plan types
resource "cloudflare_zone" "with_pro_plan" {
  account_id = var.cloudflare_account_id
  zone       = "pro-plan.example.com"
  type       = "full"
  plan       = "pro"
}

resource "cloudflare_zone" "with_business_plan" {
  account_id = var.cloudflare_account_id
  zone       = "business-plan.example.com"
  type       = "full"
  plan       = "business"
}

# ========================================
# Total Resource Count Summary
# ========================================
# Minimal: 1
# Maximal: 1
# Basic variations: 5 (paused, partial, secondary, vanity_ns, removed_attrs)
# for_each with maps: 3 instances (prod, staging, dev)
# for_each with sets: 4 instances
# count-based: 3 instances
# Conditional: 1 instance (enabled only)
# With functions: 3 instances
# Lifecycle: 3 instances
# Edge cases: 5 instances (unicode, hyphenated, deep, plan variations)
# TOTAL: 29 resource instances
