# Integration test data for zero_trust_gateway_certificate migration
# Tests comprehensive Terraform patterns and edge cases

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (unused for this resource)"
  type        = string
  default     = ""
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (unused for this resource)"
  type        = string
  default     = ""
}

locals {
  name_prefix = "cftftest"
  account_id  = var.cloudflare_account_id
}

# Test Case 1: Gateway-managed certificate with all optional fields
resource "cloudflare_zero_trust_gateway_certificate" "gateway_managed_full" {
  account_id           = var.cloudflare_account_id
  validity_period_days = 1826
  activate             = true
}

# Test Case 2: Custom certificate with ID
resource "cloudflare_zero_trust_gateway_certificate" "custom_cert" {
  account_id = var.cloudflare_account_id
  activate   = false
}

# Test Case 3: Minimal gateway-managed certificate (uses defaults)
resource "cloudflare_zero_trust_gateway_certificate" "minimal" {
  account_id = var.cloudflare_account_id
}

# Test Case 4: Long validity period
resource "cloudflare_zero_trust_gateway_certificate" "long_validity" {
  account_id           = var.cloudflare_account_id
  validity_period_days = 7300
  activate             = true
}

# Test Case 5: Short validity period
resource "cloudflare_zero_trust_gateway_certificate" "short_validity" {
  account_id           = var.cloudflare_account_id
  validity_period_days = 365
  activate             = false
}

# Test Case 6: for_each with map
resource "cloudflare_zero_trust_gateway_certificate" "map_example" {
  for_each = {
    "cert1" = {
      validity = 1826
      activate = true
    }
    "cert2" = {
      validity = 3650
      activate = false
    }
    "cert3" = {
      validity = 5475
      activate = true
    }
  }

  account_id           = var.cloudflare_account_id
  validity_period_days = each.value.validity
  activate             = each.value.activate
}

# Test Case 7: for_each with set
resource "cloudflare_zero_trust_gateway_certificate" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
  ])

  account_id           = var.cloudflare_account_id
  validity_period_days = 1826
  activate             = false
}

# Test Case 8: count-based resources
resource "cloudflare_zero_trust_gateway_certificate" "counted" {
  count = 3

  account_id           = var.cloudflare_account_id
  validity_period_days = 1826
  activate             = count.index == 0 ? true : false
}

# Test Case 9: Conditional creation (enabled)
resource "cloudflare_zero_trust_gateway_certificate" "conditional_enabled" {
  count = 1

  account_id = var.cloudflare_account_id
  activate   = true
}

# Test Case 10: Conditional creation (disabled - won't be created)
resource "cloudflare_zero_trust_gateway_certificate" "conditional_disabled" {
  count = 0

  account_id = var.cloudflare_account_id
}

# Test Case 11: Using locals
resource "cloudflare_zero_trust_gateway_certificate" "with_locals" {
  account_id = local.account_id
  activate   = true
}

# Test Case 12: Lifecycle meta-arguments
resource "cloudflare_zero_trust_gateway_certificate" "with_lifecycle" {
  account_id           = var.cloudflare_account_id
  validity_period_days = 1826

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [activate]
  }
}

# Test Case 13: Custom cert with minimal fields
resource "cloudflare_zero_trust_gateway_certificate" "custom_minimal" {
  account_id = var.cloudflare_account_id
}

# Test Case 14: Using string interpolation
resource "cloudflare_zero_trust_gateway_certificate" "with_interpolation" {
  account_id           = var.cloudflare_account_id
  validity_period_days = 1826
  activate             = true
}

# Test Case 15: Maximum validity period
resource "cloudflare_zero_trust_gateway_certificate" "max_validity" {
  account_id           = var.cloudflare_account_id
  validity_period_days = 10950
  activate             = false
}
