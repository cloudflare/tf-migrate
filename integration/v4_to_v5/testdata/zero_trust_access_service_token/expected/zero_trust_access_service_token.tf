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

# ============================================================================
# Pattern Group 1: Variables & Locals
# ============================================================================

locals {
  common_account        = var.cloudflare_account_id
  name_prefix           = "cftftest"
  tags                  = ["test", "migration", "v4_to_v5"]
  token_duration        = "8760h"
  enable_long_duration  = true
  enable_short_duration = false
}

# ============================================================================
# Pattern Group 2: for_each with Maps (5 resources)
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "map_example" {
  for_each = {
    "prod" = {
      account_id           = var.cloudflare_account_id
      name                 = "${local.name_prefix}-prod-service-token"
      duration             = "8760h"
      min_days_for_renewal = 30
    }
    "staging" = {
      account_id           = var.cloudflare_account_id
      name                 = "${local.name_prefix}-staging-service-token"
      duration             = "8760h"
      min_days_for_renewal = 15
    }
    "dev" = {
      account_id           = var.cloudflare_account_id
      name                 = "${local.name_prefix}-dev-service-token"
      duration             = "8760h"
      min_days_for_renewal = 7
    }
    "qa" = {
      account_id           = var.cloudflare_account_id
      name                 = "${local.name_prefix}-qa-service-token"
      duration             = "8760h"
      min_days_for_renewal = 10
    }
    "perf" = {
      account_id           = var.cloudflare_account_id
      name                 = "${local.name_prefix}-perf-service-token"
      duration             = "8760h"
      min_days_for_renewal = 5
    }
  }

  account_id = each.value.account_id
  name       = each.value.name
  duration   = each.value.duration
}

# ============================================================================
# Pattern Group 3: for_each with Sets (4 items)
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-set-${each.value}-token"
  duration   = local.token_duration
}

# ============================================================================
# Pattern Group 4: count-based Resources (3 instances)
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "counted" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-counted-token-${count.index}"
  duration   = "8760h"
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "conditional_enabled" {
  count = local.enable_long_duration ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-long-duration"
  duration   = "17520h" # 2 years
}

resource "cloudflare_zero_trust_access_service_token" "conditional_disabled" {
  count = local.enable_short_duration ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-short-duration"
  duration   = "8760h" # Changed from 720h to valid value
}

# ============================================================================
# Pattern Group 7: Terraform Functions
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "with_functions" {
  account_id = local.common_account

  # join() function
  name = join("-", [local.name_prefix, "function", "example"])

  # String interpolation
  duration = local.token_duration

}

resource "cloudflare_zero_trust_access_service_token" "with_interpolation" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-token-for-account-${var.cloudflare_account_id}"
  duration   = "8760h"
}

# ============================================================================
# Pattern Group 8: Lifecycle Meta-Arguments
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-test-token"
  duration   = "8760h"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [duration]
  }
}

resource "cloudflare_zero_trust_access_service_token" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-prevent-destroy-token"
  duration   = "8760h"

  lifecycle {
    prevent_destroy = false # Set to false for testing
  }
}

# ============================================================================
# Pattern Group 9: Edge Cases
# ============================================================================

# Minimal resource (only required fields)
resource "cloudflare_zero_trust_access_service_token" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal-token"
}

# Maximal resource (all fields populated)
resource "cloudflare_zero_trust_access_service_token" "maximal" {
  account_id                        = var.cloudflare_account_id
  name                              = "${local.name_prefix}-maximal-token"
  duration                          = "8760h"
  client_secret_version             = 5
  previous_client_secret_expires_at = "2025-12-31T23:59:59Z"
}

# Zero values
resource "cloudflare_zero_trust_access_service_token" "zero_values" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-zero-values-token"
  duration   = "8760h"
}


# Resource without min_days_for_renewal (deprecated field)
resource "cloudflare_zero_trust_access_service_token" "without_deprecated" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-no-deprecated-field"
  duration   = "8760h"
}

# Resource with very large client_secret_version
resource "cloudflare_zero_trust_access_service_token" "large_version" {
  account_id                        = var.cloudflare_account_id
  name                              = "${local.name_prefix}-large-version-token"
  duration                          = "8760h"
  client_secret_version             = 999
  previous_client_secret_expires_at = "2025-12-31T23:59:59Z"
}

# Resource with special characters in name
resource "cloudflare_zero_trust_access_service_token" "special_chars" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-token-with-special_chars-123"
  duration   = "8760h"
}

# Resource with minimum duration
resource "cloudflare_zero_trust_access_service_token" "min_duration" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-min-duration-token"
  duration   = "8760h" # Changed from 30m to valid minimum value
}

# Resource with maximum duration
resource "cloudflare_zero_trust_access_service_token" "max_duration" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-max-duration-token"
  duration   = "87600h" # 10 years
}

# ============================================================================
# Pattern 9: Cross-resource reference using both v4 names
# ============================================================================
# This validates that GetResourceRename() returns ALL v4 names for cross-file reference updates
# v4 name option 1: cloudflare_access_service_token
# v4 name option 2: cloudflare_zero_trust_access_service_token
# v5 name: cloudflare_zero_trust_access_service_token


# Resource using v4 name option 2
resource "cloudflare_zero_trust_access_service_token" "resourcename_opt2" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-pattern9-opt2-token"
  duration   = "8760h"
}

# Dependent resource that references option 1 via depends_on
resource "cloudflare_zero_trust_access_application" "ref_opt1" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} App depends on token opt1"
  domain     = "token-opt1.${var.cloudflare_domain}"
  type       = "self_hosted"

  depends_on                 = [cloudflare_zero_trust_access_service_token.resourcename_opt1]
  http_only_cookie_attribute = "false"
}

# Dependent resource that references option 2 via depends_on
resource "cloudflare_zero_trust_access_application" "ref_opt2" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} App depends on token opt2"
  domain     = "token-opt2.${var.cloudflare_domain}"
  type       = "self_hosted"

  depends_on                 = [cloudflare_zero_trust_access_service_token.resourcename_opt2]
  http_only_cookie_attribute = "false"
}

# Legacy resource name (cloudflare_access_service_token - deprecated)
resource "cloudflare_zero_trust_access_service_token" "legacy_name" {
  account_id                        = var.cloudflare_account_id
  name                              = "${local.name_prefix}-legacy-name-token"
  duration                          = "8760h"
  client_secret_version             = 2
  previous_client_secret_expires_at = "2024-12-31T23:59:59Z"
}

moved {
  from = cloudflare_access_service_token.legacy_name
  to   = cloudflare_zero_trust_access_service_token.legacy_name
}

# Resource using v4 name option 1
resource "cloudflare_zero_trust_access_service_token" "resourcename_opt1" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-pattern9-opt1-token"
  duration   = "8760h"
}

moved {
  from = cloudflare_access_service_token.resourcename_opt1
  to   = cloudflare_zero_trust_access_service_token.resourcename_opt1
}
