# Zero Trust Split Tunnel Migration (v4 → v5)

## Overview

In Terraform Provider v4, split tunnel configurations were managed as separate `cloudflare_split_tunnel` resources. In v5, split tunnel settings are embedded directly within device profile resources (`cloudflare_zero_trust_device_default_profile` and `cloudflare_zero_trust_device_custom_profile`).

This migration uses the **cross-resource pattern** to merge split tunnel resources into their parent device profiles during migration.

## Migration Strategy

### Cross-Resource Pattern

This migration follows the same pattern as `list`/`list_item`:

1. **Split Tunnel Migrator**: Marks `cloudflare_split_tunnel` resources for removal
2. **Device Profile Migrator**: Calls `ProcessCrossResourceConfigMigration()` to merge split tunnels
3. **Cross-Resource Function**: Scans entire file, matches split tunnels to profiles, merges data

### Resource Mapping

```
V4 Structure:
├─ cloudflare_zero_trust_device_profiles (or cloudflare_device_settings_policy)
└─ cloudflare_split_tunnel (separate resource)
   └─ policy_id (optional, references device profile)

V5 Structure:
├─ cloudflare_zero_trust_device_default_profile
│  ├─ exclude = [...]  (embedded)
│  └─ include = [...]  (embedded)
└─ cloudflare_zero_trust_device_custom_profile
   ├─ exclude = [...]  (embedded)
   └─ include = [...]  (embedded)
```

## How It Works

### 1. Device Profile Migration Triggers Cross-Resource Merge

When a device profile resource is migrated, it calls:

```go
if ctx.CFGFile != nil {
    zero_trust_split_tunnel.ProcessCrossResourceConfigMigration(ctx.CFGFile)
}
```

### 2. Cross-Resource Function Scans and Merges

The `ProcessCrossResourceConfigMigration` function:

```go
// Step 1: Collect all resources
defaultProfileBlock := findDefaultProfile()
customProfiles := findCustomProfiles()
splitTunnels := findSplitTunnels()

// Step 2: Parse policy_id references
for each split_tunnel {
    parentName := extractParentProfileName(split_tunnel)
    // No policy_id → default profile
    // Has policy_id → custom profile with that name
}

// Step 3: Merge tunnels into profiles
for each profile {
    tunnels := splitTunnelsByParent[profileName]
    mergeSplitTunnelIntoProfile(tunnels, profile)
}

// Step 4: Remove split_tunnel resources
removeAllSplitTunnelBlocks()
```

### 3. Reference Parsing

The migration parses `policy_id` references to determine which profile to merge into:

```hcl
# Example 1: No policy_id → Default Profile
resource "cloudflare_split_tunnel" "default_excludes" {
  account_id = "abc123"
  # No policy_id means applies to default profile
  mode = "exclude"
  tunnels { ... }
}

# Example 2: Direct Reference → Custom Profile
resource "cloudflare_split_tunnel" "contractor_excludes" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.contractors.id
  mode = "exclude"
  tunnels { ... }
}
```

**Supported reference formats:**
- ✅ `cloudflare_zero_trust_device_custom_profile.name.id`
- ✅ `cloudflare_zero_trust_device_profiles.name.id` (v4)
- ✅ `cloudflare_device_settings_policy.name.id` (v4 deprecated)
- ✅ `cloudflare_zero_trust_device_custom_profile["name"].id`
- ❌ `var.some_policy_id` (generates warning)
- ❌ `local.some_policy_id` (generates warning)
- ❌ `module.foo.policy_id` (generates warning)

## Migration Examples

### Example 1: Default Profile with Exclude Tunnels

**V4 Configuration:**
```hcl
resource "cloudflare_zero_trust_device_profiles" "default" {
  account_id = "abc123"
  default    = true
}

resource "cloudflare_split_tunnel" "default_excludes" {
  account_id = "abc123"
  # No policy_id means default profile
  mode = "exclude"

  tunnels {
    address     = "192.168.1.0/24"
    description = "Local network"
  }

  tunnels {
    address = "10.0.0.0/8"
  }
}
```

**V5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"

  exclude = [
    {
      address     = "192.168.1.0/24"
      description = "Local network"
    },
    {
      address = "10.0.0.0/8"
    }
  ]
}

# cloudflare_split_tunnel resource removed
```

### Example 2: Custom Profile with Multiple Modes

**V4 Configuration:**
```hcl
resource "cloudflare_zero_trust_device_profiles" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 100
}

resource "cloudflare_split_tunnel" "contractor_excludes" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_profiles.contractors.id
  mode       = "exclude"

  tunnels {
    address = "10.0.0.0/8"
  }
}

resource "cloudflare_split_tunnel" "contractor_includes" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_profiles.contractors.id
  mode       = "include"

  tunnels {
    address = "corporate.example.com/32"
  }
}
```

**V5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 1000  # Transformed: 100 + 900

  exclude = [
    {
      address = "10.0.0.0/8"
    }
  ]

  include = [
    {
      address = "corporate.example.com/32"
    }
  ]
}

# Both cloudflare_split_tunnel resources removed
```

### Example 3: Unparseable Reference (Warning Added)

