# Transform Utilities

This package provides reusable utilities for transforming Terraform provider resources during schema migrations. The utilities are organized into three main categories:

1. **State Transformations** (`state/`) - JSON state file manipulation
2. **HCL Transformations** (`hcl/`) - Configuration file (HCL) manipulation
3. **Core Framework** (`transform.go`) - Base interfaces and context

## State Transformation Utilities

Located in `internal/transform/state/`, these utilities handle JSON state file transformations using `gjson` and `sjson`.

### Batch Operations (`batch.go`)

Utilities for performing batch operations on state fields.

#### `RemoveFieldsIfExist`
Safely removes multiple fields from state if they exist.

```go
// Remove multiple fields in one call
result = state.RemoveFieldsIfExist(result, "attributes", attrs, "field1", "field2", "field3")

// Before: Multiple conditional deletions
if attrs.Get("jump_start").Exists() {
    result, _ = sjson.Delete(result, "attributes.jump_start")
}
if attrs.Get("plan").Exists() {
    result, _ = sjson.Delete(result, "attributes.plan")
}

// After: Single batch operation
result = state.RemoveFieldsIfExist(result, "attributes", attrs, "jump_start", "plan")
```

**Benefits:**
- Eliminates repetitive if-exists checks
- Reduces code by 3-4 lines per field
- Used by 25+ resources

#### `RenameField`
Renames a field in state if it exists.

```go
result = state.RenameField(result, "attributes", attrs, "old_name", "new_name")
```

#### `RenameFieldsMap`
Renames multiple fields using a map.

```go
renames := map[string]string{
    "old_field1": "new_field1",
    "old_field2": "new_field2",
}
result = state.RenameFieldsMap(result, "attributes", attrs, renames)
```

#### `SetSchemaVersion`
Sets the schema_version field in state.

```go
// Used by all 44 resources
result = state.SetSchemaVersion(result, 0)

// Replaced pattern:
// result, _ = sjson.Set(result, "schema_version", 0)
```

**Benefits:**
- Consistent schema version setting across all resources
- Centralized implementation for future changes
- Saves 1 line per resource (44 total)

### Time Normalization (`time.go`)

Utilities for normalizing time-related values.

#### `NormalizeDuration`
Simplifies Go duration strings by removing zero components.

```go
normalized := state.NormalizeDuration("24h0m0s")  // Returns "24h"
normalized := state.NormalizeDuration("1h30m0s")   // Returns "1h30m"
```

**Usage:**
```go
if duration := attrs.Get("duration"); duration.Exists() {
    result, _ = sjson.Set(result, "attributes.duration",
        state.NormalizeDuration(duration.String()))
}
```

#### `NormalizeRFC3339`
Converts various date formats to RFC3339 with UTC timezone.

```go
normalized := state.NormalizeRFC3339("2024-01-01")                    // "2024-01-01T00:00:00Z"
normalized := state.NormalizeRFC3339("2024-01-01T12:30:45+02:00")    // "2024-01-01T10:30:45Z"
```

**Usage:**
```go
if createdOn := attrs.Get("created_on"); createdOn.Exists() {
    result, _ = sjson.Set(result, "attributes.created_on",
        state.NormalizeRFC3339(createdOn.String()))
}
```

### Structure Comparison (`compare.go`)

Utilities for detecting empty or matching structures.

#### `IsEmptyStructure`
Checks if a complex nested object matches an empty template structure.

```go
emptyTemplate := `{"id": null, "version": null, "count": 0}`
if state.IsEmptyStructure(inputField, emptyTemplate) {
    // Field is effectively empty, clean it up
    result, _ = sjson.Delete(result, "attributes.input")
}
```

**Benefits:**
- Replaces custom isEmpty functions (50-60 lines → 13 lines)
- Handles complex nested structures with deep equality
- Works with any template structure

**Example from zero_trust_device_posture_rule:**
```go
// Before: ~60 lines with custom JSON unmarshaling and reflect.DeepEqual
func (m *V4ToV5Migrator) inputFieldIsEmpty(attrs gjson.Result) bool {
    emptyInput := `{...40+ fields...}`
    inputField := attrs.Get("input")
    // ... manual JSON parsing, map comparison, etc.
}

// After: 13 lines using utility
func (m *V4ToV5Migrator) inputFieldIsEmpty(inputField gjson.Result) bool {
    emptyInputTemplate := `{...40+ fields...}`
    if !inputField.Exists() {
        return false
    }
    if !inputField.IsArray() || len(inputField.Array()) == 0 {
        return true
    }
    inputObj := inputField.Array()[0]
    return state.IsEmptyStructure(inputObj, emptyInputTemplate)
}
```

### Type Conversion (`convert.go`)

Existing utilities for converting between Terraform types.

- `ConvertToFloat64` - Converts various types to float64
- `ConvertGjsonValue` - Converts gjson.Result to appropriate Go type

## HCL Transformation Utilities

