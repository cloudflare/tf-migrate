# Variables (no defaults as per pattern)
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

# URL for custom pages - this URL contains the required tokens for all error page types
locals {
  custom_pages_url = "https://custom-pages-basic.terraform-provider-acceptance-testing.workers.dev/"
}

# Basic zone-scoped custom page with real URL
resource "cloudflare_custom_pages" "error_500" {
  zone_id = var.cloudflare_zone_id
  type    = "500_errors"
  url     = local.custom_pages_url
  state   = "customized"
}
