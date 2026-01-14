# Logpull Retention Migration Guide (v4 → v5)

This guide explains how `cloudflare_logpull_retention` resources migrate from v4 to v5.

## Quick Reference

| v4 Field | v5 Field | Change Type |
|----------|----------|-------------|
| `enabled` | `flag` | Renamed |
| `zone_id` | `zone_id` | No change |


---

## Migration Examples

### Example 1: Enabled Logpull Retention

**v4 Configuration:**
```hcl
resource "cloudflare_logpull_retention" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_logpull_retention" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  flag    = true  # ← enabled renamed to flag
}
```

**What Changed:**
- `enabled` → `flag`

---

### Example 2: Disabled Logpull Retention

**v4 Configuration:**
```hcl
resource "cloudflare_logpull_retention" "disabled" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = false
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_logpull_retention" "disabled" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  flag    = false  # ← enabled renamed to flag
}
```

---

