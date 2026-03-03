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

locals {
  normalized_observatory_domain = format(
    "%s/",
    trimsuffix(
      trimprefix(
        trimprefix(trimspace(var.cloudflare_domain), "https://"),
        "http://",
      ),
      "/",
    ),
  )
}

resource "cloudflare_observatory_scheduled_test" "test_e2e" {
  zone_id   = var.cloudflare_zone_id
  url       = local.normalized_observatory_domain
  region    = "us-central1"
  frequency = "DAILY"
}
