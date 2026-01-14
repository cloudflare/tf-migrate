# Certificate Pack Migration Guide (v4 → v5)

This guide explains how `cloudflare_certificate_pack` resources migrate from v4 to v5.

## Quick Reference

| v4 Field | v5 Field | Change Type |
|----------|----------|-------------|
| `wait_for_active_status` | - | Removed |
| `validation_records` | `validation_records` | Changed to computed-only |
| `validation_errors` | `validation_errors` | Changed to computed-only |
| `validation_records.cname_target` | - | Removed from nested objects |
| `validation_records.cname_name` | - | Removed from nested objects |
| `validity_days` | `validity_days` | Type: Int → Int64 |


---

## Migration Examples

### Example 1: Basic Certificate Pack (wait_for_active_status Removed)

**v4 Configuration:**
```hcl
resource "cloudflare_certificate_pack" "example" {
  zone_id                = "0da42c8d2132a9ddaf714f9e7c920711"
  type                   = "advanced"
  hosts                  = ["example.com", "*.example.com"]
  validation_method      = "txt"
  validity_days          = 90
  certificate_authority  = "lets_encrypt"
  wait_for_active_status = true  # ← Removed in v5
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_certificate_pack" "example" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["example.com", "*.example.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
}
```

**What Changed:**
- `wait_for_active_status` removed
- All other fields preserved

---

### Example 2: Computed Fields Removed from Config

**v4 Configuration:**
```hcl
resource "cloudflare_certificate_pack" "computed" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["example.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
  validation_records    = []  # ← These were Optional+Computed in v4
  validation_errors     = []  # ← Changed to Computed-only in v5
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_certificate_pack" "computed" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["example.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
  # validation_records and validation_errors removed from config
}
```

**What Changed:**
- `validation_records` removed from config (now computed-only)
- `validation_errors` removed from config (now computed-only)
- Fields still appear in state as API-managed values

---

