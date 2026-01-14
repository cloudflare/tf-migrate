# Zero Trust Access Group Migration Guide (v4 → v5)

This guide explains how `cloudflare_access_group` resources migrate to `cloudflare_zero_trust_access_group` in v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_access_group` | `cloudflare_zero_trust_access_group` | Renamed |
| `include/exclude/require` | Blocks | Array attributes | Structure change |
| Selector arrays | `email = ["a", "b"]` | Multiple selector objects | **EXPLOSION** |
| Boolean selectors | `everyone = true` | `everyone = {}` | Empty object |
| `azure` | Selector type | `azure_ad` | Renamed |
| `github` | Simple selector | `github_organization` with teams | Renamed + expansion |
| GSuite/Azure/Okta arrays | Multiple values | **First value only** | ⚠️ Data loss |


---

## Understanding Selector Explosion

The most important concept in this migration is **selector explosion**: arrays in v4 become multiple separate selector objects in v5.

**v4 Pattern:**
```hcl
include {
  email = ["user1@example.com", "user2@example.com", "user3@example.com"]
}
```

**v5 Pattern:**
```hcl
include = [
  { email = { email = "user1@example.com" } },
  { email = { email = "user2@example.com" } },
  { email = { email = "user3@example.com" } }
]
```

**Each array element becomes a separate selector object!**

---

## Migration Examples

### Example 1: Simple Email Group

**v4 Configuration:**
```hcl
resource "cloudflare_access_group" "email_group" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Email Users"

  include {
    email = [
      "alice@example.com",
      "bob@example.com"
    ]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_group" "email_group" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Email Users"

  include = [
    {
      email = {
        email = "alice@example.com"
      }
    },
    {
      email = {
        email = "bob@example.com"
      }
    }
  ]
}
```

**What Changed:**
- `include` block → `include` array attribute
- Each email becomes a separate selector object
- Double nesting: `email = { email = "..." }`

---

### Example 2: Multiple Selector Types

**v4 Configuration:**
```hcl
resource "cloudflare_access_group" "mixed" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Mixed Selectors"

  include {
    email        = ["user@example.com"]
    email_domain = ["example.com", "company.com"]
    ip           = ["192.168.1.0/24", "10.0.0.0/8"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_group" "mixed" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Mixed Selectors"

  include = [
    {
      email = {
        email = "user@example.com"
      }
    },
    {
      email_domain = {
        domain = "example.com"
      }
    },
    {
      email_domain = {
        domain = "company.com"
      }
    },
    {
      ip = {
        ip = "192.168.1.0/24"
      }
    },
    {
      ip = {
        ip = "10.0.0.0/8"
      }
    }
  ]
}
```

**What Changed:**
- All selector arrays exploded into individual objects
- `email_domain` field renamed to `domain` inside selector
- IP addresses preserved with double nesting

---

### Example 3: Boolean Selectors (Everyone)

**v4 Configuration:**
```hcl
resource "cloudflare_access_group" "everyone" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "All Users"

  include {
    everyone = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_group" "everyone" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "All Users"

  include = [{
    everyone = {}
  }]
}
```

**What Changed:**
- `everyone = true` → `everyone = {}` (empty object)
- Boolean selectors become empty objects

---

### Example 4: GitHub Organization with Teams

**v4 Configuration:**
```hcl
resource "cloudflare_access_group" "github_teams" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "GitHub Teams"

  include {
    github {
      name  = "my-org"
      teams = ["engineering", "devops", "security"]
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_group" "github_teams" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "GitHub Teams"

  include = [
    {
      github_organization = {
        name = "my-org"
        team = "engineering"
      }
    },
    {
      github_organization = {
        name = "my-org"
        team = "devops"
      }
    },
    {
      github_organization = {
        name = "my-org"
        team = "security"
      }
    }
  ]
}
```

**What Changed:**
- `github` → `github_organization`
- `teams` array → multiple selectors with `team` (singular)
- One selector per team!

---

### Example 5: Azure AD (Data Loss Warning)

**v4 Configuration:**
```hcl
resource "cloudflare_access_group" "azure" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Azure Groups"

  include {
    azure {
      id = [
        "group-id-1",
        "group-id-2",
        "group-id-3"
      ]
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_group" "azure" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Azure Groups"

  include = [{
    azure_ad = {
      id = "group-id-1"  # ⚠️ ONLY FIRST ID KEPT
    }
  }]
}
```

**⚠️ DATA LOSS WARNING:**
- `azure` → `azure_ad`
- **Only the first ID is kept!**
- Groups `group-id-2` and `group-id-3` are lost
- API limitation: only supports single ID

**Workaround:** Create multiple groups, one per Azure AD group.

---

### Example 6: Include, Exclude, and Require

**v4 Configuration:**
```hcl
resource "cloudflare_access_group" "complex" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Complex Rules"

  include {
    email_domain = ["example.com"]
  }

  exclude {
    email = ["blocked@example.com"]
  }

  require {
    geo = ["US", "CA", "GB"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_group" "complex" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Complex Rules"

  include = [{
    email_domain = {
      domain = "example.com"
    }
  }]

  exclude = [{
    email = {
      email = "blocked@example.com"
    }
  }]

  require = [
    {
      geo = {
        country_code = "US"
      }
    },
    {
      geo = {
        country_code = "CA"
      }
    },
    {
      geo = {
        country_code = "GB"
      }
    }
  ]
}
```

**What Changed:**
- All three blocks → array attributes
- `geo` array exploded with `country_code` field

---

