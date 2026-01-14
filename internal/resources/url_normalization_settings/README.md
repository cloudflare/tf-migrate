# URL Normalization Settings Migration Guide (v4 → v5)

This guide explains how `cloudflare_url_normalization_settings` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_url_normalization_settings` | `cloudflare_url_normalization_settings` | No change |
| All fields | Unchanged | Unchanged | No change |


---

## Migration Overview

**This is a version-bump-only migration.** The url_normalization_settings resource schema is 100% backward compatible between v4 and v5.

---

## Migration Example

**v4 Configuration:**
```hcl
resource "cloudflare_url_normalization_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  type    = "cloudflare"
  scope   = "incoming"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_url_normalization_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  type    = "cloudflare"
  scope   = "incoming"
}
```

**What Changed:** Nothing - configuration is identical

---

## Field Reference

All fields remain unchanged:
- `zone_id` - Zone identifier
- `type` - Normalization type ("cloudflare" or "rfc3986")
- `scope` - Normalization scope ("incoming" or "both")

---

## Testing Your Migration

After migration, verify:

1. **Zero drift:**
   ```bash
   terraform plan
   # Should show: No changes. Your infrastructure matches the configuration.
   ```

---

## Additional Resources

- Integration tests: `integration/v4_to_v5/testdata/url_normalization_settings/`
- Migration code: `internal/resources/url_normalization_settings/v4_to_v5.go`

---


**Complexity Rating: ⭐ LOW**

Transparent pass-through migration - no field or structural changes.
