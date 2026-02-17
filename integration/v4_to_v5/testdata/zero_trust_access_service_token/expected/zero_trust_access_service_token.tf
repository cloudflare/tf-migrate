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


# ============================================================================
# Pattern Group 3: for_each with Sets (4 items)
# ============================================================================


# ============================================================================
# Pattern Group 4: count-based Resources (3 instances)
# ============================================================================


# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================



# ============================================================================
# Pattern Group 7: Terraform Functions
# ============================================================================



# ============================================================================
# Pattern Group 8: Lifecycle Meta-Arguments
# ============================================================================



# ============================================================================
# Pattern Group 9: Edge Cases
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

moved {
  from = cloudflare_access_service_token.map_example
  to   = cloudflare_zero_trust_access_service_token.map_example
}

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

moved {
  from = cloudflare_access_service_token.set_example
  to   = cloudflare_zero_trust_access_service_token.set_example
}

resource "cloudflare_zero_trust_access_service_token" "counted" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-counted-token-${count.index}"
  duration   = "8760h"
}

moved {
  from = cloudflare_access_service_token.counted
  to   = cloudflare_zero_trust_access_service_token.counted
}

resource "cloudflare_zero_trust_access_service_token" "conditional_enabled" {
  count = local.enable_long_duration ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-long-duration"
  duration   = "17520h" # 2 years
}

moved {
  from = cloudflare_access_service_token.conditional_enabled
  to   = cloudflare_zero_trust_access_service_token.conditional_enabled
}

resource "cloudflare_zero_trust_access_service_token" "conditional_disabled" {
  count = local.enable_short_duration ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-short-duration"
  duration   = "8760h" # Changed from 720h to valid value
}

moved {
  from = cloudflare_access_service_token.conditional_disabled
  to   = cloudflare_zero_trust_access_service_token.conditional_disabled
}

resource "cloudflare_zero_trust_access_service_token" "with_functions" {
  account_id = local.common_account

  # join() function
  name = join("-", [local.name_prefix, "function", "example"])

  # String interpolation
  duration = "${local.token_duration}"

}

moved {
  from = cloudflare_access_service_token.with_functions
  to   = cloudflare_zero_trust_access_service_token.with_functions
}

resource "cloudflare_zero_trust_access_service_token" "with_interpolation" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-token-for-account-${var.cloudflare_account_id}"
  duration   = "8760h"
}

moved {
  from = cloudflare_access_service_token.with_interpolation
  to   = cloudflare_zero_trust_access_service_token.with_interpolation
}

resource "cloudflare_zero_trust_access_service_token" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-test-token"
  duration   = "8760h"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [duration]
  }
}

moved {
  from = cloudflare_access_service_token.with_lifecycle
  to   = cloudflare_zero_trust_access_service_token.with_lifecycle
}

resource "cloudflare_zero_trust_access_service_token" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-prevent-destroy-token"
  duration   = "8760h"

  lifecycle {
    prevent_destroy = false # Set to false for testing
  }
}

moved {
  from = cloudflare_access_service_token.with_prevent_destroy
  to   = cloudflare_zero_trust_access_service_token.with_prevent_destroy
}

# Minimal resource (only required fields)
resource "cloudflare_zero_trust_access_service_token" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal-token"
}

moved {
  from = cloudflare_access_service_token.minimal
  to   = cloudflare_zero_trust_access_service_token.minimal
}

# Maximal resource (all fields populated)
resource "cloudflare_zero_trust_access_service_token" "maximal" {
  account_id                        = var.cloudflare_account_id
  name                              = "${local.name_prefix}-maximal-token"
  duration                          = "8760h"
  client_secret_version             = 5
  previous_client_secret_expires_at = "2025-12-31T23:59:59Z"
}

moved {
  from = cloudflare_access_service_token.maximal
  to   = cloudflare_zero_trust_access_service_token.maximal
}

# Zero values
resource "cloudflare_zero_trust_access_service_token" "zero_values" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-zero-values-token"
  duration   = "8760h"
}

moved {
  from = cloudflare_access_service_token.zero_values
  to   = cloudflare_zero_trust_access_service_token.zero_values
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

# Resource without min_days_for_renewal (deprecated field)
resource "cloudflare_zero_trust_access_service_token" "without_deprecated" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-no-deprecated-field"
  duration   = "8760h"
}

moved {
  from = cloudflare_access_service_token.without_deprecated
  to   = cloudflare_zero_trust_access_service_token.without_deprecated
}

# Resource with very large client_secret_version
resource "cloudflare_zero_trust_access_service_token" "large_version" {
  account_id                        = var.cloudflare_account_id
  name                              = "${local.name_prefix}-large-version-token"
  duration                          = "8760h"
  client_secret_version             = 999
  previous_client_secret_expires_at = "2025-12-31T23:59:59Z"
}

moved {
  from = cloudflare_access_service_token.large_version
  to   = cloudflare_zero_trust_access_service_token.large_version
}

# Resource with special characters in name
resource "cloudflare_zero_trust_access_service_token" "special_chars" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-token-with-special_chars-123"
  duration   = "8760h"
}

moved {
  from = cloudflare_access_service_token.special_chars
  to   = cloudflare_zero_trust_access_service_token.special_chars
}

# Resource with minimum duration
resource "cloudflare_zero_trust_access_service_token" "min_duration" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-min-duration-token"
  duration   = "8760h" # Changed from 30m to valid minimum value
}

moved {
  from = cloudflare_access_service_token.min_duration
  to   = cloudflare_zero_trust_access_service_token.min_duration
}

# Resource with maximum duration
resource "cloudflare_zero_trust_access_service_token" "max_duration" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-max-duration-token"
  duration   = "87600h" # 10 years
}

moved {
  from = cloudflare_access_service_token.max_duration
  to   = cloudflare_zero_trust_access_service_token.max_duration
}
