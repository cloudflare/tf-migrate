# Variables (no defaults as per pattern)
variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Basic zone-scoped custom page with real URL
resource "cloudflare_custom_pages" "error_500" {
  zone_id = var.cloudflare_zone_id
  type    = "500_errors"
  url     = "https://custom-pages-500-error-for-e2e.terraform-testing-a09.workers.dev/"
  state   = "customized"
}