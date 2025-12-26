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

# NOTE: These resources are commented out because they require mTLS certificates
# to be uploaded to Cloudflare first. Uncomment and configure certificates if needed.

# # Minimal E2E test - account-level with single settings block
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "e2e_account" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "${var.cloudflare_domain}"
    china_network                 = false
    client_certificate_forwarding = false
  }
}

# Minimal E2E test - zone-level with single settings block
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "e2e_zone" {
  zone_id = var.cloudflare_zone_id

  settings {
    hostname                      = "${var.cloudflare_domain}"
    china_network                 = false
    client_certificate_forwarding = true
  }
}
