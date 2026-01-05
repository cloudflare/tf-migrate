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

# ========================================
# Test Cases for notification_policy Migration
# ========================================

# Test Case 1: Basic notification policy with filters and single email integration
resource "cloudflare_notification_policy" "basic_with_filters" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-basic-filters"
  enabled    = true
  alert_type = "universal_ssl_event_type"

  filters {
    zones = [var.cloudflare_zone_id]
  }

  email_integration {
    id = "test-email-integration-1"
  }
}

# Test Case 2: Policy with multiple integration types
resource "cloudflare_notification_policy" "multi_integration" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-multi-integration"
  enabled    = false
  alert_type = "health_check_status_notification"

  email_integration {
    id   = "test-email-integration-2"
    name = "test@example.com"
  }

  webhooks_integration {
    id   = "test-webhook-integration-1"
    name = "My Test Webhook"
  }

  pagerduty_integration {
    id   = "test-pagerduty-integration-1"
    name = "PD Integration"
  }
}

# Test Case 3: Policy with complex filters
resource "cloudflare_notification_policy" "complex_filters" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-complex-filters"
  enabled    = true
  alert_type = "load_balancing_health_alert"

  filters {
    zones    = [var.cloudflare_zone_id]
    services = ["load_balancer"]
    status   = ["Healthy", "Unhealthy"]
  }

  email_integration {
    id = "test-email-integration-3"
  }
}

# Test Case 4: Policy without filters
resource "cloudflare_notification_policy" "no_filters" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-no-filters"
  enabled    = true
  alert_type = "billing_usage_alert"

  webhooks_integration {
    id = "test-webhook-integration-2"
  }
}

# Test Case 5: Policy with multiple email integrations
resource "cloudflare_notification_policy" "multi_email" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-multi-email"
  enabled    = true
  alert_type = "universal_ssl_event_type"

  email_integration {
    id = "test-email-integration-4"
  }

  email_integration {
    id = "test-email-integration-5"
  }
}

# Test Case 6: Policy with description
resource "cloudflare_notification_policy" "with_description" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-with-description"
  description = "This is a test notification policy with description"
  enabled     = true
  alert_type  = "universal_ssl_event_type"

  filters {
    zones = [var.cloudflare_zone_id]
  }

  email_integration {
    id   = "test-email-integration-6"
    name = "Will be removed in v5"
  }
}

# Test Case 7: Policy with enabled=false (must preserve!)
resource "cloudflare_notification_policy" "disabled_policy" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-disabled"
  enabled    = false
  alert_type = "universal_ssl_event_type"

  email_integration {
    id = "test-email-integration-7"
  }
}

# Test Case 8: Advanced filters - billing events
resource "cloudflare_notification_policy" "billing_alert" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-billing-alert"
  enabled    = true
  alert_type = "billing_usage_alert"

  webhooks_integration {
    id = "test-webhook-integration-3"
  }

  pagerduty_integration {
    id = "test-pagerduty-integration-2"
  }
}

# Test Case 9: Load balancer pool health alert
resource "cloudflare_notification_policy" "lb_pool_health" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lb-pool-health"
  enabled    = true
  alert_type = "load_balancing_pool_enablement_alert"

  filters {
    zones = [var.cloudflare_zone_id]
  }

  email_integration {
    id = "test-email-integration-8"
  }

  webhooks_integration {
    id = "test-webhook-integration-4"
  }
}

# Test Case 10: Zone-specific SSL alert with multiple emails
resource "cloudflare_notification_policy" "ssl_alert" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-ssl-alert"
  description = "SSL certificate expiration alerts"
  enabled     = true
  alert_type  = "universal_ssl_event_type"

  filters {
    zones = [var.cloudflare_zone_id]
  }

  email_integration {
    id = "test-email-integration-9"
  }

  email_integration {
    id = "test-email-integration-10"
  }
}

# Test Case 11: Health check notification
resource "cloudflare_notification_policy" "health_check_alert" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-health-check"
  enabled    = true
  alert_type = "health_check_status_notification"

  filters {
    zones = [var.cloudflare_zone_id]
  }

  email_integration {
    id = "test-email-integration-11"
  }

  pagerduty_integration {
    id   = "test-pagerduty-integration-3"
    name = "On-call team"
  }
}

# Test Case 12: Policy with all three integration types and complex filters
resource "cloudflare_notification_policy" "comprehensive" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-comprehensive"
  description = "Comprehensive test with all features"
  enabled     = true
  alert_type  = "load_balancing_health_alert"

  filters {
    zones    = [var.cloudflare_zone_id]
    services = ["load_balancer", "health_checks"]
    status   = ["Healthy", "Unhealthy", "Suspended"]
  }

  email_integration {
    id   = "test-email-integration-12"
    name = "admin@example.com"
  }

  email_integration {
    id   = "test-email-integration-13"
    name = "ops@example.com"
  }

  webhooks_integration {
    id   = "test-webhook-integration-5"
    name = "Slack Integration"
  }

  pagerduty_integration {
    id   = "test-pagerduty-integration-4"
    name = "Incident Response"
  }
}

# Test Case 13: Minimal policy (only required fields)
resource "cloudflare_notification_policy" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal"
  enabled    = true
  alert_type = "universal_ssl_event_type"

  email_integration {
    id = "test-email-integration-14"
  }
}

# Test Case 14: Policy with special characters in name
resource "cloudflare_notification_policy" "special_chars" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-policy-with-dashes_and_underscores"
  enabled    = true
  alert_type = "universal_ssl_event_type"

  email_integration {
    id = "test-email-integration-15"
  }
}

# Test Case 15: Advanced HTTP alert
resource "cloudflare_notification_policy" "rate_limit_alert" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-rate-limit"
  description = "Rate limiting notifications"
  enabled     = true
  alert_type  = "advanced_http_alert_error"

  filters {
    zones = [var.cloudflare_zone_id]
  }

  webhooks_integration {
    id = "test-webhook-integration-6"
  }
}
