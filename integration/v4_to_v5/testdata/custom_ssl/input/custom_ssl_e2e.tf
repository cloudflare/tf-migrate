# E2E test file for custom_ssl.
#
# Uses the tls provider to dynamically generate certificates for the test domain.
# This allows the test to work with any zone configured via var.cloudflare_domain.
# Note: The tls provider is included by default in the generated versions.tf.
#
# Key migration paths covered:
#   - geo_restrictions string -> { label = "..." } nested attribute  (full)
#   - minimal config (no optional fields)                           (minimal)
#   - custom_ssl_priority blocks removed on migration               (with_priority)

variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_domain" {
  type = string
}

locals {
  name_prefix = "cftftest"
}

# Generate a private key for the certificate
resource "tls_private_key" "custom_ssl" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

# Generate a self-signed certificate for the test domain
resource "tls_self_signed_cert" "custom_ssl" {
  private_key_pem = tls_private_key.custom_ssl.private_key_pem

  subject {
    common_name = var.cloudflare_domain
  }

  # Valid for 1 year
  validity_period_hours = 8760

  # Certificate for the domain and wildcard subdomain
  dns_names = [
    var.cloudflare_domain,
    "*.${var.cloudflare_domain}",
  ]

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
  ]
}

# Pattern 1: Full config - exercises geo_restrictions string -> { label } transform
resource "cloudflare_custom_ssl" "full" {
  zone_id = var.cloudflare_zone_id

  custom_ssl_options {
    certificate      = tls_self_signed_cert.custom_ssl.cert_pem
    private_key      = tls_private_key.custom_ssl.private_key_pem
    bundle_method    = "force"
    type             = "legacy_custom"
    geo_restrictions = "us"
  }
}

# Pattern 2: Minimal config - no geo_restrictions or priority blocks
resource "cloudflare_custom_ssl" "minimal" {
  zone_id = var.cloudflare_zone_id

  custom_ssl_options {
    certificate   = tls_self_signed_cert.custom_ssl.cert_pem
    private_key   = tls_private_key.custom_ssl.private_key_pem
    bundle_method = "force"
    type          = "legacy_custom"
  }
}

# Pattern 3: custom_ssl_priority blocks - must be removed on migration
resource "cloudflare_custom_ssl" "with_priority" {
  zone_id = var.cloudflare_zone_id

  custom_ssl_options {
    certificate   = tls_self_signed_cert.custom_ssl.cert_pem
    private_key   = tls_private_key.custom_ssl.private_key_pem
    bundle_method = "force"
    type          = "legacy_custom"
  }

  custom_ssl_priority {
    id       = "cert-001"
    priority = 1
  }

  custom_ssl_priority {
    id       = "cert-002"
    priority = 2
  }
}
