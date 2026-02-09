# Test authenticated_origin_pulls_certificate migration

# Per-zone certificate
resource "cloudflare_authenticated_origin_pulls_certificate" "zone_cert" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  certificate = "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----"
  type        = "per-zone"
}

# Per-hostname certificate
resource "cloudflare_authenticated_origin_pulls_certificate" "hostname_cert" {
  zone_id     = "1234567890abcdef1234567890abcdef"
  certificate = "-----BEGIN CERTIFICATE-----\nABCD...\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nXYZ...\n-----END PRIVATE KEY-----"
  type        = "per-hostname"
}
