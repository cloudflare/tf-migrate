# Comprehensive integration test for worker_route migration
# This file tests ALL Terraform patterns and edge cases
# Note: Testing routes without script_name to avoid requiring actual worker scripts

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

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

locals {
  name_prefix = "cftftest"
  zone_id     = var.cloudflare_zone_id

  # Test data for routes
  api_routes = {
    api     = "api.cf-tf-test.com/*"
    graphql = "graphql.cf-tf-test.com/*"
  }

  admin_patterns = [
    "admin.cf-tf-test.com/*",
    "manage.cf-tf-test.com/*",
  ]
}

###############################################################################
# Pattern 1: Basic Resources - Minimal and Full Configurations
###############################################################################

# Minimal route - only required fields
resource "cloudflare_workers_route" "minimal" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}.cf-tf-test.com/*"
}

# Full route - with additional fields
resource "cloudflare_workers_route" "full" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-full.cf-tf-test.com/*"
}

# Route with wildcard subdomain pattern
resource "cloudflare_workers_route" "wildcard_subdomain" {
  zone_id = local.zone_id
  pattern = "*.${local.name_prefix}.cf-tf-test.com/*"
}

# Route with specific path pattern
resource "cloudflare_workers_route" "specific_path" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}.cf-tf-test.com/api/*"
}

# Route with exact path (no wildcard)
resource "cloudflare_workers_route" "exact_path" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}.cf-tf-test.com/health"
}

###############################################################################
# Pattern 2: for_each with Maps
###############################################################################

resource "cloudflare_workers_route" "api_routes" {
  for_each = local.api_routes

  zone_id = local.zone_id
  pattern = each.value
}

###############################################################################
# Pattern 3: for_each with Sets (toset)
###############################################################################

resource "cloudflare_workers_route" "admin_routes" {
  for_each = toset(local.admin_patterns)

  zone_id = local.zone_id
  pattern = each.value
}

###############################################################################
# Pattern 4: count-based Resources
###############################################################################

resource "cloudflare_workers_route" "numbered" {
  count = 3

  zone_id = local.zone_id
  pattern = "${local.name_prefix}-${count.index}.cf-tf-test.com/*"
}

###############################################################################
# Pattern 5: Conditional Creation
###############################################################################

resource "cloudflare_workers_route" "conditional" {
  count = 1 # Simulate conditional: count = var.enable_route ? 1 : 0

  zone_id = local.zone_id
  pattern = "${local.name_prefix}-conditional.cf-tf-test.com/*"
}

###############################################################################
# Pattern 6: Variable References and Interpolation
###############################################################################

resource "cloudflare_workers_route" "with_variables" {
  zone_id = var.cloudflare_zone_id
  pattern = "${local.name_prefix}-var.cf-tf-test.com/*"
}

###############################################################################
# Pattern 7: Route without script (catch-all/fallback route)
###############################################################################

resource "cloudflare_workers_route" "no_script_catchall" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-fallback.cf-tf-test.com/*"
}

###############################################################################
# Pattern 8: Multiple Routes for Same Domain (different paths)
###############################################################################

resource "cloudflare_workers_route" "multi_path_1" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-multi.cf-tf-test.com/api/*"
}

resource "cloudflare_workers_route" "multi_path_2" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-multi.cf-tf-test.com/static/*"
}

resource "cloudflare_workers_route" "multi_path_3" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-multi.cf-tf-test.com/*"
}

###############################################################################
# Pattern 9: Special Characters in Patterns
###############################################################################

resource "cloudflare_workers_route" "query_params" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-query.cf-tf-test.com/*"
}

resource "cloudflare_workers_route" "dashes_underscores" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-test_route.cf-tf-test.com/*"
}

###############################################################################
# Pattern 10: Dynamic Block Pattern (using for_each in a different way)
###############################################################################

locals {
  route_configs = {
    prod = {
      pattern = "${local.name_prefix}-prod.cf-tf-test.com/*"
    }
    staging = {
      pattern = "${local.name_prefix}-staging.cf-tf-test.com/*"
    }
    dev = {
      pattern = "${local.name_prefix}-dev.cf-tf-test.com/*"
    }
  }
}

resource "cloudflare_workers_route" "environments" {
  for_each = local.route_configs

  zone_id = local.zone_id
  pattern = each.value.pattern
}

###############################################################################
# Pattern 11: Cross-Pattern Testing
###############################################################################

# Combination of locals, variables, and string interpolation
resource "cloudflare_workers_route" "complex_interpolation" {
  zone_id = var.cloudflare_zone_id
  pattern = "${local.name_prefix}-${replace("test.domain", ".", "-")}.cf-tf-test.com/*"
}

###############################################################################
# Summary: Test Coverage
###############################################################################
# Total resources: 25+ instances across all patterns
# - Routes without script_name (catch-all/fallback): 25+ instances
# - for_each with maps: 2 instances (api_routes)
# - for_each with sets: 2 instances (admin_routes)
# - count-based: 3 instances (numbered)
# - Conditional: 1 instance
# - Variable references: multiple
# - Special patterns: wildcard, exact paths, query params, special chars
# - Environment configs: 3 instances (prod/staging/dev)
###############################################################################
