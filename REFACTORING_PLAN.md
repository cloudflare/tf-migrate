# Terraform Provider Migration Refactoring Plan
## Step-by-Step Execution Guide

**Date Created**: 2025-12-30
**Total Resources**: 44
**Estimated Total Time**: 18-24 hours
**Lines to be Eliminated**: 330-400

---

## Pre-Execution Checklist

- [ ] Ensure all existing tests pass: `cd internal/resources && go test ./...`
- [ ] Create a feature branch: `git checkout -b refactor/consolidate-transform-utilities`
- [ ] Review the analysis document for context
- [ ] Set up test environment

---

## WAVE 1: Use Existing Utilities (2-3 hours)

**Goal**: Replace custom implementations with existing utilities
**Risk**: Low - using battle-tested utilities
**Validation**: Test after each resource

---

### Task 1.1: Verify Existing Utility Exports

**File**: `internal/transform/hcl/blocks.go`

- [ ] Verify `BuildObjectFromBlock` is exported (line 413)
- [ ] Verify `BuildResourceReference` is exported (line 33)
- [ ] Add godoc examples if missing
- [ ] Run: `go build ./internal/transform/hcl`

**File**: `internal/transform/hcl/tokens.go`

- [ ] Verify `BuildResourceReference` is exported (line 33)
- [ ] Add cross-reference to blocks.go version if duplicate
- [ ] Run: `go build ./internal/transform/hcl`

**Validation**: `cd internal/transform && go test ./...`

---

### Task 1.2: Resource - zone_setting

**File**: `internal/resources/zone_setting/migrate.go`

**Subtasks**:
- [ ] Line ~350: Locate custom `buildResourceReference` function
- [ ] Replace all calls to `buildResourceReference` with `tfhcl.BuildResourceReference`
- [ ] Remove custom `buildResourceReference` function (save ~15 lines)
- [ ] Add import: `tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"`
- [ ] Run: `go test ./internal/resources/zone_setting -v`
- [ ] Verify: Check that import blocks still use correct references

**Expected Changes**:
```go
// Before:
objectTokens := buildResourceReference("cloudflare_zone_setting", settingName)

// After:
objectTokens := tfhcl.BuildResourceReference("cloudflare_zone_setting", settingName)
```

**Lines Saved**: ~15

---

### Task 1.3: Resource - zero_trust_access_policy

**File**: `internal/resources/zero_trust_access_policy/migrate.go`

**Subtasks**:
- [ ] Line ~200-300: Locate manual object building from blocks
- [ ] Identify where block attributes are manually converted to object tokens
- [ ] Replace with `tfhcl.BuildObjectFromBlock(block)`
- [ ] Ensure import: `tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"`
- [ ] Run: `go test ./internal/resources/zero_trust_access_policy -v`
- [ ] Verify: Check complex condition transformations still work

**Expected Changes**:
```go
// Before:
var tokens hclwrite.Tokens
tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
for _, attr := range block.Body().Attributes() {
    // manual token building
}

// After:
tokens := tfhcl.BuildObjectFromBlock(block)
```

**Lines Saved**: ~20-30

---

### Task 1.4: Resource - zero_trust_dlp_custom_profile

**File**: `internal/resources/zero_trust_dlp_custom_profile/migrate.go`

**Subtasks**:
- [ ] Line ~150-250: Locate block transformation logic
- [ ] Replace manual object building with `tfhcl.BuildObjectFromBlock`
- [ ] Ensure import: `tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"`
- [ ] Run: `go test ./internal/resources/zero_trust_dlp_custom_profile -v`
- [ ] Verify: Dynamic resource naming still works correctly

**Lines Saved**: ~15-20

---

### Task 1.5: Resource - load_balancer

**File**: `internal/resources/load_balancer/migrate.go`

**Subtasks**:
- [ ] Locate manual nested block object building
- [ ] Replace with `tfhcl.BuildObjectFromBlock` where applicable
- [ ] Keep complex nested logic that's domain-specific
- [ ] Ensure import: `tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"`
- [ ] Run: `go test ./internal/resources/load_balancer -v`
- [ ] Verify: Complex origin_steering transformations still work

**Lines Saved**: ~10-15

---

### Task 1.6: Resource - pages_project

**File**: `internal/resources/pages_project/migrate.go`

**Subtasks**:
- [ ] Locate object building logic for nested structures
- [ ] Replace with `tfhcl.BuildObjectFromBlock` where applicable
- [ ] Ensure import: `tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"`
- [ ] Run: `go test ./internal/resources/pages_project -v`
- [ ] Verify: Deployment configs still transform correctly

**Lines Saved**: ~10

---

### Wave 1 Validation

- [ ] Run full test suite: `cd internal/resources && go test ./...`
- [ ] Verify all 5 resources pass tests
- [ ] Git commit: `git add -A && git commit -m "refactor(wave1): use existing utilities in 5 resources"`
- [ ] **CHECKPOINT**: All tests green before proceeding

**Wave 1 Total Lines Saved**: ~70-90

---

## WAVE 2: Create New Critical Utilities (4-5 hours)

**Goal**: Build the foundation utilities for Wave 3
**Risk**: Medium - new code requires thorough testing
**Validation**: Comprehensive tests for each new utility

---

### Task 2.1: Create Batch State Operations Utility

**File**: `internal/transform/state/batch.go` (NEW FILE)

**Subtasks**:
- [ ] Create new file with package header and imports
- [ ] Implement `RemoveFieldsIfExist` function
- [ ] Implement `RenameFieldsMap` function
- [ ] Implement `SetSchemaVersion` function
- [ ] Add comprehensive godoc comments with examples
- [ ] Run: `go build ./internal/transform/state`

**Code to write**:
```go
// Package state provides utilities for transforming Terraform state files
package state

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// RemoveFieldsIfExist removes multiple fields from state at the given path if they exist.
// This is a batch operation that prevents repeated existence checks and deletions.
//
// Example - Removing multiple deprecated fields:
//
// Before state JSON:
//   {
//     "attributes": {
//       "zone_id": "abc123",
//       "deprecated1": "value1",
//       "deprecated2": "value2",
//       "deprecated3": "value3"
//     }
//   }
//
// After calling RemoveFieldsIfExist(result, "attributes", attrs, "deprecated1", "deprecated2", "deprecated3"):
//   {
//     "attributes": {
//       "zone_id": "abc123"
//     }
//   }
func RemoveFieldsIfExist(stateJSON string, path string, instance gjson.Result, fields ...string) string {
	for _, field := range fields {
		if instance.Get(field).Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, path+"."+field)
		}
	}
	return stateJSON
}

// RenameFieldsMap renames multiple fields according to the provided mapping.
// Keys are old names, values are new names.
//
// Example - Batch renaming fields:
//
// Before:
//   {
//     "old_name1": "value1",
//     "old_name2": "value2"
//   }
//
// After calling RenameFieldsMap(result, path, instance, map[string]string{"old_name1": "new_name1", "old_name2": "new_name2"}):
//   {
//     "new_name1": "value1",
//     "new_name2": "value2"
//   }
func RenameFieldsMap(stateJSON string, path string, instance gjson.Result, renames map[string]string) string {
	for oldName, newName := range renames {
		stateJSON = RenameField(stateJSON, path, instance, oldName, newName)
	}
	return stateJSON
}

// SetSchemaVersion is a convenience helper to set the schema version in state.
// This eliminates the repeated pattern of manually setting schema_version.
//
// Example:
//
// Before:
//   result, _ = sjson.Set(result, "schema_version", 0)
//
// After:
//   result = state.SetSchemaVersion(result, 0)
func SetSchemaVersion(stateJSON string, version int) string {
	result, _ := sjson.Set(stateJSON, "schema_version", version)
	return result
}
```

