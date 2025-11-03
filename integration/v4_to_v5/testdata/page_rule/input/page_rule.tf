# Page rule example - no transformation needed
resource "cloudflare_page_rule" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  target  = "example.com/*"
  priority = 1

  actions {
    cache_level         = "cache_everything"
    edge_cache_ttl      = 7200
    browser_cache_ttl   = 3600
    always_use_https    = true
  }
}
