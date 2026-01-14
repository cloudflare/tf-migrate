# Workers KV Migration Guide (v4 → v5)

This guide explains how `cloudflare_workers_kv` resources migrate from v4 to v5.

## Quick Reference

| v4 Field | v5 Field | Change Type |
|----------|----------|-------------|
| `key` | `key_name` | Renamed |
| `namespace_id` | `namespace_id` | No change |
| `value` | `value` | No change |


---

## Migration Examples

### Example 1: Basic KV Pair

**v4 Configuration:**
```hcl
resource "cloudflare_workers_kv" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  namespace_id = "0f2ac79b498b41faac26be99057bc116"
  key          = "my-key"
  value        = "my-value"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_kv" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  namespace_id = "0f2ac79b498b41faac26be99057bc116"
  key_name     = "my-key"  # ← key renamed to key_name
  value        = "my-value"
}
```

**What Changed:**
- `key` → `key_name`

---

### Example 2: KV with JSON Value

**v4 Configuration:**
```hcl
resource "cloudflare_workers_kv" "config" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  namespace_id = "0f2ac79b498b41faac26be99057bc116"
  key          = "config"
  value        = jsonencode({
    setting1 = "value1"
    setting2 = "value2"
  })
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_kv" "config" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  namespace_id = "0f2ac79b498b41faac26be99057bc116"
  key_name     = "config"  # ← key renamed to key_name
  value        = jsonencode({
    setting1 = "value1"
    setting2 = "value2"
  })
}
```

---

