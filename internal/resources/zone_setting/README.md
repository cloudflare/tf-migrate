# Zone Setting Migration Guide (v4 → v5)

This guide explains how `cloudflare_zone_settings_override` resources migrate to `cloudflare_zone_setting` (singular) in v5 with **one-to-many resource splitting**.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_zone_settings_override` | `cloudflare_zone_setting` | Renamed (singular) |
| **Resource count** | **1 resource** | **N resources** | **SPLIT** |
| `settings` block | Single block with all settings | N separate resources | Major restructuring |
| Setting format | `settings { name = "value" }` | `setting_id = "name"; value = "..."` | Flattened |
| Nested settings | `settings { minify { ... } }` | `value = { ... }` | Object value |
| `zero_rtt` | Setting name | `0rtt` | Renamed |
| `universal_ssl` | Included | Removed | Deprecated |
| State | Migrated | **NOT migrated** | ⚠️ Manual apply needed |
| `lifecycle.ignore_changes` | Supported | **Removed** | ⚠️ Manual re-add needed |


---

## Understanding One-to-Many Splitting

**The most important concept:** This migration transforms **one v4 resource into multiple v5 resources** - one for each setting.

**v4 Pattern (1 resource):**
```hcl
resource "cloudflare_zone_settings_override" "example" {
  zone_id = "abc123"
  settings {
    tls_1_3 = "on"
    zero_rtt = "on"
    minify {
      css = "on"
      js = "on"
    }
  }
}
```

**v5 Pattern (3 resources):**
```hcl
resource "cloudflare_zone_setting" "example_tls_1_3" {
  zone_id    = "abc123"
  setting_id = "tls_1_3"
  value      = "on"
}

resource "cloudflare_zone_setting" "example_0rtt" {
  zone_id    = "abc123"
  setting_id = "0rtt"
  value      = "on"
}

