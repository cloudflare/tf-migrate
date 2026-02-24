locals {
  name_prefix = "cftftest"
}

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

# Zone DNSSEC with modified_on field (should be removed during migration)
# Tests: modified_on removal, status preservation from state
resource "cloudflare_zone_dnssec" "test" {
  zone_id     = var.cloudflare_zone_id
  modified_on = "2024-01-15T10:30:00Z"
}