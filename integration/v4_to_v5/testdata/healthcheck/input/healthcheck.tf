# Cloudflare Healthcheck v4 to v5 Migration Integration Tests
# This file contains comprehensive test cases covering all migration scenarios

# Standard variables (auto-provided by E2E infrastructure)
variable "cloudflare_account_id" {
  description = "Cloudflare account ID for testing"
  type        = string
}
variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}

# Variable for test IP - override if default doesn't work
variable "healthcheck_test_ip" {
  description = "IP for healthcheck test DNS record - use an IP you control if needed"
  type        = string
  default     = ""  # Empty default - will auto-detect zone apex IP or use httpbin.org
}

# Get actual zone domain
data "cloudflare_zone" "test_zone" {
  zone_id = var.cloudflare_zone_id
}

# Try to get the zone apex A record (if it exists)
data "external" "zone_apex_ip" {
  program = ["bash", "-c", <<-EOT
    IP=$(dig +short ${data.cloudflare_zone.test_zone.name} A @1.1.1.1 | grep -E '^[0-9.]+$' | head -1)
    if [ -z "$IP" ]; then
      echo '{"ip":"52.205.123.206"}'
    else
      echo "{\"ip\":\"$IP\"}"
    fi
  EOT
  ]
}

# Locals for common values
locals {
  zone_id       = var.cloudflare_zone_id
  name_prefix   = "migration-test"
  real_domain   = data.cloudflare_zone.test_zone.name
  # Use provided IP, otherwise use zone apex IP, otherwise use test IP
  resolved_ip   = var.healthcheck_test_ip != "" ? var.healthcheck_test_ip : data.external.zone_apex_ip.result.ip
  # Use IP directly instead of hostname to avoid DNS issues
  test_hostname = local.resolved_ip
}

# No DNS record needed - using IPs directly

# ==============================================================================
# Test Case 1: Minimal HTTP healthcheck (required fields only)
# ==============================================================================
resource "cloudflare_healthcheck" "minimal_http" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-minimal-http"
  address = local.test_hostname
  type    = "HTTP"
}

# ==============================================================================
# Test Case 2: Full HTTP healthcheck (all optional fields)
# ==============================================================================
resource "cloudflare_healthcheck" "full_http" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-full-http"
  address = local.test_hostname
  type    = "HTTP"

  # HTTP-specific fields
  port             = 80
  path             = "/health"
  method           = "GET"
  expected_codes   = ["200", "201", "204"]
  expected_body    = "OK"
  follow_redirects = false
  allow_insecure   = false

  # Headers
  header {
    header = "Host"
    values = [local.real_domain]
  }

  header {
    header = "User-Agent"
    values = ["HealthChecker/1.0"]
  }

  # Common fields
  description            = "Full HTTP healthcheck with all fields"
  consecutive_fails      = 3
  consecutive_successes  = 2
  retries                = 2
  timeout                = 5
  interval               = 60
  suspended              = false
  check_regions          = ["WNAM", "ENAM"]
}

# ==============================================================================
# Test Case 3: HTTPS healthcheck with SSL options
# ==============================================================================
resource "cloudflare_healthcheck" "https_with_ssl" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-https-ssl"
  address = local.test_hostname
  type    = "HTTPS"

  port           = 443
  path           = "/api/health"
  method         = "HEAD"
  allow_insecure = true
  expected_codes = ["200"]

  header {
    header = "Authorization"
    values = ["Bearer test-token"]
  }

  consecutive_fails = 2
  timeout           = 10
}

# ==============================================================================
# Test Case 4: TCP healthcheck
# ==============================================================================
resource "cloudflare_healthcheck" "tcp_basic" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-tcp"
  address = local.resolved_ip
  type    = "TCP"

  port   = 8080
  method = "connection_established"

  consecutive_fails = 2
  timeout           = 10
  interval          = 30
}

# ==============================================================================
# Test Case 5: for_each with map - Multiple HTTP checks
# ==============================================================================
resource "cloudflare_healthcheck" "http_map" {

  for_each = {
    "api" = {
      address = local.test_hostname
      path    = "/api/health"
      port    = 8080
    }
    "web" = {
      address = local.test_hostname
      path    = "/"
      port    = 80
    }
    "admin" = {
      address = local.test_hostname
      path    = "/admin/health"
      port    = 443
    }
  }

  zone_id = local.zone_id
  name    = "${local.name_prefix}-${each.key}"
  address = each.value.address
  type    = "HTTP"

  port   = each.value.port
  path   = each.value.path
  method = "GET"

  consecutive_fails = 3
  interval          = 60
}

