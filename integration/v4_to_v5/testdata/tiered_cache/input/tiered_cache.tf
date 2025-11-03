resource "cloudflare_tiered_cache" "smart" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  cache_type = "smart"
}

resource "cloudflare_tiered_cache" "generic" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  cache_type = "generic"
}

resource "cloudflare_tiered_cache" "off" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  cache_type = "off"
}
