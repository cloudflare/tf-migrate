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
  zone_id = var.cloudflare_zone_id
  target  = "cf-tf-test.com/minimal"
  priority = 1
  status = "active"

  actions {
    cache_level = "bypass"
  }
}

resource "cloudflare_page_rule" "with_cache_ttl" {
  zone_id = var.cloudflare_zone_id
  target  = "cf-tf-test.com/cached/*"
  priority = 2

  actions {
    cache_level = "cache_everything"
    cache_ttl_by_status {
      codes = "200"
      ttl   = 3600
    }
    cache_ttl_by_status {
      codes = "404"
      ttl   = 300
    }
  }
}

resource "cloudflare_page_rule" "with_forwarding" {
  zone_id = var.cloudflare_zone_id
  target  = "cf-tf-test.com/old/*"
  priority = 3

  actions {
    forwarding_url {
      url         = "https://cf-tf-test.com/new/$1"
      status_code = 301
    }
  }
}

resource "cloudflare_page_rule" "with_cache_key_fields" {
  zone_id = var.cloudflare_zone_id
  target  = "cf-tf-test.com/api/*"
  priority = 4

  actions {
    cache_level = "cache_everything"
    cache_key_fields {
      cookie {
        check_presence = ["sessionid"]
      }
      host {
        resolved = true
      }
      query_string {
        exclude = ["utm_*"]
      }
      user {
        device_type = true
        geo         = false
      }
    }
  }
}

resource "cloudflare_page_rule" "with_deprecated_fields" {
  zone_id = var.cloudflare_zone_id
  target  = "cf-tf-test.com/legacy/*"
  priority = 5

  actions {
    cache_level = "bypass"
    # These fields should be removed in v5
    minify {
      css  = "on"
      html = "off"
      js   = "on"
    }
    disable_railgun = false
  }
}
