# Zero Trust Gateway Settings Migration Guide (v4 → v5)

This guide explains how `cloudflare_teams_account` resources migrate to v5 with **resource splitting** and **attribute restructuring**.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_teams_account` | `cloudflare_zero_trust_gateway_settings` | Renamed |
| **Resource count** | **1 resource** | **3 resources** | **SPLIT** |
| `logging` block | Part of main resource | `cloudflare_zero_trust_gateway_logging` | Extracted to separate resource |
| `proxy` block | Part of main resource | `cloudflare_zero_trust_device_settings` | Extracted to separate resource |
| MaxItems:1 blocks | Nested blocks | Nested attribute objects | Syntax change |
| Computed read-only blocks | `payload_log`, `ssh_session_log` | Removed | No longer in schema |
| `antivirus` | Block | Attribute object | Syntax change |
| `block_page` | Block | Attribute object | Syntax change |
| `body_scanning` | Block | Attribute object | Syntax change |
| `browser_isolation` | Block | Attribute object | Syntax change |
| `certificate` | Block | Attribute object | Syntax change |
| `custom_certificate` | Block | Attribute object | Syntax change |
| `extended_email_matching` | Block | Attribute object | Syntax change |
| `fips` | Block | Attribute object | Syntax change |
| Empty optional fields | Stored as `""` or `false` | Transformed to `null` | Value transformation |

## Understanding Resource Splitting

**The most important concept:** This migration transforms **one v4 resource into three v5 resources**:
1. `cloudflare_zero_trust_gateway_settings` - Core gateway settings
2. `cloudflare_zero_trust_gateway_logging` - Logging configuration (from deprecated `logging` block)
3. `cloudflare_zero_trust_device_settings` - Device/proxy configuration (from deprecated `proxy` block)

**v4 Pattern (1 resource):**
```hcl
resource "cloudflare_teams_account" "main" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  tls_decrypt_enabled = true

  logging {
    redact_pii = true
    settings_by_rule_type {
      dns { log_all = true }
      http { log_all = true }
      l4 { log_all = false }
    }
  }

  proxy {
    tcp = true
    udp = true
  }

  block_page {
    enabled = true
    name = "Custom Block Page"
  }
}
```

**v5 Pattern (3 resources):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "main" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    tls_decrypt = {
      enabled = true
    }
    block_page = {
      enabled = true
      name    = "Custom Block Page"
    }
  }
}

resource "cloudflare_zero_trust_gateway_logging" "main_logging" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  redact_pii = true

  settings_by_rule_type = {
    dns  = { log_all = true }
    http = { log_all = true }
    l4   = { log_all = false }
  }
}

resource "cloudflare_zero_trust_device_settings" "main_device_settings" {
  account_id              = "f037e56e89293a057740de681ac9abbe"
  gateway_proxy_enabled   = true
  gateway_udp_proxy_enabled = true
}
```

---

## Migration Examples

### Example 1: Basic Gateway Settings

**v4 Configuration:**
```hcl
resource "cloudflare_teams_account" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  tls_decrypt_enabled       = true
  activity_log_enabled      = true
  url_browser_isolation_enabled = false

  block_page {
    enabled     = true
    name        = "Access Denied"
    footer_text = "Contact IT"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    tls_decrypt = {
      enabled = true
    }
    activity_log = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
    }
    block_page = {
      enabled     = true
      name        = "Access Denied"
      footer_text = "Contact IT"
    }
  }
}
```

**What Changed:**
- Resource type: `cloudflare_teams_account` → `cloudflare_zero_trust_gateway_settings`
- Top-level boolean fields → nested objects under `settings`
- `tls_decrypt_enabled` → `settings.tls_decrypt.enabled`
- `activity_log_enabled` → `settings.activity_log.enabled`
- `url_browser_isolation_enabled` → `settings.browser_isolation.url_browser_isolation_enabled`
- `block_page` block → `settings.block_page` attribute object

---

### Example 2: With Logging Configuration

**v4 Configuration:**
```hcl
resource "cloudflare_teams_account" "with_logging" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  tls_decrypt_enabled = true

  logging {
    redact_pii = true
    settings_by_rule_type {
      dns {
        log_all    = true
        log_blocks = false
      }
      http {
        log_all    = true
        log_blocks = true
      }
      l4 {
        log_all    = false
        log_blocks = true
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "with_logging" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    tls_decrypt = {
      enabled = true
    }
  }
}

resource "cloudflare_zero_trust_gateway_logging" "with_logging_logging" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  redact_pii = true

  settings_by_rule_type = {
    dns = {
      log_all    = true
      log_blocks = false
    }
    http = {
      log_all    = true
      log_blocks = true
    }
    l4 = {
      log_all    = false
      log_blocks = true
    }
  }
}
```

