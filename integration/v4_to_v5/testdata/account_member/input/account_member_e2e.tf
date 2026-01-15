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

resource "cloudflare_account_member" "basic_am" {
    account_id = var.cloudflare_account_id
    status = "accepted"
    email_address = "terraform-test-user-b@cfapi.net"
    role_ids = [
        "e58cefd75d7adae0b761796c28815e5c",
        "a4154d230e664f8b3e6e5c95a8cc812f"
    ]
}
