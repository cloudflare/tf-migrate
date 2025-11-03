# Argo resource with smart_routing only
resource "cloudflare_argo" "smart_routing_example" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"
}

# Argo resource with tiered_caching only
resource "cloudflare_argo" "tiered_caching_example" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  tiered_caching = "on"
}

# Argo resource with both smart_routing and tiered_caching
resource "cloudflare_argo" "both_example" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing  = "on"
  tiered_caching = "on"
}

# Argo resource with no attributes (defaults to smart_routing off)
resource "cloudflare_argo" "default_example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}

# Argo resource with lifecycle block
resource "cloudflare_argo" "lifecycle_example" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"

  lifecycle {
    prevent_destroy = true
  }
}
