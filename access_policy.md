# Access Policy Resources: Investigation & Migration Analysis

This document captures the investigation findings about the three access policy resources
in the Cloudflare Terraform ecosystem and their migration compatibility.

## The Three Resources

There are **three distinct access policy resources** that are often confused:

| Resource Name | Provider Version | Has `application_id`? | API Endpoint | Scope |
|---------------|------------------|----------------------|--------------|-------|
| `cloudflare_access_policy` | v4 (legacy) | **Yes** (required) | `POST /access/apps/{app_id}/policies/` | Application-scoped |
| `cloudflare_zero_trust_access_policy` | v4 (devstack) | **No** | `POST /access/policies/` | Account-level |
| `cloudflare_zero_trust_access_policy` | v5 | **No** | `POST /access/policies/` | Account-level |

## Migration Compatibility

| Source Resource | Has `application_id`? | Target Resource | Migrates? |
|-----------------|----------------------|-----------------|-----------|
| `cloudflare_access_policy` | Yes | Inline `policies` in `cloudflare_zero_trust_access_application` | **NO** (manual) |
| `cloudflare_zero_trust_access_policy` (v4) | No | `cloudflare_zero_trust_access_policy` (v5) | **YES** |

## v5 Equivalent for Application-Scoped Policies

In v5, there is **no separate resource** for application-scoped policies. Instead, they are defined
**inline** within the `cloudflare_zero_trust_access_application` resource using the `policies` attribute.

### The `policies` Attribute

The v5 `cloudflare_zero_trust_access_application.policies` attribute supports two modes:

**1. Reference existing account-level policy by ID:**
```hcl
resource "cloudflare_zero_trust_access_application" "app" {
  account_id = "..."
  name       = "My App"
  domain     = "app.example.com"
  
  policies = [
    { id = cloudflare_zero_trust_access_policy.my_reusable_policy.id }
  ]
}
```

**2. Define inline application-exclusive policy:**
```hcl
resource "cloudflare_zero_trust_access_application" "app" {
  account_id = "..."
  name       = "My App"
  domain     = "app.example.com"
  
  policies = [
    {
      decision   = "allow"
      precedence = 1
      include    = [{ email = { email = "user@example.com" } }]
    }
  ]
}
```

### Migration Example: Application-Scoped Policy

**v4 (separate resources):**
```hcl
resource "cloudflare_access_application" "app" {
  account_id = "..."
  name       = "My App"
  domain     = "app.example.com"
}

resource "cloudflare_access_policy" "policy" {
  account_id     = "..."
  application_id = cloudflare_access_application.app.id
  name           = "Allow Users"
  decision       = "allow"
  precedence     = 1
  
  include {
    email = ["user@example.com"]
  }
}
```

**v5 (inline policy):**
```hcl
resource "cloudflare_zero_trust_access_application" "app" {
  account_id = "..."
  name       = "My App"
  domain     = "app.example.com"
  
  policies = [
    {
      decision   = "allow"
      precedence = 1
      include    = [{ email = { email = "user@example.com" } }]
    }
  ]
}
```

### Migration Complexity Summary

| v4 Resource | v5 Equivalent | Automated? | Complexity |
|-------------|---------------|------------|------------|
| `cloudflare_access_policy` (with `application_id`) | Inline `policies` in `cloudflare_zero_trust_access_application` | **No** | High - requires merging resources |
| `cloudflare_zero_trust_access_policy` (no `application_id`) | `cloudflare_zero_trust_access_policy` | **Yes** | Low - simple schema changes |

### Why Automated Migration is Difficult

Migrating `cloudflare_access_policy` (with `application_id`) to inline policies requires:

1. **Cross-resource transformation** - The policy must be merged INTO the application resource
2. **Resource deletion** - The standalone policy resource must be removed
3. **State manipulation** - Terraform state must be updated to reflect the merge
4. **Reference updates** - Any references to the old policy resource must be removed

This is fundamentally different from a simple rename + schema transformation.

## Why `cloudflare_access_policy` Cannot Migrate

### The Problem

The legacy `cloudflare_access_policy` resource creates **application-scoped policies** 
bound to a specific Access Application. These are fundamentally different API resources 
from account-level policies.

