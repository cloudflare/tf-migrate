# Zone Migration Guide (v4 → v5)

This guide explains how `cloudflare_zone` resources migrate to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_zone` | `cloudflare_zone` | No change |
| `zone` | Attribute | `name` | Renamed |
| `account_id` | String attribute | `account = { id = "..." }` | Nested object |
| `jump_start` | Optional | Removed | Deprecated |
| `plan` | Optional string | Removed (computed only) | Changed to computed |


---

## Migration Examples

### Example 1: Basic Zone

**v4 Configuration:**
```hcl
resource "cloudflare_zone" "example" {
  zone       = "example.com"
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "full"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone" "example" {
  name = "example.com"
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
  type = "full"
}
```

**What Changed:**
- `zone` → `name`
- `account_id` → `account.id` (nested object)

---

### Example 2: With Variable Account ID

**v4 Configuration:**
```hcl
variable "cloudflare_account_id" {
  type = string
}

resource "cloudflare_zone" "site" {
  zone       = "site.com"
  account_id = var.cloudflare_account_id
  type       = "full"
  paused     = false
}
```

**v5 Configuration (After Migration):**
```hcl
variable "cloudflare_account_id" {
  type = string
}

resource "cloudflare_zone" "site" {
  name = "site.com"
  account = {
    id = var.cloudflare_account_id
  }
  type   = "full"
  paused = false
}
```

**What Changed:**
- `zone` → `name`
- `account_id = var...` → `account = { id = var... }`
- Variable reference preserved (not converted to string)

---

### Example 3: With Deprecated Fields

**v4 Configuration:**
```hcl
resource "cloudflare_zone" "legacy" {
  zone       = "legacy.com"
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "full"
  jump_start = true
  plan       = "free"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zone" "legacy" {
  name = "legacy.com"
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
  type = "full"
  # jump_start removed
  # plan removed (now computed-only)
}
```

**What Changed:**
- `zone` → `name`
- `account_id` → `account.id`
- `jump_start` removed (deprecated)
- `plan` removed (changed to computed-only in v5)

---

### Example 4: With Data Source Reference

**v4 Configuration:**
```hcl
data "cloudflare_accounts" "main" {
  name = "My Company"
}

resource "cloudflare_zone" "app" {
  zone       = "app.example.com"
  account_id = data.cloudflare_accounts.main.accounts[0].id
  type       = "full"
}
```

**v5 Configuration (After Migration):**
```hcl
data "cloudflare_accounts" "main" {
  name = "My Company"
}

resource "cloudflare_zone" "app" {
  name = "app.example.com"
  account = {
    id = data.cloudflare_accounts.main.accounts[0].id
  }
  type = "full"
}
```

**What Changed:**
- `zone` → `name`
- `account_id` with data reference → `account.id` with reference preserved

---

