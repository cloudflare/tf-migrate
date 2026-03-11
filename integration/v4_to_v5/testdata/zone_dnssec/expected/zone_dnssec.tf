locals {
  name_prefix = "cftftest"
}

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

resource "cloudflare_zone_dnssec" "example_active" {
  zone_id = var.cloudflare_zone_id
}

resource "cloudflare_zone_dnssec" "example_disabled" {
  zone_id = var.cloudflare_zone_id
}

resource "cloudflare_zone_dnssec" "example_minimal" {
  zone_id = var.cloudflare_zone_id
}

resource "cloudflare_zone_dnssec" "example_null_status" {
  zone_id = var.cloudflare_zone_id
}
