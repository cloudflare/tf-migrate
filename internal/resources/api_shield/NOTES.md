# API Shield v4 to v5 Migration Notes

## Migration Type
**Type A (Simple)** - Estimated 4-6 hours

## Summary
The `cloudflare_api_shield` resource migration from v4 to v5 involves converting the `auth_id_characteristics` field from block syntax to attribute syntax. This is a straightforward migration with no field renames, type conversions, or state structure changes beyond setting the schema version.

## Pattern 0 Detection Results

### Pattern 1: TypeList MaxItems:1 Fields
**Found:** 0 fields
- No MaxItems:1 fields requiring array→object transformation

### Pattern 2: Computed vs User-Provided Fields
**Classification:**
- `zone_id` - User-provided (Required)
- `auth_id_characteristics` - User-provided (changes from Optional to Required in v5)
- `auth_id_characteristics.type` - User-provided (Required)
- `auth_id_characteristics.name` - User-provided (Required)
- `id` - Computed

**Analysis:** No computed fields within `auth_id_characteristics`, so no risk of accidentally modifying API-managed data.

### Pattern 3: v5 Default Values
**Found:** None
- No new default values introduced in v5 schema

### Pattern 4: Block vs Attribute Syntax Changes
**Found:** 1 change
- `auth_id_characteristics` changes from block syntax to attribute syntax (list of objects)

## Schema Changes

### v4 Schema (Block Syntax)
```hcl
resource "cloudflare_api_shield" "example" {
  zone_id = "..."

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "session_id"
  }
}
```

### v5 Schema (Attribute Syntax)
```hcl
resource "cloudflare_api_shield" "example" {
  zone_id = "..."

  auth_id_characteristics = [
    {
      type = "header"
      name = "authorization"
    },
    {
      type = "cookie"
      name = "session_id"
    }
  ]
}
```

### Additional v5 Changes
- `auth_id_characteristics.type` gains new value `"jwt"` (backward compatible - doesn't affect v4 configs)
- `auth_id_characteristics` changes from Optional to Required
- `auth_id_characteristics.name` and `.type` change from Optional to Required

## Implementation Details

### Config Transformation (HCL)
**Approach:** Use existing `ConvertBlocksToAttributeList` helper
```go
tfhcl.ConvertBlocksToAttributeList(body, "auth_id_characteristics", nil)
```

**Rationale:**
- Reuses battle-tested helper function
- Handles multiple blocks correctly
- Preserves ordering and values
- No custom logic needed

### State Transformation (JSON)
**Approach:** No-op (handled by provider)

**Implementation:**
```go
// Return state unchanged - provider handles state migration via StateUpgraders
return stateJSON.String(), nil
```

**Rationale:**
- State migration is handled by the v5 provider's StateUpgraders
- Provider UpgradeFromV4 handles v4 (schema_version=0) → v5 (version=500) transformation
- Provider UpgradeFromV5 handles v5 (version=1) → v5 (version=500) version bump
- tf-migrate only handles config (HCL) transformations
- State passes through unchanged and is upgraded by provider on next apply

**Provider StateUpgraders:**
- Location: `cloudflare-terraform-next/internal/services/api_shield/migration/v500/`
- Transform function handles:
  - Direct field copies (id, zone_id)
  - Array conversion (auth_id_characteristics SourceModel → TargetModel)
  - Empty array handling (v4 Optional → v5 Required field)

## Testing Coverage

### tf-migrate Unit Tests (5 total)
**Config Transformation Tests (5):**
1. Single header characteristic
2. Single cookie characteristic
3. Multiple characteristics
4. Multiple resources
5. Missing auth_id_characteristics - sets empty array

**State Transformation Tests:**
- Removed (state migration handled by provider)
- See provider migration tests for state transformation coverage

### tf-migrate Integration Tests (20 instances)
Test cases cover:
- **Basic scenarios:** Single header, single cookie
- **Multiple characteristics:** 2-4 per resource
- **Mixed types:** Headers and cookies together
- **Naming patterns:** Hyphens, underscores, capitalization
- **Real-world use cases:**
  - OAuth flow (Authorization + X-OAuth-Token + oauth_state)
  - JWT authentication (Authorization + X-JWT-Token)
  - API key authentication (X-API-Key + X-API-Secret)
  - Multi-factor authentication (Authorization + X-MFA-Token + mfa_session)
  - Bearer token authentication
  - Custom authentication schemes

All tests passing ✓

### Provider Migration Tests (6 test cases)
**Location:** `cloudflare-terraform-next/internal/services/api_shield/migration/v500/migrations_test.go`

Test cases cover state migration:
1. Single header characteristic (v4→v5)
2. Single cookie characteristic (v4→v5)
3. Multiple mixed characteristics (v4→v5)
4. Special characters in names (v4→v5)
5. OAuth flow simulation (v4→v5)
6. Empty auth_id_characteristics - Optional→Required (v4→v5)

These tests verify:
- StateUpgraders correctly transform v4 state to v5 state
- Empty array handling for Optional→Required migration
- Field value preservation (zone_id, type, name)
- Array ordering preservation
- Special character handling

## Migration Complexity Assessment

### Low Complexity Factors
✓ No field renames
✓ No type conversions
✓ No MaxItems:1 complexities
✓ No computed field modifications
✓ No default value handling
✓ Simple block→attribute conversion
✓ Existing helper function available
✓ State structure unchanged

### Why This is Type A (Simple)
- Single transformation type (block→attribute)
- Well-established pattern in codebase
- No edge cases requiring special handling
- Minimal state transformation logic
- No API interactions needed for migration

## Known Issues / Limitations
None identified.

## Verification Steps
1. ✓ tf-migrate unit tests pass (5 config transformation tests)
2. ✓ tf-migrate integration tests pass (20 test instances)
3. ✓ Provider migration tests pass (6 state transformation tests)
4. ✓ Manual migration test verified
5. ✓ Build succeeds with race detector
6. ✓ All repository tests pass

## References
- v4 Schema: `cloudflare-terraform-v4/internal/sdkv2provider/schema_cloudflare_api_shield.go`
- v5 Documentation: `cloudflare-terraform-next/docs/resources/api_shield.md`
- Helper Function: `internal/transform/hcl/blocks.go:ConvertBlocksToAttributeList`

## Migration Date
December 30, 2024

## Notes for Future Maintainers
- The `auth_id_characteristics` field uses a simple list structure - no nested complexity
- If additional fields are added to `auth_id_characteristics` in future versions, the same helper function should work for config transformation
- The `jwt` type value added in v5 is backward compatible and doesn't require migration logic
- Consider this migration as a template for other simple block→attribute conversions

**State Migration Architecture (as of February 2026):**
- tf-migrate handles **config** (HCL) transformations only (TransformConfig)
- Provider handles **state** (JSON) transformations via StateUpgraders (UpgradeState)
- This separation of concerns eliminates duplicate transformation logic
- State transformation tests are in the provider, not tf-migrate
- tf-migrate's TransformState is a no-op that passes state through unchanged
- Provider upgrades state on first apply/refresh with TF_MIG_TEST=1
