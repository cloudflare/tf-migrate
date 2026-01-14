# API Token Migration Guide (v4 → v5)

This guide explains how `cloudflare_api_token` resources migrate from v4 to v5.

## Quick Reference

| v4 Field | v5 Field | Change Type |
|----------|----------|-------------|
| `policy` (block) | `policies` (array) | Renamed + syntax change |
| `permission_groups` (list of strings) | `permission_groups` (list of objects) | Type change |
| `resources` (map) | `resources` (JSON string) | Type change + jsonencode |
| `effect` (optional) | `effect` (required, defaults to "allow") | Required field |
| `condition` (block) | `condition` (object) | Syntax change |
| - | `last_used_on` | Added (computed) |


---

## Migration Examples

### Example 1: Basic Single Policy Token

**v4 Configuration:**
```hcl
resource "cloudflare_api_token" "basic" {
  name = "terraform-token"

  policy {
    effect = "allow"
    permission_groups = ["c8fed203ed3043cba015a93ad1616f1f"]
    resources = {
      "com.cloudflare.api.account.*" = "*"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_api_token" "basic" {
  name = "terraform-token"

  policies = [{
    effect = "allow"
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
  }]
}
```

**What Changed:**
- `policy` block → `policies` array attribute
- `permission_groups` strings → objects with `id` field
- `resources` map → `jsonencode()` wrapped JSON string

---

### Example 2: Multiple Policies with IP Conditions

**v4 Configuration:**
```hcl
resource "cloudflare_api_token" "restricted" {
  name = "multi-policy-token"

  policy {
    effect = "allow"
    permission_groups = ["c8fed203ed3043cba015a93ad1616f1f"]
    resources = {
      "com.cloudflare.api.account.*" = "*"
    }
  }

  policy {
    effect = "deny"
    permission_groups = ["82e64a83756745bbbb1c9c2701bf816b"]
    resources = {
      "com.cloudflare.api.account.billing.*" = "*"
    }
  }

  condition {
    request_ip {
      in     = ["192.168.1.0/24"]
      not_in = ["192.168.1.100/32"]
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_api_token" "restricted" {
  name = "multi-policy-token"

  policies = [{
    effect = "allow"
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
  }, {
    effect = "deny"
    permission_groups = [{
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
    resources = jsonencode({
      "com.cloudflare.api.account.billing.*" = "*"
    })
  }]

  condition = {
    request_ip = {
      in     = ["192.168.1.0/24"]
      not_in = ["192.168.1.100/32"]
    }
  }
}
```

**What Changed:**
- Multiple `policy` blocks → Array of policy objects in `policies`
- Nested `condition` block → Object attribute
- Nested `request_ip` block → Object within condition

---

### Example 3: Policy Without Explicit Effect (Auto-defaults)

**v4 Configuration:**
```hcl
resource "cloudflare_api_token" "implicit" {
  name = "api-token-read"

  policy {
    # No effect specified
    permission_groups = ["e086da7e2179491d91ee5f35b3ca210a"]
    resources = {
      "com.cloudflare.api.account.*" = "*"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_api_token" "implicit" {
  name = "api-token-read"

  policies = [{
    effect = "allow"  # ← Auto-added default
    permission_groups = [{
      id = "e086da7e2179491d91ee5f35b3ca210a"
    }]
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
  }]
}
```

**What Changed:**
- Missing `effect` field defaults to `"allow"`
- All other transformations applied

---

### Example 4: Token with Dynamic References

**v4 Configuration:**
```hcl
variable "user_id" {
  default = "12345"
}

resource "cloudflare_api_token" "dynamic" {
  name = "user-specific-token"

  policy {
    permission_groups = ["c8fed203ed3043cba015a93ad1616f1f"]
    resources = {
      "com.cloudflare.api.user.${var.user_id}" = "*"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
variable "user_id" {
  default = "12345"
}

resource "cloudflare_api_token" "dynamic" {
  name = "user-specific-token"

  policies = [{
    effect = "allow"
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
    resources = jsonencode({
      "com.cloudflare.api.user.${var.user_id}" = "*"
    })
  }]
}
```

**What Changed:**
- String interpolations preserved in `jsonencode()`
- Variable references remain functional

---

