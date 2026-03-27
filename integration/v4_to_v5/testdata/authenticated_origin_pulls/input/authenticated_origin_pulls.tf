# Test authenticated_origin_pulls migration

# Variables
variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}

# Basic zone-wide AOP
resource "cloudflare_authenticated_origin_pulls" "zone_wide" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}

# TKT-001: cert_id should reference the renamed resource type
# In v4: cloudflare_authenticated_origin_pulls_certificate (per-hostname type)
# In v5: cloudflare_authenticated_origin_pulls_hostname_certificate
resource "cloudflare_authenticated_origin_pulls_certificate" "hostname_cert" {
  zone_id     = var.cloudflare_zone_id
  certificate = "-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqhkiG\n-----END CERTIFICATE-----\n"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIBIjANBgkqhkiG\n-----END PRIVATE KEY-----\n"
  type        = "per-hostname"
}

resource "cloudflare_authenticated_origin_pulls" "hostname_aop" {
  zone_id                                = var.cloudflare_zone_id
  authenticated_origin_pulls_certificate = cloudflare_authenticated_origin_pulls_certificate.hostname_cert.id
  hostname                               = "example.cloudflare.com"
  enabled                                = true
}

# TKT-001: for_each variant — cert_id referencing per-hostname cert
locals {
  hostnames = { "api" = "api.example.com", "web" = "web.example.com" }
}

resource "cloudflare_authenticated_origin_pulls_certificate" "multi_cert" {
  for_each    = local.hostnames
  zone_id     = var.cloudflare_zone_id
  certificate = "-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqhkiG\n-----END CERTIFICATE-----\n"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIBIjANBgkqhkiG\n-----END PRIVATE KEY-----\n"
  type        = "per-hostname"
}

resource "cloudflare_authenticated_origin_pulls" "multi_aop" {
  for_each                               = local.hostnames
  zone_id                                = var.cloudflare_zone_id
  authenticated_origin_pulls_certificate = cloudflare_authenticated_origin_pulls_certificate.multi_cert[each.key].id
  hostname                               = each.value
  enabled                                = true
}
