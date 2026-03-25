variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (unused)"
  type        = string
  default     = ""
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (unused)"
  type        = string
  default     = ""
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}

locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
}

# Only 1 gateway-managed certificate — API rate limit is 3 per 24 hours account-wide.
# Keep to minimum to avoid exhausting the quota across concurrent or repeated runs.
resource "cloudflare_zero_trust_gateway_certificate" "managed" {
  account_id           = var.cloudflare_account_id
  validity_period_days = 1826
  activate             = true

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [activate]
  }
}