Located in `internal/transform/hcl/`, these utilities handle HCL configuration file transformations using `hclwrite`.

### Meta-Arguments (`meta.go`)

Standardized handling of Terraform meta-arguments across resources.

#### `MetaArguments` Struct
Represents all possible Terraform meta-arguments.

```go
type MetaArguments struct {
    Count     *hclwrite.Attribute  // count = 5
    ForEach   *hclwrite.Attribute  // for_each = var.items
    Lifecycle *hclwrite.Block      // lifecycle { ... }
    DependsOn *hclwrite.Attribute  // depends_on = [...]
    Provider  *hclwrite.Attribute  // provider = cloudflare.alt
    Timeouts  *hclwrite.Block      // timeouts { ... }
}
```

#### `ExtractMetaArguments`
Extracts all meta-arguments from a resource block.

```go
// Replaces custom extraction logic
metaArgs := tfhcl.ExtractMetaArguments(block)
```

#### `CopyMetaArgumentsToBlock`
Copies meta-arguments to a resource block.

```go
tfhcl.CopyMetaArgumentsToBlock(newBlock, metaArgs)
```

#### `CopyMetaArgumentsToImport`
Copies meta-arguments to an import block (excludes timeouts and lifecycle).

```go
tfhcl.CopyMetaArgumentsToImport(importBlock, metaArgs)
```

**Benefits:**
- Eliminates custom meta-argument handling (50+ lines per resource)
- Consistent behavior across all resources
- Centralized for future meta-argument additions

**Example from zone_setting:**
```go
// Before: Custom struct, extraction, and copy functions (54 lines)
type metaArguments struct {
    count     *hclwrite.Attribute
    forEach   *hclwrite.Attribute
    lifecycle *hclwrite.Block
    dependsOn *hclwrite.Attribute
    provider  *hclwrite.Attribute
    timeouts  *hclwrite.Block
}

func (m *V4ToV5Migrator) extractMetaArguments(block *hclwrite.Block) *metaArguments {
    // ... 28 lines ...
}

func (m *V4ToV5Migrator) copyIterationMetaArguments(dstImport *hclwrite.Block, meta *metaArguments) {
    // ... 18 lines ...
}

// After: Using shared utilities (0 custom lines)
metaArgs := tfhcl.ExtractMetaArguments(block)
tfhcl.CopyMetaArgumentsToImport(importBlock, metaArgs)
tfhcl.CopyMetaArgumentsToBlock(newBlock, metaArgs)
```

### Block Manipulation

Existing utilities for manipulating HCL blocks and attributes:

- `FindBlockByType` - Find first block of a type
- `FindBlocksByType` - Find all blocks of a type
- `RenameResourceType` - Rename resource type in block labels
- `ConvertBlocksToAttribute` - Convert nested blocks to attributes
- `MergeAttributeAndBlocksToObjectArray` - Merge blocks into object array
- `CreateNestedAttributeFromFields` - Create nested attribute from field map
- `MoveAttributesToNestedObject` - Move attributes into nested object
- `RemoveAttributes` - Remove multiple attributes
- `RemoveBlocksByType` - Remove all blocks of a type

## Core Framework

Located in `internal/transform/`, the core framework provides base interfaces and context.

### Key Interfaces

- `ResourceTransformer` - Main interface for resource migrations
- `ResourceRenamer` - Interface for resources that rename during migration
- `Context` - Execution context with config and state data

### TransformResult

Return type for config transformations:

```go
type TransformResult struct {
    Blocks         []*hclwrite.Block  // Transformed/new blocks
    RemoveOriginal bool               // Whether to remove original block
}
```

## Migration Patterns

### Standard v4→v5 Migration Structure

```go
package myresource

import (
    "github.com/hashicorp/hcl/v2/hclwrite"
    "github.com/tidwall/gjson"
    "github.com/tidwall/sjson"

    "github.com/cloudflare/tf-migrate/internal"
    "github.com/cloudflare/tf-migrate/internal/transform"
    tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
    "github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
    migrator := &V4ToV5Migrator{}
    internal.RegisterMigrator("cloudflare_myresource", "v4", "v5", migrator)
    return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
    return "cloudflare_myresource"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
    return resourceType == "cloudflare_myresource"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
    // String-level transformations before HCL parsing
    return content
}

func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
    // Return same name if no rename, or (oldName, newName) if renaming
    return "cloudflare_myresource", "cloudflare_myresource"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
    body := block.Body()

    // Use HCL utilities for config transformations
    tfhcl.ConvertBlocksToAttribute(body, "input", "input", func(inputBlock *hclwrite.Block) {
        tfhcl.RemoveAttributes(inputBlock.Body(), "deprecated_field")
    })

    return &transform.TransformResult{
        Blocks:         []*hclwrite.Block{block},
        RemoveOriginal: false,
    }, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
    result := stateJSON.String()
    attrs := stateJSON.Get("attributes")

    if !attrs.Exists() {
        result = state.SetSchemaVersion(result, 0)
        return result, nil
    }

    // Use state utilities for transformations
    result = state.RemoveFieldsIfExist(result, "attributes", attrs, "deprecated1", "deprecated2")

    // Convert numeric fields
    if count := attrs.Get("count"); count.Exists() {
        floatVal := state.ConvertToFloat64(count)
        result, _ = sjson.Set(result, "attributes.count", floatVal)
    }

    // Set schema version
    result = state.SetSchemaVersion(result, 0)

    return result, nil
}

func init() {
    NewV4ToV5Migrator()
}
```

