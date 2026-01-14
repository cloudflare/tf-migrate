# Zero Trust Device Profile Migration Guide (v4 → v5)

This guide explains how `cloudflare_zero_trust_device_profiles` resources migrate from v4 to v5.

## Quick Reference

| v4 Attributes | v5 Resource Type |
|---------------|------------------|
| `default = true` | `cloudflare_zero_trust_device_default_profile` |
| `default = false` + `match` + `precedence` | `cloudflare_zero_trust_device_custom_profile` |
| No `default` + `match` + `precedence` | `cloudflare_zero_trust_device_custom_profile` |
| `default = false` (missing `match` or `precedence`) | `cloudflare_zero_trust_device_default_profile` |
| No attributes that identify custom profile | `cloudflare_zero_trust_device_default_profile` |

## Migration Examples

### Example 1: Default Profile (Explicit `default = true`)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_device_profiles" "corp" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Corporate Profile"
  description = "Profile for corporate devices"
  default     = true

  allow_mode_switch = true
  auto_connect      = 0
  captive_portal    = 180
  tunnel_protocol   = "wireguard"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile" "corp" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  # name, description, default removed

  allow_mode_switch = true
  auto_connect      = 0
  captive_portal    = 180
  tunnel_protocol   = "wireguard"

  # New required fields
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}
```

**What Changed:**
- Resource type: `cloudflare_zero_trust_device_profiles` → `cloudflare_zero_trust_device_default_profile`
- Removed fields: `name`, `description`, `default`, `enabled`
- Added fields: `register_interface_ip_with_dns`, `sccm_vpn_boundary_support`

---

### Example 2: Custom Profile (Has `match` and `precedence`)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_device_profiles" "contractors" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Contractor Profile"
  description = "Profile for contractor devices"
  match       = "identity.email == \"contractor@example.com\""
  precedence  = 100

  allow_mode_switch = false
  auto_connect      = 15
  tunnel_protocol   = "wireguard"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Contractor Profile"
  description = "Profile for contractor devices"
  match       = "identity.email == \"contractor@example.com\""
  precedence  = 1000  # ← TRANSFORMED: 100 + 900

  allow_mode_switch = false
  auto_connect      = 15
  tunnel_protocol   = "wireguard"
}
```

**What Changed:**
- Resource type: `cloudflare_zero_trust_device_profiles` → `cloudflare_zero_trust_device_custom_profile`
- Removed fields: `default`, `enabled`
- **Precedence transformed:** Original value + 900 (to avoid API conflicts)
- Kept fields: `name`, `description`, `match`, `precedence` (all required for custom profiles)

---

### Example 3: Default Profile (Implicit - No Routing Attributes)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_device_profiles" "simple" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  name           = "Simple Profile"
  auto_connect   = 0
  captive_portal = 300
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile" "simple" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  auto_connect   = 0
  captive_portal = 300

  # New required fields
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}
```

**What Changed:**
- Routes to default profile because no `match`/`precedence` present
- Removed: `name` field
- Added: `register_interface_ip_with_dns`, `sccm_vpn_boundary_support`

---

### Example 4: Service Mode v2 Transformation

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_device_profiles" "proxy" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  default              = true
  service_mode_v2_mode = "proxy"
  service_mode_v2_port = 8080
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile" "proxy" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  # Flat fields merged into nested object
  service_mode_v2 = {
    mode = "proxy"
    port = 8080
  }

  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}
```

**What Changed:**
- `service_mode_v2_mode` + `service_mode_v2_port` → `service_mode_v2` nested object

**Note:** If `service_mode_v2_mode = "warp"` (v4 default) and port is not set, the entire field is omitted.

---

### Example 5: Fallback Domains (Removed from State)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_device_profiles" "with_fallback" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  default    = true

  # Note: fallback_domains in v4 state but not in config
  # (managed by separate cloudflare_zero_trust_local_fallback_domain resource)
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile" "with_fallback" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false

  # fallback_domains removed from state during migration
  # Use separate cloudflare_zero_trust_device_default_profile_local_domain_fallback resource
}
```

**Migration Behavior:**
- `fallback_domains` is removed from state
- `exclude = []` is removed from state
- These fields are `Computed` in v5 - the API manages them

**Post-Migration Action Required:**
If you had fallback domains in v4, create the new v5 resource:

```hcl
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "corp_domains" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix      = "corp.example.com"
      description = "Corporate network"
      dns_server  = ["10.0.0.1", "10.0.0.2"]
    }
  ]
}
```

---

### Example 6: Old Resource Name (Deprecated v4 Name)

**v4 Configuration:**
```hcl
resource "cloudflare_device_settings_policy" "legacy" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  default        = true
  auto_connect   = 0
  captive_portal = 180
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile" "legacy" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  auto_connect   = 0
  captive_portal = 180

  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}
```

**What Changed:**
- Both `cloudflare_zero_trust_device_profiles` and `cloudflare_device_settings_policy` migrate the same way

---

