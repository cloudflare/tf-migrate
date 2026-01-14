# Zero Trust DLP Custom Profile Migration Guide (v4 → v5)

This guide explains how `cloudflare_dlp_profile` / `cloudflare_zero_trust_dlp_profile` resources migrate to v5 **with conditional dual-type splitting**.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_dlp_profile` | **Conditional:** Custom OR Predefined | **SPLIT** |
| Alt resource name | `cloudflare_zero_trust_dlp_profile` | **Conditional:** Custom OR Predefined | **SPLIT** |
| **Custom profile** | `type = "custom"` | `cloudflare_zero_trust_dlp_custom_profile` | New name |
| **Predefined profile** | `type = "predefined"` | `cloudflare_zero_trust_dlp_predefined_profile` | New name |
| `type` attribute | Required field | Removed | No longer needed |
| `entry` blocks (custom) | Multiple blocks | `entries` array attribute | Structure change |
| `entry.id` (custom) | Included | Removed | Not needed in v5 |
| `pattern` (custom) | Block | Attribute object | Syntax change |
| `entry` blocks (predefined) | Enabled/disabled list | `enabled_entries` array | Logic change |
| `id` (predefined) | Resource ID | `profile_id` | Field rename |
| Dynamic `entry` blocks | Supported | **NOT SUPPORTED** | ⚠️ Manual refactor needed |


---

## Understanding the Split

**The most important concept:** This migration transforms **one v4 resource type into TWO different v5 resource types** based on the `type` attribute value.

### Decision Logic

**type = "custom"?** → Custom Profile
```hcl
# v4
resource "cloudflare_dlp_profile" "custom" {
  type = "custom"
  name = "Credit Card Detection"
  # entry blocks with patterns
}

# v5
resource "cloudflare_zero_trust_dlp_custom_profile" "custom" {
  name = "Credit Card Detection"
  # entries array with patterns
}
```

**type = "predefined"?** → Predefined Profile
```hcl
# v4
resource "cloudflare_dlp_profile" "predefined" {
  type = "predefined"
  id   = "aws-keys-uuid"
  # entry blocks with enabled/disabled
}

