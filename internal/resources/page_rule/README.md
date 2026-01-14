# Page Rule Migration Guide (v4 → v5)

This guide explains how `cloudflare_page_rule` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_page_rule` | `cloudflare_page_rule` | No change |
| `actions` | Block (MaxItems:1) | Attribute object | Syntax change |
| `status` default | "active" | "disabled" | Default changed (migration preserves "active") |
| Numeric fields | Int | Int64 (float64 in state) | Type conversion |
| `minify` | Supported | Removed | Deprecated action |
| `disable_railgun` | Supported | Removed | Deprecated action |
| `cache_ttl_by_status` | Multiple blocks | Map attribute | Structure change |
| Nested blocks in `cache_key_fields` | Blocks | Attribute objects | Syntax change |


---

## Migration Examples

### Example 1: Basic Page Rule with Caching

**v4 Configuration:**
```hcl
resource "cloudflare_page_rule" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/static/*"
  priority = 1
  status   = "active"

  actions {
    cache_level         = "cache_everything"
    edge_cache_ttl      = 7200
    browser_cache_ttl   = "3600"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_page_rule" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/static/*"
  priority = 1
  status   = "active"

  actions = {
    cache_level         = "cache_everything"
    edge_cache_ttl      = 7200
    browser_cache_ttl   = "3600"
  }
}
```

**What Changed:**
- `actions { }` block → `actions = { }` attribute

---

### Example 2: Forwarding URL Rule

**v4 Configuration:**
```hcl
resource "cloudflare_page_rule" "redirect" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "example.com/old-path/*"
  priority = 1

  actions {
    forwarding_url {
      url         = "https://example.com/new-path/$1"
      status_code = 301
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_page_rule" "redirect" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "example.com/old-path/*"
  priority = 1

  actions = {
    forwarding_url = {
      url         = "https://example.com/new-path/$1"
      status_code = 301
    }
  }
}
```

**What Changed:**
- Nested `forwarding_url` block → attribute object

---

### Example 3: Cache TTL by Status Code

**v4 Configuration:**
```hcl
resource "cloudflare_page_rule" "cache_by_status" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/api/*"
  priority = 1

  actions {
    cache_level = "cache_everything"

    cache_ttl_by_status {
      codes = "200-299"
      ttl   = 3600
    }

    cache_ttl_by_status {
      codes = "400-499"
      ttl   = 60
    }

    cache_ttl_by_status {
      codes = "500-599"
      ttl   = 0
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_page_rule" "cache_by_status" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/api/*"
  priority = 1

  actions = {
    cache_level = "cache_everything"

    cache_ttl_by_status = {
      "200-299" = 3600
      "400-499" = 60
      "500-599" = 0
    }
  }
}
```

**What Changed:**
- Multiple `cache_ttl_by_status` blocks → single map attribute
- Block `codes` becomes map key, `ttl` becomes map value

---

### Example 4: Advanced Cache Key Fields

**v4 Configuration:**
```hcl
resource "cloudflare_page_rule" "cache_key" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/api/*"
  priority = 1

  actions {
    cache_level = "cache_everything"

    cache_key_fields {
      cookie {
        check_presence = ["session_id", "auth_token"]
        include        = ["user_pref"]
      }

      header {
        check_presence = ["Authorization"]
        include        = ["Accept-Language"]
        exclude        = ["User-Agent"]
      }

      host {
        resolved = true
      }

      query_string {
        include = ["page", "filter"]
        exclude = ["utm_*"]
      }

      user {
        device_type = true
        geo         = true
        lang        = false
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_page_rule" "cache_key" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/api/*"
  priority = 1

  actions = {
    cache_level = "cache_everything"

    cache_key_fields = {
      cookie = {
        check_presence = ["session_id", "auth_token"]
        include        = ["user_pref"]
      }

      header = {
        check_presence = ["Authorization"]
        include        = ["Accept-Language"]
        exclude        = ["User-Agent"]
      }

      host = {
        resolved = true
      }

      query_string = {
        include = ["page", "filter"]
        exclude = ["utm_*"]
      }

      user = {
        device_type = true
        geo         = true
        lang        = false
      }
    }
  }
}
```

**What Changed:**
- All nested blocks converted to attribute objects
- 5 levels of nested blocks → 5 levels of nested attributes

---

### Example 5: Deprecated Features (Removed)

**v4 Configuration:**
```hcl
resource "cloudflare_page_rule" "deprecated" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/*"
  priority = 1

  actions {
    cache_level = "cache_everything"

    # ⚠️ Deprecated in v5
    minify {
      html = "on"
      css  = "on"
      js   = "on"
    }

    # ⚠️ Deprecated in v5
    disable_railgun = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_page_rule" "deprecated" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/*"
  priority = 1

  actions = {
    cache_level = "cache_everything"
    # minify and disable_railgun removed - no longer supported in v5
  }
}
```

**What Changed:**
- `minify` block completely removed
- `disable_railgun` attribute removed
- Use separate resources (e.g., `cloudflare_zone_settings_override`) for minification

---

### Example 6: Status Default Handling

**v4 Configuration:**
```hcl
resource "cloudflare_page_rule" "implicit_active" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/*"
  priority = 1
  # status not specified - v4 default is "active"

  actions {
    cache_level = "cache_everything"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_page_rule" "implicit_active" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  target   = "*.example.com/*"
  priority = 1
  status   = "active"  # Explicitly added to preserve v4 behavior

  actions = {
    cache_level = "cache_everything"
  }
}
```

**What Changed:**
- `status = "active"` explicitly added if missing
- v5 default is "disabled", so migration preserves v4 "active" default

---

