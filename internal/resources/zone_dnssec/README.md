# Zone DNSSEC Migration Guide (v4 → v5)

This guide explains how `cloudflare_zone_dnssec` resources migrate to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_zone_dnssec` | `cloudflare_zone_dnssec` | No change |
| `status` | Computed only | Optional | Now configurable |
| `status` values | 4 values | 2 values | Normalized |
| `modified_on` | Optional+computed | Computed only | Removed from config |
| `modified_on` format | Custom | RFC3339 | Format change (state) |
| `flags` | Int | Int64 (float64) | Type conversion (state) |
| `key_tag` | Int | Int64 (float64) | Type conversion (state) |


---

## Migration Examples

### Example 1: Basic DNSSEC Enablement

**v4 Configuration:**
```hcl
resource "cloudflare_zone_dnssec" "example" {
  zone_id = cloudflare_zone.example.id
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_dnssec" "example" {
  zone_id = cloudflare_zone.example.id
  status  = "active"  # Added from state
}
```

**What Changed:**
- `status` attribute added from state (was computed-only in v4)
- If state had `status = "active"` or `"pending"`, config gets `status = "active"`

---

### Example 2: Disabled DNSSEC

**v4 Configuration:**
```hcl
resource "cloudflare_zone_dnssec" "disabled" {
  zone_id = cloudflare_zone.site.id
  # Status is computed-only, cannot be set
}
```

**v5 Configuration (After Migration, if state shows disabled):**
```hcl
resource "cloudflare_zone_dnssec" "disabled" {
  zone_id = cloudflare_zone.site.id
  status  = "disabled"  # Added from state
}
```

**What Changed:**
- If state had `status = "disabled"` or `"pending-disabled"`, config gets `status = "disabled"`

---

### Example 3: With modified_on (Deprecated)

**v4 Configuration:**
```hcl
resource "cloudflare_zone_dnssec" "legacy" {
  zone_id     = cloudflare_zone.legacy.id
  modified_on = "Tue, 04 Nov 2025 21:52:44 +0000"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone_dnssec" "legacy" {
  zone_id = cloudflare_zone.legacy.id
  status  = "active"  # Added from state
  # modified_on removed (now computed-only)
}
```

**What Changed:**
- `modified_on` removed from configuration
- Becomes computed-only field in v5
- State still tracks it (in RFC3339 format)

---

