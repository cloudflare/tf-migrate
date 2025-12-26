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

resource "cloudflare_argo_smart_routing" "both_with_lifecycle" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
  lifecycle {
    ignore_changes = [value]
  }
}
moved {
  from = cloudflare_argo.both_with_lifecycle
  to   = cloudflare_argo_smart_routing.both_with_lifecycle
}
resource "cloudflare_argo_tiered_caching" "both_with_lifecycle_tiered" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
  lifecycle {
    ignore_changes = [value]
  }
}
