# R2 Bucket Migration Guide (v4 → v5)

This guide explains how `cloudflare_r2_bucket` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_r2_bucket` | `cloudflare_r2_bucket` | No change |
| Config fields | All preserved | All preserved | No change |
| State defaults | - | `jurisdiction`, `storage_class` | Added with defaults |


---

## Migration Overview

**No configuration changes required.** The v5 provider adds two new fields with default values in state to prevent plan changes after migration.

---

## Migration Example

**v4 Configuration:**
```hcl
resource "cloudflare_r2_bucket" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-bucket"
  location   = "WNAM"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_r2_bucket" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-bucket"
  location   = "WNAM"
}
```

**What Changed in Config:** Nothing

**What Changed in State:**
```json
{
  "account_id": "f037e56e89293a057740de681ac9abbe",
  "name": "my-bucket",
  "location": "WNAM",
  "jurisdiction": "default",
  "storage_class": "Standard"
}
```

---

## Field Reference

### Existing Fields (Unchanged)
- `account_id` - Account identifier
- `name` - Bucket name
- `location` - Bucket location (optional)

### New Fields (v5 - State Only)
- **`jurisdiction`** - Data jurisdiction (default: `"default"`)
- **`storage_class`** - Storage class (default: `"Standard"`)

These fields are added to state with default values during migration to prevent drift.

---

## Testing Your Migration

After migration, verify:

1. **Zero drift:**
   ```bash
   terraform plan
   # Should show: No changes. Your infrastructure matches the configuration.
   ```

2. **New state fields:**
   ```bash
   terraform state show 'cloudflare_r2_bucket.example'
   # Will show jurisdiction and storage_class
   ```

---

## Optional: Explicit Configuration

If you want to explicitly set these values in v5:

```hcl
resource "cloudflare_r2_bucket" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-bucket"
  location   = "WNAM"

  # Optional in v5
  jurisdiction  = "default"
  storage_class = "Standard"
}
```

---

## Additional Resources

- Integration tests: `integration/v4_to_v5/testdata/r2_bucket/`
- Migration code: `internal/resources/r2_bucket/v4_to_v5.go`

---


**Complexity Rating: ⭐ LOW**

Transparent migration - no config changes needed. Defaults added to state prevent drift.
