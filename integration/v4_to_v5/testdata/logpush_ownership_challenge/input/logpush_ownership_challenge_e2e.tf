# E2E test for logpush_ownership_challenge migration
# This resource has 3rd party storage requirements, and you can only have 1 per configured storage.
# The integration tests have various storage URIs

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
resource "cloudflare_logpush_ownership_challenge" "gcs_zone" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = "gs://cf-terraform-provider-acct-test/ownership_challenges"
}
