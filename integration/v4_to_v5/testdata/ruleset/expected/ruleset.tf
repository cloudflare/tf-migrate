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

# Test Case 12: WAF managed ruleset with overrides
resource "cloudflare_ruleset" "waf_with_overrides" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-waf-overrides"
  kind        = "zone"
  phase       = "http_request_firewall_managed"
  description = "WAF ruleset with basic overrides"

  rules = [
    {
      action      = "execute"
      expression  = "true"
      description = "Execute OWASP ruleset with sensitivity override"
      enabled     = true
      action_parameters = {
        id = "4814384a9e5d4991b9815dcfc25d2f1f"
        overrides = {
          enabled = true
          action  = "log"
        }
      }
    }
  ]
}

# Test Case 13: Cache settings with cache_reserve and query_string
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

# Test Case 17: DDoS with algorithms override
resource "cloudflare_ruleset" "ddos_algorithms" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-ddos-algorithms"
  kind        = "root"
  phase       = "ddos_l7"
  description = "DDoS protection with algorithm override"

  rules = [
    {
      action      = "execute"
      expression  = "true"
      description = "Execute DDoS rules with sensitivity"
      enabled     = true
      action_parameters = {
        id = "4d21379b4f9f4bb088e0729962c8b3cf"
        overrides = {
          sensitivity_level = "medium"
        }
      }
    }
  ]
}
