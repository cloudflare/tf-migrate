# Ruleset Migration Guide (v4 → v5)

This guide explains how `cloudflare_ruleset` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_ruleset` | `cloudflare_ruleset` | No change |
| `rules` | Multiple blocks | Array attribute | Structure change |
| `action_parameters` | Nested blocks | Nested objects | Syntax change |
| `headers` (in action_parameters) | Multiple blocks with `name` | Map with name as key | Structure change |
| `query_string` (in cache_key) | Multiple blocks | Merged single object | Structure change |
| Log custom fields | String arrays | Array of objects with `name` | Structure change |
| Dynamic rules | `dynamic "rules"` block | `for` expression | Syntax change |
| `disable_railgun` | Supported | Removed | Deprecated |


---

## Migration Examples

### Example 1: Basic Firewall Rule

**v4 Configuration:**
```hcl
resource "cloudflare_ruleset" "firewall" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "Block malicious traffic"
  description = "Firewall rules"
  kind        = "zone"
  phase       = "http_request_firewall_custom"

  rules {
    action      = "block"
    expression  = "(http.host eq \"example.com\" and ip.geoip.country eq \"CN\")"
    description = "Block China"
    enabled     = true
  }

  rules {
    action      = "challenge"
    expression  = "(cf.threat_score gt 14)"
    description = "Challenge medium threats"
    enabled     = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_ruleset" "firewall" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "Block malicious traffic"
  description = "Firewall rules"
  kind        = "zone"
  phase       = "http_request_firewall_custom"

  rules = [
    {
      action      = "block"
      expression  = "(http.host eq \"example.com\" and ip.geoip.country eq \"CN\")"
      description = "Block China"
      enabled     = true
    },
    {
      action      = "challenge"
      expression  = "(cf.threat_score gt 14)"
      description = "Challenge medium threats"
      enabled     = true
    }
  ]
}
```

**What Changed:**
- Multiple `rules` blocks → single `rules` array attribute

---

### Example 2: Transform Rules with Headers

**v4 Configuration:**
```hcl
resource "cloudflare_ruleset" "transform" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "HTTP request headers"
  kind    = "zone"
  phase   = "http_request_transform"

  rules {
    action = "rewrite"
    expression = "true"

    action_parameters {
      headers {
        name      = "X-Custom-Header"
        operation = "set"
        value     = "custom-value"
      }

      headers {
        name      = "X-Another-Header"
        operation = "set"
        value     = "another-value"
      }

      headers {
        name      = "X-Remove-This"
        operation = "remove"
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_ruleset" "transform" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "HTTP request headers"
  kind    = "zone"
  phase   = "http_request_transform"

  rules = [{
    action = "rewrite"
    expression = "true"

    action_parameters = {
      headers = {
        "X-Custom-Header" = {
          operation = "set"
          value     = "custom-value"
        }
        "X-Another-Header" = {
          operation = "set"
          value     = "another-value"
        }
        "X-Remove-This" = {
          operation = "remove"
        }
      }
    }
  }]
}
```

**What Changed:**
- `action_parameters` block → attribute object
- Multiple `headers` blocks → single `headers` map
- Header `name` becomes map key

---

### Example 3: Cache Rules with Custom Key

**v4 Configuration:**
```hcl
resource "cloudflare_ruleset" "cache" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Cache rules"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action = "set_cache_settings"
    expression = "(http.host eq \"example.com\")"

    action_parameters {
      cache = true

      cache_key {
        custom_key {
          query_string {
            include = ["page", "sort"]
          }

          query_string {
            exclude = ["utm_*"]
          }

          cookie {
            check_presence = ["session"]
          }

          header {
            include = ["Accept-Language"]
          }
        }
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_ruleset" "cache" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Cache rules"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [{
    action = "set_cache_settings"
    expression = "(http.host eq \"example.com\")"

    action_parameters = {
      cache = true

      cache_key = {
        custom_key = {
          query_string = {
            include = {
              list = ["page", "sort"]
            }
            exclude = {
              list = ["utm_*"]
            }
          }

          cookie = {
            check_presence = ["session"]
          }

          header = {
            include = ["Accept-Language"]
          }
        }
      }
    }
  }]
}
```

**What Changed:**
- Multiple `query_string` blocks → single merged object
- `include/exclude` arrays → objects with `list` field
- All blocks → attribute objects

---

### Example 4: Log Custom Fields

**v4 Configuration:**
```hcl
resource "cloudflare_ruleset" "logging" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Custom logging"
  kind    = "zone"
  phase   = "http_log_custom_fields"

  rules {
    action = "log_custom_field"
    expression = "true"

    action_parameters {
      cookie_fields   = ["session_id", "user_id"]
      request_fields  = ["content-type", "user-agent"]
      response_fields = ["content-length", "x-custom-header"]
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_ruleset" "logging" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Custom logging"
  kind    = "zone"
  phase   = "http_log_custom_fields"

  rules = [{
    action = "log_custom_field"
    expression = "true"

    action_parameters = {
      cookie_fields = [
        { name = "session_id" },
        { name = "user_id" }
      ]
      request_fields = [
        { name = "content-type" },
        { name = "user-agent" }
      ]
      response_fields = [
        { name = "content-length" },
        { name = "x-custom-header" }
      ]
    }
  }]
}
```

**What Changed:**
- String arrays → arrays of objects with `name` field
- Arrays alphabetically sorted

---

### Example 5: Dynamic Rules (For Expression)

**v4 Configuration:**
```hcl
variable "firewall_rules" {
  default = [
    { expr = "ip.geoip.country eq \"CN\"", action = "block" },
    { expr = "ip.geoip.country eq \"RU\"", action = "block" }
  ]
}

resource "cloudflare_ruleset" "dynamic" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Dynamic firewall"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  dynamic "rules" {
    for_each = var.firewall_rules
    content {
      action     = rules.value.action
      expression = rules.value.expr
      enabled    = true
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
variable "firewall_rules" {
  default = [
    { expr = "ip.geoip.country eq \"CN\"", action = "block" },
    { expr = "ip.geoip.country eq \"RU\"", action = "block" }
  ]
}

resource "cloudflare_ruleset" "dynamic" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Dynamic firewall"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [for rules in var.firewall_rules : {
    action     = rules.action
    expression = rules.expr
    enabled    = true
  }]
}
```

**What Changed:**
- `dynamic "rules"` block → `for` expression
- `rules.value.field` → `rules.field` reference

---

