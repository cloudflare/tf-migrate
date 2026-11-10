locals {
  name_prefix = "cftftest"
}

# ========================================
# Variables
# ========================================
variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_domain" {
  type        = string
  description = "Domain for testing"
}

# Test Case 2: Ruleset with headers transformation
resource "cloudflare_ruleset" "with_headers" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-headers-ruleset"
  kind        = "zone"
  phase       = "http_request_late_transform"
  description = "Ruleset with header transformations"

  rules {
    action      = "rewrite"
    expression  = "true"
    description = "Modify request headers"
    enabled     = true

    action_parameters {
      headers {
        name      = "X-Custom-Header"
        operation = "set"
        value     = "custom-value"
      }

      headers {
        name       = "X-Dynamic-Header"
        operation  = "set"
        expression = "cf.bot_management.score"
      }

      headers {
        name      = "X-Remove-Header"
        operation = "remove"
      }
    }
  }
}

# Test Case 4: Block action with custom response
resource "cloudflare_ruleset" "block_with_response" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-block-ruleset"
  kind        = "zone"
  phase       = "http_request_firewall_custom"
  description = "Ruleset with block action and custom response"

  rules {
    action      = "block"
    expression  = "(http.request.uri.path contains \"/admin\")"
    description = "Block admin access with custom response"
    enabled     = true

    action_parameters {
      response {
        content      = "{\"error\": \"Access denied\"}"
        content_type = "application/json"
        status_code  = 403
      }
    }
  }
}

# Test Case 5: Redirect action with dynamic target
resource "cloudflare_ruleset" "redirect" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-redirect-ruleset"
  kind        = "zone"
  phase       = "http_request_dynamic_redirect"
  description = "Ruleset with redirect action"

  rules {
    action      = "redirect"
    expression  = "(http.request.uri.path eq \"/old-page\")"
    description = "Redirect old page to new location"
    enabled     = true

    action_parameters {
      from_value {
        status_code           = 301
        preserve_query_string = true

        target_url {
          value = "https://example.com/new-page"
        }
      }
    }
  }
}

# Test Case 6: Rewrite action with URI modification
resource "cloudflare_ruleset" "rewrite_uri" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-rewrite-uri-ruleset"
  kind        = "zone"
  phase       = "http_request_transform"
  description = "Ruleset with URI rewriting"

  rules {
    action      = "rewrite"
    expression  = "(http.request.uri.path contains \"/api/v1/\")"
    description = "Rewrite API v1 to v2"
    enabled     = true

    action_parameters {
      uri {
        path {
          value = "/api/v2/endpoint"
        }

        query {
          expression = "concat(\"version=2&\", http.request.uri.query)"
        }
      }
    }
  }
}

# Test Case 8: Set config action with multiple settings
resource "cloudflare_ruleset" "set_config" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-config-ruleset"
  kind        = "zone"
  phase       = "http_config_settings"
  description = "Ruleset with configuration overrides"

  rules {
    action      = "set_config"
    expression  = "(http.host eq \"staging.example.com\")"
    description = "Apply staging configuration"
    enabled     = true

    action_parameters {
      automatic_https_rewrites = true
      email_obfuscation        = true
      security_level           = "medium"
      ssl                      = "strict"

      autominify {
        css  = true
        html = true
        js   = true
      }
    }
  }
}

# Test Case 11: Log custom fields transformation
resource "cloudflare_ruleset" "log_custom_fields" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-log-custom-fields"
  kind        = "zone"
  phase       = "http_log_custom_fields"
  description = "Ruleset with custom field logging"

  rules {
    action      = "log_custom_field"
    expression  = "true"
    description = "Log custom fields from request"
    enabled     = true

    action_parameters {
      cookie_fields   = ["session_id", "user_token", "preferences"]
      request_fields  = ["cf.bot_score", "cf.threat_score"]
      response_fields = ["cf.cache_status"]
    }
  }
}

