variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

resource "cloudflare_argo" "smart_only" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"
}

resource "cloudflare_argo" "tiered_only" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  tiered_caching = "on"
}

resource "cloudflare_argo" "both" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing  = "on"
  tiered_caching = "on"
}

resource "cloudflare_argo" "neither" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}

resource "cloudflare_argo" "with_lifecycle" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing]
  }
}

resource "cloudflare_argo" "with_vars" {
  zone_id       = var.cloudflare_zone_id
  smart_routing = "on"
}
