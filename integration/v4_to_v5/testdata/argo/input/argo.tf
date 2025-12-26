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

resource "cloudflare_argo" "both_with_lifecycle" {
  zone_id        = var.cloudflare_zone_id
  smart_routing  = "on"
  tiered_caching = "on"

  lifecycle {
    ignore_changes = [smart_routing, tiered_caching]
  }
}
