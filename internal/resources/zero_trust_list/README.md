# Zero Trust List Migration Guide (v4 → v5)

This guide explains how `cloudflare_teams_list` / `cloudflare_zero_trust_list` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_teams_list` | `cloudflare_zero_trust_list` | Renamed |
| Alt resource name | `cloudflare_zero_trust_list` | `cloudflare_zero_trust_list` | No change |
| `items` (string array) | `["value1", "value2"]` | Merged into objects | Structure change |
| `items_with_description` (blocks) | Blocks | Merged into objects | Structure change |
| Unified `items` | - | `[{value: "...", description: "..."}]` | New unified format |


---

## Migration Examples

### Example 1: Simple Items (String Array)

**v4 Configuration:**
```hcl
resource "cloudflare_teams_list" "simple" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked-domains"
  type       = "DOMAIN"
  items      = ["malicious.com", "phishing.com", "spam.com"]
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_list" "simple" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked-domains"
  type       = "DOMAIN"

  items = [
    { value = "malicious.com" },
    { value = "phishing.com" },
    { value = "spam.com" }
  ]
}
```

**What Changed:**
- Resource type: `cloudflare_teams_list` → `cloudflare_zero_trust_list`
- String array → Array of objects with `value` field

---

### Example 2: Items with Descriptions (Blocks)

**v4 Configuration:**
```hcl
resource "cloudflare_teams_list" "with_descriptions" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "threat-list"
  type       = "IP"

  items_with_description {
    value       = "192.0.2.1"
    description = "Known attack source"
  }

  items_with_description {
    value       = "192.0.2.2"
    description = "Malware C2 server"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_list" "with_descriptions" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "threat-list"
  type       = "IP"

  items = [
    {
      value       = "192.0.2.1"
      description = "Known attack source"
    },
    {
      value       = "192.0.2.2"
      description = "Malware C2 server"
    }
  ]
}
```

**What Changed:**
- `items_with_description` blocks → unified `items` array
- Block structure preserved as objects

---

### Example 3: Mixed Items and Descriptions

**v4 Configuration:**
```hcl
resource "cloudflare_teams_list" "mixed" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "combined-list"
  type       = "DOMAIN"

  items = ["simple1.com", "simple2.com"]

  items_with_description {
    value       = "important.com"
    description = "Critical domain to block"
  }

  items_with_description {
    value       = "another.com"
    description = "Also important"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_list" "mixed" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "combined-list"
  type       = "DOMAIN"

  items = [
    # Items with descriptions listed first (API ordering)
    {
      value       = "important.com"
      description = "Critical domain to block"
    },
    {
      value       = "another.com"
      description = "Also important"
    },
    # Simple items without descriptions
    { value = "simple1.com" },
    { value = "simple2.com" }
  ]
}
```

**What Changed:**
- Both `items` and `items_with_description` merged into single `items` array
- Items with descriptions listed first to match API order
- Simple items converted to objects without `description` field

---

