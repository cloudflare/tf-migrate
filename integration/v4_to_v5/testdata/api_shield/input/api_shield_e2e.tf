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

# Test Case 8: Multiple characteristics with varied order
resource "cloudflare_api_shield" "varied_order" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "cookie"
    name = "auth_cookie"
  }

  auth_id_characteristics {
    type = "header"
    name = "Authorization"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "session"
  }

  auth_id_characteristics {
    type = "header"
    name = "X-Forwarded-For"
  }
}

