# E2E test for regional_tiered_cache migration
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
# Note: Regional Tiered Cache requires Enterprise plan
resource "cloudflare_regional_tiered_cache" "test" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}
