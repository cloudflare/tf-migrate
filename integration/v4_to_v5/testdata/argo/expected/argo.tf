resource "cloudflare_argo_smart_routing" "smart_routing_example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.smart_routing_example
  to   = cloudflare_argo_smart_routing.smart_routing_example
}
resource "cloudflare_argo_tiered_caching" "tiered_caching_example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.tiered_caching_example
  to   = cloudflare_argo_tiered_caching.tiered_caching_example
}
resource "cloudflare_argo_smart_routing" "both_example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.both_example
  to   = cloudflare_argo_smart_routing.both_example
}
resource "cloudflare_argo_tiered_caching" "both_example_tiered" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.both_example
  to   = cloudflare_argo_tiered_caching.both_example_tiered
}
resource "cloudflare_argo_smart_routing" "default_example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"
}
moved {
  from = cloudflare_argo.default_example
  to   = cloudflare_argo_smart_routing.default_example
}
resource "cloudflare_argo_smart_routing" "lifecycle_example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
  lifecycle {
    prevent_destroy = true
  }
}
moved {
  from = cloudflare_argo.lifecycle_example
  to   = cloudflare_argo_smart_routing.lifecycle_example
}
