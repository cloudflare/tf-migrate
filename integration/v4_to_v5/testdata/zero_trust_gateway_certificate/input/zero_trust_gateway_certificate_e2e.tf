# E2E test data for zero_trust_gateway_certificate migration
# Minimal test set to avoid API rate limits (3 certs per 24 hours)

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (unused for this resource)"
  type        = string
  default     = ""
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (unused for this resource)"
  type        = string
  default     = ""
}

resource "cloudflare_zero_trust_gateway_certificate" "with_lifecycle" {
  account_id           = var.cloudflare_account_id
  gateway_managed      = true
  validity_period_days = 1826
  activate             = true

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [activate]
  }
}
