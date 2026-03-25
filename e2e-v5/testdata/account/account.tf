locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
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

resource "cloudflare_account" "basic_account" {
  name = "${local.name_prefix}-account-e2e"
  settings = {
    enforce_twofactor = false
  }
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
