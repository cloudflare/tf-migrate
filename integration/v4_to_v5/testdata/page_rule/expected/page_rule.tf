# ========================================
# Variables
# ========================================
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

resource "cloudflare_page_rule" "minimal" {
  zone_id  = var.cloudflare_zone_id
  target   = "cftftest.com/minimal"
  priority = 1
  status   = "active"

  actions = {
    cache_level = "bypass"
  }
}

resource "cloudflare_page_rule" "with_cache_ttl" {
  zone_id  = var.cloudflare_zone_id
  target   = "cftftest.com/cached/*"
  priority = 2

  status = "active"
  actions = {
    cache_level = "cache_everything"
    cache_ttl_by_status = {
      "200" = "3600"
      "404" = "300"
    }
  }
}

resource "cloudflare_page_rule" "with_forwarding" {
  zone_id  = var.cloudflare_zone_id
  target   = "cftftest.com/old/*"
  priority = 3

  status = "active"
  actions = {
    forwarding_url = {
      url         = "https://cftftest.com/new/$1"
      status_code = 301
    }
  }
}

resource "cloudflare_page_rule" "with_cache_key_fields" {
  zone_id  = var.cloudflare_zone_id
  target   = "cftftest.com/api/*"
  priority = 4

  status = "active"
  actions = {
    cache_level = "cache_everything"
    cache_key_fields = {
      cookie = {
        check_presence = ["sessionid"]
      }
      host = {
        resolved = true
      }
      query_string = {
        exclude = ["utm_*"]
      }
      user = {
        device_type = true
        geo         = false
        lang        = false
      }
    }
  }
}

resource "cloudflare_page_rule" "with_deprecated_fields" {
  zone_id  = var.cloudflare_zone_id
  target   = "cftftest.com/legacy/*"
  priority = 5

  status = "active"
  actions = {
    cache_level = "bypass"
  }
}
