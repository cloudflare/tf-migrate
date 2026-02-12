# Test authenticated_origin_pulls migration

# Variables
variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}


# Basic zone-wide AOP
resource "cloudflare_authenticated_origin_pulls_settings" "zone_wide" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}

moved {
  from = cloudflare_authenticated_origin_pulls.zone_wide
  to   = cloudflare_authenticated_origin_pulls_settings.zone_wide
}
