# Custom Pages Migration Guide (v4 → v5)

This guide explains how `cloudflare_custom_pages` resources migrate from v4 to v5.

## Quick Reference

| v4 Field | v5 Field | Change Type |
|----------|----------|-------------|
| `type` | `identifier` | Renamed |
| `state` | `state` | Now required (defaults to "default") |
| `zone_id` | `zone_id` | No change |
| `account_id` | `account_id` | No change |
| `url` | `url` | No change |


---

## Migration Examples

### Example 1: Zone-Scoped Custom Error Page

**v4 Configuration:**
```hcl
resource "cloudflare_custom_pages" "error_500" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  type    = "500_errors"
  url     = "https://example.com/errors/500.html"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_custom_pages" "error_500" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  identifier = "500_errors"  # ← type renamed to identifier
  url        = "https://example.com/errors/500.html"
  state      = "default"     # ← Added if missing
}
```

**What Changed:**
- `type` → `identifier`
- `state` field added with default value

---

### Example 2: Account-Scoped Challenge Page

**v4 Configuration:**
```hcl
resource "cloudflare_custom_pages" "challenge" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "basic_challenge"
  url        = "https://example.com/challenge.html"
  state      = "customized"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_custom_pages" "challenge" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  identifier = "basic_challenge"  # ← type renamed to identifier
  url        = "https://example.com/challenge.html"
  state      = "customized"       # ← Preserved if present
}
```

**What Changed:**
- `type` → `identifier`
- `state` preserved as-is

---

### Example 3: Multiple Custom Pages

**v4 Configuration:**
```hcl
resource "cloudflare_custom_pages" "error_404" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  type    = "404_errors"
  url     = "https://example.com/errors/404.html"
}

resource "cloudflare_custom_pages" "waf_challenge" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  type    = "waf_challenge"
  url     = "https://example.com/waf.html"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_custom_pages" "error_404" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  identifier = "404_errors"
  url        = "https://example.com/errors/404.html"
  state      = "default"
}

resource "cloudflare_custom_pages" "waf_challenge" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  identifier = "waf_challenge"
  url        = "https://example.com/waf.html"
  state      = "default"
}
```

---