# Test Case: WAF managed ruleset with skip rules (action_parameters.rules map)
# Tests conversion of comma-separated rule IDs to lists
resource "cloudflare_ruleset" "waf_with_skip_rules" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-waf-skip"
  kind        = "zone"
  phase       = "http_request_firewall_managed"
  description = "WAF managed ruleset with skip rules"

  rules {
    action = "skip"
    action_parameters {
      rules = {
        efb7b8c949ac4650a09736fc376e9aee = "0c054d4e4dd5455c9ff8f01efe5abb10,3536f964ccc345308b6445e8dc29b753,e7e4b386797e417c998d872956c390a1"
      }
    }
    description = "Exempt firewall rule expressions from log4j WAF block"
    enabled     = true
    expression  = "(http.request.uri.path contains \"/filters\")"
    logging {
      enabled = true
    }
  }

  rules {
    action = "skip"
    action_parameters {
      rules = {
        efb7b8c949ac4650a09736fc376e9aee = "5f6744fa026a4638bda5b3d7d5e015dd"
      }
    }
    description = "Single rule skip"
    enabled     = true
    expression  = "(http.request.uri.path contains \"/graphql\")"
    logging {
      enabled = true
    }
  }

  rules {
    action = "skip"
    action_parameters {
      rules = {
        "4814384a9e5d4991b9815dcfc25d2f1f" = "6179ae15870a4bb7b2d480d4843b323c"
        "efb7b8c949ac4650a09736fc376e9aee" = "0242110ae62e44028a13bf4834780914,6b1cc72dff9746469d4695a474430f12"
      }
    }
    description = "Multi-ruleset skip with mixed values"
    enabled     = true
    expression  = "(ip.src in {64.39.96.0/20})"
    logging {
      enabled = true
    }
  }

  rules {
    action = "skip"
    action_parameters {
      ruleset = "current"
    }
    description = "Skip current ruleset"
    enabled     = true
    expression  = "(http.request.uri.path contains \"/exempt\")"
    logging {
      enabled = true
    }
  }

  rules {
    action      = "execute"
    enabled     = true
    description = "Execute managed ruleset"
    expression  = "true"
    action_parameters {
      id = "efb7b8c949ac4650a09736fc376e9aee"
      overrides {
        rules {
          enabled = false
          id      = "5de7edfa648c4d6891dc3e7f84534ffa"
        }
        rules {
          enabled = false
          id      = "e3a567afc347477d9702d9047e97d760"
        }
      }
    }
  }
}

# Test Case 12: Cache settings with cache_reserve and query_string
resource "cloudflare_ruleset" "cache_with_reserve" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-cache-reserve"
  kind        = "zone"
  phase       = "http_request_cache_settings"
  description = "Cache settings with cache reserve"

  rules {
    action      = "set_cache_settings"
    expression  = "(http.request.uri.path contains \"/static\")"
    description = "Cache static content with reserve"
    enabled     = true

    action_parameters {
      cache = true

      edge_ttl {
        mode    = "override_origin"
        default = 86400

        status_code_ttl {
          status_code = 200
          value       = 2592000
        }

        status_code_ttl {
          value = 300
          status_code_range {
            from = 400
            to   = 499
          }
        }
      }

      cache_key {
        cache_by_device_type = true

        custom_key {
          query_string {
            include = ["utm_source", "utm_medium", "page"]
          }
        }
      }

      cache_reserve {
        eligible          = true
        minimum_file_size = 10485760
      }

      serve_stale {
        disable_stale_while_updating = true
      }
    }
  }

  rules {
    action      = "set_cache_settings"
    expression  = "(http.request.uri.path contains \"/nocache\")"
    description = "Bypass cache key for all query parameters (exclude wildcard)"
    enabled     = true

    action_parameters {
      cache = true

      cache_key {
        custom_key {
          query_string {
            exclude = ["*"]
          }
        }
      }
    }
  }

  rules {
    action      = "set_cache_settings"
    expression  = "(http.request.uri.path contains \"/passthrough\")"
    description = "Bypass cache key for all query parameters (commented exclude wildcard)"
    enabled     = true

    action_parameters {
      cache = true

      cache_key {
        custom_key {
          query_string {
            exclude = [
              # Exclude all query parameters from cache key.
              "*",
            ]
          }
        }
      }
    }
  }

  rules {
    action      = "set_cache_settings"
    expression  = "(http.request.uri.path contains \"/vary\")"
    description = "Cache varies on all query parameters (include wildcard)"
    enabled     = true

    action_parameters {
      cache = true

      cache_key {
        custom_key {
          query_string {
            include = ["*"]
          }
        }
      }
    }
  }
}

