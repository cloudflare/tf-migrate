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

variable "cache_type" {
  type    = string
  default = "off"
}






resource "cloudflare_tiered_cache" "smart" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}

resource "cloudflare_argo_tiered_caching" "smart" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}

resource "cloudflare_tiered_cache" "off" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

resource "cloudflare_argo_tiered_caching" "off" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

resource "cloudflare_tiered_cache" "generic" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

resource "cloudflare_argo_tiered_caching" "generic" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}

resource "cloudflare_tiered_cache" "variable" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

resource "cloudflare_argo_tiered_caching" "variable" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

resource "cloudflare_tiered_cache" "lifecycle" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_argo_tiered_caching" "lifecycle" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
  lifecycle {
    create_before_destroy = true
  }
}
