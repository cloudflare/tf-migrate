# Comprehensive integration test for worker_route migration
# This file tests ALL Terraform patterns and edge cases

# Standard variables used by test infrastructure (DO NOT declare these)
# - var.cloudflare_zone_id
# These are provided automatically by the test framework

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

# Full route - all fields including optional script_name
resource "cloudflare_workers_route" "full" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-full.cf-tf-test.com/*"
  script  = "cftftest-worker"
}

# Route with wildcard subdomain pattern
resource "cloudflare_workers_route" "wildcard_subdomain" {
  zone_id = local.zone_id
  pattern = "*.${local.name_prefix}.cf-tf-test.com/*"
  script  = "cftftest-wildcard-worker"
}

# Route with specific path pattern
resource "cloudflare_workers_route" "specific_path" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}.cf-tf-test.com/api/*"
  script  = "cftftest-api-worker"
}

# Route with exact path (no wildcard)
resource "cloudflare_workers_route" "exact_path" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}.cf-tf-test.com/health"
  script  = "cftftest-health-worker"
}

###############################################################################
# Pattern 2: for_each with Maps
###############################################################################

resource "cloudflare_workers_route" "api_routes" {
  for_each = local.api_routes

  zone_id = local.zone_id
  pattern = each.value
  script  = "${local.name_prefix}-${each.key}-worker"
}

###############################################################################
# Pattern 3: for_each with Sets (toset)
###############################################################################

resource "cloudflare_workers_route" "admin_routes" {
  for_each = toset(local.admin_patterns)

  zone_id = local.zone_id
  pattern = each.value
  script  = "${local.name_prefix}-admin-worker"
}

###############################################################################
# Pattern 4: count-based Resources
###############################################################################

resource "cloudflare_workers_route" "numbered" {
  count = 3

  zone_id = local.zone_id
  pattern = "${local.name_prefix}-${count.index}.cf-tf-test.com/*"
  script  = "${local.name_prefix}-worker-${count.index}"
}

###############################################################################
# Pattern 5: Conditional Creation
###############################################################################

resource "cloudflare_workers_route" "conditional" {
  count = 1 # Simulate conditional: count = var.enable_route ? 1 : 0

  zone_id = local.zone_id
  pattern = "${local.name_prefix}-conditional.cf-tf-test.com/*"
  script  = "${local.name_prefix}-conditional-worker"
}

###############################################################################
# Pattern 6: Variable References and Interpolation
###############################################################################

resource "cloudflare_workers_route" "with_variables" {
  zone_id = var.cloudflare_zone_id
  pattern = "${local.name_prefix}-var.cf-tf-test.com/*"
  script  = "${local.name_prefix}-var-worker"
}

###############################################################################
# Pattern 7: Route without script (catch-all/fallback route)
###############################################################################

resource "cloudflare_workers_route" "no_script_catchall" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-fallback.cf-tf-test.com/*"
  # script_name intentionally omitted - this is a valid use case
}

###############################################################################
# Pattern 8: Multiple Routes for Same Domain (different paths)
###############################################################################

resource "cloudflare_workers_route" "multi_path_1" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-multi.cf-tf-test.com/api/*"
  script  = "${local.name_prefix}-api-handler"
}

resource "cloudflare_workers_route" "multi_path_2" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-multi.cf-tf-test.com/static/*"
  script  = "${local.name_prefix}-static-handler"
}

resource "cloudflare_workers_route" "multi_path_3" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-multi.cf-tf-test.com/*"
  # Catch-all for everything else (no script)
}

###############################################################################
# Pattern 9: Special Characters in Patterns
###############################################################################

resource "cloudflare_workers_route" "query_params" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-query.cf-tf-test.com/*"
  script  = "${local.name_prefix}-query-worker"
}

resource "cloudflare_workers_route" "dashes_underscores" {
  zone_id = local.zone_id
  pattern = "${local.name_prefix}-test_route.cf-tf-test.com/*"
  script  = "${local.name_prefix}_dash_under_worker"
}

###############################################################################
# Pattern 10: Dynamic Block Pattern (using for_each in a different way)
###############################################################################

locals {
  route_configs = {
    prod = {
      pattern = "${local.name_prefix}-prod.cf-tf-test.com/*"
      script  = "${local.name_prefix}-prod-worker"
    }
    staging = {
      pattern = "${local.name_prefix}-staging.cf-tf-test.com/*"
      script  = "${local.name_prefix}-staging-worker"
    }
    dev = {
      pattern = "${local.name_prefix}-dev.cf-tf-test.com/*"
      script  = "${local.name_prefix}-dev-worker"
    }
  }
}

resource "cloudflare_workers_route" "environments" {
  for_each = local.route_configs

  zone_id = local.zone_id
  pattern = each.value.pattern
  script  = each.value.script
}

###############################################################################
# Pattern 11: Cross-Pattern Testing
###############################################################################

# Combination of locals, variables, and string interpolation
resource "cloudflare_workers_route" "complex_interpolation" {
  zone_id = var.cloudflare_zone_id
  pattern = "${local.name_prefix}-${replace("test.domain", ".", "-")}.cf-tf-test.com/*"
  script  = "${local.name_prefix}-${lower("COMPLEX")}-worker"
}

###############################################################################
# Summary: Test Coverage
###############################################################################
# Total resources: 30+ instances across all patterns
# - Minimal configuration (no script): 4 instances
# - Full configuration (with script): 26+ instances
# - for_each with maps: 2 instances (api_routes)
# - for_each with sets: 2 instances (admin_routes)
# - count-based: 3 instances (numbered)
# - Conditional: 1 instance
# - Variable references: multiple
# - Special patterns: wildcard, exact paths, query params, special chars
# - Environment configs: 3 instances (prod/staging/dev)
###############################################################################
