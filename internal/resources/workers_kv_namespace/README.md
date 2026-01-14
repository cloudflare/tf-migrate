# Workers KV Namespace Migration Guide (v4 → v5)

This guide explains how `cloudflare_workers_kv_namespace` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_workers_kv_namespace` | `cloudflare_workers_kv_namespace` | No change |
| All fields | Unchanged | Unchanged | No change |
| New computed field | - | `supports_url_encoding` | Added (computed) |


---

## Migration Overview

**This is a version-bump-only migration.** The workers_kv_namespace resource schema is 100% backward compatible. The v5 provider adds a computed `supports_url_encoding` field that will be populated on first refresh.

---

## Migration Example

**v4 Configuration:**
```hcl
resource "cloudflare_workers_kv_namespace" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "My KV Namespace"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_kv_namespace" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "My KV Namespace"
}
```

**What Changed:** Nothing in configuration

**State After First Refresh:**
```json
{
  "account_id": "f037e56e89293a057740de681ac9abbe",
  "title": "My KV Namespace",
  "supports_url_encoding": true
}
```

---

## Field Reference

### Existing Fields (Unchanged)
- `account_id` - Account identifier
- `title` - Namespace title
- `id` - Namespace identifier

### New Computed Field (v5)
- `supports_url_encoding` - Whether the namespace supports URL encoding (populated by provider)

---

## Testing Your Migration

After migration, verify:

1. **Zero drift:**
   ```bash
   terraform plan
   # Should show: No changes. Your infrastructure matches the configuration.
   ```

2. **New computed field (after refresh):**
   ```bash
   terraform state show 'cloudflare_workers_kv_namespace.example'
   # Will show supports_url_encoding field
   ```

---

## Additional Resources

- Integration tests: `integration/v4_to_v5/testdata/workers_kv_namespace/`
- Migration code: `internal/resources/workers_kv_namespace/v4_to_v5.go`

---


**Complexity Rating: ⭐ LOW**

Transparent pass-through migration with no config changes. New computed field auto-populated by provider.