**Validation**: Proceed to Task 2.2 (tests created together)

---

### Task 2.2: Create Tests for Batch Operations

**File**: `internal/transform/state/batch_test.go` (NEW FILE)

**Subtasks**:
- [ ] Create test file with package and imports
- [ ] Write `TestRemoveFieldsIfExist` with 5+ test cases
- [ ] Write `TestRenameFieldsMap` with 5+ test cases
- [ ] Write `TestSetSchemaVersion` with 3+ test cases
- [ ] Run: `go test ./internal/transform/state -v -run Batch`
- [ ] Ensure 100% coverage for batch.go

**Test cases to include**:
- RemoveFieldsIfExist:
  - All fields exist
  - Some fields exist
  - No fields exist
  - Empty field list
  - Nested path
- RenameFieldsMap:
  - Multiple renames
  - Field doesn't exist
  - Field already has new name
  - Empty map
- SetSchemaVersion:
  - Set to 0
  - Set to non-zero
  - Overwrite existing version

**Validation**:
- [ ] All tests pass
- [ ] Run: `go test ./internal/transform/state -v -cover`
- [ ] Coverage should be 100% for batch.go

---

### Task 2.3: Create Meta-Arguments Utility

**File**: `internal/transform/hcl/meta.go` (NEW FILE)

**Subtasks**:
- [ ] Create new file with package header and imports
- [ ] Define `MetaArguments` struct
- [ ] Implement `ExtractMetaArguments` function
- [ ] Implement `CopyMetaArgumentsToBlock` function
- [ ] Implement `CopyMetaArgumentsToImport` function
- [ ] Add comprehensive godoc comments with examples
- [ ] Run: `go build ./internal/transform/hcl`

**Code to write**:
```go
// Package hcl provides utilities for transforming HCL configuration files
package hcl

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// MetaArguments holds Terraform meta-arguments that should be preserved during migrations.
// These are special arguments that affect resource behavior but aren't part of the resource schema.
type MetaArguments struct {
	// count defines how many instances of this resource to create
	Count *hclwrite.Attribute
	// for_each creates an instance for each item in a map or set
	ForEach *hclwrite.Attribute
	// lifecycle controls resource lifecycle behavior
	Lifecycle *hclwrite.Block
	// depends_on explicitly defines dependencies
	DependsOn *hclwrite.Attribute
	// provider selects which provider configuration to use
	Provider *hclwrite.Attribute
	// timeouts customizes operation timeouts
	Timeouts *hclwrite.Block
}

// ExtractMetaArguments extracts all meta-arguments from a resource block.
// Returns a MetaArguments struct with pointers to the attributes/blocks found.
// Fields will be nil if the corresponding meta-argument doesn't exist.
//
// Example:
//
//   resource "cloudflare_zone_setting" "example" {
//     zone_id = "abc123"
//     count   = 5
//
//     lifecycle {
//       ignore_changes = [modified_on]
//     }
//   }
//
//   meta := ExtractMetaArguments(block)
//   // meta.Count is non-nil
//   // meta.Lifecycle is non-nil
//   // meta.ForEach is nil
func ExtractMetaArguments(block *hclwrite.Block) *MetaArguments {
	body := block.Body()
	return &MetaArguments{
		Count:     body.GetAttribute("count"),
		ForEach:   body.GetAttribute("for_each"),
		Lifecycle: FindBlockByType(body, "lifecycle"),
		DependsOn: body.GetAttribute("depends_on"),
		Provider:  body.GetAttribute("provider"),
		Timeouts:  FindBlockByType(body, "timeouts"),
	}
}

// CopyMetaArgumentsToBlock copies meta-arguments to a destination block.
// Only non-nil meta-arguments are copied.
//
// Example - Creating a derived resource with preserved meta-arguments:
//
//   originalBlock := ... // has count and lifecycle
//   meta := ExtractMetaArguments(originalBlock)
//
//   newBlock := hclwrite.NewBlock("resource", []string{"cloudflare_new_type", "example"})
//   CopyMetaArgumentsToBlock(newBlock, meta)
//   // newBlock now has count and lifecycle
func CopyMetaArgumentsToBlock(dstBlock *hclwrite.Block, meta *MetaArguments) {
	if meta == nil {
		return
	}

	dstBody := dstBlock.Body()

	if meta.Count != nil {
		dstBody.SetAttributeRaw("count", meta.Count.Expr().BuildTokens(nil))
	}
	if meta.ForEach != nil {
		dstBody.SetAttributeRaw("for_each", meta.ForEach.Expr().BuildTokens(nil))
	}
	if meta.DependsOn != nil {
		dstBody.SetAttributeRaw("depends_on", meta.DependsOn.Expr().BuildTokens(nil))
	}
	if meta.Provider != nil {
		dstBody.SetAttributeRaw("provider", meta.Provider.Expr().BuildTokens(nil))
	}

	// Copy blocks
	if meta.Lifecycle != nil {
		lifecycleBlock := dstBody.AppendNewBlock("lifecycle", nil)
		// Copy all attributes from the lifecycle block
		for name, attr := range meta.Lifecycle.Body().Attributes() {
			lifecycleBlock.Body().SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
		}
	}
	if meta.Timeouts != nil {
		timeoutsBlock := dstBody.AppendNewBlock("timeouts", nil)
		// Copy all attributes from the timeouts block
		for name, attr := range meta.Timeouts.Body().Attributes() {
			timeoutsBlock.Body().SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
		}
	}
}

// CopyMetaArgumentsToImport copies compatible meta-arguments to an import block.
// Import blocks support for_each but not count, and don't support lifecycle/timeouts.
//
// Example - Creating import block with for_each:
//
//   meta := ExtractMetaArguments(resourceBlock)
//   importBlock := hclwrite.NewBlock("import", nil)
//   CopyMetaArgumentsToImport(importBlock, meta)
//   // importBlock has for_each if original had it
func CopyMetaArgumentsToImport(importBlock *hclwrite.Block, meta *MetaArguments) {
	if meta == nil {
		return
	}

	body := importBlock.Body()

	// Import blocks can have for_each but not count
	if meta.ForEach != nil {
		body.SetAttributeRaw("for_each", meta.ForEach.Expr().BuildTokens(nil))
	}

	// provider is also supported on import blocks
	if meta.Provider != nil {
		body.SetAttributeRaw("provider", meta.Provider.Expr().BuildTokens(nil))
	}
}
```

