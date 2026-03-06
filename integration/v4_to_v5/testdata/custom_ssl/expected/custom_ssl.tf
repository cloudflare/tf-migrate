# Terraform v4 to v5 Migration Integration Test - Custom SSL
# This file contains 20+ custom_ssl instances using various Terraform patterns.
#
# Key v4 → v5 changes under test:
#   - custom_ssl_options block → flat top-level attributes
#   - geo_restrictions string → { label = "..." } nested attribute
#   - custom_ssl_priority blocks → removed

# Variables without defaults (passed at runtime)
variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_account_id" {
  type = string
}

# Resource-specific variables with defaults
variable "cert_pem" {
  type    = string
  default = "-----BEGIN CERTIFICATE-----\nMIID..."
}

variable "key_pem" {
  type    = string
  default = "-----BEGIN RSA PRIVATE KEY-----\nMIIE..."
}

# Locals
locals {
  name_prefix = "cftftest"

  geo_regions = ["us", "eu", "highest_security"]

  ssl_configs = {
    alpha = {
      bundle_method = "ubiquitous"
      type          = "legacy_custom"
      geo           = "us"
    }
    beta = {
      bundle_method = "force"
      type          = "legacy_custom"
      geo           = "eu"
    }
    gamma = {
      bundle_method = "optimal"
      type          = "sni_custom"
      geo           = ""
    }
  }
}

# Pattern 1: Full config with all fields including geo_restrictions
resource "cloudflare_custom_ssl" "full" {
  zone_id = var.cloudflare_zone_id

  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = "ubiquitous"
  type          = "legacy_custom"
  geo_restrictions = {
    label = "us"
  }
}

# Pattern 2: Minimal config - only certificate and private_key
resource "cloudflare_custom_ssl" "minimal" {
  zone_id = var.cloudflare_zone_id

  certificate = var.cert_pem
  private_key = var.key_pem
}

# Pattern 3: Config without geo_restrictions but with bundle_method and type
resource "cloudflare_custom_ssl" "no_geo" {
  zone_id = var.cloudflare_zone_id

  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = "force"
  type          = "legacy_custom"
}

# Pattern 4: Config with custom_ssl_priority (must be removed on migration)
resource "cloudflare_custom_ssl" "with_priority" {
  zone_id = var.cloudflare_zone_id



  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = "ubiquitous"
  type          = "legacy_custom"
}

# Pattern 5: geo_restrictions with EU region
resource "cloudflare_custom_ssl" "eu_geo" {
  zone_id = var.cloudflare_zone_id

  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = "ubiquitous"
  type          = "legacy_custom"
  geo_restrictions = {
    label = "eu"
  }
}

# Pattern 6: geo_restrictions with highest_security region
resource "cloudflare_custom_ssl" "highest_security_geo" {
  zone_id = var.cloudflare_zone_id

  certificate = var.cert_pem
  private_key = var.key_pem
  geo_restrictions = {
    label = "highest_security"
  }
}

# Pattern 7: for_each with maps (3 resources)
resource "cloudflare_custom_ssl" "by_config" {
  for_each = local.ssl_configs

  zone_id = var.cloudflare_zone_id

  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = each.value.bundle_method
  type          = each.value.type
}

# Pattern 8: for_each with sets (3 resources)
resource "cloudflare_custom_ssl" "by_geo" {
  for_each = toset(local.geo_regions)

  zone_id = var.cloudflare_zone_id

  certificate = var.cert_pem
  private_key = var.key_pem
  geo_restrictions = {
    label = each.value
  }
}

# Pattern 9: count-based resources (3 resources)
resource "cloudflare_custom_ssl" "count_based" {
  count = 3

  zone_id = var.cloudflare_zone_id

  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = element(["ubiquitous", "force", "optimal"], count.index)
  type          = "legacy_custom"
}

# Pattern 10: Conditional resource creation
resource "cloudflare_custom_ssl" "conditional" {
  count = length(var.cloudflare_zone_id) > 0 ? 1 : 0

  zone_id = var.cloudflare_zone_id

  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = "ubiquitous"
  geo_restrictions = {
    label = "us"
  }
}

# Pattern 11: Lifecycle meta-arguments
resource "cloudflare_custom_ssl" "with_lifecycle" {
  zone_id = var.cloudflare_zone_id


  lifecycle {
    create_before_destroy = true
    prevent_destroy       = false
  }
  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = "ubiquitous"
  type          = "legacy_custom"
}

# Pattern 12: Terraform functions (base64decode, upper, etc.)
resource "cloudflare_custom_ssl" "with_functions" {
  zone_id = var.cloudflare_zone_id

  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = lower("UBIQUITOUS")
  type          = "legacy_custom"
  geo_restrictions = {
    label = lower("US")
  }
}

# Pattern 13: sni_custom type
resource "cloudflare_custom_ssl" "sni_custom" {
  zone_id = var.cloudflare_zone_id

  certificate = var.cert_pem
  private_key = var.key_pem
  type        = "sni_custom"
}

# Pattern 14: optimal bundle method
resource "cloudflare_custom_ssl" "optimal_bundle" {
  zone_id = var.cloudflare_zone_id

  certificate   = var.cert_pem
  private_key   = var.key_pem
  bundle_method = "optimal"
  type          = "legacy_custom"
}

# Total: 22+ resources (counting for_each/count expansions)