### Failure Sequence

When tf-migrate processes a `cloudflare_access_policy` resource:

1. **tf-migrate** renames `cloudflare_access_policy` → `cloudflare_zero_trust_access_policy`
2. **tf-migrate** removes `application_id`, `precedence`, `zone_id`, `session_duration`
3. **tf-migrate** generates a `moved` block
4. **Terraform** runs with v5 provider
5. **v5 provider** attempts `PUT /access/policies/{policy_id}`
6. **API returns 404** - the policy ID only exists at `/access/apps/{app_id}/policies/{policy_id}`

### Root Cause

Application-scoped policies and account-level policies are **different API resources**:

- **Application-scoped**: `GET/PUT/DELETE /access/apps/{application_id}/policies/{policy_id}`
- **Account-level**: `GET/PUT/DELETE /access/policies/{policy_id}`

The policy ID created under an application does not exist at the account-level endpoint.
They share a naming convention but are separate resources in the Cloudflare API.

## Schema Comparison

### `cloudflare_access_policy` (v4 Legacy)

```hcl
resource "cloudflare_access_policy" "example" {
  account_id     = "..."           # Optional (account OR zone)
  zone_id        = "..."           # Optional (account OR zone)
  application_id = "..."           # REQUIRED - ties policy to application
  name           = "..."           # Required
  decision       = "allow"         # Required
  precedence     = 1               # Required - order within application
  session_duration = "24h"         # Optional
  
  include { ... }                  # Block syntax
  exclude { ... }                  # Block syntax  
  require { ... }                  # Block syntax
  approval_group { ... }           # Block syntax (singular)
}
```

**Key characteristics:**
- `application_id` is required
- Creates policy at `/access/apps/{app_id}/policies/`
- `precedence` controls order within the application's policy list

### `cloudflare_zero_trust_access_policy` (v4 DevStack)

```hcl
resource "cloudflare_zero_trust_access_policy" "example" {
  account_id = "..."               # Required
  name       = "..."               # Required
  decision   = "allow"             # Required
  # NO application_id
  # NO precedence
  # NO zone_id
  # NO session_duration
  
  include { ... }                  # Block syntax (same as legacy)
  exclude { ... }
  require { ... }
  approval_group { ... }
}
```

**Key characteristics:**
- NO `application_id` - account-level resource
- Creates policy at `/access/policies/`
- Reusable across multiple applications

### `cloudflare_zero_trust_access_policy` (v5)

```hcl
resource "cloudflare_zero_trust_access_policy" "example" {
  account_id = "..."               # Required
  name       = "..."               # Required
  decision   = "allow"             # Required
  
  include = [{ ... }]              # Array attribute syntax
  exclude = [{ ... }]              # Array attribute syntax
  require = [{ ... }]              # Array attribute syntax
  approval_groups = [{ ... }]      # Renamed, array attribute
  
  # New fields
  session_duration = "24h"
  connection_rules = { ... }       # Object attribute
}
```

**Key characteristics:**
- Same API endpoint as v4 devstack (`/access/policies/`)
- Condition syntax changed (blocks → array attributes)
- `approval_group` → `approval_groups`

## Migration Paths

### For `cloudflare_zero_trust_access_policy` (v4 → v5)

**Automatic migration supported via tf-migrate:**
- HCL transformation handles syntax changes
- Provider state upgrader handles state migration
- `moved` blocks generated for resource rename (if applicable)

See `internal/resources/zero_trust_access_policy/README.md` for detailed transformation examples.

### For `cloudflare_access_policy` (v4 Legacy with `application_id`)

**Manual migration required.** See "v5 Equivalent for Application-Scoped Policies" section above for details.

**Options:**

1. **Convert to inline policies** (Recommended for app-specific policies)
   - Move policy definition into `cloudflare_zero_trust_access_application.policies` attribute
   - Delete the standalone `cloudflare_access_policy` resource
   - Run `terraform state rm` on the old policy before applying
   - See migration example above