**Validation**: Proceed to Task 2.4

---

### Task 2.4: Create Tests for Meta-Arguments

**File**: `internal/transform/hcl/meta_test.go` (NEW FILE)

**Subtasks**:
- [ ] Create test file with package and imports
- [ ] Write `TestExtractMetaArguments` with 8+ test cases
- [ ] Write `TestCopyMetaArgumentsToBlock` with 6+ test cases
- [ ] Write `TestCopyMetaArgumentsToImport` with 4+ test cases
- [ ] Run: `go test ./internal/transform/hcl -v -run Meta`
- [ ] Ensure 100% coverage for meta.go

**Test cases to include**:
- ExtractMetaArguments:
  - Block with all meta-arguments
  - Block with some meta-arguments
  - Block with no meta-arguments
  - Block with count
  - Block with for_each
  - Block with lifecycle
  - Block with depends_on
  - Block with provider and timeouts
- CopyMetaArgumentsToBlock:
  - Copy all meta-arguments
  - Copy nil meta
  - Copy only count
  - Copy only lifecycle
  - Copy nested lifecycle attributes (ignore_changes, etc.)
  - Copy timeouts with multiple timeout types
- CopyMetaArgumentsToImport:
  - Copy for_each (should work)
  - Copy count (should NOT copy)
  - Copy provider (should work)
  - Copy lifecycle (should NOT copy)

**Validation**:
- [ ] All tests pass
- [ ] Run: `go test ./internal/transform/hcl -v -cover`
- [ ] Coverage should be 100% for meta.go

---

### Task 2.5: Create Time/Duration Utilities

**File**: `internal/transform/state/time.go` (NEW FILE)

**Subtasks**:
- [ ] Create new file with package header and imports
- [ ] Implement `NormalizeDuration` function
- [ ] Implement `NormalizeRFC3339` function
- [ ] Add comprehensive godoc comments with examples
- [ ] Run: `go build ./internal/transform/state`

**Code to write**:
```go
// Package state provides utilities for transforming Terraform state files
package state

import (
	"strings"
	"time"
)

// NormalizeDuration normalizes Go duration strings by removing unnecessary zero components.
// This is useful when state contains verbose duration strings like "24h0m0s" that should be "24h".
//
// Example transformations:
//   "24h0m0s" -> "24h"
//   "1h30m0s" -> "1h30m"
//   "45m0s"   -> "45m"
//   "30s"     -> "30s" (unchanged)
//
// Usage in migrations:
//   duration := attrs.Get("timeout").String()
//   normalized := NormalizeDuration(duration)
//   result, _ = sjson.Set(result, "attributes.timeout", normalized)
func NormalizeDuration(duration string) string {
	// Remove unnecessary zero components from right to left
	duration = strings.ReplaceAll(duration, "h0m0s", "h")
	duration = strings.ReplaceAll(duration, "m0s", "m")
	// Don't modify if it's just seconds
	return duration
}

// NormalizeRFC3339 normalizes date strings to RFC3339 format.
// Attempts to parse various common date formats and converts them to RFC3339.
// Returns the original string if it cannot be parsed.
//
// Supported input formats:
//   - RFC3339: "2006-01-02T15:04:05Z07:00"
//   - ISO8601 with Z: "2006-01-02T15:04:05Z"
//   - Date only: "2006-01-02"
//   - RFC3339Nano: "2006-01-02T15:04:05.999999999Z07:00"
//
// Example:
//   "2024-01-01" -> "2024-01-01T00:00:00Z"
//   "2024-01-01T12:30:45Z" -> "2024-01-01T12:30:45Z"
//   "invalid" -> "invalid" (unchanged)
//
// Usage in migrations:
//   createdOn := attrs.Get("created_on").String()
//   normalized := NormalizeRFC3339(createdOn)
//   result, _ = sjson.Set(result, "attributes.created_on", normalized)
func NormalizeRFC3339(dateStr string) string {
	// List of formats to try parsing
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format(time.RFC3339)
		}
	}

	// If we can't parse it, return as-is
	return dateStr
}
```

**Validation**: Proceed to Task 2.6

---

### Task 2.6: Create Tests for Time Utilities

**File**: `internal/transform/state/time_test.go` (NEW FILE)

**Subtasks**:
- [ ] Create test file with package and imports
- [ ] Write `TestNormalizeDuration` with 8+ test cases
- [ ] Write `TestNormalizeRFC3339` with 8+ test cases
- [ ] Run: `go test ./internal/transform/state -v -run Time`
- [ ] Ensure 100% coverage for time.go

