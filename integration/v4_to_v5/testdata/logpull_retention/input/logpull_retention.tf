# Integration Test for cloudflare_logpull_retention v4 → v5 Migration
# Simplified to single resource since logpull_retention is a zone-level setting

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

# Single logpull retention resource
# Tests the basic enabled → flag migration
resource "cloudflare_logpull_retention" "basic" {
  zone_id = var.cloudflare_zone_id
  enabled = true
}
