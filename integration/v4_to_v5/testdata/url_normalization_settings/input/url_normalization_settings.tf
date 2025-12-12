# Terraform v4 to v5 Migration Integration Tests
# Resource: cloudflare_url_normalization_settings
#
# MINIMAL test suite for this resource (simplest possible migration)
# Note: This is a SINGLETON resource - only ONE instance per zone
# Target: 1 resource instance

# Variables (provided by test infrastructure)
variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test: Single instance (singleton resource - only one per zone)
resource "cloudflare_url_normalization_settings" "test" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"
}

# Total: 1 resource instance
# Note: url_normalization_settings is a zone-level singleton
# There can only be one configuration per zone
