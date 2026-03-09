# Test authenticated_origin_pulls migration

# Variables
variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}

# Basic zone-wide AOP
resource "cloudflare_authenticated_origin_pulls" "zone_wide" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}
