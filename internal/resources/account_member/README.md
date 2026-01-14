# Account Member Migration Guide (v4 → v5)

This guide explains how `cloudflare_account_member` resources migrate from v4 to v5.

## Quick Reference

| v4 Field | v5 Field | Change Type |
|----------|----------|-------------|
| `email_address` | `email` | Renamed |
| `role_ids` | `roles` | Renamed |
| `account_id` | `account_id` | No change |
| `status` | `status` | No change |


---

## Migration Examples

### Example 1: Basic Account Member

**v4 Configuration:**
```hcl
resource "cloudflare_account_member" "developer" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "developer@example.com"
  role_ids      = ["68b329da9893e34099c7d8ad5cb9c940"]
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_account_member" "developer" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  email      = "developer@example.com"
  roles      = ["68b329da9893e34099c7d8ad5cb9c940"]
}
```

**What Changed:**
- `email_address` → `email`
- `role_ids` → `roles`
- Resource type unchanged: `cloudflare_account_member`

---

### Example 2: Account Member with Multiple Roles

**v4 Configuration:**
```hcl
resource "cloudflare_account_member" "admin" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "admin@example.com"
  role_ids = [
    "68b329da9893e34099c7d8ad5cb9c940",  # Administrator
    "d784fa8b6d98d27699781bd9a7cf19f0"   # Super Administrator
  ]
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_account_member" "admin" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  email      = "admin@example.com"
  roles = [
    "68b329da9893e34099c7d8ad5cb9c940",
    "d784fa8b6d98d27699781bd9a7cf19f0"
  ]
}
```

**What Changed:**
- `email_address` → `email`
- `role_ids` → `roles`

---

### Example 3: Account Member with Status

**v4 Configuration:**
```hcl
resource "cloudflare_account_member" "invited" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "newuser@example.com"
  role_ids      = ["68b329da9893e34099c7d8ad5cb9c940"]
  status        = "accepted"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_account_member" "invited" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  email      = "newuser@example.com"
  roles      = ["68b329da9893e34099c7d8ad5cb9c940"]
  status     = "accepted"
}
```

**What Changed:**
- `email_address` → `email`
- `role_ids` → `roles`
- `status` preserved as-is

---