**V4 Configuration:**
```hcl
variable "some_policy_id" {
  default = "policy-abc123"
}

resource "cloudflare_split_tunnel" "variable_ref" {
  account_id = "abc123"
  policy_id  = var.some_policy_id  # Cannot parse
  mode       = "exclude"

  tunnels {
    address = "192.168.0.0/16"
  }
}
```

**V5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
  # MIGRATION WARNING: Split tunnel "variable_ref" has unparseable policy_id reference - manual migration required
}

# Split tunnel removed - user must manually migrate
```

## Warning Conditions

The migration adds warnings (but still removes resources) in these cases:

### 1. Unparseable policy_id Reference

```hcl
# MIGRATION WARNING: Split tunnel "name" has unparseable policy_id reference - manual migration required
```

**Causes:**
- Variable references: `policy_id = var.some_policy_id`
- Local references: `policy_id = local.some_policy_id`
- Module outputs: `policy_id = module.foo.policy_id`
- Complex expressions: `policy_id = condition ? id1 : id2`

**Action Required:** User must manually add split tunnel configuration to the appropriate device profile.

### 2. Referenced Profile Not Found

```hcl
# MIGRATION WARNING: Split tunnel "name" references profile "nonexistent" which was not found - manual migration required
```

**Causes:**
- Profile defined in a different file
- Profile name typo in reference
- Profile doesn't exist

**Action Required:** User must ensure the referenced profile exists and manually merge the split tunnel configuration.

### 3. No Default Profile Found

```hcl
# MIGRATION WARNING: No default device profile found - create cloudflare_zero_trust_device_default_profile resource first
```

**Causes:**
- Split tunnel with no `policy_id` but no default profile resource exists

**Action Required:** User must create a default profile resource first, then re-run migration.

## Testing

### Unit Tests

All tests are in `v4_to_v5_test.go`:

```bash
go test -v ./internal/resources/zero_trust_split_tunnel/...
```

**Test Coverage:**
- ✅ Resource removal (split_tunnel marked for deletion)
- ✅ Default profile exclude merge
- ✅ Default profile include merge
- ✅ Custom profile merge (multiple tunnels)
- ✅ Multiple modes for same profile (exclude + include)
- ✅ Unparseable reference warnings
- ✅ Missing profile warnings

### Integration Testing

Integration tests should be added in a separate test file to verify the end-to-end migration flow with the device profile migrator.

## Implementation Details

### Key Functions

**`ProcessCrossResourceConfigMigration(file *hclwrite.File)`**
- Scans entire file for device profiles and split tunnels
- Matches split tunnels to profiles via `policy_id` parsing
- Merges tunnel configuration into profile blocks
- Removes split tunnel blocks
- Adds warnings for unmergeable tunnels

**`extractParentProfileName(block *hclwrite.Block) string`**
- Parses `policy_id` attribute to extract resource name
- Returns empty string for default profile (no `policy_id`)
- Returns empty string if reference cannot be parsed

**`mergeSplitTunnelIntoProfile(tunnelBlock, profileBlock *hclwrite.Block)`**
- Extracts mode (`exclude` or `include`)
- Builds tunnel objects from `tunnels` blocks
- Sets appropriate attribute on profile

**State Merging Functions:**

**`ProcessCrossResourceStateMigration(stateJSON string) string`**
- Scans state for device profiles and split tunnels
- Matches split tunnels to profiles via policy_id/account_id
- Merges tunnel state into profile state
- Removes split tunnel resources from state

**`extractSplitTunnelStateInfo(resource gjson.Result, index int) *splitTunnelStateInfo`**
- Extracts split tunnel state data from resource instance
- Parses policy_id, mode, and tunnels array

**`mergeTunnelsIntoProfileState(jsonStr string, profileIndex int, mode string, tunnels []splitTunnelStateInfo) string`**
- Merges tunnel data into device profile's exclude/include attributes in state

**`removeSplitTunnelResourcesFromState(jsonStr string) string`**
- Removes all cloudflare_split_tunnel resources from state

### Design Decisions

1. **Idempotent**: Both config and state migration functions can be called multiple times safely
2. **Option 2 Warnings**: Adds explicit warnings for unparseable references (doesn't silently drop data)
3. **Full State Merging**: Split tunnel state is merged into device profile state (matching list_item pattern)
4. **File-Level Warnings**: Warnings added to default profile or file level comments
5. **No Refresh Required**: State is correctly merged - users can run `terraform apply` immediately

### Limitations

1. **Variable References**: Cannot parse variable/local/module references
2. **Cross-File References**: Only merges resources within same file
3. **Existing Tunnels**: Does not merge if profile already has `exclude`/`include` attributes

## Related Resources

- **Device Profile Migration**: `internal/resources/zero_trust_device_default_profile/`
- **List/List Item Pattern**: `internal/resources/list_item/` (similar cross-resource pattern)
- **Documentation**: See `/prompts/notes/list_list_item_migration_analysis.md`

## Maintenance Notes

- If device profile schema changes, update `mergeSplitTunnelIntoProfile()`
- If new v4 device profile resource names added, update `ProcessCrossResourceConfigMigration()`
- Warning messages should clearly indicate manual action required
