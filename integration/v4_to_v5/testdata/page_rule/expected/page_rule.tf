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

resource "cloudflare_page_rule" "with_server_side_exclude" {
  zone_id  = var.cloudflare_zone_id
  target   = "cftftest.com/server-side/*"
  priority = 6

  status = "active"
  actions = {
    cache_level = "bypass"
  }
}

resource "cloudflare_page_rule" "with_query_string_ignore" {
  zone_id  = var.cloudflare_zone_id
  target   = "cftftest.com/query-ignore/*"
  priority = 7

  status = "active"
  actions = {
    cache_level = "cache_everything"
    cache_key_fields = {
      query_string = {
        include = []
        exclude = ["*"]
      }
      user = {
        device_type = true
        geo         = true
        lang        = false
      }
    }
  }
}

variable "cftftest_cache_ttl" {
  type    = number
  default = 1800
}

locals {
  cftftest_status_codes = "301"
}

# Regression coverage: expression values in cache_ttl_by_status used to collapse
# to their first identifier, rendering as "200-299" = "var" and "local" = "max".
resource "cloudflare_page_rule" "with_cache_ttl_expressions" {
  zone_id  = var.cloudflare_zone_id
  target   = "cftftest.com/expr/*"
  priority = 5

  status = "active"
  actions = {
    cache_level = "cache_everything"
    cache_ttl_by_status = {
      "200-299"                     = var.cftftest_cache_ttl
      "404"                         = "300"
      (local.cftftest_status_codes) = max(300, var.cftftest_cache_ttl)
    }
  }
}
