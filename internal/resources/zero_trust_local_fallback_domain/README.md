# Zero Trust Local Fallback Domain Migration Guide (v4 → v5)

This guide explains how `cloudflare_zero_trust_local_fallback_domain` / `cloudflare_fallback_domain` resources migrate to v5 **with conditional dual-variant splitting**.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_zero_trust_local_fallback_domain` | **Conditional:** Default OR Custom | **SPLIT** |
| Alt resource name | `cloudflare_fallback_domain` | **Conditional:** Default OR Custom | **SPLIT** |
| **Default profile** | With no `policy_id` | `cloudflare_zero_trust_device_default_profile_local_domain_fallback` | New name |
| **Custom profile** | With `policy_id` | `cloudflare_zero_trust_device_custom_profile_local_domain_fallback` | New name |
| `domains` | Multiple blocks | Array attribute | Structure change |
| ID (custom only) | `account_id/policy_id` | `policy_id` | Format change |
| `policy_id = null` | Explicitly set | Removed | Cleanup |


---

## Understanding the Split

**The most important concept:** This migration transforms **one v4 resource type into TWO different v5 resource types** based on whether `policy_id` is present.

### Decision Logic

**Has policy_id?** → Custom Profile
```hcl
# v4
resource "cloudflare_zero_trust_local_fallback_domain" "custom" {
  policy_id = cloudflare_zero_trust_device_custom_profile.prod.id
  # ...
}

# v5
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "custom" {
  policy_id = cloudflare_zero_trust_device_custom_profile.prod.id
  # ...
}
```

**No policy_id?** → Default Profile
```hcl
# v4
resource "cloudflare_zero_trust_local_fallback_domain" "default" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  # no policy_id
  # ...
}

# v5
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "default" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  # ...
}
```

---

## Migration Examples

### Example 1: Default Profile (No Policy ID)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_local_fallback_domain" "default" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix      = "example.com"
    description = "Internal domain"
    dns_server  = ["1.1.1.1", "8.8.8.8"]
  }

  domains {
    suffix = "internal.local"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "default" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix      = "example.com"
      description = "Internal domain"
      dns_server  = ["1.1.1.1", "8.8.8.8"]
    },
    {
      suffix = "internal.local"
    }
  ]
}
```

**What Changed:**
- Resource type → `device_default_profile_local_domain_fallback`
- Multiple `domains` blocks → single `domains` array attribute

---

### Example 2: Custom Profile (With Policy ID)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_local_fallback_domain" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = cloudflare_zero_trust_device_custom_profile.prod.id

  domains {
    suffix      = "corp.example.com"
    description = "Corporate network"
    dns_server  = ["10.0.0.1", "10.0.0.2"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = cloudflare_zero_trust_device_custom_profile.prod.id

  domains = [
    {
      suffix      = "corp.example.com"
      description = "Corporate network"
      dns_server  = ["10.0.0.1", "10.0.0.2"]
    }
  ]
}
```

**What Changed:**
- Resource type → `device_custom_profile_local_domain_fallback`
- `domains` blocks → array attribute
- `policy_id` preserved

---

### Example 3: Deprecated Resource Name

**v4 Configuration:**
```hcl
resource "cloudflare_fallback_domain" "legacy" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "old.example.com"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "legacy" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "old.example.com"
    }
  ]
}
```

**What Changed:**
- Deprecated name `cloudflare_fallback_domain` → new naming convention
- Same transformation as modern v4 name

---

### Example 4: Explicit policy_id = null

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_local_fallback_domain" "explicit_null" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = null  # Explicitly set to null

  domains {
    suffix = "example.com"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "explicit_null" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  # policy_id removed (was null)

  domains = [
    {
      suffix = "example.com"
    }
  ]
}
```

**What Changed:**
- Treated as default profile (no policy_id)
- Explicit `policy_id = null` removed from configuration

---

### Example 5: Multiple Domains with All Fields

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_local_fallback_domain" "comprehensive" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix      = "app1.internal"
    description = "Application 1 domain"
    dns_server  = ["10.1.0.1", "10.1.0.2"]
  }

  domains {
    suffix      = "app2.internal"
    description = "Application 2 domain"
    dns_server  = ["10.2.0.1"]
  }

  domains {
    suffix = "app3.internal"
    # No description or dns_server
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "comprehensive" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix      = "app1.internal"
      description = "Application 1 domain"
      dns_server  = ["10.1.0.1", "10.1.0.2"]
    },
    {
      suffix      = "app2.internal"
      description = "Application 2 domain"
      dns_server  = ["10.2.0.1"]
    },
    {
      suffix = "app3.internal"
    }
  ]
}
```

**What Changed:**
- Multiple blocks → array of objects
- All optional fields preserved where present
- Missing fields remain omitted

---

### Example 6: Cross-Resource References (Preprocessing)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_device_profiles" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Custom Policy"
  match      = "user.email == \"admin@example.com\""
  precedence = 10
}

resource "cloudflare_zero_trust_local_fallback_domain" "linked" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = cloudflare_zero_trust_device_profiles.custom.id

  domains {
    suffix = "internal.example.com"
  }
}
```

**v5 Configuration (After Migration - with preprocessing):**
```hcl
resource "cloudflare_zero_trust_device_custom_profile" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Custom Policy"
  match      = "user.email == \"admin@example.com\""
  precedence = 10
}

resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "linked" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = cloudflare_zero_trust_device_custom_profile.custom.id  # Reference updated!

  domains = [
    {
      suffix = "internal.example.com"
    }
  ]
}
```

**What Changed:**
- Device profile resource migrated first
- Reference updated from `cloudflare_zero_trust_device_profiles` → `cloudflare_zero_trust_device_custom_profile`
- Fallback domain resource then migrated with updated reference

---

