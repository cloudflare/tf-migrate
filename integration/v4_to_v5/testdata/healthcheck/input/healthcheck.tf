# Cloudflare Healthcheck v4 to v5 Migration Integration Tests
# This file contains comprehensive test cases covering all migration scenarios

# Standard variables (auto-provided by E2E infrastructure)
variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}

# Locals for common values
locals {
  zone_id     = var.cloudflare_zone_id
  name_prefix = "migration-test"
  test_domain = "example.com"
}

# ==============================================================================
# Test Case 1: Minimal HTTP healthcheck (required fields only)
# ==============================================================================
resource "cloudflare_healthcheck" "minimal_http" {
  zone_id = local.zone_id
  name    = "${local.name_prefix}-minimal-http"
  address = local.test_domain
  type    = "HTTP"
}

# ==============================================================================
# Test Case 2: Full HTTP healthcheck (all optional fields)
# ==============================================================================
resource "cloudflare_healthcheck" "full_http" {
  zone_id = local.zone_id
  name    = "${local.name_prefix}-full-http"
  address = "api.${local.test_domain}"
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
    values = [local.test_domain]
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
  address = "secure.${local.test_domain}"
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
  address = "10.0.0.1"
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
      address = "api.${local.test_domain}"
      path    = "/api/health"
      port    = 8080
    }
    "web" = {
      address = "www.${local.test_domain}"
      path    = "/"
      port    = 80
    }
    "admin" = {
      address = "admin.${local.test_domain}"
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
  address = "${each.value}.internal"
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
  address = "server-${count.index}.${local.test_domain}"
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
  address = "monitor.${local.test_domain}"
  type    = "HTTP"

  port   = 80
  path   = "/health"
  method = "GET"
}

resource "cloudflare_healthcheck" "conditional_disabled" {
  count = local.enable_debug ? 1 : 0

  zone_id = local.zone_id
  name    = "${local.name_prefix}-conditional-disabled"
  address = "debug.${local.test_domain}"
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
  address = "${local.test_domain}"
  type    = "HTTPS"

  port        = 443
  path        = "/health"
  method      = "POST"
  description = "Healthcheck for zone ${local.zone_id}"

  expected_codes = tolist(["200", "202"])
  check_regions  = tolist(["WNAM", "EEUR"])

  interval = 60
}

# ==============================================================================
# Test Case 10: Lifecycle meta-arguments
# ==============================================================================
resource "cloudflare_healthcheck" "with_lifecycle" {
  zone_id = local.zone_id
  name    = "${local.name_prefix}-lifecycle"
  address = "lifecycle.${local.test_domain}"
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
  address = "important.${local.test_domain}"
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
  address = "api.${local.test_domain}"
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
  address = "empty.${local.test_domain}"
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
  address     = "null.${local.test_domain}"
  type        = "HTTP"
  description = null
}

# ==============================================================================
# Test Case 14: Suspended healthcheck
# ==============================================================================
resource "cloudflare_healthcheck" "suspended" {
  zone_id   = local.zone_id
  name      = "${local.name_prefix}-suspended"
  address   = "suspended.${local.test_domain}"
  type      = "HTTP"
  suspended = true

  path   = "/health"
  method = "GET"
}

# ==============================================================================
# Test Case 15: Different HTTP methods
# ==============================================================================
locals {
  http_methods = ["GET", "HEAD", "POST"]
}

resource "cloudflare_healthcheck" "http_methods" {
  for_each = toset(local.http_methods)

  zone_id = local.zone_id
  name    = "${local.name_prefix}-method-${lower(each.value)}"
  address = "methods.${local.test_domain}"
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
  address       = "us.${local.test_domain}"
  type          = "HTTP"
  check_regions = ["WNAM"]
}

resource "cloudflare_healthcheck" "regions_multi" {
  zone_id       = local.zone_id
  name          = "${local.name_prefix}-region-multi"
  address       = "global.${local.test_domain}"
  type          = "HTTP"
  check_regions = ["WNAM", "ENAM", "WEU", "EEU", "SEAS", "WEAS"]
}

# ==============================================================================
# Test Case 17: String interpolation
# ==============================================================================
resource "cloudflare_healthcheck" "interpolation" {
  zone_id     = local.zone_id
  name        = "${local.name_prefix}-${local.test_domain}-interpolated"
  address     = "${local.test_domain}"
  type        = "HTTP"
  description = "Health check for ${local.test_domain} in zone ${local.zone_id}"

  path   = "/health"
  method = "GET"
}

# ==============================================================================
# Test Case 18: Numeric edge values
# ==============================================================================
resource "cloudflare_healthcheck" "numeric_edges" {
  zone_id = local.zone_id
  name    = "${local.name_prefix}-numeric-edges"
  address = "numeric.${local.test_domain}"
  type    = "HTTP"

  consecutive_fails     = 1  # Minimum
  consecutive_successes = 1  # Minimum
  retries               = 5  # Higher value
  timeout               = 30 # Higher value
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
  address = "dynamic.${local.test_domain}"
  type    = "HTTP"

  port   = 80
  path   = "/health"
  method = "GET"

  dynamic "header" {
    for_each = local.custom_headers
    content {
      header = header.key
      values = header.value
    }
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