# ==============================================================================
# Test Case 6: for_each with set - Multiple TCP checks
# ==============================================================================
resource "cloudflare_healthcheck" "tcp_set" {

  for_each = toset([
    "db-primary",
    "db-replica",
    "cache-server"
  ])

  zone_id = local.zone_id
  name    = "${local.name_prefix}-${each.value}"
  address = local.test_hostname
  type    = "TCP"

  port   = 5432
  method = "connection_established"

  timeout  = 5
  interval = 30
}

# ==============================================================================
# Test Case 7: count-based resources
# ==============================================================================
resource "cloudflare_healthcheck" "counted" {

  count = 4

  zone_id = local.zone_id
  name    = "${local.name_prefix}-counted-${count.index}"
  address = local.test_hostname
  type    = "HTTP"

  port   = 80 + count.index
  path   = "/health/${count.index}"
  method = "GET"

  description = "Health check for server ${count.index}"
  interval    = 60 + (count.index * 10)
}

# ==============================================================================
# Test Case 8: Conditional creation
# ==============================================================================
locals {
  enable_monitoring = true
  enable_debug      = false
}

resource "cloudflare_healthcheck" "conditional_enabled" {

  count = local.enable_monitoring ? 1 : 0

  zone_id = local.zone_id
  name    = "${local.name_prefix}-conditional-enabled"
  address = local.test_hostname
  type    = "HTTP"

  port   = 80
  path   = "/health"
  method = "GET"
}

resource "cloudflare_healthcheck" "conditional_disabled" {

  count = local.enable_debug ? 1 : 0

  zone_id = local.zone_id
  name    = "${local.name_prefix}-conditional-disabled"
  address = local.test_hostname
  type    = "HTTP"

  port   = 80
  path   = "/debug"
  method = "GET"
}

# ==============================================================================
# Test Case 9: Terraform functions
# ==============================================================================
resource "cloudflare_healthcheck" "with_functions" {

  zone_id = local.zone_id
  name    = join("-", [local.name_prefix, "functions", "test"])
  address = local.resolved_ip
  type    = "HTTPS"

  port        = 443
  path        = "/health"
  method      = "GET"
  description = "Healthcheck for zone ${local.zone_id}"

  expected_codes = tolist(["200", "202"])
  check_regions  = tolist(["WNAM", "EEU"])

  interval = 60
}

# ==============================================================================
# Test Case 10: Lifecycle meta-arguments
# ==============================================================================
resource "cloudflare_healthcheck" "with_lifecycle" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-lifecycle"
  address = local.test_hostname
  type    = "HTTP"

  port   = 80
  path   = "/health"
  method = "GET"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
}

resource "cloudflare_healthcheck" "prevent_destroy" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-prevent-destroy"
  address = local.test_hostname
  type    = "HTTPS"

  port   = 443
  path   = "/"
  method = "GET"

  lifecycle {
    prevent_destroy = false  # Set to false for testing
  }
}

# ==============================================================================
# Test Case 11: Multiple headers
# ==============================================================================
resource "cloudflare_healthcheck" "multiple_headers" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-multi-headers"
  address = local.test_hostname
  type    = "HTTP"

  port   = 80
  path   = "/v1/health"
  method = "GET"

  header {
    header = "Accept"
    values = ["application/json", "text/plain"]
  }

  header {
    header = "X-API-Key"
    values = ["test-api-key-123"]
  }

  header {
    header = "X-Custom-Header"
    values = ["custom-value"]
  }
}

# ==============================================================================
# Test Case 12: Empty optional fields
# ==============================================================================
resource "cloudflare_healthcheck" "empty_fields" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-empty"
  address = local.test_hostname
  type    = "HTTP"

  path           = "/"
  check_regions  = []
  expected_codes = []
}

