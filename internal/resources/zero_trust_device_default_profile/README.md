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

## Field Mapping Reference

### Fields Removed During Migration

#### For Default Profiles:
- `name` - Not supported on default profile
- `description` - Not supported on default profile
- `match` - Custom profile only
- `precedence` - Custom profile only
- `enabled` - Always true for default profile
- `default` - Implicit (it's the default profile resource)

#### For Custom Profiles:
- `default` - Must be false for custom profiles
- `enabled` - Managed by API

#### For Both Profile Types:
- `fallback_domains` - Use separate `cloudflare_zero_trust_device_*_profile_local_domain_fallback` resource
- `exclude = []` - Removed if empty (Optional+Computed field)
- `include = []` - Removed if empty (Optional+Computed field)

### Fields Added During Migration

#### For Default Profiles Only:
- `register_interface_ip_with_dns` (required, default: `true`)
- `sccm_vpn_boundary_support` (required, default: `false`)

### Fields Transformed During Migration

#### For All Profiles:
- `service_mode_v2_mode` + `service_mode_v2_port` → `service_mode_v2 { mode, port }`

#### For Custom Profiles Only:
- `precedence` → `precedence + 900` (to avoid API conflicts)

### Fields Preserved As-Is

- `account_id`
- `allow_mode_switch`
- `allow_updates`
- `allowed_to_leave`
- `auto_connect`
- `captive_portal`
- `disable_auto_fallback`
- `exclude_office_ips`
- `support_url`
- `switch_locked`
- `tunnel_protocol`
- `exclude` (if not empty)
- `include` (if not empty)

---

## Routing Logic Decision Tree

```
Is default = true explicitly?
├─ YES → cloudflare_zero_trust_device_default_profile
└─ NO
   │
   ├─ Has match AND precedence?
   │  ├─ YES → cloudflare_zero_trust_device_custom_profile
   │  └─ NO → cloudflare_zero_trust_device_default_profile
   │
   └─ No default field present?
      ├─ Has match AND precedence?
      │  ├─ YES → cloudflare_zero_trust_device_custom_profile
      │  └─ NO → cloudflare_zero_trust_device_default_profile
```

**Priority:** The `default` field takes precedence. If `default = true`, it's always a default profile, even if `match`/`precedence` are present (though that would be invalid config).

---

## State Transformations

### Numeric Type Conversions

All integer fields in v4 state are converted to float64 in v5 state:
- `auto_connect`: Int → Float64
- `captive_portal`: Int → Float64
- `precedence` (custom profiles): Int → Float64

### ID Format Changes

#### Default Profile:
- v4 ID: `f037e56e89293a057740de681ac9abbe/profile_id` (composite)
- v5 ID: `f037e56e89293a057740de681ac9abbe` (account_id only)

#### Custom Profile:
- v4 ID: `f037e56e89293a057740de681ac9abbe/profile_id` (composite)
- v5 ID: `profile_id` (just the profile portion)
- v5 also adds: `policy_id` attribute with the profile_id value

---

## Common Migration Scenarios

### Scenario 1: You have one default profile
✅ Migrates smoothly to `cloudflare_zero_trust_device_default_profile`

### Scenario 2: You have multiple custom profiles with different precedence values
✅ Migrates smoothly - each gets +900 added to precedence to avoid conflicts
- Original precedence 10 → 910
- Original precedence 100 → 1000
- Original precedence 999 → 1899

### Scenario 3: You have fallback_domains configured
⚠️ Action required after migration:
- Create separate `cloudflare_zero_trust_device_default_profile_local_domain_fallback` or
- Create `cloudflare_zero_trust_device_custom_profile_local_domain_fallback` resources

### Scenario 4: You have `exclude` or `include` split tunneling configured
✅ Migrates smoothly - these fields are preserved if they have values
- Empty arrays (`exclude = []`) are removed from state
- Non-empty arrays are preserved

---

## Testing Your Migration

After migration, verify:

1. **Resource count matches:**
   ```bash
   terraform state list | grep device_profile | wc -l
   ```

2. **No unexpected changes:**
   ```bash
   terraform plan
   # Should show: No changes. Your infrastructure matches the configuration.
   ```

3. **For custom profiles, check precedence:**
   ```bash
   terraform state show 'cloudflare_zero_trust_device_custom_profile.example'
   # precedence should be original + 900
   ```

4. **For default profiles, check new required fields:**
   ```bash
   terraform state show 'cloudflare_zero_trust_device_default_profile.example'
   # Should have register_interface_ip_with_dns and sccm_vpn_boundary_support
   ```

---

## Troubleshooting

### "Missing required argument: precedence"
This happens with the registry v5 provider. The migration now handles this by:
- Keeping `precedence` in config for custom profiles
- Setting it to a high value (900 + original) to avoid conflicts

### "Resource produced drift"
Expected drift during migration:
- `fallback_domains` being removed/updated
- `exclude = []` being removed
- `default` and `gateway_unique_id` changing to computed values

These are covered by drift exemptions in e2e tests.

### "500 Internal Server Error"
If you see this during `terraform apply`:
- This was an issue with the v5 provider's handling of `exclude`/`include` fields
- **Fixed in provider** by making these fields `Optional+Computed`
- Ensure you're using the updated provider build

---

## Additional Resources

- Integration tests: `integration/v4_to_v5/testdata/zero_trust_device_default_profile/`
- Unit tests: `internal/resources/zero_trust_device_default_profile/v4_to_v5_test.go`
- Migration notes: `MIGRATION_NOTES_FALLBACK_DOMAINS.md` (root of tf-migrate)
- Provider fix: `PROVIDER_FIX_EXCLUDE_INCLUDE.md` (root of tf-migrate)
