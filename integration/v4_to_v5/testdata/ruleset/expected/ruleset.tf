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

  rules = [
    {
      action      = "rewrite"
      expression  = "true"
      description = "Modify request headers"
      enabled     = true
      action_parameters = {
        headers = {
          "X-Custom-Header" = {
            operation = "set"
            value     = "custom-value"
          }
          "X-Dynamic-Header" = {
            expression = "cf.bot_management.score"
            operation  = "set"
          }
          "X-Remove-Header" = {
            operation = "remove"
          }
        }
      }
    }
  ]
}

# Test Case 4: Block action with custom response
resource "cloudflare_ruleset" "block_with_response" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-block-ruleset"
  kind        = "zone"
  phase       = "http_request_firewall_custom"
  description = "Ruleset with block action and custom response"

  rules = [
    {
      action      = "block"
      expression  = "(http.request.uri.path contains \"/admin\")"
      description = "Block admin access with custom response"
      enabled     = true
      action_parameters = {
        response = {
          content      = "{\"error\": \"Access denied\"}"
          content_type = "application/json"
          status_code  = 403
        }
      }
    }
  ]
}

# Test Case 5: Redirect action with dynamic target
resource "cloudflare_ruleset" "redirect" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-redirect-ruleset"
  kind        = "zone"
  phase       = "http_request_dynamic_redirect"
  description = "Ruleset with redirect action"

  rules = [
    {
      action      = "redirect"
      expression  = "(http.request.uri.path eq \"/old-page\")"
      description = "Redirect old page to new location"
      enabled     = true
      action_parameters = {
        from_value = {
          status_code           = 301
          preserve_query_string = true
          target_url = {
            value = "https://example.com/new-page"
          }
        }
      }
    }
  ]
}

# Test Case 6: Rewrite action with URI modification
resource "cloudflare_ruleset" "rewrite_uri" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-rewrite-uri-ruleset"
  kind        = "zone"
  phase       = "http_request_transform"
  description = "Ruleset with URI rewriting"

  rules = [
    {
      action      = "rewrite"
      expression  = "(http.request.uri.path contains \"/api/v1/\")"
      description = "Rewrite API v1 to v2"
      enabled     = true
      action_parameters = {
        uri = {
          path = {
            value = "/api/v2/endpoint"
          }
          query = {
            expression = "concat(\"version=2&\", http.request.uri.query)"
          }
        }
      }
    }
  ]
}

# Test Case 8: Set config action with multiple settings
resource "cloudflare_ruleset" "set_config" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-config-ruleset"
  kind        = "zone"
  phase       = "http_config_settings"
  description = "Ruleset with configuration overrides"

  rules = [
    {
      action      = "set_config"
      expression  = "(http.host eq \"staging.example.com\")"
      description = "Apply staging configuration"
      enabled     = true
      action_parameters = {
        automatic_https_rewrites = true
        email_obfuscation        = true
        security_level           = "medium"
        ssl                      = "strict"
        autominify = {
          css  = true
          html = true
          js   = true
        }
      }
    }
  ]
}

# Test Case 11: Log custom fields transformation
resource "cloudflare_ruleset" "log_custom_fields" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-log-custom-fields"
  kind        = "zone"
  phase       = "http_log_custom_fields"
  description = "Ruleset with custom field logging"

  rules = [
    {
      action      = "log_custom_field"
      expression  = "true"
      description = "Log custom fields from request"
      enabled     = true
      action_parameters = {
        cookie_fields = [
          { name = "preferences" },
          { name = "session_id" },
          { name = "user_token" },
        ]
        request_fields = [
          { name = "cf.bot_score" },
          { name = "cf.threat_score" },
        ]
        response_fields = [
          { name = "cf.cache_status" },
        ]
      }
    }
  ]
}

