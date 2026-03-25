# Test authenticated_origin_pulls migration

# Variables
variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}


# Basic zone-wide AOP
resource "cloudflare_authenticated_origin_pulls_settings" "zone_wide" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}


variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
