# Comprehensive Zone Migration Integration Test
# Covers all Terraform patterns and zone-specific edge cases

# ========================================
# Variables
# ========================================
variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# ========================================
# Pattern Group 1: Locals
# ========================================

locals {
  common_account = var.cloudflare_account_id
  name_prefix    = "cftftest"
  zone_types     = ["full", "partial", "secondary"]
  test_domains   = ["cf-tf-test.com", "test.com", "demo.com"]
}

# ========================================
# Pattern Group 2: Basic Zone Configurations
# ========================================

# Test Case 1: Minimal zone (only required fields)
resource "cloudflare_zone" "minimal" {
  name = "${local.name_prefix}-minimal.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case 2: Zone with paused = true
resource "cloudflare_zone" "paused_zone" {
  paused = true
  type   = "full"
  name   = "${local.name_prefix}-paused.cf-tf-test.com"
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
      domain = "cftftest-prod.cf-tf-test.com"
      type   = "full"
      paused = false
    }
    "staging" = {
      domain = "cftftest-staging.cf-tf-test.com"
      type   = "full"
      paused = false
    }
    # "dev" = {
    #   domain = "cftftest-dev.cf-tf-test.com"
    #   type   = "partial"
    #   paused = true
    # }
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
    "cftftest-alpha.cf-tf-test.com",
    "cftftest-beta.cf-tf-test.com",
    "cftftest-gamma.cf-tf-test.com",
    "cftftest-delta.cf-tf-test.com"
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
  name   = "${local.name_prefix}-zone-${count.index}.cf-tf-test.com"
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
  name = "${local.name_prefix}-conditional-enabled.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "conditional_disabled" {
  count = local.enable_feature_zone ? 1 : 0

  type = "full"
  name = "${local.name_prefix}-conditional-disabled.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# ========================================
# Pattern Group 7: Terraform Functions
# ========================================

resource "cloudflare_zone" "with_functions" {
  type = "full"
  name = join("-", ["cftftest", "function", "test.cf-tf-test.com"])
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_interpolation" {
  type   = "full"
  paused = false
  name   = "${local.name_prefix}-interpolated.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_locals" {
  type = local.zone_types[0]
  name = "${local.name_prefix}-with-locals.cf-tf-test.com"
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
  name = "${local.name_prefix}-lifecycle-test.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_ignore_changes" {
  type = "full"

  lifecycle {
    ignore_changes = [paused]
  }
  name = "${local.name_prefix}-ignore-changes.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_prevent_destroy" {
  type = "full"

  lifecycle {
    prevent_destroy = false
  }
  name = "${local.name_prefix}-prevent-destroy.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# ========================================
# Pattern Group 9: Edge Cases
# ========================================

# Test Case: Hyphenated domain
resource "cloudflare_zone" "hyphenated_domain" {
  type = "full"
  name = "${local.name_prefix}-hyphen-domain.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case: Subdomain with many levels
resource "cloudflare_zone" "deep_subdomain" {
  type = "full"
  name = "${local.name_prefix}-level4.level3.level2.level1.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case: Zone with removed attributes (jump_start and plan)
resource "cloudflare_zone" "with_removed_attrs" {
  type = "full"
  name = "${local.name_prefix}-removed-attrs.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# Test Case: Zone with all plan types
resource "cloudflare_zone" "with_pro_plan" {
  type = "full"
  name = "${local.name_prefix}-pro-plan.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_zone" "with_business_plan" {
  type = "full"
  name = "${local.name_prefix}-business-plan.cf-tf-test.com"
  account = {
    id = var.cloudflare_account_id
  }
}

# ========================================
# Total Resource Count Summary
# ========================================
# Minimal: 1
# Basic variations: 2 (paused, removed_attrs)
# for_each with maps: 2 instances (prod, staging)
# for_each with sets: 4 instances
# count-based: 3 instances
# Conditional: 1 instance (enabled only)
# With functions: 3 instances
# Lifecycle: 3 instances
# Edge cases: 4 instances (hyphenated, deep, plan variations)
# TOTAL: 23 resource instances
#
# REMOVED (require Business/Enterprise plans):
# - maximal (vanity name servers)
# - partial_zone (partial zone type)
# - secondary_zone (secondary zone type)
# - with_vanity_ns (vanity name servers)
# - map_zones["dev"] (partial zone type)
# - unicode_domain (commented out by linter)
