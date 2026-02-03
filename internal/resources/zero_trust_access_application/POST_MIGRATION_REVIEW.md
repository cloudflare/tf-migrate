# Post-Migration Review: zero_trust_access_application

## Migration Summary
- **Resource**: `cloudflare_access_application` → `cloudflare_zero_trust_access_application`
- **Migration Type**: v4 to v5 (resource rename + schema changes)
- **Complexity**: High (multiple nested block transformations, default value changes, type conversions)
- **Status**: COMPLETE

---

## What Went Well

1. **Comprehensive test coverage**: Unit tests covered 25+ transformation scenarios including edge cases
2. **Integration test data**: Full testdata directory with input/expected HCL files
3. **Dual registration**: Successfully registered migrator for both old (`cloudflare_access_application`) and new (`cloudflare_zero_trust_access_application`) resource names
4. **Provider-side MoveState**: Implemented full MoveState handler in the provider for seamless state migration
5. **Sweepers already existed**: Test sweepers were already in place from previous work

---

## Critical Pitfalls & Gotchas

### 1. Moved Block Generation Bug (CRITICAL)

**Symptom**: Test failure with `no schema available for cloudflare_access_policy`

**Root Cause**: Resource name was captured AFTER the rename operation instead of BEFORE.

```go
// WRONG - captures the new type name, not the resource name
tfhcl.RenameResourceType(block, "cloudflare_access_policy", "cloudflare_zero_trust_access_policy")
resourceName := block.Labels()[0]  // Returns "cloudflare_zero_trust_access_policy"!

// CORRECT - capture resource name BEFORE rename
resourceName := tfhcl.GetResourceName(block)  // Returns "test"
tfhcl.RenameResourceType(block, "cloudflare_access_policy", "cloudflare_zero_trust_access_policy")
```

**Impact**: Moved blocks were generated as:
```hcl
# WRONG
moved {
  from = cloudflare_access_policy.cloudflare_zero_trust_access_policy
  to   = cloudflare_zero_trust_access_policy.cloudflare_zero_trust_access_policy
}

# CORRECT
moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}
```

**Prevention**: Always capture identifying information BEFORE modifying the block.

---

### 2. Default Value Change: `http_only_cookie_attribute`

**v4 Behavior**: Defaults to `false`
**v5 Behavior**: Defaults to `true`

**Problem**: If migration doesn't explicitly set this attribute, existing applications would have their cookie behavior changed.

**Solution**: Migration explicitly sets `http_only_cookie_attribute = "false"` for applicable app types to preserve v4 behavior:

```go
// Only applicable for types: self_hosted, ssh, vnc, rdp, mcp_portal
appType := tfhcl.ExtractStringFromAttribute(body.GetAttribute("type"))
if appType == "self_hosted" || appType == "ssh" || appType == "vnc" || appType == "rdp" || appType == "mcp_portal" {
    tfhcl.EnsureAttribute(body, "http_only_cookie_attribute", "false")
}
```

**Gotcha**: This attribute is NOT applicable for `saas` or `warp` application types.

---

### 3. `self_hosted_domains` Set to List + Sorting

**v4**: Set type (unordered)
**v5**: Set type but provider returns sorted alphabetically

**Problem**: Plan drift if domains in config don't match provider's sorted order.

**Solution**: Sort the array alphabetically during config transformation:

```go
tfhcl.SortStringArrayAttribute(body, "self_hosted_domains")
```

---

### 4. Block to Attribute Syntax Changes

Multiple nested blocks changed from block syntax to attribute syntax:

| Field | v4 Syntax | v5 Syntax |
|-------|-----------|-----------|
| `cors_headers` | `cors_headers { ... }` | `cors_headers = { ... }` |
| `saas_app` | `saas_app { ... }` | `saas_app = { ... }` |
| `scim_config` | `scim_config { ... }` | `scim_config = { ... }` |
| `landing_page_design` | `landing_page_design { ... }` | `landing_page_design = { ... }` |
| `destinations` | `destinations { ... }` | `destinations = [{ ... }]` |
| `footer_links` | `footer_links { ... }` | `footer_links = [{ ... }]` |

**Gotcha**: Deeply nested blocks require recursive transformation:
- `saas_app.custom_attribute` → `saas_app.custom_attributes` (rename + plural)
- `saas_app.custom_claim` → `saas_app.custom_claims` (rename + plural)
- `saas_app.custom_attribute.source` → block to attribute
- `scim_config.authentication` → block to attribute
- `scim_config.mappings.operations` → block to attribute

---

### 5. `policies` Array Structure Change

