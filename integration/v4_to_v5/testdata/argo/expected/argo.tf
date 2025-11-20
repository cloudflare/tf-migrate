variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}






resource "cloudflare_argo_smart_routing" "smart_only" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.smart_only
  to   = cloudflare_argo_smart_routing.smart_only
}
resource "cloudflare_argo_tiered_caching" "tiered_only" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.tiered_only
  to   = cloudflare_argo_tiered_caching.tiered_only
}
resource "cloudflare_argo_smart_routing" "both" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.both
  to   = cloudflare_argo_smart_routing.both
}
resource "cloudflare_argo_tiered_caching" "both_tiered" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.both
  to   = cloudflare_argo_tiered_caching.both_tiered
}
resource "cloudflare_argo_smart_routing" "neither" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"
}
moved {
  from = cloudflare_argo.neither
  to   = cloudflare_argo_smart_routing.neither
}
resource "cloudflare_argo_smart_routing" "with_lifecycle" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
  lifecycle {
    ignore_changes = [smart_routing]
  }
}
moved {
  from = cloudflare_argo.with_lifecycle
  to   = cloudflare_argo_smart_routing.with_lifecycle
}
resource "cloudflare_argo_smart_routing" "with_vars" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}
moved {
  from = cloudflare_argo.with_vars
  to   = cloudflare_argo_smart_routing.with_vars
}
