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

resource "cloudflare_account_member" "basic_am" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "v5-upgrade-test-user@cfapi.net"
  roles = [
    "e58cefd75d7adae0b761796c28815e5c",
    "a4154d230e664f8b3e6e5c95a8cc812f"
  ]
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
