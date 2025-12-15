variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cache_type" {
  type        = string
  default = "off"
}

resource "cloudflare_tiered_cache" "smart" {
  zone_id    = var.cloudflare_zone_id
  cache_type = "smart"
}

resource "cloudflare_tiered_cache" "off" {
  zone_id    = var.cloudflare_zone_id
  cache_type = "off"
}

resource "cloudflare_tiered_cache" "generic" {
  zone_id    = var.cloudflare_zone_id
  cache_type = "generic"
}

resource "cloudflare_tiered_cache" "variable" {
  zone_id    = var.cloudflare_zone_id
  cache_type = var.cache_type
}

resource "cloudflare_tiered_cache" "lifecycle" {
  zone_id    = var.cloudflare_zone_id
  cache_type = "generic"

  lifecycle {
    create_before_destroy = true
  }
}
