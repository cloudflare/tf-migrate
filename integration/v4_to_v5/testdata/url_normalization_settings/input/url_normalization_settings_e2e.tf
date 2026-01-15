# E2E test for url_normalization_settings migration
# This is a simplified version for actual API testing (singleton resource)

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

# Test: Single instance (singleton resource - only one per zone)
# Note: This is a SINGLETON resource - only ONE instance per zone
resource "cloudflare_url_normalization_settings" "test" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"
}
