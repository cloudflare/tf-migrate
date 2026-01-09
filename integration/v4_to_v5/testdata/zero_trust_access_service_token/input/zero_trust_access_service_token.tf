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
  common_account = var.cloudflare_account_id
  name_prefix                        = "cftftest"
  tags           = ["test", "migration", "v4_to_v5"]
  token_duration = "8760h"
  enable_long_duration = true
  enable_short_duration = false
}

# ============================================================================
# Pattern Group 2: for_each with Maps (5 resources)
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "map_example" {
  for_each = {
    "prod" = {
      account_id           = var.cloudflare_account_id
      name = "${local.name_prefix}-prod-service-token"
      duration             = "8760h"
      min_days_for_renewal = 30
    }
    "staging" = {
      account_id           = var.cloudflare_account_id
      name = "${local.name_prefix}-staging-service-token"
      duration             = "8760h"
      min_days_for_renewal = 15
    }
    "dev" = {
      account_id           = var.cloudflare_account_id
      name = "${local.name_prefix}-dev-service-token"
      duration             = "8760h"
      min_days_for_renewal = 7
    }
    "qa" = {
      account_id           = var.cloudflare_account_id
      name = "${local.name_prefix}-qa-service-token"
      duration             = "8760h"
      min_days_for_renewal = 10
    }
    "perf" = {
      account_id           = var.cloudflare_account_id
      name = "${local.name_prefix}-perf-service-token"
      duration             = "8760h"
      min_days_for_renewal = 5
    }
  }

  account_id           = each.value.account_id
  name                 = each.value.name
  duration             = each.value.duration
  min_days_for_renewal = each.value.min_days_for_renewal
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
  name = "${local.name_prefix}-set-${each.value}-token"
  duration   = local.token_duration
  min_days_for_renewal = 30
}

# ============================================================================
# Pattern Group 4: count-based Resources (3 instances)
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "counted" {
  count = 3

  account_id            = var.cloudflare_account_id
  name = "${local.name_prefix}-counted-token-${count.index}"
  duration              = "8760h"
  min_days_for_renewal  = count.index * 10 + 10
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "conditional_enabled" {
  count = local.enable_long_duration ? 1 : 0

  account_id           = var.cloudflare_account_id
  name = "${local.name_prefix}-conditional-long-duration"
  duration             = "17520h"  # 2 years
  min_days_for_renewal = 60
}

resource "cloudflare_zero_trust_access_service_token" "conditional_disabled" {
  count = local.enable_short_duration ? 1 : 0

  account_id           = var.cloudflare_account_id
  name = "${local.name_prefix}-conditional-short-duration"
  duration             = "8760h"  # Changed from 720h to valid value
  min_days_for_renewal = 7
}

# ============================================================================
# Pattern Group 7: Terraform Functions
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "with_functions" {
  account_id = local.common_account

  # join() function
  name = join("-", [local.name_prefix, "function", "example"])

  # String interpolation
  duration = "${local.token_duration}"

  min_days_for_renewal = 30
}

resource "cloudflare_zero_trust_access_service_token" "with_interpolation" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-token-for-account-${var.cloudflare_account_id}"
  duration   = "8760h"
  min_days_for_renewal = 30
}

# ============================================================================
# Pattern Group 8: Lifecycle Meta-Arguments
# ============================================================================

resource "cloudflare_zero_trust_access_service_token" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-lifecycle-test-token"
  duration   = "8760h"
  min_days_for_renewal = 30

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [duration]
  }
}

resource "cloudflare_zero_trust_access_service_token" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-prevent-destroy-token"
  duration   = "8760h"
  min_days_for_renewal = 30

  lifecycle {
    prevent_destroy = false  # Set to false for testing
  }
}

# ============================================================================
# Pattern Group 9: Edge Cases
# ============================================================================

# Minimal resource (only required fields)
resource "cloudflare_zero_trust_access_service_token" "minimal" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-minimal-token"
}

# Maximal resource (all fields populated)
resource "cloudflare_zero_trust_access_service_token" "maximal" {
  account_id                        = var.cloudflare_account_id
  name = "${local.name_prefix}-maximal-token"
  duration                          = "8760h"
  min_days_for_renewal              = 30
  client_secret_version             = 5
  previous_client_secret_expires_at = "2025-12-31T23:59:59Z"
}

# Zero values
resource "cloudflare_zero_trust_access_service_token" "zero_values" {
  account_id           = var.cloudflare_account_id
  name = "${local.name_prefix}-zero-values-token"
  duration             = "8760h"
  min_days_for_renewal = 0
}

# Legacy resource name (cloudflare_access_service_token - deprecated)
resource "cloudflare_access_service_token" "legacy_name" {
  account_id                        = var.cloudflare_account_id
  name = "${local.name_prefix}-legacy-name-token"
  duration                          = "8760h"
  min_days_for_renewal              = 30
  client_secret_version             = 2
  previous_client_secret_expires_at = "2024-12-31T23:59:59Z"
}

# Resource without min_days_for_renewal (deprecated field)
resource "cloudflare_zero_trust_access_service_token" "without_deprecated" {
  account_id            = var.cloudflare_account_id
  name = "${local.name_prefix}-no-deprecated-field"
  duration              = "8760h"
}

# Resource with very large client_secret_version
resource "cloudflare_zero_trust_access_service_token" "large_version" {
  account_id                        = var.cloudflare_account_id
  name                              = "${local.name_prefix}-large-version-token"
  duration                          = "8760h"
  min_days_for_renewal              = 30
  client_secret_version             = 999
  previous_client_secret_expires_at = "2025-12-31T23:59:59Z"
}

# Resource with special characters in name
resource "cloudflare_zero_trust_access_service_token" "special_chars" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-token-with-special_chars-123"
  duration             = "8760h"
  min_days_for_renewal = 30
}

# Resource with minimum duration
resource "cloudflare_zero_trust_access_service_token" "min_duration" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-min-duration-token"
  duration             = "8760h"  # Changed from 30m to valid minimum value
  min_days_for_renewal = 1
}

# Resource with maximum duration
resource "cloudflare_zero_trust_access_service_token" "max_duration" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-max-duration-token"
  duration             = "87600h"  # 10 years
  min_days_for_renewal = 365
}
