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

# Use Cloudflare trace endpoint for webhook testing
# URL: https://www.cloudflare.com/cdn-cgi/trace
# This endpoint responds with 200 OK to all requests for webhook validation

# ========================================
# Locals
# ========================================
locals {
  common_account   = var.cloudflare_account_id
  name_prefix = "cftftest"
  webhook_base_url = "https://www.cloudflare.com/cdn-cgi/trace"
  enable_backup    = true
  enable_test      = false
}

# ========================================
# Basic Resources
# ========================================

# Test Case 1: Basic webhook with minimal fields
resource "cloudflare_notification_policy_webhooks" "basic_webhook" {
  account_id = var.cloudflare_account_id
  name       = "basic-webhook"
  url        = "https://www.cloudflare.com/cdn-cgi/trace"
}

# Test Case 2: Full webhook with all fields
resource "cloudflare_notification_policy_webhooks" "full_webhook" {
  account_id = var.cloudflare_account_id
  name       = "production-webhook"
  url        = "https://www.cloudflare.com/cdn-cgi/trace"
  secret     = "webhook-secret-token-12345"
}

# ========================================
# for_each with Maps Pattern (3 resources)
# ========================================
resource "cloudflare_notification_policy_webhooks" "map_example" {
  for_each = {
    "alerts" = {
      name   = "alerts-webhook"
      secret = "alerts-secret-123"
    }
    "monitoring" = {
      name   = "monitoring-webhook"
      secret = "monitoring-secret-456"
    }
    "security" = {
      name   = "security-webhook"
      secret = "security-secret-789"
    }
  }

  account_id = local.common_account
  name       = each.value.name
  url        = local.webhook_base_url
  secret     = each.value.secret
}

# ========================================
# for_each with Sets Pattern (4 resources)
# ========================================
resource "cloudflare_notification_policy_webhooks" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  account_id = var.cloudflare_account_id
  name       = "set-${each.value}"
  url        = local.webhook_base_url
}

# ========================================
# count-based Resources (3 resources)
# ========================================
resource "cloudflare_notification_policy_webhooks" "counted" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "webhook-${count.index}"
  url        = local.webhook_base_url
}

# ========================================
# Conditional Creation
# ========================================
resource "cloudflare_notification_policy_webhooks" "conditional_enabled" {
  count = local.enable_backup ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "conditional-enabled"
  url        = local.webhook_base_url
}

resource "cloudflare_notification_policy_webhooks" "conditional_disabled" {
  count = local.enable_test ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "conditional-disabled"
  url        = local.webhook_base_url
}

# ========================================
# Terraform Functions
# ========================================
resource "cloudflare_notification_policy_webhooks" "with_functions" {
  account_id = local.common_account
  name       = join("-", [local.name_prefix, "function", "example"])
  url        = local.webhook_base_url
  secret     = "function-test-secret"
}

# ========================================
# Lifecycle Meta-Arguments
# ========================================
resource "cloudflare_notification_policy_webhooks" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "lifecycle-test"
  url        = local.webhook_base_url

  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_notification_policy_webhooks" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  name       = "prevent-destroy-test"
  url        = local.webhook_base_url

  lifecycle {
    prevent_destroy = false
  }
}

# ========================================
# Edge Cases
# ========================================

# Minimal resource (only required fields)
resource "cloudflare_notification_policy_webhooks" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "minimal"
  url        = local.webhook_base_url
}

# Maximal resource (all fields populated)
resource "cloudflare_notification_policy_webhooks" "maximal" {
  account_id = var.cloudflare_account_id
  name       = "maximal-webhook-with-all-fields"
  url        = local.webhook_base_url
  secret     = "maximal-secret-token-with-special-chars-!@#$"
}

# URL with special characters
resource "cloudflare_notification_policy_webhooks" "special_chars" {
  account_id = var.cloudflare_account_id
  name       = "special-chars-test"
  url        = local.webhook_base_url
}
