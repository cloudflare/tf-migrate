# Zero Trust DEX Test Migration Guide (v4 → v5)

This guide explains how `cloudflare_device_dex_test` / `cloudflare_zero_trust_dex_test` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_device_dex_test` | `cloudflare_zero_trust_dex_test` | Renamed |
| `data` structure | Block (MaxItems:1) | Attribute object | Syntax change |
| Added field | - | `test_id` | Copied from `id` |
| Removed fields | `updated`, `created` | - | Timestamp fields removed |


---

## Migration Examples

### Example 1: HTTP DEX Test

**v4 Configuration:**
```hcl
resource "cloudflare_device_dex_test" "http_test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "HTTP Health Check"
  description = "Test HTTP connectivity"
  interval    = "0h30m0s"
  enabled     = true

  data {
    host   = "https://example.com"
    kind   = "http"
    method = "GET"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_dex_test" "http_test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "HTTP Health Check"
  description = "Test HTTP connectivity"
  interval    = "0h30m0s"
  enabled     = true

  data = {
    host   = "https://example.com"
    kind   = "http"
    method = "GET"
  }
}
```

**What Changed:**
- Resource type: `cloudflare_device_dex_test` → `cloudflare_zero_trust_dex_test`
- `data { }` block → `data = { }` attribute

---

### Example 2: Traceroute DEX Test

**v4 Configuration:**
```hcl
resource "cloudflare_device_dex_test" "traceroute" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Network Path Test"
  interval   = "1h0m0s"
  enabled    = true

  data {
    host = "8.8.8.8"
    kind = "traceroute"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_dex_test" "traceroute" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Network Path Test"
  interval   = "1h0m0s"
  enabled    = true

  data = {
    host = "8.8.8.8"
    kind = "traceroute"
  }
}
```

---

