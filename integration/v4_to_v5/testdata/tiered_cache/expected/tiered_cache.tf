resource "cloudflare_tiered_cache" "smart" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}


resource "cloudflare_tiered_cache" "off" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"
}
resource "cloudflare_argo_tiered_caching" "generic" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_tiered_cache.generic
  to   = cloudflare_argo_tiered_caching.generic
}
