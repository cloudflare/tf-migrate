# E2E test for authenticated_origin_pulls migration
# NOTE: Only includes the zone-wide AOP resource for e2e testing.
# The cert_id reference fix (per-hostname certificates) is validated
# by integration tests only — the API rejects fake certificate data and real
# certificates cannot be included in testdata.

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}

# Basic zone-wide AOP — sufficient to validate migration
resource "cloudflare_authenticated_origin_pulls" "zone_wide" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}