resource "cloudflare_zone_setting" "example_minify" {
  zone_id    = "abc123"
  setting_id = "minify"
  value = {
    css = "on"
    js  = "on"
  }
}
```

---

## Migration Examples

### Example 1: Simple Settings

**v4 Configuration:**
```hcl
resource "cloudflare_zone_settings_override" "site" {
  zone_id = cloudflare_zone.example.id

  settings {
    tls_1_3               = "on"
    automatic_https_rewrites = "on"
    opportunistic_encryption = "on"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_setting" "site_tls_1_3" {
  zone_id    = cloudflare_zone.example.id
  setting_id = "tls_1_3"
  value      = "on"
}

resource "cloudflare_zone_setting" "site_automatic_https_rewrites" {
  zone_id    = cloudflare_zone.example.id
  setting_id = "automatic_https_rewrites"
  value      = "on"
}

resource "cloudflare_zone_setting" "site_opportunistic_encryption" {
  zone_id    = cloudflare_zone.example.id
  setting_id = "opportunistic_encryption"
  value      = "on"
}
```

**What Changed:**
- 1 resource → 3 resources
- Each setting becomes a separate resource
- Resource names: `{original_name}_{setting_name}`
- `zone_id` copied to all resources

---

### Example 2: With Nested Settings (Minify)

**v4 Configuration:**
```hcl
resource "cloudflare_zone_settings_override" "optimized" {
  zone_id = "abc123"

  settings {
    brotli = "on"

    minify {
      css  = "on"
      js   = "on"
      html = "off"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_setting" "optimized_brotli" {
  zone_id    = "abc123"
  setting_id = "brotli"
  value      = "on"
}

resource "cloudflare_zone_setting" "optimized_minify" {
  zone_id    = "abc123"
  setting_id = "minify"
  value = {
    css  = "on"
    html = "off"
    js   = "on"
  }
}
```

**What Changed:**
- Nested `minify` block → object `value`
- Object keys sorted alphabetically
- Simple settings remain scalar values

---

### Example 3: With zero_rtt Rename

**v4 Configuration:**
```hcl
resource "cloudflare_zone_settings_override" "performance" {
  zone_id = "abc123"

  settings {
    zero_rtt   = "on"
    http2      = "on"
    http3      = "on"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_setting" "performance_0rtt" {
  zone_id    = "abc123"
  setting_id = "0rtt"
  value      = "on"
}

resource "cloudflare_zone_setting" "performance_http2" {
  zone_id    = "abc123"
  setting_id = "http2"
  value      = "on"
}

resource "cloudflare_zone_setting" "performance_http3" {
  zone_id    = "abc123"
  setting_id = "http3"
  value      = "on"
}
```

**What Changed:**
- `zero_rtt` → `0rtt` in setting_id
- Resource name uses original name pattern

---

### Example 4: With Security Headers (Double Nesting)

**v4 Configuration:**
```hcl
resource "cloudflare_zone_settings_override" "secure" {
  zone_id = "abc123"

  settings {
    security_header {
      enabled            = true
      max_age            = 31536000
      include_subdomains = true
      preload            = true
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_setting" "secure_security_header" {
  zone_id    = "abc123"
  setting_id = "security_header"
  value = {
    strict_transport_security = {
      enabled            = true
      include_subdomains = true
      max_age            = 31536000
      preload            = true
    }
  }
}
```

**What Changed:**
- Double-wrapped: `value.strict_transport_security`
- Attributes sorted alphabetically
- Special case for security headers

---

### Example 5: With Mobile Redirect

**v4 Configuration:**
```hcl
resource "cloudflare_zone_settings_override" "mobile" {
  zone_id = "abc123"

  settings {
    mobile_redirect {
      status           = "on"
      mobile_subdomain = "m.example.com"
      strip_uri        = false
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_setting" "mobile_mobile_redirect" {
  zone_id    = "abc123"
  setting_id = "mobile_redirect"
  value = {
    mobile_subdomain = "m.example.com"
    status           = "on"
    strip_uri        = false
  }
}
```

**What Changed:**
- Nested block → object value
- Attributes alphabetically sorted

---

### Example 6: With Meta-Arguments

**v4 Configuration:**
```hcl
resource "cloudflare_zone_settings_override" "counted" {
  count   = 2
  zone_id = element(cloudflare_zone.sites[*].id, count.index)

  settings {
    tls_1_3 = "on"
    ssl     = "strict"
  }

  depends_on = [cloudflare_zone.sites]
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_setting" "counted_tls_1_3" {
  count      = 2
  zone_id    = element(cloudflare_zone.sites[*].id, count.index)
  setting_id = "tls_1_3"
  value      = "on"

  depends_on = [cloudflare_zone.sites]
}

resource "cloudflare_zone_setting" "counted_ssl" {
  count      = 2
  zone_id    = element(cloudflare_zone.sites[*].id, count.index)
  setting_id = "ssl"
  value      = "strict"

  depends_on = [cloudflare_zone.sites]
}
```

**What Changed:**
- Meta-arguments (`count`, `depends_on`) copied to all generated resources
- Each setting resource inherits meta-arguments

---

### Example 7: With Deprecated universal_ssl

**v4 Configuration:**
```hcl
resource "cloudflare_zone_settings_override" "legacy" {
  zone_id = "abc123"

  settings {
    tls_1_3       = "on"
    universal_ssl = "on"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_setting" "legacy_tls_1_3" {
  zone_id    = "abc123"
  setting_id = "tls_1_3"
  value      = "on"
}

# universal_ssl removed (deprecated)
```

**What Changed:**
- `universal_ssl` setting skipped (no longer exists)
- Only valid settings migrated

---

### Example 8: With lifecycle.ignore_changes (⚠️ Manual Action Required)

**v4 Configuration:**
```hcl
resource "cloudflare_zone_settings_override" "managed" {
  zone_id = "abc123"

  settings {
    tls_1_3 = "on"
    minify {
      css = "on"
    }
  }

  lifecycle {
    ignore_changes = [
      settings[0].minify
    ]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_setting" "managed_tls_1_3" {
  zone_id    = "abc123"
  setting_id = "tls_1_3"
  value      = "on"
}

resource "cloudflare_zone_setting" "managed_minify" {
  zone_id    = "abc123"
  setting_id = "minify"
  value = {
    css = "on"
  }

  # WARNING: lifecycle blocks with ignore_changes cannot be automatically migrated.
  # V4 paths like settings[0].minify are invalid in V5.
  # Please manually re-add lifecycle blocks after migration.
}
```

**What Changed:**
- `lifecycle` blocks with `ignore_changes` **intentionally removed**
- v4 paths don't map to v5 structure
- Manual re-add required after migration

---

