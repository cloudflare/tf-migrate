# Zero Trust Access Service Token Migration Guide (v4 → v5)

This guide explains how `cloudflare_access_service_token` / `cloudflare_zero_trust_access_service_token` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_access_service_token` | `cloudflare_zero_trust_access_service_token` | Renamed |
| Alt resource name | `cloudflare_zero_trust_access_service_token` | `cloudflare_zero_trust_access_service_token` | No change |
| Removed field | `min_days_for_renewal` | - | Deprecated |
| Type conversion | `client_secret_version` int | float64 | Numeric type |


---

## Migration Examples

### Example 1: Basic Service Token

**v4 Configuration:**
```hcl
resource "cloudflare_access_service_token" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "API Service Token"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_service_token" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "API Service Token"
}
```

**What Changed:**
- Resource type: `cloudflare_access_service_token` → `cloudflare_zero_trust_access_service_token`

---

### Example 2: With Duration and Deprecated Field

**v4 Configuration:**
```hcl
resource "cloudflare_access_service_token" "expiring" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "Temporary Token"
  duration             = "8760h"  # 1 year
  min_days_for_renewal = 30       # ← Removed in v5
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_service_token" "expiring" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Temporary Token"
  duration   = "8760h"
  # min_days_for_renewal removed (deprecated)
}
```

**What Changed:**
- Resource type renamed
- `min_days_for_renewal` field removed

---

### Example 3: Zone-Scoped Token

**v4 Configuration:**
```hcl
resource "cloudflare_access_service_token" "zone_token" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Zone Service Token"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_service_token" "zone_token" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Zone Service Token"
}
```

---