# v5
resource "cloudflare_zero_trust_dlp_predefined_profile" "predefined" {
  profile_id      = "aws-keys-uuid"
  enabled_entries = ["aws-access-key", "aws-session-token"]
}
```

---

## Migration Examples

### Example 1: Custom Profile with Single Entry

**v4 Configuration:**
```hcl
resource "cloudflare_dlp_profile" "credit_cards" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  type        = "custom"
  name        = "Credit Card Detection"
  description = "Detects credit card numbers"

  entry {
    id      = "abc123"
    name    = "Visa Card"
    enabled = true

    pattern {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_dlp_custom_profile" "credit_cards" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Credit Card Detection"
  description = "Detects credit card numbers"

  entries = [{
    name    = "Visa Card"
    enabled = true

    pattern = {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
    }
  }]
}
```

**What Changed:**
- Resource type → `cloudflare_zero_trust_dlp_custom_profile`
- `type` attribute removed
- `entry` blocks → `entries` array attribute
- `entry.id` field removed
- `pattern` block → `pattern` attribute object

---

### Example 2: Custom Profile with Multiple Entries

**v4 Configuration:**
```hcl
resource "cloudflare_dlp_profile" "financial" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "custom"
  name       = "Financial Data"

  entry {
    id      = "entry-1"
    name    = "Visa"
    enabled = true
    pattern {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
    }
  }

  entry {
    id      = "entry-2"
    name    = "Mastercard"
    enabled = true
    pattern {
      regex      = "5[1-5][0-9]{14}"
      validation = "luhn"
    }
  }

  entry {
    id      = "entry-3"
    name    = "Amex"
    enabled = false
    pattern {
      regex      = "3[47][0-9]{13}"
      validation = "luhn"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_dlp_custom_profile" "financial" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Financial Data"

  entries = [
    {
      name    = "Visa"
      enabled = true
      pattern = {
        regex      = "4[0-9]{12}(?:[0-9]{3})?"
        validation = "luhn"
      }
    },
    {
      name    = "Mastercard"
      enabled = true
      pattern = {
        regex      = "5[1-5][0-9]{14}"
        validation = "luhn"
      }
    },
    {
      name    = "Amex"
      enabled = false
      pattern = {
        regex      = "3[47][0-9]{13}"
        validation = "luhn"
      }
    }
  ]
}
```

**What Changed:**
- All `entry` blocks → elements in `entries` array
- Each entry's ID removed
- Each pattern block → pattern object

---

### Example 3: Predefined Profile with Enabled Entries

**v4 Configuration:**
```hcl
resource "cloudflare_dlp_profile" "aws_keys" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "predefined"
  id         = "aws-keys-profile-uuid"
  name       = "AWS Keys Detection"

  entry {
    id      = "aws-access-key"
    name    = "AWS Access Key ID"
    enabled = true
  }

  entry {
    id      = "aws-secret-key"
    name    = "AWS Secret Access Key"
    enabled = true
  }

  entry {
    id      = "aws-session-token"
    name    = "AWS Session Token"
    enabled = false
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_dlp_predefined_profile" "aws_keys" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  profile_id = "aws-keys-profile-uuid"
  name       = "AWS Keys Detection"

  enabled_entries = [
    "aws-access-key",
    "aws-secret-key"
  ]
}
```

**What Changed:**
- Resource type → `cloudflare_zero_trust_dlp_predefined_profile`
- `type` attribute removed
- `id` → `profile_id`
- All `entry` blocks removed
- Only **enabled** entry IDs collected into `enabled_entries` array
- Entry names and other fields discarded

---

### Example 4: Predefined Profile with All Disabled

**v4 Configuration:**
```hcl
resource "cloudflare_dlp_profile" "disabled_profile" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "predefined"
  id         = "profile-uuid"
  name       = "Disabled Profile"

  entry {
    id      = "pattern-1"
    enabled = false
  }

  entry {
    id      = "pattern-2"
    enabled = false
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_dlp_predefined_profile" "disabled_profile" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  profile_id = "profile-uuid"
  name       = "Disabled Profile"

  # No enabled_entries (all were disabled)
}
```

**What Changed:**
- Empty `enabled_entries` not included (no enabled entries)

---

### Example 5: Deprecated Resource Name

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_dlp_profile" "legacy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "custom"
  name       = "Legacy Profile"

  entry {
    name    = "SSN"
    enabled = true
    pattern {
      regex = "[0-9]{3}-[0-9]{2}-[0-9]{4}"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_dlp_custom_profile" "legacy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Legacy Profile"

  entries = [{
    name    = "SSN"
    enabled = true
    pattern = {
      regex = "[0-9]{3}-[0-9]{2}-[0-9]{4}"
    }
  }]
}
```

**What Changed:**
- Both v4 names (`cloudflare_dlp_profile` and `cloudflare_zero_trust_dlp_profile`) handled identically

---

### Example 6: Dynamic Entry Blocks (⚠️ Manual Refactor Required)

**v4 Configuration:**
```hcl
variable "patterns" {
  type = list(object({
    name    = string
    enabled = bool
    regex   = string
  }))
}

resource "cloudflare_dlp_profile" "dynamic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "custom"
  name       = "Dynamic Profile"

  dynamic "entry" {
    for_each = var.patterns
    content {
      name    = entry.value.name
      enabled = entry.value.enabled
      pattern {
        regex = entry.value.regex
      }
    }
  }
}
```

**⚠️ Migration Result (Warning Comments Added):**
```hcl
resource "cloudflare_zero_trust_dlp_custom_profile" "dynamic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Dynamic Profile"

  # WARNING: Dynamic entry blocks cannot be automatically migrated to v5.
  # The v5 provider uses 'entries' as a list attribute, which doesn't support dynamic blocks.
  # Please manually convert dynamic entries to a static list or use for_each at the resource level.

  dynamic "entry" {
    for_each = var.patterns
    content {
      name    = entry.value.name
      enabled = entry.value.enabled
      pattern {
        regex = entry.value.regex
      }
    }
  }
}
```

**Manual Refactoring Option:**
```hcl
resource "cloudflare_zero_trust_dlp_custom_profile" "dynamic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Dynamic Profile"

  entries = [
    for pattern in var.patterns : {
      name    = pattern.name
      enabled = pattern.enabled
      pattern = {
        regex = pattern.regex
      }
    }
  ]
}
```

**What Changed:**
- Dynamic blocks **cannot** be automatically migrated
- Must manually convert to for-expression
- v5 uses list attribute (not blocks)

---

