# Zero Trust Gateway Policy Migration Guide (v4 → v5)

This guide explains how `cloudflare_teams_rule` resources migrate to `cloudflare_zero_trust_gateway_policy` in v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_teams_rule` | `cloudflare_zero_trust_gateway_policy` | Renamed |
| `rule_settings` | Block | Attribute object | Syntax change |
| **11 nested blocks** | Blocks | Attribute objects | Massive nesting change |
| `block_page_reason` | Field name | `block_reason` | Renamed |
| `notification_settings.message` | Field name | `notification_settings.msg` | Renamed |
| `l4override.port` | Int | Int64 (float64) | Type conversion |
| `dns_resolvers.*.port` | Int | Int64 (float64) | Type conversion (arrays) |
| Duration fields | `"24h0m0s"` | `"24h"` | Format normalized |
| `biso_admin_controls.*` | 8 deprecated fields | Removed | Deprecated |
| `precedence` / `version` | Int | Int64 (float64) | Type conversion |


---

## Migration Examples

### Example 1: Basic Rule without Settings

**v4 Configuration:**
```hcl
resource "cloudflare_teams_rule" "basic" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Block Social Media"
  description = "Block access to social media sites"
  precedence  = 1000
  action      = "block"
  filters     = ["dns"]
  traffic     = "domain.suffix in {\"facebook.com\" \"twitter.com\"}"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_policy" "basic" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Block Social Media"
  description = "Block access to social media sites"
  precedence  = 1000
  action      = "block"
  filters     = ["dns"]
  traffic     = "domain.suffix in {\"facebook.com\" \"twitter.com\"}"
}
```

**What Changed:**
- Resource type renamed
- No other changes (minimal rule)

---

### Example 2: With Rule Settings Block

**v4 Configuration:**
```hcl
resource "cloudflare_teams_rule" "with_settings" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block with Custom Page"
  action     = "block"
  filters    = ["http"]
  traffic    = "http.request.uri matches \".*\"<br>  precedence = 2000

  rule_settings {
    block_page_reason = "Access denied by policy"
    block_page_enabled = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_policy" "with_settings" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block with Custom Page"
  action     = "block"
  filters    = ["http"]
  traffic    = "http.request.uri matches \".*\""
  precedence = 2000

  rule_settings = {
    block_reason       = "Access denied by policy"  # Renamed!
    block_page_enabled = true
  }
}
```

**What Changed:**
- `rule_settings` block → attribute object
- `block_page_reason` → `block_reason`

---

### Example 3: With Notification Settings

**v4 Configuration:**
```hcl
resource "cloudflare_teams_rule" "with_notifications" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block with Notification"
  action     = "block"
  filters    = ["http"]
  traffic    = "http.host == \"blocked.example.com\""
  precedence = 3000

  rule_settings {
    block_page_enabled = true

    notification_settings {
      enabled = true
      message = "This site is blocked by IT policy"
      support_url = "https://support.example.com"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_policy" "with_notifications" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block with Notification"
  action     = "block"
  filters    = ["http"]
  traffic    = "http.host == \"blocked.example.com\""
  precedence = 3000

  rule_settings = {
    block_page_enabled = true

    notification_settings = {
      enabled     = true
      msg         = "This site is blocked by IT policy"  # Renamed!
      support_url = "https://support.example.com"
    }
  }
}
```

**What Changed:**
- Both blocks → attribute objects
- `notification_settings.message` → `notification_settings.msg`

---

### Example 4: With L4 Override

**v4 Configuration:**
```hcl
resource "cloudflare_teams_rule" "l4_override" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Override L4"
  action     = "l4_override"
  filters    = ["l4"]
  traffic    = "net.dst.ip == \"192.168.1.0/24\""
  precedence = 4000

  rule_settings {
    l4override {
      ip   = "10.0.0.1"
      port = 8080
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_policy" "l4_override" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Override L4"
  action     = "l4_override"
  filters    = ["l4"]
  traffic    = "net.dst.ip == \"192.168.1.0/24\""
  precedence = 4000

  rule_settings = {
    l4override = {
      ip   = "10.0.0.1"
      port = 8080
    }
  }
}
```

**What Changed:**
- Nested blocks → attribute objects
- Port value same in config (int → float64 in state)

---

### Example 5: With DNS Resolvers (Complex Nested Arrays)

**v4 Configuration:**
```hcl
resource "cloudflare_teams_rule" "dns_override" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Custom DNS Resolvers"
  action     = "resolve"
  filters    = ["dns"]
  traffic    = "dns.fqdn == \"internal.example.com\""
  precedence = 5000

  rule_settings {
    dns_resolvers {
      ipv4 {
        ip   = "1.1.1.1"
        port = 53
      }
      ipv4 {
        ip   = "8.8.8.8"
        port = 5353
      }
      ipv6 {
        ip   = "2606:4700:4700::1111"
        port = 53
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_policy" "dns_override" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Custom DNS Resolvers"
  action     = "resolve"
  filters    = ["dns"]
  traffic    = "dns.fqdn == \"internal.example.com\""
  precedence = 5000

  rule_settings = {
    dns_resolvers = {
      ipv4 = [
        {
          ip   = "1.1.1.1"
          port = 53
        },
        {
          ip   = "8.8.8.8"
          port = 5353
        }
      ]
      ipv6 = [
        {
          ip   = "2606:4700:4700::1111"
          port = 53
        }
      ]
    }
  }
}
```

**What Changed:**
- Triple nesting: rule_settings → dns_resolvers → ipv4/ipv6 arrays
- All blocks → attribute objects/arrays
- Port values (int → float64 in state)

---

### Example 6: With Check Session (Duration Normalization)

**v4 Configuration:**
```hcl
resource "cloudflare_teams_rule" "session_check" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Check Session Duration"
  action     = "allow"
  filters    = ["http"]
  traffic    = "http.host == \"app.example.com\""
  precedence = 6000

  rule_settings {
    check_session {
      enforce  = true
      duration = "24h0m0s"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_policy" "session_check" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Check Session Duration"
  action     = "allow"
  filters    = ["http"]
  traffic    = "http.host == \"app.example.com\""
  precedence = 6000

  rule_settings = {
    check_session = {
      enforce  = true
      duration = "24h"  # Normalized!
    }
  }
}
```

**What Changed:**
- Blocks → attribute objects
- Duration format normalized: `"24h0m0s"` → `"24h"`

---

### Example 7: With Deprecated BISO Fields

**v4 Configuration:**
```hcl
resource "cloudflare_teams_rule" "biso_legacy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "BISO Controls"
  action     = "isolate"
  filters    = ["http"]
  traffic    = "http.host == \"external.example.com\""
  precedence = 7000

  rule_settings {
    biso_admin_controls {
      disable_printing             = true
      disable_copy_paste           = true
      disable_download             = false
      disable_upload               = false
      disable_clipboard_redirection = true
      disable_keyboard             = false
      dp                           = true
      dd                           = false
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_policy" "biso_legacy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "BISO Controls"
  action     = "isolate"
  filters    = ["http"]
  traffic    = "http.host == \"external.example.com\""
  precedence = 7000

  rule_settings = {
    biso_admin_controls = {
      # All 8 deprecated fields removed
      # Use new v5 fields instead (printing, copy_paste, download, upload, clipboard_redirection, keyboard)
    }
  }
}
```

**What Changed:**
- Block → attribute object
- **8 deprecated fields removed:**
  - `disable_printing`, `disable_copy_paste`, `disable_download`, `disable_upload`
  - `disable_clipboard_redirection`, `disable_keyboard`
  - `dp`, `dd` (legacy shorthands)

---

### Example 8: Comprehensive Rule with Multiple Nested Structures

**v4 Configuration:**
```hcl
resource "cloudflare_teams_rule" "comprehensive" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Comprehensive Rule"
  action     = "allow"
  filters    = ["http"]
  traffic    = "http.host == \"app.example.com\""
  precedence = 8000

  rule_settings {
    block_page_reason = "Blocked"

    audit_ssh {
      command_logging = true
    }

    egress {
      ipv4          = "192.168.1.1"
      ipv4_fallback = "192.168.1.2"
    }

    payload_log {
      enabled = true
    }

    untrusted_cert {
      action = "block"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_policy" "comprehensive" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Comprehensive Rule"
  action     = "allow"
  filters    = ["http"]
  traffic    = "http.host == \"app.example.com\""
  precedence = 8000

  rule_settings = {
    block_reason = "Blocked"  # Renamed

    audit_ssh = {
      command_logging = true
    }

    egress = {
      ipv4          = "192.168.1.1"
      ipv4_fallback = "192.168.1.2"
    }

    payload_log = {
      enabled = true
    }

    untrusted_cert = {
      action = "block"
    }
  }
}
```

**What Changed:**
- 5 nested blocks all converted to attribute objects
- Field rename at top level

---

