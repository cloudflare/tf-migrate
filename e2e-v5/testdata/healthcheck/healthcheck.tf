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
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
  hc_prefix   = "${local.name_prefix}-e2e-hc"
  hc_address  = "httpbin.org"
}

# Test 1: Minimal HTTP healthcheck
# Covers: type=HTTP, check_regions (top-level), expected_codes → http_config
resource "cloudflare_healthcheck" "e2e_http_minimal" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.hc_prefix}-http-minimal"
  address = local.hc_address
  type    = "HTTP"

  check_regions = ["WNAM"]
  http_config = {
    expected_codes = ["200"]
  }
}

# Test 2: Full HTTP healthcheck
# Covers: all HTTP-specific fields → http_config, common fields stay top-level,
#         header blocks → http_config.header map
resource "cloudflare_healthcheck" "e2e_http_full" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.hc_prefix}-http-full"
  address = local.hc_address
  type    = "HTTP"



  # Common fields (stay top-level in v5)
  description           = "E2E full HTTP healthcheck"
  consecutive_fails     = 2
  consecutive_successes = 2
  retries               = 2
  timeout               = 5
  interval              = 60
  suspended             = false
  check_regions         = ["WNAM"]
  http_config = {
    allow_insecure   = false
    expected_codes   = ["200"]
    follow_redirects = false
    header = {
      "User-Agent" = ["HealthChecker/1.0"]
    }
    method = "GET"
    path   = "/get"
    port   = 80
  }
}

# Test 3: TCP healthcheck
# Covers: type=TCP, method + port → tcp_config, common fields stay top-level
resource "cloudflare_healthcheck" "e2e_tcp" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.hc_prefix}-tcp"
  address = local.hc_address
  type    = "TCP"


  # Common fields (stay top-level in v5)
  consecutive_fails = 2
  timeout           = 5
  interval          = 30
  check_regions     = ["WNAM"]
  tcp_config = {
    method = "connection_established"
    port   = 80
  }
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
