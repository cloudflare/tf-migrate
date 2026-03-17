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
# E2E Tests: Healthcheck v4 → v5 Migration
#
# These tests cover the key migration paths:
#   1. HTTP healthcheck: top-level HTTP fields → http_config nested attribute
#   2. HTTP with headers: header blocks → http_config.header map
#   3. TCP healthcheck: top-level TCP fields → tcp_config nested attribute
#   4. Common fields (check_regions, consecutive_fails, etc.) stay top-level
#
# Note: cloudflare_healthcheck is a zone-scoped resource. Each test uses a
# distinct name to avoid conflicts. The address uses httpbin.org which is a
# stable public endpoint suitable for healthcheck testing.
# ============================================================================

locals {
  hc_prefix  = "cftftest-e2e-hc"
  hc_address = "httpbin.org"
}

# Test 1: Minimal HTTP healthcheck
# Covers: type=HTTP, check_regions (top-level), expected_codes → http_config
resource "cloudflare_healthcheck" "e2e_http_minimal" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.hc_prefix}-http-minimal"
  address = local.hc_address
  type    = "HTTP"

  check_regions  = ["WNAM"]
  expected_codes = ["200"]
}

# Test 2: Full HTTP healthcheck
# Covers: all HTTP-specific fields → http_config, common fields stay top-level,
#         header blocks → http_config.header map
resource "cloudflare_healthcheck" "e2e_http_full" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.hc_prefix}-http-full"
  address = local.hc_address
  type    = "HTTP"

  # HTTP-specific fields (migrate into http_config)
  port             = 80
  path             = "/get"
  method           = "GET"
  expected_codes   = ["200"]
  follow_redirects = false
  allow_insecure   = false

  # Header block (migrates into http_config.header map)
  header {
    header = "User-Agent"
    values = ["HealthChecker/1.0"]
  }

  # Common fields (stay top-level in v5)
  description           = "E2E full HTTP healthcheck"
  consecutive_fails     = 2
  consecutive_successes = 2
  retries               = 2
  timeout               = 5
  interval              = 60
  suspended             = false
  check_regions         = ["WNAM"]
}

# Test 3: TCP healthcheck
# Covers: type=TCP, method + port → tcp_config, common fields stay top-level
resource "cloudflare_healthcheck" "e2e_tcp" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.hc_prefix}-tcp"
  address = local.hc_address
  type    = "TCP"

  # TCP-specific fields (migrate into tcp_config)
  port   = 80
  method = "connection_established"

  # Common fields (stay top-level in v5)
  consecutive_fails = 2
  timeout           = 5
  interval          = 30
  check_regions     = ["WNAM"]
}
