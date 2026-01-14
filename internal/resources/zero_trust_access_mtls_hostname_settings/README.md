# Zero Trust Access mTLS Hostname Settings Migration Guide (v4 → v5)

This guide explains how `cloudflare_zero_trust_access_mtls_hostname_settings` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_zero_trust_access_mtls_hostname_settings` | `cloudflare_zero_trust_access_mtls_hostname_settings` | No change |
| `settings` | Blocks | Array attribute | Syntax change |
| Field defaults | Optional | Required with defaults | Added defaults |


---

## Migration Examples

### Example 1: Account-Level Settings

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "account" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings {
    hostname                      = "app.example.com"
    china_network                 = false
    client_certificate_forwarding = false
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "account" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = [{
    hostname                      = "app.example.com"
    china_network                 = false
    client_certificate_forwarding = false
  }]
}
```

**What Changed:**
- `settings { }` block → `settings = [{ }]` array attribute

---

### Example 2: Multiple Settings with Defaults

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "zone" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings {
    hostname = "api.example.com"
  }

  settings {
    hostname                      = "admin.example.com"
    client_certificate_forwarding = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "zone" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings = [
    {
      hostname                      = "api.example.com"
      china_network                 = false  # ← Default added
      client_certificate_forwarding = false  # ← Default added
    },
    {
      hostname                      = "admin.example.com"
      china_network                 = false  # ← Default added
      client_certificate_forwarding = true
    }
  ]
}
```

**What Changed:**
- Multiple blocks → Array with multiple objects
- Missing fields get default values: `false`

---

