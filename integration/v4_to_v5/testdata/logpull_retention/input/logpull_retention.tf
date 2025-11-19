variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test Case 1: Basic logpull retention enabled
resource "cloudflare_logpull_retention" "enabled_zone" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}

# Test Case 2: Logpull retention disabled
resource "cloudflare_logpull_retention" "disabled_zone" {
  zone_id = var.cloudflare_zone_id
  enabled = false
}