2. **Convert to account-level reusable policies**
   - Create new `cloudflare_zero_trust_access_policy` resources (without `application_id`)
   - Reference them from applications via `policies = [{ id = ... }]`
   - Delete old application-scoped policies from state
   - Best for policies shared across multiple applications

3. **Hybrid approach**
   - Use inline policies for app-specific rules
   - Use account-level policies for reusable rules shared across apps

**Migration steps:**
```bash
# 1. Remove old policy from state (don't destroy - it will be replaced)
terraform state rm cloudflare_access_policy.my_policy

# 2. Update HCL to use inline policy in application resource

# 3. Apply - creates inline policy, application manages it
terraform apply
```

## Developer Notes

### Relevant Source Files

**tf-migrate:**
- `internal/resources/zero_trust_access_policy/v4_to_v5.go` - HCL transformation
  - Line 83: Removes `application_id`, `precedence`, `zone_id`, `session_duration`
  - Line 46: Returns resource rename mapping
- `internal/resources/zero_trust_access_policy/README.md` - Detailed transformation docs

**Provider (terraform-provider-cloudflare):**
- `internal/services/zero_trust_access_policy/migrations.go` - State upgrader registration
- `internal/services/zero_trust_access_policy/migration/v500/handler.go` - MoveState handler
- `internal/services/zero_trust_access_policy/migration/v500/transform.go` - State transformation
- `internal/services/zero_trust_access_policy/migration/v500/model.go` - Source/target models

**Reference (v4 provider schemas):**
- `cloudflare-terraform-v4/internal/services/zero_trust_access_policy/schema.go`
  - v4 `cloudflare_zero_trust_access_policy` schema (no `application_id`)
- `cloudflare-terraform-v4/internal/services/zero_trust_access_policy/migration/v500/source_schema.go`
  - Legacy `cloudflare_access_policy` schema (has `application_id`)

### E2E Test Impact

The test file `integration/v4_to_v5/testdata/zero_trust_access_application/input/zero_trust_access_application.tf` 
contains `cloudflare_access_policy` resources with `application_id` (lines 439-462):

```hcl
resource "cloudflare_access_policy" "ref_opt1" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.resourcename_opt1.id  # <-- Problem
  name           = "Policy referencing pattern9 opt1"
  precedence     = 1
  decision       = "allow"
  include { email = ["user@example.com"] }
}
```

**What happens:**
1. tf-migrate transforms this to `cloudflare_zero_trust_access_policy`
2. tf-migrate removes `application_id` and generates `moved` block
3. e2e test runs `terraform apply`
4. v5 provider tries `PUT /access/policies/{policy_id}`
5. **404 error** - policy doesn't exist at that endpoint

### Recommendations

1. **For e2e tests:** Remove or skip `cloudflare_access_policy` test cases that have `application_id`

2. **For tf-migrate:** Consider adding detection logic to:
   - Warn users when `cloudflare_access_policy` with `application_id` is detected
   - Skip generating `moved` blocks for these resources
   - Output guidance for manual migration

3. **For documentation:** Clearly communicate this limitation to end users before they attempt migration

## Open Questions

1. Should tf-migrate emit a warning when it encounters `cloudflare_access_policy` with `application_id`?
2. Should the migration skip these resources entirely, or still transform the HCL (without `moved` block)?
3. ~~Is there a path to support application-scoped policy migration in the future?~~
   - **Answered:** Yes, but it requires complex cross-resource transformation to merge policies into
     the application's inline `policies` attribute. This is not a simple rename operation.

## Future Enhancement: Automated Application-Scoped Policy Migration

If tf-migrate were to support automated migration of `cloudflare_access_policy` (with `application_id`),
it would need to:

1. **Detect** `cloudflare_access_policy` resources with `application_id` attribute
2. **Find** the corresponding `cloudflare_access_application` resource (by matching `application_id` reference)
3. **Transform** the policy into the v5 inline `policies` syntax
4. **Inject** the policy into the application resource's `policies` attribute
5. **Remove** the standalone policy resource block
6. **NOT generate** a `moved` block (resource is being merged, not moved)
7. **Emit** state migration instructions for users to run `terraform state rm`

This is significantly more complex than current tf-migrate transformations and may be out of scope.
