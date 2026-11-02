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
  status  = "active"  # Added by tf-migrate
}
```

**What Changed:**
- `status = "active"` added when the config has no `status`. In v4 the resource always enabled DNSSEC and `status` was computed-only. In v5 `status` is optional and not computed, so without it every plan shows `status = "active" -> null`.

---

### Example 2: DNSSEC disabled outside Terraform

v4 always set DNSSEC to active on create, so tf-migrate writes `status = "active"`. If DNSSEC was later disabled outside Terraform, the first v5 plan shows `status = "disabled" -> "active"`. Set `status = "disabled"` in the config if you want to keep it off.

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
  status  = "active"  # Added by tf-migrate
  # modified_on removed (now computed-only)
}
```

**What Changed:**
- `modified_on` removed from configuration
- Becomes computed-only field in v5
- State still tracks it (in RFC3339 format)

---