**What Changed:**
- 1 resource → 2 resources
- `logging` block extracted to separate `cloudflare_zero_trust_gateway_logging` resource
- Resource name pattern: `{original_name}_logging`
- `settings_by_rule_type` structure preserved
- Block syntax → attribute syntax for nested objects

---

### Example 3: With Proxy/Device Settings

**v4 Configuration:**
```hcl
resource "cloudflare_teams_account" "with_proxy" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  tls_decrypt_enabled = true

  proxy {
    tcp = true
    udp = true
    root_ca = true
    virtual_ip = false
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "with_proxy" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    tls_decrypt = {
      enabled = true
    }
  }
}

resource "cloudflare_zero_trust_device_settings" "with_proxy_device_settings" {
  account_id                          = "f037e56e89293a057740de681ac9abbe"
  gateway_proxy_enabled               = true
  gateway_udp_proxy_enabled           = true
  root_certificate_installation_enabled = true
  use_zt_virtual_ip                   = false
}
```

**What Changed:**
- `proxy` block extracted to separate `cloudflare_zero_trust_device_settings` resource
- Resource name pattern: `{original_name}_device_settings`
- Field mappings:
  - `proxy.tcp` → `gateway_proxy_enabled`
  - `proxy.udp` → `gateway_udp_proxy_enabled`
  - `proxy.root_ca` → `root_certificate_installation_enabled`
  - `proxy.virtual_ip` → `use_zt_virtual_ip`

---

### Example 4: Comprehensive Configuration

**v4 Configuration:**
```hcl
resource "cloudflare_teams_account" "comprehensive" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  tls_decrypt_enabled       = true
  activity_log_enabled      = true
  protocol_detection_enabled = true
  url_browser_isolation_enabled = false
  non_identity_browser_isolation_enabled = false

  block_page {
    enabled         = true
    name            = "Corporate Block Page"
    header_text     = "Access Denied"
    footer_text     = "Contact IT Security"
    logo_path       = "https://example.com/logo.png"
    background_color = "#1a1a1a"
  }

  body_scanning {
    inspection_mode = "deep"
  }

  fips {
    tls = true
  }

  logging {
    redact_pii = true
    settings_by_rule_type {
      dns {
        log_all    = true
        log_blocks = false
      }
      http {
        log_all    = true
        log_blocks = true
      }
      l4 {
        log_all    = false
        log_blocks = true
      }
    }
  }

  proxy {
    tcp     = true
    udp     = true
    root_ca = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "comprehensive" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = true
    }
    protocol_detection = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled          = true
      name             = "Corporate Block Page"
      header_text      = "Access Denied"
      footer_text      = "Contact IT Security"
      logo_path        = "https://example.com/logo.png"
      background_color = "#1a1a1a"
    }
    body_scanning = {
      inspection_mode = "deep"
    }
    fips = {
      tls = true
    }
  }
}

resource "cloudflare_zero_trust_gateway_logging" "comprehensive_logging" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  redact_pii = true

  settings_by_rule_type = {
    dns = {
      log_all    = true
      log_blocks = false
    }
    http = {
      log_all    = true
      log_blocks = true
    }
    l4 = {
      log_all    = false
      log_blocks = true
    }
  }
}

resource "cloudflare_zero_trust_device_settings" "comprehensive_device_settings" {
  account_id                            = "f037e56e89293a057740de681ac9abbe"
  gateway_proxy_enabled                 = true
  gateway_udp_proxy_enabled             = true
  root_certificate_installation_enabled = true
}
```

**What Changed:**
- 1 resource → 3 resources
- All MaxItems:1 blocks → nested attribute objects under `settings`
- `logging` and `proxy` blocks → separate resources
- Field restructuring and renaming as documented

---

### Example 5: With Antivirus Settings

**v4 Configuration:**
```hcl
resource "cloudflare_teams_account" "with_av" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  tls_decrypt_enabled = true

  antivirus {
    enabled_download_phase = true
    enabled_upload_phase   = false
    fail_closed            = true
    notification_settings {
      enabled = true
      msg     = "Malware detected"
      support_url = "https://support.example.com"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "with_av" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    tls_decrypt = {
      enabled = true
    }
    antivirus = {
      enabled_download_phase = true
      enabled_upload_phase   = false
      fail_closed            = true
      notification_settings = {
        enabled     = true
        msg         = "Malware detected"
        support_url = "https://support.example.com"
      }
    }
  }
}
```

**What Changed:**
- `antivirus` block → `settings.antivirus` attribute object
- Nested `notification_settings` block → attribute object (double nesting)
- All field values preserved

---

### Example 6: With Custom Certificate

**v4 Configuration:**
```hcl
resource "cloudflare_teams_account" "custom_cert" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  tls_decrypt_enabled = true

  custom_certificate {
    enabled = true
    id      = "cert-abc123"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "custom_cert" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    tls_decrypt = {
      enabled = true
    }
    custom_certificate = {
      enabled = true
      id      = "cert-abc123"
    }
  }
}
```

