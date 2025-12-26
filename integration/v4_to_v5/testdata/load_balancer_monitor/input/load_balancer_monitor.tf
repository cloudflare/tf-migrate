# Terraform v4 Load Balancer Monitor Test Configuration
# This file tests the migration from cloudflare_load_balancer_monitor (v4) to cloudflare_load_balancer_monitor (v5)
# NOTE: Resource name stays the same, but field structures change

# Variables - auto-provided by test infrastructure

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

# Locals for DRY and testing
locals {
  name_prefix    = "cftftest"
  common_account = var.cloudflare_account_id
  monitor_types  = ["http", "https", "tcp"]
}

# Test Case 1: Minimal monitor (only required fields)
resource "cloudflare_load_balancer_monitor" "minimal" {
  account_id = var.cloudflare_account_id
  expected_codes = "200"
}

# Test Case 3: HTTPS monitor with multiple headers
resource "cloudflare_load_balancer_monitor" "https_with_headers" {
  account_id     = var.cloudflare_account_id
  type           = "https"
  method         = "GET"
  path           = "/api/health"
  expected_codes = "200"
  expected_body  = "OK"
  interval       = 45
  retries        = 3
  timeout        = 10
  port           = 443
  header {
    header = "Host"
    values = ["api.cf-tf-test.com"]
  }
  header {
    header = "Authorization"
    values = ["Bearer token123"]
  }
}

# Test Case 4: TCP monitor (no headers)
resource "cloudflare_load_balancer_monitor" "tcp" {
  account_id = var.cloudflare_account_id
  type       = "tcp"
  method     = "connection_established"
  port       = 8080
  interval   = 60
  retries    = 2
  timeout    = 5
}

# Test Case 5: Monitor with all optional fields
resource "cloudflare_load_balancer_monitor" "maximal" {
  account_id       = var.cloudflare_account_id
  description      = "Comprehensive test monitor"
  type             = "https"
  method           = "GET"
  path             = "/status"
  port             = 8443
  interval         = 120
  retries          = 5
  timeout          = 10
  expected_codes   = "2xx,3xx"
  expected_body    = "healthy"
  follow_redirects = true
  allow_insecure   = false
  consecutive_down = 3
  consecutive_up   = 2
  header {
    header = "Host"
    values = ["status.cf-tf-test.com"]
  }
  header {
    header = "X-Custom-Header"
    values = ["custom-value"]
  }
}

# Test Case 7: for_each with set
resource "cloudflare_load_balancer_monitor" "set_monitors" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  account_id  = var.cloudflare_account_id
  description = "${local.name_prefix}-monitor-${each.value}"
  type        = "http"
  method      = "GET"
  path        = "/healthz"
  interval    = 60
  expected_codes = "200"
}

# Test Case 8: count-based resources
resource "cloudflare_load_balancer_monitor" "counted" {
  count = 3

  account_id  = var.cloudflare_account_id
  description = "${local.name_prefix}-monitor-${count.index}"
  type        = "http"
  method      = "GET"
  path        = "/health${count.index}"
  port        = 8080 + count.index
  interval    = 30 + (count.index * 10)
  expected_codes = "200"
}

# Test Case 9: Conditional creation
resource "cloudflare_load_balancer_monitor" "conditional_enabled" {
  count = true ? 1 : 0

  account_id  = var.cloudflare_account_id
  description = "${local.name_prefix}-conditional-enabled"
  type        = "http"
  method      = "GET"
  path        = "/enabled"
  expected_codes = "200"
}

resource "cloudflare_load_balancer_monitor" "conditional_disabled" {
  count = false ? 1 : 0

  account_id  = var.cloudflare_account_id
  description = "${local.name_prefix}-conditional-disabled"
  type        = "http"
  method      = "GET"
  path        = "/disabled"
}

# Test Case 10: Using Terraform functions
resource "cloudflare_load_balancer_monitor" "with_functions" {
  account_id  = var.cloudflare_account_id
  description = join("-", [local.name_prefix, "function", "test"])
  type        = "https"
  method      = "GET"
  path        = "/check"
  interval    = 60
  expected_codes = "200"

  header {
    header = "Host"
    values = [join(".", [local.name_prefix, "example", "com"])]
  }
}

# Test Case 11: Lifecycle meta-arguments
resource "cloudflare_load_balancer_monitor" "with_lifecycle" {
  account_id  = var.cloudflare_account_id
  description = "${local.name_prefix}-lifecycle-test"
  type        = "http"
  method      = "GET"
  path        = "/lifecycle"
  expected_codes = "200"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
}

# Test Case 12: Monitor with empty optional fields
resource "cloudflare_load_balancer_monitor" "empty_optionals" {
  account_id = var.cloudflare_account_id
  type       = "http"
  method     = "GET"
  expected_codes = "200"
  # No path, no port, no headers - test default handling
}

# Test Case 13: SMTP monitor type
resource "cloudflare_load_balancer_monitor" "smtp" {
  account_id = var.cloudflare_account_id
  type       = "smtp"
  port       = 25
  interval   = 60
  retries    = 3
  timeout    = 10
}

# Test Case 14: UDP-ICMP monitor type
resource "cloudflare_load_balancer_monitor" "udp_icmp" {
  account_id = var.cloudflare_account_id
  type       = "udp_icmp"
  port       = 53
  interval   = 30
  timeout    = 1
}

# Test Case 15: ICMP-PING monitor type
resource "cloudflare_load_balancer_monitor" "icmp_ping" {
  account_id = var.cloudflare_account_id
  type       = "icmp_ping"
  interval   = 60
  retries    = 2
  timeout    = 1
}

# Test Case 16: Header with multiple values
resource "cloudflare_load_balancer_monitor" "multi_value_header" {
  account_id = var.cloudflare_account_id
  type       = "https"
  method     = "GET"
  path       = "/multi"
  expected_codes = "200"
  header {
    header = "Accept"
    values = ["application/json", "application/xml", "text/html"]
  }
  header {
    header = "Accept-Language"
    values = ["en-US", "en", "fr"]
  }
}

# Test Case 17: String interpolation
resource "cloudflare_load_balancer_monitor" "with_interpolation" {
  account_id  = var.cloudflare_account_id
  description = "Monitor for account ${var.cloudflare_account_id}"
  type        = "http"
  method      = "GET"
  path        = "/check"
  expected_codes = "200"
}

# Test Case 18: Special characters in fields
resource "cloudflare_load_balancer_monitor" "special_chars" {
  account_id    = var.cloudflare_account_id
  description   = "Test: Special & chars <> in \"description\""
  type          = "https"
  method        = "GET"
  path          = "/api/v2/status?detailed=true&format=json"
  expected_body = "Status: OK | Ready: true"
  expected_codes = "200"
  header {
    header = "X-Api-Key"
    values = ["key_with-dashes.and_underscores"]
  }
}

# Test Case 19: Edge case - consecutive values
resource "cloudflare_load_balancer_monitor" "consecutive_values" {
  account_id       = var.cloudflare_account_id
  type             = "http"
  method           = "GET"
  path             = "/check"
  consecutive_down = 5
  consecutive_up   = 3
  interval         = 45
  retries          = 4
  timeout          = 8
  expected_codes = "200"
}
