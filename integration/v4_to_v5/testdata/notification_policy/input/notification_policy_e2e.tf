# E2E Test: cloudflare_notification_policy
# Creates webhook integrations and notification policies that can be applied with real infrastructure
# Note: Email and PagerDuty integrations require external setup and cannot be tested here

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

# ========================================
# Locals
# ========================================
locals {
  name_prefix = "cftftest"
}

##########################
# E2E TEST WEBHOOKS
##########################
# Create webhook integrations to use with notification policies

resource "cloudflare_notification_policy_webhooks" "e2e_webhook_1" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-webhook-1"
  url        = "https://www.cloudflare.com/cdn-cgi/trace"
}

resource "cloudflare_notification_policy_webhooks" "e2e_webhook_2" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-webhook-2"
  url        = "https://www.cloudflare.com/cdn-cgi/trace"
}

resource "cloudflare_notification_policy_webhooks" "e2e_webhook_3" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-webhook-3"
  url        = "https://www.cloudflare.com/cdn-cgi/trace"
}

##########################
# E2E TEST NOTIFICATION POLICIES
##########################

# Test Case 1: Expiring service token alert (account-level, no filters required)
resource "cloudflare_notification_policy" "e2e_billing" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-billing"
  enabled    = true
  alert_type = "expiring_service_token_alert"

  webhooks_integration {
    id = cloudflare_notification_policy_webhooks.e2e_webhook_1.id
  }
}

# Test Case 2: Expiring service token alert (account-level)
resource "cloudflare_notification_policy" "e2e_service_token" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-service-token"
  enabled    = true
  alert_type = "expiring_service_token_alert"

  webhooks_integration {
    id = cloudflare_notification_policy_webhooks.e2e_webhook_1.id
  }
}

# Test Case 3: Policy with multiple webhooks
resource "cloudflare_notification_policy" "e2e_multi_webhook" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-multi-webhook"
  enabled    = true
  alert_type = "expiring_service_token_alert"

  webhooks_integration {
    id = cloudflare_notification_policy_webhooks.e2e_webhook_1.id
  }

  webhooks_integration {
    id = cloudflare_notification_policy_webhooks.e2e_webhook_2.id
  }
}

# Test Case 4: Policy with description
resource "cloudflare_notification_policy" "e2e_with_description" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-e2e-description"
  description = "E2E test notification policy with description"
  enabled     = true
  alert_type  = "expiring_service_token_alert"

  webhooks_integration {
    id = cloudflare_notification_policy_webhooks.e2e_webhook_2.id
  }
}

# Test Case 5: Disabled policy (enabled=false - CRITICAL to preserve during migration)
resource "cloudflare_notification_policy" "e2e_disabled" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-disabled"
  enabled    = false
  alert_type = "expiring_service_token_alert"

  webhooks_integration {
    id = cloudflare_notification_policy_webhooks.e2e_webhook_3.id
  }
}

# Test Case 6: Policy with special characters in name
resource "cloudflare_notification_policy" "e2e_special_chars" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-policy_with-dashes"
  enabled    = true
  alert_type = "expiring_service_token_alert"

  webhooks_integration {
    id = cloudflare_notification_policy_webhooks.e2e_webhook_1.id
  }
}