# Test Case 16: Origin routing with SNI
resource "cloudflare_ruleset" "origin_with_sni" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-origin-sni"
  kind        = "zone"
  phase       = "http_request_origin"
  description = "Route to origin with SNI override"

  rules {
    action      = "route"
    expression  = "(http.host eq \"api.${var.cloudflare_domain}\")"
    description = "Route API traffic to origin with custom SNI"
    enabled     = true

    action_parameters {
      origin {
        host = var.cloudflare_domain
        port = 8443
      }

      sni {
        value = "secure.${var.cloudflare_domain}"
      }
    }
  }
}

# Note: DDoS L7 ruleset test removed - Cloudflare only allows one ddos_l7 ruleset per account
# and it cannot be deleted once created. Testing DDoS rulesets requires manual setup.

# Integration-only cases below: ruleset_e2e.tf holds the cases above for the
# e2e run, so these are migrated and compared here but never applied.

# Test Case 17: dynamic action_parameters with static and dynamic headers
resource "cloudflare_ruleset" "cftftest_dynamic_action_parameters" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-dynamic-action-parameters"
  kind    = "zone"
  phase   = "http_response_headers_transform"

  dynamic "rules" {
    for_each = local.cftftest_header_rules
    content {
      action     = "rewrite"
      expression = rules.value.expression
      dynamic "action_parameters" {
        for_each = rules.value.extra_headers != null ? [rules.value.extra_headers] : []
        content {
          headers {
            name      = "X-Powered-By"
            operation = "remove"
          }
          dynamic "headers" {
            for_each = action_parameters.value
            content {
              name      = headers.key
              operation = "set"
              value     = headers.value
            }
          }
        }
      }
    }
  }
}

# Test Case 18: static and dynamic categories, either/or dynamic ratelimit
resource "cloudflare_ruleset" "cftftest_dynamic_same_type" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-dynamic-same-type"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules {
    action     = "execute"
    expression = "true"
    action_parameters {
      id = "efb7b8c949ac4650a09736fc376e9aee"
      overrides {
        categories {
          category = "drupal"
          action   = "log"
        }
        dynamic "categories" {
          for_each = ["wordpress"]
          content {
            category = categories.value
            action   = "block"
          }
        }
        dynamic "categories" {
          for_each = ["joomla"]
          content {
            category = categories.value
            action   = "log"
          }
        }
      }
    }
  }
}

resource "cloudflare_ruleset" "cftftest_dynamic_ratelimit_either_or" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-dynamic-ratelimit"
  kind    = "zone"
  phase   = "http_ratelimit"

  rules {
    action     = "block"
    expression = "true"
    dynamic "ratelimit" {
      for_each = local.cftftest_strict_ratelimit ? [1] : []
      content {
        characteristics     = ["ip.src", "cf.colo.id"]
        period              = 10
        requests_per_period = 10
        mitigation_timeout  = 10
      }
    }
    dynamic "ratelimit" {
      for_each = local.cftftest_strict_ratelimit ? [] : [1]
      content {
        characteristics     = ["ip.src", "cf.colo.id"]
        period              = 60
        requests_per_period = 100
        mitigation_timeout  = 60
      }
    }
  }
}

# Test Case 19: dynamic cache_key and dynamic query_string with wildcards
resource "cloudflare_ruleset" "cftftest_dynamic_cache_key" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-dynamic-cache-key"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action     = "set_cache_settings"
    expression = "(http.host eq \"a.${var.cloudflare_domain}\")"
    action_parameters {
      cache = true
      dynamic "cache_key" {
        for_each = [1]
        content {
          custom_key {
            query_string {
              exclude = ["*"]
            }
          }
        }
      }
    }
  }

  rules {
    action     = "set_cache_settings"
    expression = "(http.host eq \"b.${var.cloudflare_domain}\")"
    action_parameters {
      cache = true
      cache_key {
        custom_key {
          dynamic "query_string" {
            for_each = [1]
            content {
              include = ["*"]
            }
          }
        }
      }
    }
  }
}

locals {
  cftftest_header_rules = [
    {
      expression    = "true"
      extra_headers = { "X-Frame-Options" = "DENY" }
    },
  ]
  cftftest_strict_ratelimit = true
}