**v4**: Array of strings (policy IDs)
```hcl
policies = [
  cloudflare_access_policy.allow.id,
  cloudflare_access_policy.deny.id
]
```

**v5**: Array of objects with `id` and `precedence`
```hcl
policies = [
  {
    id         = cloudflare_zero_trust_access_policy.allow.id
    precedence = 1
  },
  {
    id         = cloudflare_zero_trust_access_policy.deny.id
    precedence = 2
  }
]
```

**Implementation**: Use index-based precedence assignment:

```go
tfhcl.ConvertArrayAttributeToObjectArray(body, "policies", func(element hclwrite.Tokens, index int) map[string]hclwrite.Tokens {
    return map[string]hclwrite.Tokens{
        "id": element,
        "precedence": {
            &hclwrite.Token{
                Type:  hclsyntax.TokenNumberLit,
                Bytes: []byte(strconv.Itoa(index + 1)),
            },
        },
    }
})
```

---

### 6. `toset()` Function Wrapper Removal

**v4**: Many array fields wrapped in `toset()`
```hcl
allowed_idps = toset(["idp1", "idp2"])
```

**v5**: Direct array syntax
```hcl
allowed_idps = ["idp1", "idp2"]
```

**Affected Fields**:
- `allowed_idps`
- `custom_pages`
- `self_hosted_domains`

---

### 7. Deprecated Field Removal

**`domain_type`**: Completely removed in v5

```go
tfhcl.RemoveAttributes(body, "domain_type")
```

---

### 8. State Transformation: MaxItems:1 Array to Object

In Terraform state, `MaxItems:1` blocks are stored as single-element arrays. v5 expects objects.

```json
// v4 state
"cors_headers": [{ "max_age": 3600 }]

// v5 state
"cors_headers": { "max_age": 3600 }
```

---

### 9. Empty Value Normalization

**Problem**: v4 provider stores empty arrays `[]` and `false` booleans, but v5 normalizes them to `null`.

**Solution**: Remove default/empty values from config to prevent drift:

```go
func removeDefaultValueAttributes(body *hclwrite.Body) {
    // Boolean attributes that should be removed if false
    boolAttrs := []string{
        "auto_redirect_to_identity",
        "enable_binding_cookie",
        "options_preflight_bypass",
        "service_auth_401_redirect",
        "skip_interstitial",
    }
    // ... removal logic
}
```

---

### 10. OIDC Scopes Ordering

**Problem**: Provider returns OIDC scopes in a specific order (openid first, then profile, email, etc.)

**Solution**: Custom sort function for OIDC spec ordering:

```go
func sortOIDCScopes(strings []string) {
    scopeOrder := map[string]int{
        "openid":         1,
        "profile":        2,
        "email":          3,
        "address":        4,
        "phone":          5,
        "offline_access": 6,
    }
    // ... sort logic
}
```

---

### 11. SAML `name_by_idp` Structure Change

**v4**: Map type
```hcl
source {
  name_by_idp = {
    azure = "department"
    okta  = "dept"
  }
}
```

**v5**: Array of objects
```hcl
source = {
  name_by_idp = [
    { idp_id = "azure", source_name = "department" },
    { idp_id = "okta", source_name = "dept" }
  ]
}
```

**Note**: This only applies to SAML `custom_attributes`. OIDC `custom_claims` keeps the map format.

---

### 12. `target_criteria.target_attributes` Structure Change

**v4**: Array of `{ name, values }` objects
```hcl
target_criteria {
  target_attributes {
    name   = "hostname"
    values = ["server1", "server2"]
  }
}
```

**v5**: Map where keys are names
```hcl
target_criteria = [{
  target_attributes = {
    "hostname" = ["server1", "server2"]
  }
}]
```

---

## Type Conversion Gotchas

| Field | v4 Type | v5 Type |
|-------|---------|---------|
| `cors_headers.max_age` | `int64` | `float64` |
| `approval_groups.approvals_needed` | `int64` | `float64` |

---

## Provider Integration: MoveState vs UpgradeState

**MoveState**: Used when resources are renamed via `moved` block:
```hcl
moved {
  from = cloudflare_access_application.example
  to   = cloudflare_zero_trust_access_application.example
}
```

**UpgradeState**: Used for direct provider upgrades (v5.12 → latest):
- Schema version 0: No-op upgrade (v5 schema compatible)
- Schema version 2: v4 state format upgrade

---

## Test Categories Added

### Unit Tests (v4_to_v5_test.go)
1. Policy array transformation (references, literals, mixed)
2. `domain_type` removal
3. Block to attribute conversions (cors_headers, saas_app, scim_config)
4. `toset()` wrapper removal
5. `self_hosted_domains` sorting
6. Resource type rename + moved block generation
7. Empty value removal
8. SAAS app nested structures (custom_attributes, custom_claims)
9. Meta-arguments (count, for_each)

