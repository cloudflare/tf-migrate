# Queue Migration Guide (v4 → v5)

This guide explains how `cloudflare_queue` resources migrate from v4 to v5.

## Quick Reference

| v4 Field | v5 Field | Change Type |
|----------|----------|-------------|
| `name` | `queue_name` | Renamed |
| `account_id` | `account_id` | No change |


---

## Migration Examples

### Example 1: Basic Queue

**v4 Configuration:**
```hcl
resource "cloudflare_queue" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-queue"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_queue" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  queue_name = "my-queue"  # ← name renamed to queue_name
}
```

**What Changed:**
- `name` → `queue_name`

---