**Test cases to include**:
- NormalizeDuration:
  - "24h0m0s" -> "24h"
  - "1h30m0s" -> "1h30m"
  - "45m0s" -> "45m"
  - "30s" -> "30s"
  - "0h0m0s" -> "0h"
  - "1h0m30s" -> "1h0m30s" (can't simplify further)
  - "" -> ""
  - "invalid" -> "invalid"
- NormalizeRFC3339:
  - "2024-01-01" -> "2024-01-01T00:00:00Z"
  - "2024-01-01T12:30:45Z" -> "2024-01-01T12:30:45Z"
  - "2024-01-01T12:30:45+02:00" -> proper RFC3339
  - "2024-01-01T12:30:45.123456Z" -> proper RFC3339
  - "" -> ""
  - "invalid" -> "invalid"
  - Already RFC3339 -> unchanged
  - Various timezone formats

**Validation**:
- [ ] All tests pass
- [ ] Run: `go test ./internal/transform/state -v -cover`
- [ ] Coverage should be 100% for time.go

---

### Wave 2 Validation

- [ ] Run full transform tests: `cd internal/transform && go test ./... -v -cover`
- [ ] Verify all new utilities have 100% test coverage
- [ ] Verify documentation is complete
- [ ] Git commit: `git add -A && git commit -m "feat(wave2): add batch, meta, and time utilities"`
- [ ] **CHECKPOINT**: All tests green before proceeding

**Wave 2 Summary**:
- **New Files**: 6 (3 source, 3 test)
- **New Functions**: 8
- **Test Coverage**: 100% for new code

---

## WAVE 3: Refactor Resources to Use New Utilities (6-8 hours)

**Goal**: Update all resources to use Wave 2 utilities
**Risk**: Medium - bulk changes need careful validation
**Validation**: Test after every 5 resources

---

### BATCH 3.1: Simple Resources with Field Removal (5 resources)

---

#### Task 3.1.1: Resource - account_member

**File**: `internal/resources/account_member/state.go`

**Subtasks**:
- [ ] Locate field removal logic (around RemoveFields calls)
- [ ] Replace with `state.RemoveFieldsIfExist`
- [ ] Replace schema version setting with `state.SetSchemaVersion`
- [ ] Run: `go test ./internal/resources/account_member -v`

**Expected Changes**:
```go
// Before:
if attrs.Get("role_ids").Exists() {
    result, _ = sjson.Delete(result, "attributes.role_ids")
}
result, _ = sjson.Set(result, "schema_version", 0)

// After:
result = state.RemoveFieldsIfExist(result, "attributes", attrs, "role_ids")
result = state.SetSchemaVersion(result, 0)
```

**Lines Saved**: ~3-5

---

#### Task 3.1.2: Resource - api_token

**File**: `internal/resources/api_token/state.go`

**Subtasks**:
- [ ] Locate multiple field removals
- [ ] Replace with single `state.RemoveFieldsIfExist` call
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/api_token -v`

**Expected Changes**:
```go
// Before:
if attrs.Get("field1").Exists() {
    result, _ = sjson.Delete(result, "attributes.field1")
}
if attrs.Get("field2").Exists() {
    result, _ = sjson.Delete(result, "attributes.field2")
}
result, _ = sjson.Set(result, "schema_version", 0)

// After:
result = state.RemoveFieldsIfExist(result, "attributes", attrs, "field1", "field2")
result = state.SetSchemaVersion(result, 0)
```

**Lines Saved**: ~5-8

---

#### Task 3.1.3: Resource - argo

**File**: `internal/resources/argo/state.go`

**Subtasks**:
- [ ] Locate field removal logic
- [ ] Replace with `state.RemoveFieldsIfExist`
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/argo -v`

**Lines Saved**: ~3-5

---

#### Task 3.1.4: Resource - bot_management

**File**: `internal/resources/bot_management/state.go`

**Subtasks**:
- [ ] Locate field removal logic
- [ ] Replace with `state.RemoveFieldsIfExist`
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/bot_management -v`

**Lines Saved**: ~3-5

---

#### Task 3.1.5: Resource - logpull_retention

**File**: `internal/resources/logpull_retention/state.go`

**Subtasks**:
- [ ] Locate field removal logic
- [ ] Replace with `state.RemoveFieldsIfExist`
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/logpull_retention -v`

**Lines Saved**: ~3-5

---

### Batch 3.1 Validation

- [ ] Run: `go test ./internal/resources/{account_member,api_token,argo,bot_management,logpull_retention} -v`
- [ ] All tests pass
- [ ] Git commit: `git add -A && git commit -m "refactor(wave3.1): use batch utilities in 5 simple resources"`

**Batch 3.1 Lines Saved**: ~17-28

---

### BATCH 3.2: Resources with Field Removal + Rename (5 resources)

---

#### Task 3.2.1: Resource - dns_record

**File**: `internal/resources/dns_record/state.go`

**Subtasks**:
- [ ] Locate multiple field removals
- [ ] Replace with `state.RemoveFieldsIfExist`
- [ ] Check if any rename operations could use `state.RenameFieldsMap`
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/dns_record -v`

**Expected Changes**:
```go
// Before:
result, _ = sjson.Delete(result, "attributes.field1")
result, _ = sjson.Delete(result, "attributes.field2")
result = state.RenameField(result, "attributes", attrs, "old1", "new1")
result = state.RenameField(result, "attributes", attrs, "old2", "new2")
result, _ = sjson.Set(result, "schema_version", 0)

// After:
result = state.RemoveFieldsIfExist(result, "attributes", attrs, "field1", "field2")
result = state.RenameFieldsMap(result, "attributes", attrs, map[string]string{
    "old1": "new1",
    "old2": "new2",
})
result = state.SetSchemaVersion(result, 0)
```

**Lines Saved**: ~5-10

---

#### Task 3.2.2: Resource - healthcheck

**File**: `internal/resources/healthcheck/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/healthcheck -v`

**Lines Saved**: ~5-10

---

#### Task 3.2.3: Resource - load_balancer

**File**: `internal/resources/load_balancer/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep complex nested transformations as-is
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/load_balancer -v`

**Lines Saved**: ~5-10

---

#### Task 3.2.4: Resource - page_rule

**File**: `internal/resources/page_rule/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/page_rule -v`

**Lines Saved**: ~5-10

---

#### Task 3.2.5: Resource - regional_hostname

**File**: `internal/resources/regional_hostname/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/regional_hostname -v`

**Lines Saved**: ~3-5

---

### Batch 3.2 Validation

- [ ] Run: `go test ./internal/resources/{dns_record,healthcheck,load_balancer,page_rule,regional_hostname} -v`
- [ ] All tests pass
- [ ] Git commit: `git add -A && git commit -m "refactor(wave3.2): use batch utilities in 5 resources with renames"`

**Batch 3.2 Lines Saved**: ~23-45

---

### BATCH 3.3: More Complex Resources (5 resources)

---

#### Task 3.3.1: Resource - load_balancer_monitor

**File**: `internal/resources/load_balancer_monitor/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep type conversion logic (ConvertToFloat64)
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/load_balancer_monitor -v`

**Lines Saved**: ~5-8

---

#### Task 3.3.2: Resource - load_balancer_pool

**File**: `internal/resources/load_balancer_pool/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep type conversion logic
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/load_balancer_pool -v`

**Lines Saved**: ~5-8

---

#### Task 3.3.3: Resource - pages_project

**File**: `internal/resources/pages_project/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep TransformEmptyValuesToNull logic
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/pages_project -v`

**Lines Saved**: ~8-12

---

#### Task 3.3.4: Resource - r2_bucket

**File**: `internal/resources/r2_bucket/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/r2_bucket -v`

**Lines Saved**: ~3-5

---

#### Task 3.3.5: Resource - tiered_cache

**File**: `internal/resources/tiered_cache/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/tiered_cache -v`

**Lines Saved**: ~3-5

---

### Batch 3.3 Validation

- [ ] Run: `go test ./internal/resources/{load_balancer_monitor,load_balancer_pool,pages_project,r2_bucket,tiered_cache} -v`
- [ ] All tests pass
- [ ] Git commit: `git add -A && git commit -m "refactor(wave3.3): use batch utilities in 5 complex resources"`

**Batch 3.3 Lines Saved**: ~24-38

---

### BATCH 3.4: Zero Trust Resources (5 resources)

---

#### Task 3.4.1: Resource - zero_trust_access_identity_provider

**File**: `internal/resources/zero_trust_access_identity_provider/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep TransformEmptyValuesToNull and TransformFieldArrayToObject
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/zero_trust_access_identity_provider -v`

**Lines Saved**: ~5-8

---

#### Task 3.4.2: Resource - zero_trust_device_posture_rule

**File**: `internal/resources/zero_trust_device_posture_rule/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep custom transformation logic
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/zero_trust_device_posture_rule -v`

**Lines Saved**: ~5-8

---

#### Task 3.4.3: Resource - zero_trust_gateway_policy

**File**: `internal/resources/zero_trust_gateway_policy/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Locate duration normalization logic
- [ ] Replace with `state.NormalizeDuration`
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/zero_trust_gateway_policy -v`

**Expected Changes for Duration**:
```go
// Before:
if strings.Contains(duration, "0m0s") {
    duration = strings.ReplaceAll(duration, "h0m0s", "h")
    duration = strings.ReplaceAll(duration, "m0s", "m")
}

// After:
duration = state.NormalizeDuration(duration)
```

**Lines Saved**: ~8-12

---

#### Task 3.4.4: Resource - zero_trust_access_application

**File**: `internal/resources/zero_trust_access_application/state.go`

**Subtasks**:
- [ ] Locate field operations (if any)
- [ ] Replace with batch utilities where applicable
- [ ] Keep TransformEmptyValuesToNull
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/zero_trust_access_application -v`

**Lines Saved**: ~3-5

---

#### Task 3.4.5: Resource - zero_trust_access_group

**File**: `internal/resources/zero_trust_access_group/state.go`

**Subtasks**:
- [ ] Locate field operations (if any)
- [ ] Replace with batch utilities where applicable
- [ ] Keep TransformEmptyValuesToNull
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/zero_trust_access_group -v`

**Lines Saved**: ~3-5

---

### Batch 3.4 Validation

- [ ] Run: `go test ./internal/resources/zero_trust_* -v`
- [ ] All tests pass
- [ ] Git commit: `git add -A && git commit -m "refactor(wave3.4): use batch and time utilities in 5 zero trust resources"`

**Batch 3.4 Lines Saved**: ~24-38

---

### BATCH 3.5: Remaining Resources (5 resources)

---

#### Task 3.5.1: Resource - workers_script

**File**: `internal/resources/workers_script/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/workers_script -v`

**Lines Saved**: ~8-12

---

#### Task 3.5.2: Resource - zone

**File**: `internal/resources/zone/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep custom nested account transformation
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/zone -v`

**Lines Saved**: ~5-8

---

#### Task 3.5.3: Resource - zone_dnssec

**File**: `internal/resources/zone_dnssec/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Verify `state.NormalizeRFC3339` is being used
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/zone_dnssec -v`

**Lines Saved**: ~5-8

---

#### Task 3.5.4: Resource - spectrum_application

**File**: `internal/resources/spectrum_application/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep custom protocol handling
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/spectrum_application -v`

**Lines Saved**: ~5-8

---

#### Task 3.5.5: Resource - logpush_job

**File**: `internal/resources/logpush_job/state.go`

**Subtasks**:
- [ ] Locate field operations
- [ ] Replace with batch utilities
- [ ] Keep custom output_options transformation
- [ ] Replace schema version setting
- [ ] Run: `go test ./internal/resources/logpush_job -v`

**Lines Saved**: ~5-8

---

### Batch 3.5 Validation

- [ ] Run: `go test ./internal/resources/{workers_script,zone,zone_dnssec,spectrum_application,logpush_job} -v`
- [ ] All tests pass
- [ ] Git commit: `git add -A && git commit -m "refactor(wave3.5): use batch utilities in 5 remaining resources"`

**Batch 3.5 Lines Saved**: ~28-44

---

### Wave 3 Final Validation

- [ ] Run full resource test suite: `cd internal/resources && go test ./... -v`
- [ ] Verify all 25 resources pass tests
- [ ] Git commit: `git add -A && git commit -m "refactor(wave3): completed batch utility refactoring for 25 resources"`
- [ ] **CHECKPOINT**: All tests green before proceeding

**Wave 3 Total Lines Saved**: ~116-193

---

## WAVE 4: Meta-Arguments Refactoring (2-3 hours)

**Goal**: Replace custom meta-argument handling with new utility
**Risk**: Medium - affects resource splitting logic
**Validation**: Careful testing of derived resources

---

### Task 4.1: Resource - zone_setting

**File**: `internal/resources/zone_setting/migrate.go`

**Subtasks**:
- [ ] Locate custom `metaArguments` struct definition (~line 50-60)
- [ ] Locate `extractMetaArguments` function (~line 350)
- [ ] Locate `copyMetaArguments` function (~line 370)
- [ ] Locate `copyMetaArgumentsToImport` function (~line 400)
- [ ] Replace all usages with `tfhcl.ExtractMetaArguments`, `tfhcl.CopyMetaArgumentsToBlock`, `tfhcl.CopyMetaArgumentsToImport`
- [ ] Remove custom functions and struct
- [ ] Ensure import: `tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"`
- [ ] Run: `go test ./internal/resources/zone_setting -v`
- [ ] Verify: Check that split resources preserve meta-arguments correctly

**Expected Changes**:
```go
// Before:
type metaArguments struct {
    count     *hclwrite.Attribute
    forEach   *hclwrite.Attribute
    lifecycle *hclwrite.Block
    // ...
}

func (m *Migrator) extractMetaArguments(block *hclwrite.Block) *metaArguments { ... }
func (m *Migrator) copyMetaArguments(newBlock *hclwrite.Block, meta *metaArguments) { ... }

meta := m.extractMetaArguments(block)
m.copyMetaArguments(newBlock, meta)
m.copyMetaArgumentsToImport(importBlock, meta)

// After:
meta := tfhcl.ExtractMetaArguments(block)
tfhcl.CopyMetaArgumentsToBlock(newBlock, meta)
tfhcl.CopyMetaArgumentsToImport(importBlock, meta)
```

**Lines Saved**: ~120-150

---

### Task 4.2: Resource - zero_trust_dlp_custom_profile

**File**: `internal/resources/zero_trust_dlp_custom_profile/migrate.go`

**Subtasks**:
- [ ] Locate where new resources are created from original
- [ ] Add meta-argument extraction after block is identified
- [ ] Add meta-argument copying to new blocks
- [ ] Ensure import: `tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"`
- [ ] Run: `go test ./internal/resources/zero_trust_dlp_custom_profile -v`
- [ ] Verify: Check that renamed resources preserve meta-arguments

**Expected Changes**:
```go
// Before (no meta-argument handling):
newBlock := hclwrite.NewBlock("resource", []string{newType, resourceName})
// copy attributes
// missing: meta-arguments are lost

// After:
meta := tfhcl.ExtractMetaArguments(block)
newBlock := hclwrite.NewBlock("resource", []string{newType, resourceName})
// copy attributes
tfhcl.CopyMetaArgumentsToBlock(newBlock, meta)
```

**Lines Saved**: ~0 (adds functionality, doesn't remove code, but improves correctness)
**Functionality Added**: Meta-arguments now preserved

---

### Task 4.3: Resource - pages_project

**File**: `internal/resources/pages_project/migrate.go`

**Subtasks**:
- [ ] Audit for any manual meta-argument handling
- [ ] If found, replace with `tfhcl.ExtractMetaArguments` and `tfhcl.CopyMetaArgumentsToBlock`
- [ ] If not found, verify meta-arguments don't need preservation
- [ ] Run: `go test ./internal/resources/pages_project -v`

**Lines Saved**: ~0-20 (depends on what's found)

---

### Wave 4 Validation

- [ ] Run: `go test ./internal/resources/{zone_setting,zero_trust_dlp_custom_profile,pages_project} -v`
- [ ] Manual test: Create a zone_setting with count=2, verify split resources preserve count
- [ ] Manual test: Create a zone_setting with lifecycle, verify split resources preserve lifecycle
- [ ] All tests pass
- [ ] Git commit: `git add -A && git commit -m "refactor(wave4): use meta-arguments utility in 3 resources"`
- [ ] **CHECKPOINT**: All tests green before proceeding

**Wave 4 Lines Saved**: ~120-170

---

## WAVE 5: Empty Structure Detection (2-3 hours)

**Goal**: Create and deploy empty structure checker utility
**Risk**: Low - isolated utility
**Validation**: Test new utility and affected resources

---

### Task 5.1: Create Empty Structure Utility

**File**: `internal/transform/state/compare.go` (NEW FILE)

**Subtasks**:
- [ ] Create new file with package header and imports
- [ ] Implement `IsEmptyStructure` function
- [ ] Add comprehensive godoc comments with examples
- [ ] Run: `go build ./internal/transform/state`

**Code to write**:
```go
// Package state provides utilities for transforming Terraform state files
package state

import (
	"encoding/json"
	"reflect"

	"github.com/tidwall/gjson"
)

// IsEmptyStructure checks if a gjson.Result matches an empty template structure.
// This is useful for detecting when a complex nested object has only null/empty values.
//
// The comparison is deep equality - both structure and values must match.
//
// Example - Detecting empty input structure:
//
// Template:
//   {"id": null, "version": null}
//
// Actual value 1:
//   {"id": null, "version": null}
//   -> Returns true (matches template)
//
// Actual value 2:
//   {"id": "abc123", "version": null}
//   -> Returns false (id has a value)
//
// Actual value 3:
//   {"id": null, "version": null, "extra": "field"}
//   -> Returns false (has extra field)
//
// Usage in migrations:
//   emptyTemplate := `{"id":null,"version":null}`
//   if state.IsEmptyStructure(input, emptyTemplate) {
//       // This input is effectively empty, clean it up
//       result, _ = sjson.Delete(result, "attributes.input")
//   }
func IsEmptyStructure(actual gjson.Result, emptyTemplate string) bool {
	if !actual.Exists() {
		return true
	}

	var actualMap map[string]interface{}
	var templateMap map[string]interface{}

	// Parse actual value
	if err := json.Unmarshal([]byte(actual.Raw), &actualMap); err != nil {
		return false
	}

	// Parse template
	if err := json.Unmarshal([]byte(emptyTemplate), &templateMap); err != nil {
		return false
	}

	// Deep equality check
	return reflect.DeepEqual(actualMap, templateMap)
}
```

**Validation**: Proceed to Task 5.2

---

### Task 5.2: Create Tests for Compare Utility

**File**: `internal/transform/state/compare_test.go` (NEW FILE)

**Subtasks**:
- [ ] Create test file with package and imports
- [ ] Write `TestIsEmptyStructure` with 12+ test cases
- [ ] Run: `go test ./internal/transform/state -v -run Compare`
- [ ] Ensure 100% coverage for compare.go

**Test cases to include**:
- Exact match (empty template matches actual)
- Has value in one field
- Has value in multiple fields
- Extra field in actual
- Missing field in actual
- Nested objects match
- Nested objects don't match
- Non-existent actual (doesn't exist)
- Empty string template
- Invalid JSON in template
- Array fields
- Mixed types

**Validation**:
- [ ] All tests pass
- [ ] Run: `go test ./internal/transform/state -v -cover`
- [ ] Coverage should be 100% for compare.go

---

### Task 5.3: Resource - zero_trust_device_posture_rule

**File**: `internal/resources/zero_trust_device_posture_rule/state.go`

**Subtasks**:
- [ ] Locate `inputFieldIsEmpty` function definition
- [ ] Replace all calls to `inputFieldIsEmpty` with `state.IsEmptyStructure`
- [ ] Remove `inputFieldIsEmpty` function
- [ ] Run: `go test ./internal/resources/zero_trust_device_posture_rule -v`
- [ ] Verify: Empty input detection still works correctly

**Expected Changes**:
```go
// Before:
func inputFieldIsEmpty(input gjson.Result) bool {
    return input.Raw == `{"id":null,"version":null}`
}

if inputFieldIsEmpty(inputField) {
    // handle empty
}

// After:
if state.IsEmptyStructure(inputField, `{"id":null,"version":null}`) {
    // handle empty
}
```

**Lines Saved**: ~10-15 (function definition + cleaner usage)

---

### Task 5.4: Resource - zero_trust_access_group

**File**: `internal/resources/zero_trust_access_group/state.go`

**Subtasks**:
- [ ] Audit for custom empty structure checks
- [ ] If found, replace with `state.IsEmptyStructure`
- [ ] Run: `go test ./internal/resources/zero_trust_access_group -v`

**Lines Saved**: ~5-10 (if custom checks exist)

---

### Task 5.5: Audit Other Resources for Empty Checks

**Files**: Scan remaining resources

**Subtasks**:
- [ ] Grep for patterns like `== "{}"`, `== "[]"`, `Raw ==`, custom isEmpty functions
- [ ] Run: `cd internal/resources && grep -r "Raw ==" --include="*.go"`
- [ ] Run: `cd internal/resources && grep -r "isEmpty" --include="*.go"`
- [ ] For each match, evaluate if `state.IsEmptyStructure` applies
- [ ] Update if applicable

**Potential Resources**:
- managed_transforms
- custom_pages
- snippet

---

### Wave 5 Validation

- [ ] Run: `go test ./internal/transform/state -v`
- [ ] Run: `go test ./internal/resources/zero_trust_device_posture_rule -v`
- [ ] Run: `go test ./internal/resources/zero_trust_access_group -v`
- [ ] All tests pass
- [ ] Git commit: `git add -A && git commit -m "feat(wave5): add empty structure detection utility"`
- [ ] **CHECKPOINT**: All tests green before proceeding

**Wave 5 Lines Saved**: ~15-25 (plus utility for future use)

---

## WAVE 6: Documentation & Final Validation (2-3 hours)

**Goal**: Document the utilities and validate all changes
**Risk**: Low - documentation and verification
**Validation**: Full test suite + manual review

---

### Task 6.1: Create Transform Utilities Guide

**File**: `internal/transform/README.md` (NEW FILE)

**Subtasks**:
- [ ] Create comprehensive guide covering all utilities
- [ ] Include sections: HCL transformations, State transformations, Common patterns
- [ ] Add code examples for each utility
- [ ] Add "When to use which utility" decision tree
- [ ] Add migration patterns section
- [ ] Run: Review for completeness

**Content Structure**:
```markdown
# Transform Utilities Guide

## Table of Contents
1. Overview
2. HCL Transformations
   - Attribute Operations
   - Block Operations
   - Meta-Arguments
   - Token Building
3. State Transformations
   - Field Operations
   - Batch Operations
   - Array Operations
   - Type Conversions
   - Time Utilities
   - Comparison Utilities
4. Common Migration Patterns
5. Decision Tree: Which Utility to Use
6. Examples
7. Testing Guidelines
```

**Sections to include**:
- [ ] Overview of transform package structure
- [ ] When to use HCL vs State utilities
- [ ] Complete list of all utilities with signatures
- [ ] Code examples for each utility (copy from godocs)
- [ ] Common migration patterns (field rename, block to attribute, etc.)
- [ ] Decision tree flowchart (text-based)
- [ ] Links to test files for more examples
- [ ] Guidelines for adding new utilities

**Validation**: Internal review for completeness

---

### Task 6.2: Enhance Godoc Comments

**Files**: All utility files in `internal/transform/`

**Subtasks**:
- [ ] Review `internal/transform/hcl/attributes.go` - enhance examples
- [ ] Review `internal/transform/hcl/blocks.go` - enhance examples
- [ ] Review `internal/transform/hcl/arrays.go` - enhance examples
- [ ] Review `internal/transform/hcl/expressions.go` - enhance examples
- [ ] Review `internal/transform/hcl/tokens.go` - enhance examples
- [ ] Review `internal/transform/hcl/meta.go` - verify examples are clear
- [ ] Review `internal/transform/state/fields.go` - enhance examples
- [ ] Review `internal/transform/state/arrays.go` - enhance examples
- [ ] Review `internal/transform/state/batch.go` - verify examples are clear
- [ ] Review `internal/transform/state/time.go` - verify examples are clear
- [ ] Review `internal/transform/state/compare.go` - verify examples are clear
- [ ] Review `internal/transform/state/type_converters.go` - enhance examples

**For Each File**:
- [ ] Ensure package-level documentation exists
- [ ] Ensure each function has clear godoc with example
- [ ] Add "See also" cross-references to related functions
- [ ] Verify code examples are correct and runnable
- [ ] Run: `go doc -all ./internal/transform/...` to preview

**Validation**: Generate and review documentation

---

### Task 6.3: Create Migration Template

**File**: `internal/resources/_template/migrate.go` (NEW FILE)

**Subtasks**:
- [ ] Create template directory structure
- [ ] Create `migrate.go` template showing best practices
- [ ] Create `state.go` template showing best practices
- [ ] Create `README.md` explaining the template
- [ ] Include comments with common patterns

**Template Content**:
```go
// Package template provides migration logic for cloudflare_template resource
// from v4 to v5 of the Cloudflare Terraform provider.
//
// Migration changes:
// - [List specific changes this migration performs]
// - [e.g., "Renames resource from cloudflare_old to cloudflare_new"]
// - [e.g., "Converts config block to config attribute"]
// - [e.g., "Removes deprecated fields: field1, field2"]
package template

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

const (
	// V4 resource type name
	resourceTypeV4 = "cloudflare_old_name"
	// V5 resource type name
	resourceTypeV5 = "cloudflare_new_name"
)

// Migrator implements the migration logic for cloudflare_template
type Migrator struct{}

// CanHandle returns true if this migrator can handle the given resource type
func (m *Migrator) CanHandle(resourceType string) bool {
	return resourceType == resourceTypeV4 || resourceType == resourceTypeV5
}

// TransformHCL transforms the HCL configuration from v4 to v5
func (m *Migrator) TransformHCL(ctx *transform.Context, file *hclwrite.File) ([]*hclwrite.File, error) {
	// Pattern 1: Rename resource type
	for _, block := range file.Body().Blocks() {
		if block.Type() == "resource" && tfhcl.GetResourceType(block) == resourceTypeV4 {
			tfhcl.RenameResourceType(block, resourceTypeV4, resourceTypeV5)

			body := block.Body()

			// Pattern 2: Remove deprecated attributes
			tfhcl.RemoveAttributes(body, "deprecated_field1", "deprecated_field2")

			// Pattern 3: Rename attributes
			tfhcl.RenameAttribute(body, "old_name", "new_name")

			// Pattern 4: Convert blocks to attributes
			tfhcl.ConvertBlocksToAttribute(body, "config", "config", nil)

			// Add more transformations as needed
		}
	}

	return []*hclwrite.File{file}, nil
}

// TransformState transforms the Terraform state from v4 to v5
func (m *Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		return state.SetSchemaVersion(result, 0), nil
	}

	// Pattern 1: Remove deprecated fields (batch operation)
	result = state.RemoveFieldsIfExist(result, "attributes", attrs,
		"deprecated_field1", "deprecated_field2")

	// Pattern 2: Rename fields (batch operation)
	result = state.RenameFieldsMap(result, "attributes", attrs, map[string]string{
		"old_name1": "new_name1",
		"old_name2": "new_name2",
	})

	// Pattern 3: Convert array to object (if applicable)
	result = state.ConvertMaxItemsOneArrayToObject(result, "attributes", attrs, "config")

	// Pattern 4: Type conversions (if applicable)
	// if field := attrs.Get("numeric_field"); field.Exists() {
	//     result, _ = sjson.Set(result, "attributes.numeric_field",
	//         state.ConvertToInt64(field))
	// }

	// Add more transformations as needed

	// Always set schema version last
	return state.SetSchemaVersion(result, 0), nil
}
```

**Also Create**: `_template/state.go`, `_template/README.md`

**Validation**: Template should compile without errors

---

### Task 6.4: Run Full Test Suite

**Subtasks**:
- [ ] Run all transform tests: `cd internal/transform && go test ./... -v -cover`
- [ ] Verify 100% coverage for new utilities
- [ ] Run all resource tests: `cd internal/resources && go test ./... -v`
- [ ] Verify all 44 resources pass
- [ ] Check for test failures, investigate and fix
- [ ] Run race detector: `go test ./... -race`

**Commands**:
```bash
# Full test suite
cd /Users/tjozsa/cf-repos/sdks/migration-work/agent-a/tf-migrate
go test ./internal/transform/... -v -cover
go test ./internal/resources/... -v
go test ./... -race

# Generate coverage report
go test ./internal/transform/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

**Validation**:
- [ ] All tests pass
- [ ] No race conditions detected
- [ ] Coverage for new utilities is 100%

---

### Task 6.5: Manual Validation Sampling

**Subtasks**:
- [ ] Pick 3 simple resources (e.g., account_member, list, r2_bucket)
- [ ] Pick 3 complex resources (e.g., zone_setting, load_balancer, zero_trust_gateway_policy)
- [ ] For each resource:
  - [ ] Review the migrate.go changes
  - [ ] Review the state.go changes
  - [ ] Verify no behavioral changes (only refactoring)
  - [ ] Verify readability improved
  - [ ] Check for any missed refactoring opportunities

**Documentation Review**:
- [ ] Review README.md for completeness
- [ ] Review godoc output: `godoc -http=:6060` and browse to package
- [ ] Check all links and cross-references work
- [ ] Verify examples are clear and correct

---

### Task 6.6: Code Review Preparation

**File**: `REFACTORING_SUMMARY.md` (NEW FILE)

**Subtasks**:
- [ ] Create summary document for code review
- [ ] Include: Goals, approach, changes made, testing performed
- [ ] Include: Before/after statistics (files, lines, test coverage)
- [ ] Include: Known issues or limitations (if any)
- [ ] Include: Future improvement suggestions

**Content Structure**:
```markdown
# Refactoring Summary

## Goals
[Brief description of refactoring goals]

## Approach
[Describe the wave-based approach]

## Changes Made

### New Utilities Created
- internal/transform/state/batch.go (3 functions)
- internal/transform/hcl/meta.go (3 functions + struct)
- internal/transform/state/time.go (2 functions)
- internal/transform/state/compare.go (1 function)

### Resources Refactored
- Wave 1: 5 resources (existing utilities)
- Wave 3: 25 resources (batch utilities)
- Wave 4: 3 resources (meta-arguments)
- Wave 5: 2 resources (empty structure)

## Statistics

### Lines Eliminated
- Wave 1: ~70-90 lines
- Wave 3: ~116-193 lines
- Wave 4: ~120-170 lines
- Wave 5: ~15-25 lines
- **Total: ~321-478 lines eliminated**

### Files Changed
- New files: 10 (6 source + 4 test)
- Modified resources: 30+ resources
- Documentation: 2 new docs (README.md, template)

### Test Coverage
- All new utilities: 100% coverage
- All resource tests: passing
- No race conditions detected

## Testing Performed
[List all testing performed]

## Known Issues
[List any known issues or limitations]

## Future Improvements
[Suggestions for future work]
```

**Validation**: Document is complete and accurate

---

### Task 6.7: Final Git Operations

**Subtasks**:
- [ ] Review all commits made during refactoring
- [ ] Ensure commit messages follow conventions
- [ ] Squash commits if needed (optional)
- [ ] Create summary commit with all changes
- [ ] Push branch: `git push origin refactor/consolidate-transform-utilities`
- [ ] Create draft pull request with REFACTORING_SUMMARY.md in description

**Git Commands**:
```bash
# Review commit history
git log --oneline

# Final commit
git add -A
git commit -m "docs(wave6): add comprehensive documentation and validation"

# Push
git push origin refactor/consolidate-transform-utilities
```

---

### Wave 6 Validation

- [ ] All tests pass
- [ ] Documentation is complete
- [ ] Template is available for future migrations
- [ ] Summary document is ready for review
- [ ] Branch is pushed
- [ ] **FINAL CHECKPOINT**: Ready for code review

---

## Post-Refactoring Checklist

### Completeness
- [ ] All 6 waves completed
- [ ] All planned resources refactored (30+ resources)
- [ ] All new utilities created (4 utilities)
- [ ] All tests passing
- [ ] Documentation complete

### Quality
- [ ] No duplicated patterns remain
- [ ] Code is more readable
- [ ] Test coverage maintained or improved
- [ ] No behavioral changes (pure refactoring)
- [ ] Meta-arguments preserved correctly

### Deliverables
- [ ] 4 new utility files (batch.go, meta.go, time.go, compare.go)
- [ ] 4 new test files (100% coverage)
- [ ] 1 comprehensive README.md
- [ ] 1 migration template
- [ ] 1 refactoring summary document
- [ ] 30+ refactored resources
- [ ] Clean git history with descriptive commits

### Metrics
- **Lines Eliminated**: 321-478 lines
- **Files Created**: 10 (6 source + 4 test)
- **Resources Refactored**: 30+ out of 44
- **Test Coverage**: 100% for new utilities
- **Time Spent**: 18-24 hours estimated

---

## Rollback Plan

If issues are discovered:

1. **Immediate Issues**: Revert last commit
   ```bash
   git revert HEAD
   git push origin refactor/consolidate-transform-utilities
   ```

2. **Wave-Level Issues**: Revert to previous wave
   ```bash
   git log --oneline  # Find wave checkpoint commit
   git revert <commit-hash>..HEAD
   ```

3. **Full Rollback**: Delete branch and restart
   ```bash
   git checkout main
   git branch -D refactor/consolidate-transform-utilities
   # Create new branch and start over
   ```

---

## Success Criteria

The refactoring is successful when:

✅ All 44 resource tests pass
✅ 300+ lines of duplicated code eliminated
✅ 4 new reusable utilities created
✅ 100% test coverage for new utilities
✅ Comprehensive documentation exists
✅ Template available for future migrations
✅ No behavioral changes introduced
✅ Code is more maintainable and readable
✅ Future duplication is prevented

---

## Notes

- **Estimated Total Time**: 18-24 hours
- **Risk Level**: Low to Medium
- **Priority**: High value (improves maintainability)
- **Breaking Changes**: None (internal refactoring only)
- **User Impact**: None (pure refactoring)

---

**Document Version**: 1.0
**Last Updated**: 2025-12-30
**Status**: Ready for Execution
