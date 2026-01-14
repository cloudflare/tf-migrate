# Zero Trust Access Policy Migration Guide (v4 → v5)

This guide explains how `cloudflare_access_policy` resources migrate to `cloudflare_zero_trust_access_policy` in v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_access_policy` | `cloudflare_zero_trust_access_policy` | Renamed |
| `application_id` | Required | Removed | Moved to application |
| `precedence` | Supported | Removed | Moved to application |
| `zone_id` | Supported | Removed | Use on application |
| `session_duration` | Supported | Removed | Moved to application |
| `include/exclude/require` | Blocks | Array attributes | Structure change |
| Condition arrays | `email = ["a", "b"]` | Multiple condition objects | **EXPLOSION** |
| Boolean conditions | `everyone = true` | `everyone = {}` | Empty object |
| IP addresses | "192.168.1.1" | "192.168.1.1/32" | CIDR normalization |
| `approval_group` | Block | `approval_groups` array | Renamed + structure change |
| `connection_rules` | Block | Attribute object | Syntax change |


---

## Understanding Condition Explosion

Like Access Groups, policies use **condition explosion**: arrays in v4 become multiple separate condition objects in v5.

**v4 Pattern:**
```hcl
include {
  email = ["user1@example.com", "user2@example.com"]
}
```

**v5 Pattern:**
```hcl
include = [
  { email = { email = "user1@example.com" } },
  { email = { email = "user2@example.com" } }
]
```

---

## Migration Examples

### Example 1: Basic Allow Policy

**v4 Configuration:**
```hcl
resource "cloudflare_access_policy" "allow_users" {
  application_id = cloudflare_access_application.app.id
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  name           = "Allow Users"
  decision       = "allow"
  precedence     = 1

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
resource "cloudflare_zero_trust_access_policy" "allow_users" {
  name     = "Allow Users"
  decision = "allow"
  # application_id, zone_id, precedence moved to application resource

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
- `application_id`, `zone_id`, `precedence` removed (managed in application)
- Email array exploded into separate condition objects

---

### Example 2: Policy with IP Addresses

**v4 Configuration:**
```hcl
resource "cloudflare_access_policy" "office_network" {
  application_id = cloudflare_access_application.app.id
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  name           = "Office Network"
  decision       = "allow"
  precedence     = 1

  include {
    ip = [
      "192.168.1.1",
      "10.0.0.0/8",
      "203.0.113.5"
    ]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_policy" "office_network" {
  name     = "Office Network"
  decision = "allow"

  include = [
    {
      ip = {
        ip = "192.168.1.1/32"  # Single IPs get /32
      }
    },
    {
      ip = {
        ip = "10.0.0.0/8"  # CIDRs preserved
      }
    },
    {
      ip = {
        ip = "203.0.113.5/32"  # Single IPs get /32
      }
    }
  ]
}
```

**What Changed:**
- Single IP addresses automatically get `/32` suffix
- CIDR ranges preserved as-is

---

### Example 3: Block Policy with Exclusions

**v4 Configuration:**
```hcl
resource "cloudflare_access_policy" "block_contractors" {
  application_id = cloudflare_access_application.app.id
  account_id     = "f037e56e89293a057740de681ac9abbe"
  name           = "Block Contractors"
  decision       = "deny"
  precedence     = 10

  include {
    email_domain = ["contractor.com"]
  }

  exclude {
    email = ["approved@contractor.com"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_policy" "block_contractors" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block Contractors"
  decision   = "deny"

  include = [{
    email_domain = {
      domain = "contractor.com"
    }
  }]

  exclude = [{
    email = {
      email = "approved@contractor.com"
    }
  }]
}
```

**What Changed:**
- Fields removed from policy (moved to application)
- All condition blocks → arrays

---

### Example 4: MFA Policy with Approval

**v4 Configuration:**
```hcl
resource "cloudflare_access_policy" "mfa_required" {
  application_id = cloudflare_access_application.app.id
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  name           = "Require MFA"
  decision       = "allow"
  precedence     = 5

  include {
    everyone = true
  }

  require {
    device_posture = ["posture-id-1", "posture-id-2"]
  }

  approval_group {
    email_addresses    = ["manager@example.com"]
    approvals_needed   = 1
    email_list_uuid    = ""
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_policy" "mfa_required" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  name     = "Require MFA"
  decision = "allow"

  include = [{
    everyone = {}
  }]

  require = [
    {
      device_posture = {
        integration_uid = "posture-id-1"
      }
    },
    {
      device_posture = {
        integration_uid = "posture-id-2"
      }
    }
  ]

  approval_groups = [{
    email_addresses  = ["manager@example.com"]
    approvals_needed = 1
    email_list_uuid  = ""
  }]
}
```

**What Changed:**
- `everyone = true` → `everyone = {}`
- `device_posture` array exploded
- `approval_group` block → `approval_groups` array

---

### Example 5: GitHub Organization Policy

**v4 Configuration:**
```hcl
resource "cloudflare_access_policy" "github" {
  application_id = cloudflare_access_application.app.id
  account_id     = "f037e56e89293a057740de681ac9abbe"
  name           = "GitHub Teams"
  decision       = "allow"
  precedence     = 1

  include {
    github {
      name  = "my-org"
      teams = ["engineering", "devops"]
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_policy" "github" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "GitHub Teams"
  decision   = "allow"

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
    }
  ]
}
```

**What Changed:**
- `github` → `github_organization`
- Teams array → one condition per team

---

### Example 6: Connection Rules (SSH)

**v4 Configuration:**
```hcl
resource "cloudflare_access_policy" "ssh_policy" {
  application_id = cloudflare_access_application.app.id
  account_id     = "f037e56e89293a057740de681ac9abbe"
  name           = "SSH Access"
  decision       = "allow"
  precedence     = 1

  include {
    email = ["admin@example.com"]
  }

  connection_rules {
    ssh {
      usernames = ["root", "admin"]
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_policy" "ssh_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SSH Access"
  decision   = "allow"

  include = [{
    email = {
      email = "admin@example.com"
    }
  }]

  connection_rules = {
    ssh = {
      usernames = ["root", "admin"]
    }
  }
}
```

**What Changed:**
- `connection_rules { }` block → `connection_rules = { }` attribute
- Nested `ssh` block → object

---