## Common Transformation Patterns

### Pattern 1: Batch Field Removal

**When:** Removing multiple deprecated or computed fields

```go
// Instead of:
if attrs.Get("field1").Exists() {
    result, _ = sjson.Delete(result, "attributes.field1")
}
if attrs.Get("field2").Exists() {
    result, _ = sjson.Delete(result, "attributes.field2")
}

// Use:
result = state.RemoveFieldsIfExist(result, "attributes", attrs, "field1", "field2")
```

### Pattern 2: Field Renaming

**When:** Renaming fields during schema migration

```go
// Single rename:
result = state.RenameField(result, "attributes", attrs, "old_name", "new_name")

// Multiple renames:
renames := map[string]string{
    "old1": "new1",
    "old2": "new2",
}
result = state.RenameFieldsMap(result, "attributes", attrs, renames)
```

### Pattern 3: Empty Structure Detection

**When:** Checking if complex nested objects are effectively empty

```go
// Define what "empty" looks like for your structure
emptyTemplate := `{"id": null, "version": null, "enabled": false}`

inputField := attrs.Get("input")
if state.IsEmptyStructure(inputField, emptyTemplate) {
    // Remove the empty structure
    result, _ = sjson.Delete(result, "attributes.input")
}
```

### Pattern 4: Meta-Arguments Handling

**When:** Creating new blocks while preserving count, for_each, etc.

```go
// Extract meta-arguments from source block
metaArgs := tfhcl.ExtractMetaArguments(sourceBlock)

// Apply to new resource block
tfhcl.CopyMetaArgumentsToBlock(newResourceBlock, metaArgs)

// Apply to import block (excludes lifecycle and timeouts)
tfhcl.CopyMetaArgumentsToImport(importBlock, metaArgs)
```

### Pattern 5: Time Normalization

**When:** Normalizing duration or date fields

```go
// Duration normalization
if duration := attrs.Get("timeout"); duration.Exists() {
    result, _ = sjson.Set(result, "attributes.timeout",
        state.NormalizeDuration(duration.String()))
}

// RFC3339 date normalization
if createdOn := attrs.Get("created_on"); createdOn.Exists() {
    result, _ = sjson.Set(result, "attributes.created_on",
        state.NormalizeRFC3339(createdOn.String()))
}
```

## Testing Your Migration

Always test both config and state transformations:

```go
func TestV4ToV5Transformation(t *testing.T) {
    migrator := NewV4ToV5Migrator()

    t.Run("ConfigTransformation", func(t *testing.T) {
        input := `
resource "cloudflare_myresource" "example" {
  name = "test"
}
`
        expected := `
resource "cloudflare_myresource" "example" {
  name = "test"
}
`
        // Test config transformation
    })

    t.Run("StateTransformation", func(t *testing.T) {
        input := `{
            "schema_version": 4,
            "attributes": {
                "name": "test",
                "deprecated_field": "value"
            }
        }`

        expected := `{
            "schema_version": 0,
            "attributes": {
                "name": "test"
            }
        }`
        // Test state transformation
    })
}
```

## Performance Considerations

- **Batch operations** reduce the number of sjson operations, improving performance for resources with many fields
- **Conditional checks** before deletions prevent unnecessary sjson.Delete calls that would fail
- **Template-based comparison** in `IsEmptyStructure` uses efficient JSON parsing and reflect.DeepEqual

## Lines Saved Summary

Through these refactorings, we've eliminated significant code duplication:

- **SetSchemaVersion**: 44 resources × 1 line = 44 lines
- **RemoveFieldsIfExist**: ~25 resources × 3-4 lines per field = 75-100 lines
- **Meta-arguments**: 1 resource × 54 lines = 54 lines (3 more resources identified)
- **IsEmptyStructure**: 1 resource × 47 lines = 47 lines

**Total: 220-245 lines eliminated** with utilities that improve maintainability and consistency.

## Future Enhancements

Potential areas for additional utilities:

1. **Nested array/object transformations** - Common pattern in several resources
2. **Conditional field moves** - Moving fields based on other field values
3. **Validation utilities** - Common validation patterns
4. **Diff utilities** - Comparing before/after transformations

## Contributing

When adding new utilities:

1. Place in appropriate package (`state/` or `hcl/`)
2. Add comprehensive godoc comments with examples
3. Create thorough unit tests with edge cases
4. Update this README with usage examples
5. Update affected resources to use the new utility