**What Changed:**
- `custom_certificate` block → `settings.custom_certificate` attribute object
- Field structure preserved

---

### Example 7: Minimal Configuration (Empty Blocks)

**v4 Configuration:**
```hcl
resource "cloudflare_teams_account" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  tls_decrypt_enabled = true

  block_page {
    enabled = true
    name    = "Block Page"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    tls_decrypt = {
      enabled = true
    }
    block_page = {
      enabled = true
      name    = "Block Page"
    }
  }
}
```

**What Changed:**
- Optional fields not in config are stored as `null` in v5 state (vs `""` or `false` in v4)
- This matches v5 provider's expected behavior for Optional+Computed fields

---

### Example 8: With Meta-Arguments

**v4 Configuration:**
```hcl
resource "cloudflare_teams_account" "depends" {
  account_id = var.account_id

  tls_decrypt_enabled = true

  logging {
    redact_pii = true
    settings_by_rule_type {
      dns { log_all = true }
    }
  }

  depends_on = [cloudflare_zero_trust_access_application.app]
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_settings" "depends" {
  account_id = var.account_id

  settings = {
    tls_decrypt = {
      enabled = true
    }
  }

  depends_on = [cloudflare_zero_trust_access_application.app]
}

resource "cloudflare_zero_trust_gateway_logging" "depends_logging" {
  account_id = var.account_id
  redact_pii = true

  settings_by_rule_type = {
    dns = { log_all = true }
  }

  depends_on = [cloudflare_zero_trust_access_application.app]
}
```

**What Changed:**
- Meta-arguments (`depends_on`) copied to all generated resources
- Variable references preserved

---

## Important Notes

### Empty Value Transformation

The migration uses `TransformEmptyValuesToNull` to handle optional fields that aren't explicitly set in your v4 configuration:

- **V4 behavior**: API returns empty strings (`""`) or default booleans (`false`) for unset optional fields
- **V5 behavior**: Unset optional fields are stored as `null` in state
- **Migration behavior**: Empty values returned by the API are transformed to `null` if not explicitly set in your config

This ensures your migrated state matches what would be created by a fresh v5 configuration.

### Known Issue: block_page Perpetual Drift

⚠️ **There is a known issue with the v5 provider** (not a migration bug) affecting the following optional `block_page` fields:
- `include_context`
- `mailto_address`
- `mailto_subject`
- `mode`
- `suppress_footer`
- `target_uri`

**Symptom**: If these fields are not explicitly set in your configuration, they will show perpetual drift on every `terraform plan`, toggling between API-returned defaults and `null`.

**Root cause**: The Cloudflare API returns server-side defaults for these optional fields, but the v5 provider doesn't properly handle Optional+Computed fields with API defaults.

**Workaround**: If experiencing drift, explicitly set these fields in your configuration:
```hcl
settings = {
  block_page = {
    enabled          = true
    name             = "Block Page"
    include_context  = false
    mailto_address   = ""
    mailto_subject   = ""
    mode             = "customized_block_page"
    suppress_footer  = false
    target_uri       = ""
  }
}
```

This issue affects **any v5 configuration** (migrated or created fresh) and has been documented for provider team investigation.

---

## Field Mapping Reference

### Top-Level to settings Mappings

| v4 Field | v5 Field |
|----------|----------|
| `tls_decrypt_enabled` | `settings.tls_decrypt.enabled` |
| `activity_log_enabled` | `settings.activity_log.enabled` |
| `protocol_detection_enabled` | `settings.protocol_detection.enabled` |
| `url_browser_isolation_enabled` | `settings.browser_isolation.url_browser_isolation_enabled` |
| `non_identity_browser_isolation_enabled` | `settings.browser_isolation.non_identity_enabled` |

### Proxy to Device Settings Mappings

| v4 Field | v5 Resource | v5 Field |
|----------|-------------|----------|
| `proxy.tcp` | `cloudflare_zero_trust_device_settings` | `gateway_proxy_enabled` |
| `proxy.udp` | `cloudflare_zero_trust_device_settings` | `gateway_udp_proxy_enabled` |
| `proxy.root_ca` | `cloudflare_zero_trust_device_settings` | `root_certificate_installation_enabled` |
| `proxy.virtual_ip` | `cloudflare_zero_trust_device_settings` | `use_zt_virtual_ip` |

### Blocks Converted to Attribute Objects

All MaxItems:1 blocks nested under `settings`:
- `antivirus`
- `block_page`
- `body_scanning`
- `browser_isolation`
- `certificate`
- `custom_certificate`
- `extended_email_matching`
- `fips`

### Removed Read-Only Blocks

The following computed blocks are no longer in the v5 schema:
- `payload_log`
- `ssh_session_log`

---