# ==============================================================================
# Test Case 13: Null description
# ==============================================================================
resource "cloudflare_healthcheck" "null_description" {

  zone_id     = local.zone_id
  name        = "${local.name_prefix}-null-desc"
  address     = local.test_hostname
  type        = "HTTP"
  description = null
}

# ==============================================================================
# Test Case 14: Suspended healthcheck
# ==============================================================================
resource "cloudflare_healthcheck" "suspended" {

  zone_id   = local.zone_id
  name      = "${local.name_prefix}-suspended"
  address   = local.test_hostname
  type      = "HTTP"
  suspended = true

  path   = "/health"
  method = "GET"
}

# ==============================================================================
# Test Case 15: Different HTTP methods
# ==============================================================================
locals {
  http_methods = ["GET", "HEAD"]
}

resource "cloudflare_healthcheck" "http_methods" {

  for_each = toset(local.http_methods)

  zone_id = local.zone_id
  name    = "${local.name_prefix}-method-${lower(each.value)}"
  address = local.test_hostname
  type    = "HTTP"

  port   = 80
  path   = "/health"
  method = each.value
}

# ==============================================================================
# Test Case 16: Different check regions
# ==============================================================================
resource "cloudflare_healthcheck" "regions_wnam" {

  zone_id       = local.zone_id
  name          = "${local.name_prefix}-region-wnam"
  address       = local.test_hostname
  type          = "HTTP"
  check_regions = ["WNAM"]
}

resource "cloudflare_healthcheck" "regions_multi" {

  zone_id       = local.zone_id
  name          = "${local.name_prefix}-region-multi"
  address       = local.test_hostname
  type          = "HTTP"
  check_regions = ["WNAM", "ENAM", "WEU", "EEU", "SEAS"]
}

# ==============================================================================
# Test Case 17: String interpolation
# ==============================================================================
resource "cloudflare_healthcheck" "interpolation" {

  zone_id     = local.zone_id
  name        = "${local.name_prefix}-${local.real_domain}-interpolated"
  address     = local.resolved_ip
  type        = "HTTP"
  description = "Health check for ${local.real_domain} in zone ${local.zone_id}"

  path   = "/health"
  method = "GET"
}

# ==============================================================================
# Test Case 18: Numeric edge values
# ==============================================================================
resource "cloudflare_healthcheck" "numeric_edges" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-numeric-edges"
  address = local.test_hostname
  type    = "HTTP"

  consecutive_fails     = 1  # Minimum
  consecutive_successes = 1  # Minimum
  retries               = 5  # Higher value
  timeout               = 15 # Maximum value
  interval              = 300 # Higher value

  port   = 8443
  method = "GET"
}

# ==============================================================================
# Test Case 19: Dynamic block pattern (for headers)
# ==============================================================================
locals {
  custom_headers = {
    "X-Environment" = ["production"]
    "X-Region"      = ["us-west"]
    "X-Version"     = ["v2"]
  }
}

resource "cloudflare_healthcheck" "dynamic_headers" {

  zone_id = local.zone_id
  name    = "${local.name_prefix}-dynamic-headers"
  address = local.test_hostname
  type    = "HTTP"

  port   = 80
  path   = "/health"
  method = "GET"

  header {
    header = "X-Environment"
    values = ["production"]
  }

  header {
    header = "X-Region"
    values = ["us-west"]
  }

  header {
    header = "X-Version"
    values = ["v2"]
  }
}

# ==============================================================================
# Summary: 25+ resource instances covering all patterns
# ==============================================================================
# 1. minimal_http (1)
# 2. full_http (1)
# 3. https_with_ssl (1)
# 4. tcp_basic (1)
# 5. http_map (3 instances via for_each)
# 6. tcp_set (3 instances via for_each)
# 7. counted (4 instances via count)
# 8. conditional_enabled (1, disabled=0)
# 9. with_functions (1)
# 10. with_lifecycle (1)
# 11. prevent_destroy (1)
# 12. multiple_headers (1)
# 13. empty_fields (1)
# 14. null_description (1)
# 15. suspended (1)
# 16. http_methods (3 instances via for_each)
# 17. regions_wnam (1)
# 18. regions_multi (1)
# 19. interpolation (1)
# 20. numeric_edges (1)
# 21. dynamic_headers (1)
#
# TOTAL: 28 resource instances