### Integration Tests (testdata/)
- 28 resource patterns covering:
  - Minimal/maximal configurations
  - SAAS apps (SAML, OIDC)
  - SSH apps with target criteria
  - Self-hosted domains
  - Domain type removal
  - Count/for_each meta-arguments
  - Special characters in names

### Provider Migration Tests (migrations_test.go)
- `TestMigrateZeroTrustAccessApplication_V4toV5_Basic`
- `TestMigrateZeroTrustAccessApplication_V4toV5_SaaSApp`
- `TestMigrateZeroTrustAccessApplication_V4toV5_WithCORSHeaders`
- `TestMigrateZeroTrustAccessApplication_V4toV5_WithPolicies`
- `TestMigrateZeroTrustAccessApplication_FromV5_12`
- `TestMigrateZeroTrustAccessApplication_FromV5_14`
- `TestMigrateZeroTrustAccessApplication_FromV5_15`

---

## Mistakes Made & Lessons Learned

| Mistake | Impact | Root Cause | Prevention |
|---------|--------|------------|------------|
| Captured resource name AFTER rename | Moved blocks generated with wrong `from` address | Misunderstood when `block.Labels()` values change | Always capture identifiers before mutating blocks |
| Initially added v5.x tests to wrong package | Tests weren't running | User was running `zero_trust_access_application` tests but I edited `zero_trust_access_policy` | Double-check which package user is testing |
| Missing `http_only_cookie_attribute` explicit set | Would have changed app behavior silently | Didn't check v4 vs v5 default value differences | Always compare default values between versions |

---

## Reusable Patterns Discovered

### 1. Safe Resource Name Capture
```go
resourceName := tfhcl.GetResourceName(block)  // Capture FIRST
tfhcl.RenameResourceType(block, oldType, newType)  // Then rename
```

### 2. Conditional Default Value Setting
```go
appType := tfhcl.ExtractStringFromAttribute(body.GetAttribute("type"))
if appType == "self_hosted" || appType == "ssh" {
    tfhcl.EnsureAttribute(body, "attribute_name", "default_value")
}
```

### 3. Array Sorting for Provider Normalization
```go
tfhcl.SortStringArrayAttribute(body, "self_hosted_domains")
```

### 4. Nested Block Transformation Chain
```go
// Process inner blocks first, then convert parent
innerBlocks := tfhcl.FindBlocksByType(parentBody, "inner_block")
for _, innerBlock := range innerBlocks {
    // Transform inner block
}
tfhcl.ConvertSingleBlockToAttribute(parentBody, "inner_block", "inner_block")
```

---

## Files Modified/Created

### tf-migrate
- `internal/resources/zero_trust_access_application/v4_to_v5.go` - Main migrator
- `internal/resources/zero_trust_access_application/v4_to_v5_test.go` - Unit tests
- `internal/resources/zero_trust_access_application/README.md` - User documentation
- `internal/registry/registry.go` - Migrator registration
- `integration/v4_to_v5/testdata/zero_trust_access_application/` - Integration tests
- `e2e/tf/v4/zero_trust_access_application/` - E2E test configs

### terraform-provider-cloudflare
- `internal/services/zero_trust_access_application/v4_model.go` - v4 state model
- `internal/services/zero_trust_access_application/v4_schema.go` - v4 schema
- `internal/services/zero_trust_access_application/migrations.go` - MoveState/UpgradeState
- `internal/services/zero_trust_access_application/migrations_test.go` - Provider tests

---

## Recommendations for Future Migrations

1. **Always check default value changes** between v4 and v5 schemas
2. **Capture block identifiers BEFORE mutation** - use helper functions
3. **Test deeply nested transformations** separately before integrating
4. **Check provider normalization behavior** for arrays/sets to prevent drift
5. **Add v5.x upgrade tests** for each minor version that had schema changes
6. **Document type conversions** (int64 → float64 is common)
7. **Test with real Terraform** via E2E tests, not just unit tests

---

## Cleanup Commands

### Run Test Sweepers
```bash
cd terraform-provider-cloudflare
export CLOUDFLARE_ACCOUNT_ID=<account-id>
export CLOUDFLARE_EMAIL=<email>
export CLOUDFLARE_API_KEY=<api-key>
go test ./internal/services/zero_trust_access_application/ -v -sweep=all
```

### Verify Migration
```bash
cd tf-migrate
go test ./internal/resources/zero_trust_access_application/... -v
go test ./integration/v4_to_v5/... -v -run TestZeroTrustAccessApplication
```
