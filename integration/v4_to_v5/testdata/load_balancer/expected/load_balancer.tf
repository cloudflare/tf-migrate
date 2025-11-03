# Load balancer example - state transformation focused
# Key transformations:
# - fallback_pool_id → fallback_pool (state only)
# - default_pool_ids → default_pools (state only)
# Config transformation is minimal

resource "cloudflare_load_balancer" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "example-lb.example.com"

  fallback_pool_id = "pool-id-fallback"
  default_pool_ids = ["pool-id-1", "pool-id-2"]

  description      = "Example load balancer"
  ttl              = 30
  steering_policy  = "geo"
  proxied          = true
  session_affinity = "cookie"

  country_pools {
    country  = "US"
    pool_ids = ["pool-id-us"]
  }

  region_pools {
    region   = "WNAM"
    pool_ids = ["pool-id-wnam"]
  }
}
