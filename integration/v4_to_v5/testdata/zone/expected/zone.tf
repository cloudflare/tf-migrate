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
  name = "minimal.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case 2: Full zone with all v4 attributes
resource "cloudflare_zone" "maximal" {
  paused              = false
  type                = "full"
  vanity_name_servers = ["ns1.custom.example.com", "ns2.custom.example.com"]
  name                = "maximal.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case 3: Zone with paused = true
resource "cloudflare_zone" "paused_zone" {
  paused = true
  type   = "full"
  name   = "paused.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case 4: Partial zone type
resource "cloudflare_zone" "partial_zone" {
  type   = "partial"
  paused = false
  name   = "partial.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case 5: Secondary zone type
resource "cloudflare_zone" "secondary_zone" {
  type = "secondary"
  name = "secondary.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case 6: Zone with vanity name servers
resource "cloudflare_zone" "with_vanity_ns" {
  type                = "full"
  vanity_name_servers = ["ns1.vanity.example.com", "ns2.vanity.example.com", "ns3.vanity.example.com"]
  name                = "vanity.example.com"
  account = {
    id = var.cloudflare_account_id
  }
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

  type   = each.value.type
  paused = each.value.paused
  name   = each.value.domain
  account = {
    id = var.cloudflare_account_id
  }
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

  type = "full"
  name = each.value
  account = {
    id = var.cloudflare_account_id
  }
}

# ========================================
# Pattern Group 5: count-based Resources
# ========================================

resource "cloudflare_zone" "counted_zones" {
  count = 3

  type   = "full"
  paused = count.index == 1 ? true : false
  name   = "zone-${count.index}.example.com"
  account = {
    id = var.cloudflare_account_id
  }
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

  type = "full"
  name = "conditional-enabled.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "conditional_disabled" {
  count = local.enable_feature_zone ? 1 : 0

  type = "full"
  name = "conditional-disabled.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# ========================================
# Pattern Group 7: Terraform Functions
# ========================================

resource "cloudflare_zone" "with_functions" {
  type = "full"
  name = join("-", ["function", "test", "example.com"])
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_interpolation" {
  type   = "full"
  paused = false
  name   = "${local.name_prefix}-interpolated.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_locals" {
  type = local.zone_types[0]
  name = "with-locals.example.com"
  account = {
    id = local.common_account
  }
}

# ========================================
# Pattern Group 8: Lifecycle Meta-Arguments
# ========================================

resource "cloudflare_zone" "with_lifecycle" {
  type   = "full"
  paused = false

  lifecycle {
    create_before_destroy = true
  }
  name = "lifecycle-test.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_ignore_changes" {
  type = "full"

  lifecycle {
    ignore_changes = [paused]
  }
  name = "ignore-changes.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_prevent_destroy" {
  type = "full"

  lifecycle {
    prevent_destroy = false
  }
  name = "prevent-destroy.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# ========================================
# Pattern Group 9: Edge Cases
# ========================================

# Test Case: Unicode domain name
resource "cloudflare_zone" "unicode_domain" {
  type = "full"
  name = "例え.テスト"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case: Hyphenated domain
resource "cloudflare_zone" "hyphenated_domain" {
  type = "full"
  name = "test-hyphen-domain.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case: Subdomain with many levels
resource "cloudflare_zone" "deep_subdomain" {
  type = "full"
  name = "level4.level3.level2.level1.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case: Zone with removed attributes (jump_start and plan)
resource "cloudflare_zone" "with_removed_attrs" {
  type = "full"
  name = "removed-attrs.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case: Zone with all plan types
resource "cloudflare_zone" "with_pro_plan" {
  type = "full"
  name = "pro-plan.example.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_business_plan" {
  type = "full"
  name = "business-plan.example.com"
  account = {
    id = var.cloudflare_account_id
  }
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
