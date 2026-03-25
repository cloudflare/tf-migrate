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

# Test Case 8: Multiple characteristics with varied order
resource "cloudflare_api_shield" "varied_order" {
  zone_id = var.cloudflare_zone_id




  auth_id_characteristics = [
    {
      type = "cookie"
      name = "auth_cookie"
    },
    {
      type = "header"
      name = "Authorization"
    },
    {
      type = "cookie"
      name = "session"
    },
    {
      type = "header"
      name = "X-Forwarded-For"
    }
  ]
}


variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
