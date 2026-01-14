# Regional Hostname Migration Guide (v4 → v5)

This guide explains how `cloudflare_regional_hostname` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_regional_hostname` | `cloudflare_regional_hostname` | No change |
| Fields | All preserved | All preserved | No change |
| `timeouts` block | Supported | Removed | Deprecated feature |


---

## Migration Examples

### Example 1: Basic Regional Hostname

**v4 Configuration:**
```hcl
resource "cloudflare_regional_hostname" "example" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "eu"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_regional_hostname" "example" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "eu"
}
```

**What Changed:** Nothing

---

### Example 2: With Timeouts Block (Removed)

**v4 Configuration:**
```hcl
resource "cloudflare_regional_hostname" "example" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "us"

  timeouts {
    create = "30m"
    update = "30m"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_regional_hostname" "example" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "us"
}
```

**What Changed:**
- `timeouts` block removed (no longer supported in v5)

---