# Test Case: WAF managed ruleset with skip rules (action_parameters.rules map)
# Tests conversion of comma-separated rule IDs to lists
resource "cloudflare_ruleset" "waf_with_skip_rules" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-waf-skip"
  kind        = "zone"
  phase       = "http_request_firewall_managed"
  description = "WAF managed ruleset with skip rules"





  rules = [
    {
      action      = "skip"
      description = "Exempt firewall rule expressions from log4j WAF block"
      enabled     = true
      expression  = "(http.request.uri.path contains \"/filters\")"
      action_parameters = {
        rules = {
          efb7b8c949ac4650a09736fc376e9aee = ["0c054d4e4dd5455c9ff8f01efe5abb10", "3536f964ccc345308b6445e8dc29b753", "e7e4b386797e417c998d872956c390a1"]
        }
      }
      logging = {
        enabled = true
      }
    },
    {
      action      = "skip"
      description = "Single rule skip"
      enabled     = true
      expression  = "(http.request.uri.path contains \"/graphql\")"
      action_parameters = {
        rules = {
          efb7b8c949ac4650a09736fc376e9aee = ["5f6744fa026a4638bda5b3d7d5e015dd"]
        }
      }
      logging = {
        enabled = true
      }
    },
    {
      action      = "skip"
      description = "Multi-ruleset skip with mixed values"
      enabled     = true
      expression  = "(ip.src in {64.39.96.0/20})"
      action_parameters = {
        rules = {
          "4814384a9e5d4991b9815dcfc25d2f1f" = ["6179ae15870a4bb7b2d480d4843b323c"]
          "efb7b8c949ac4650a09736fc376e9aee" = ["0242110ae62e44028a13bf4834780914", "6b1cc72dff9746469d4695a474430f12"]
        }
      }
      logging = {
        enabled = true
      }
    },
    {
      action      = "skip"
      description = "Skip current ruleset"
      enabled     = true
      expression  = "(http.request.uri.path contains \"/exempt\")"
      action_parameters = {
        ruleset = "current"
      }
      logging = {
        enabled = true
      }
    },
    {
      action      = "execute"
      enabled     = true
      description = "Execute managed ruleset"
      expression  = "true"
      action_parameters = {
        id = "efb7b8c949ac4650a09736fc376e9aee"
        overrides = {
          rules = [
            {
              enabled = false
              id      = "5de7edfa648c4d6891dc3e7f84534ffa"
            },
            {
              enabled = false
              id      = "e3a567afc347477d9702d9047e97d760"
            }
          ]
        }
      }
    }
  ]
}

# Test Case 12: Cache settings with cache_reserve and query_string
resource "cloudflare_ruleset" "cache_with_reserve" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-cache-reserve"
  kind        = "zone"
  phase       = "http_request_cache_settings"
  description = "Cache settings with cache reserve"

  rules = [
    {
      action      = "set_cache_settings"
      expression  = "(http.request.uri.path contains \"/static\")"
      description = "Cache static content with reserve"
      enabled     = true
      action_parameters = {
        cache = true
        cache_key = {
          cache_by_device_type = true
          custom_key = {
            query_string = {
              include = {
                list = ["utm_source", "utm_medium", "page"]
              }
            }
          }
        }
        cache_reserve = {
          eligible          = true
          minimum_file_size = 10485760
        }
        edge_ttl = {
          mode    = "override_origin"
          default = 86400
          status_code_ttl = [
            {
              status_code = 200
              value       = 2592000
            },
            {
              value = 300
              status_code_range = {
                from = 400
                to   = 499
              }
            }
          ]
        }
        serve_stale = {
          disable_stale_while_updating = true
        }
      }
    }
  ]
}

# Test Case 16: Origin routing with SNI
resource "cloudflare_ruleset" "origin_with_sni" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-origin-sni"
  kind        = "zone"
  phase       = "http_request_origin"
  description = "Route to origin with SNI override"

  rules = [
    {
      action      = "route"
      expression  = "(http.host eq \"api.${var.cloudflare_domain}\")"
      description = "Route API traffic to origin with custom SNI"
      enabled     = true
      action_parameters = {
        origin = {
          host = var.cloudflare_domain
          port = 8443
        }
        sni = {
          value = "secure.${var.cloudflare_domain}"
        }
      }
    }
  ]
}

# Note: DDoS L7 ruleset test removed - Cloudflare only allows one ddos_l7 ruleset per account
# and it cannot be deleted once created. Testing DDoS rulesets requires manual setup.
