variable "cloudflare_account_id" {
  type    = string
  default = "f037e56e89293a057740de681ac9abbe"
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (not used by Zero Trust resources)"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (not used by Zero Trust resources)"
  type        = string
}

locals {
  name_prefix = "cftftest"
}

# Test 1: Basic HTTP test
resource "cloudflare_zero_trust_dex_test" "http_basic" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-basic-http-test"
  description = "Test HTTP connectivity to example.com"
  interval    = "0h30m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://example.com"
    method = "GET"
  }
}

# Test 2: Traceroute test (no method field)
resource "cloudflare_zero_trust_dex_test" "traceroute" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-traceroute-to-dns"
  description = "Test network path to Google DNS"
  interval    = "1h0m0s"
  enabled     = true

  data = {
    kind = "traceroute"
    host = "8.8.8.8"
  }
}

# Test 3: Disabled test
resource "cloudflare_zero_trust_dex_test" "disabled" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-disabled-test"
  description = "Currently disabled for maintenance"
  interval    = "0h15m0s"
  enabled     = false

  data = {
    kind   = "http"
    host   = "https://internal.example.com"
    method = "GET"
  }
}

# Test 4: HTTP test with short interval
resource "cloudflare_zero_trust_dex_test" "http_frequent" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-frequent-http-test"
  description = "High-frequency monitoring"
  interval    = "0h5m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://api.example.com/health"
    method = "GET"
  }
}

# Test 5: Traceroute to hostname
resource "cloudflare_zero_trust_dex_test" "traceroute_hostname" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-traceroute-to-hostname"
  description = "Test path to cloudflare.com"
  interval    = "2h0m0s"
  enabled     = true

  data = {
    kind = "traceroute"
    host = "cloudflare.com"
  }
}

# Test 6: for_each with map
locals {
  dex_tests = {
    "production-api" = {
      host        = "https://api.prod.example.com"
      description = "Production API monitoring"
      interval    = "0h5m0s"
    }
    "staging-api" = {
      host        = "https://api.staging.example.com"
      description = "Staging API monitoring"
      interval    = "0h15m0s"
    }
    "dev-api" = {
      host        = "https://api.dev.example.com"
      description = "Dev API monitoring"
      interval    = "0h30m0s"
    }
  }
}

resource "cloudflare_zero_trust_dex_test" "api_tests" {
  for_each = local.dex_tests

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-api-test-${each.key}"
  description = each.value.description
  interval    = each.value.interval
  enabled     = true

  data = {
    kind   = "http"
    host   = each.value.host
    method = "GET"
  }
}

# Test 7: for_each with set
resource "cloudflare_zero_trust_dex_test" "dns_tests" {
  for_each = toset(["1.1.1.1", "8.8.8.8", "8.8.4.4"])

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-dns-test-${each.value}"
  description = "Test connectivity to ${each.value}"
  interval    = "1h0m0s"
  enabled     = true

  data = {
    kind = "traceroute"
    host = each.value
  }
}

# Test 8: count pattern
resource "cloudflare_zero_trust_dex_test" "regional_tests" {
  count = 3

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-regional-test-${count.index + 1}"
  description = "Test endpoint in region ${count.index + 1}"
  interval    = "0h30m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://region${count.index + 1}.example.com"
    method = "GET"
  }
}

# Test 9: Conditional creation
variable "enable_backup_tests" {
  type    = bool
  default = true
}

resource "cloudflare_zero_trust_dex_test" "backup_test" {
  count = var.enable_backup_tests ? 1 : 0

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-backup-system-test"
  description = "Test backup system availability"
  interval    = "4h0m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://backup.example.com/status"
    method = "GET"
  }
}

# Test 10: Using dynamic values
locals {
  monitoring_config = {
    interval_minutes = 30
    enabled          = true
  }
}

resource "cloudflare_zero_trust_dex_test" "dynamic_config" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-dynamic-configuration-test"
  description = "Test with dynamic configuration values"
  interval    = "0h${local.monitoring_config.interval_minutes}m0s"
  enabled     = local.monitoring_config.enabled

  data = {
    kind   = "http"
    host   = "https://metrics.example.com"
    method = "GET"
  }
}

# Test 11: HTTP test with HTTPS URL
resource "cloudflare_zero_trust_dex_test" "https_test" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-https-connectivity-test"
  description = "Verify HTTPS connectivity"
  interval    = "0h10m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://secure.example.com"
    method = "GET"
  }
}

# Test 12: Internal network test
resource "cloudflare_zero_trust_dex_test" "internal" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-internal-network-test"
  description = "Test internal resource availability"
  interval    = "0h5m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://internal.corp.example.com/healthcheck"
    method = "GET"
  }
}

# Test 13: Long description
resource "cloudflare_zero_trust_dex_test" "long_description" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-test-with-long-description"
  description = <<-EOT
    This is a comprehensive DEX test with a very detailed description
    that spans multiple lines and contains important information about
    what this test does, why it exists, and how it should be monitored.
  EOT
  interval    = "1h0m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://example.com/detailed"
    method = "GET"
  }
}

# Test 14: IPv6 traceroute
resource "cloudflare_zero_trust_dex_test" "ipv6_traceroute" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-ipv6-traceroute-test"
  description = "Test IPv6 connectivity"
  interval    = "2h0m0s"
  enabled     = true

  data = {
    kind = "traceroute"
    host = "2001:4860:4860::8888"
  }
}

# Test 15: API endpoint with path
resource "cloudflare_zero_trust_dex_test" "api_with_path" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-api-endpoint-test"
  description = "Test specific API endpoint"
  interval    = "0h10m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://api.example.com/v1/status"
    method = "GET"
  }
}
