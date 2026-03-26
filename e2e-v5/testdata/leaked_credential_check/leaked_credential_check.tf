# E2E test for leaked_credential_check migration
# This resource is a singleton (one per zone)

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

resource "cloudflare_leaked_credential_check" "test" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
