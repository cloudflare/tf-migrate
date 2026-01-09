# Comprehensive Integration Tests for workers_kv_namespace v4 to v5 Migration
# Covers all mandatory patterns from subtask 08

# Pattern Group 1: Variables & Locals (Lines 1-25)
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
  common_account = var.cloudflare_account_id
  name_prefix    = "cftftest"
  tags           = ["test", "migration", "v4_to_v5"]
  environments   = ["dev", "staging", "prod"]
}

# Pattern Group 2: for_each with Maps (3 instances)
resource "cloudflare_workers_kv_namespace" "map_example" {
  for_each = {
    "cache" = {
      title = "Cache Store"
    }
    "session" = {
      title = "Session Store"
    }
    "config" = {
      title = "Configuration Store"
    }
  }

  account_id = local.common_account
  title      = "${local.name_prefix}-${each.value.title}"
}

# Pattern Group 3: for_each with Sets (4 instances)
resource "cloudflare_workers_kv_namespace" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-set-${each.value}-namespace"
}

# Pattern Group 4: count-based Resources (3 instances)
resource "cloudflare_workers_kv_namespace" "counted" {
  count = 3

  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-counted-namespace-${count.index}"
}

# Pattern Group 5: Conditional Creation (1 instance created, 1 not created)
locals {
  enable_feature = true
  enable_test    = false
}

resource "cloudflare_workers_kv_namespace" "conditional_enabled" {
  count = local.enable_feature ? 1 : 0

  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-conditional-enabled-namespace"
}

resource "cloudflare_workers_kv_namespace" "conditional_disabled" {
  count = local.enable_test ? 1 : 0

  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-conditional-disabled-namespace"
}

# Pattern Group 6: Terraform Functions (3 instances)
resource "cloudflare_workers_kv_namespace" "with_join" {
  account_id = var.cloudflare_account_id
  title      = join("-", [local.name_prefix, "workers", "kv", "joined"])
}

resource "cloudflare_workers_kv_namespace" "with_format" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-formatted-namespace-001"
}

resource "cloudflare_workers_kv_namespace" "with_interpolation" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-namespace-for-account-${var.cloudflare_account_id}"
}

# Pattern Group 7: Lifecycle Meta-Arguments (2 instances)
resource "cloudflare_workers_kv_namespace" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-lifecycle-test-namespace"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [title]
  }
}

resource "cloudflare_workers_kv_namespace" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-prevent-destroy-namespace"

  lifecycle {
    prevent_destroy = false
  }
}

# Pattern Group 8: Edge Cases

# Minimal resource (only required fields)
resource "cloudflare_workers_kv_namespace" "minimal" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-minimal"
}

# Resource with special characters in title
resource "cloudflare_workers_kv_namespace" "special_chars" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}_namespace-with.special-chars!2024"
}

# Resource with spaces in title
resource "cloudflare_workers_kv_namespace" "with_spaces" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix} My Workers KV Namespace With Spaces"
}

# Resource with very long title
resource "cloudflare_workers_kv_namespace" "long_title" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-very-long-namespace-title-that-tests-maximum-length-handling"
}

# Resource with unicode characters
resource "cloudflare_workers_kv_namespace" "unicode" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-namespace-with-émojis-🚀-and-spëcial-chàrs"
}

# Pattern Group 9: Production-like Patterns

# Using locals for common values
resource "cloudflare_workers_kv_namespace" "from_locals" {
  account_id = local.common_account
  title      = "${local.name_prefix}-from-locals"
}

# Variable-driven configuration
resource "cloudflare_workers_kv_namespace" "variable_driven" {
  account_id = var.cloudflare_account_id
  title      = "${local.name_prefix}-namespace-${var.cloudflare_zone_id}"
}

# Total instances: 23
# - map_example: 3 instances (cache, session, config)
# - set_example: 4 instances (alpha, beta, gamma, delta)
# - counted: 3 instances (0, 1, 2)
# - conditional_enabled: 1 instance
# - conditional_disabled: 0 instances (not created)
# - with_join: 1 instance
# - with_format: 1 instance
# - with_interpolation: 1 instance
# - with_lifecycle: 1 instance
# - with_prevent_destroy: 1 instance
# - minimal: 1 instance
# - special_chars: 1 instance
# - with_spaces: 1 instance
# - long_title: 1 instance
# - unicode: 1 instance
# - from_locals: 1 instance
# - variable_driven: 1 instance
# Total: 23 instances (excluding conditional_disabled which has count = 0)
