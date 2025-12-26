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

resource "cloudflare_tiered_cache" "generic_with_lifecycle" {
  zone_id    = var.cloudflare_zone_id
  cache_type = "generic"

  lifecycle {
    create_before_destroy = true
  }
}
