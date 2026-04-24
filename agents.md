# tf-migrate - Claude Context Guide

This document provides accurate, code-verified context about the tf-migrate project for AI agents starting with empty context. All patterns, function names, and interfaces are taken directly from the source code.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [Core Interfaces](#core-interfaces)
4. [How Migrations Work](#how-migrations-work)
5. [Resource Transformers](#resource-transformers)
6. [HCL Utilities Reference](#hcl-utilities-reference)
7. [Testing System](#testing-system)
8. [Drift Exemptions System](#drift-exemptions-system)
9. [Development Guide](#development-guide)
10. [Common Patterns](#common-patterns)

---

## Project Overview

**tf-migrate** is a CLI tool for automatically migrating Terraform configurations between different versions of the Cloudflare Terraform Provider. State migration is handled by the provider's built-in `UpgradeState`/`MoveState` mechanisms — tf-migrate only transforms config (`.tf` files).

### What It Does

- **Transforms `.tf` configuration files** — Updates resource types, attribute names, and block structures
- **Generates `moved {}` blocks** — For resource renames so state is updated without recreation
- **Generates `import {}` blocks** — For new v5 resources that don't exist in state yet
- **Generates `removed {}` blocks** — For resources removed from v5 that need state cleanup
- **Rewrites cross-file references** — Updates resource type names, attribute references, and computed attribute references across all files in the directory
- **Updates `required_providers`** — Fetches latest provider version from GitHub and updates provider version constraint
- **Phased migration** — Handles resources with no v5 schema via a two-phase workflow (comment out + `removed {}` first, full migration second)

### Supported Migration Paths

- **v4 → v5**: Cloudflare Provider v4 to v5 (80+ resource types, 6+ data sources)
- **v5 → v5**: Bypass mode — generates `moved {}` blocks only (used for provider-level schema moves within v5)

### Technology Stack

- **Language**: Go 1.25+
- **HCL Parsing**: `github.com/hashicorp/hcl/v2`
- **JSON manipulation**: `github.com/tidwall/gjson` + `github.com/tidwall/sjson` (used in state value transforms)
- **CLI Framework**: `github.com/spf13/cobra`
- **No Cloudflare API client** — tf-migrate does not call the Cloudflare API. The only outbound call is to the GitHub API to fetch the latest provider version.

---

## Architecture

### Pipeline Design

Config file transformation follows a four-handler chain of responsibility:

```
Input bytes
  → PreprocessHandler   (string-level transforms before HCL parsing)
  → ParseHandler        (bytes → hclwrite.File AST)
  → ResourceTransformHandler  (calls migrator.TransformConfig per resource block)
  → FormatterHandler    (AST → formatted bytes)
Output bytes
```

Built by `BuildConfigPipeline(log, provider)` in `internal/pipeline/pipeline.go`.

### Directory Structure

```
tf-migrate/
├── cmd/
│   ├── tf-migrate/          # Main CLI binary (migrate, verify-drift, version commands)
│   └── e2e/                 # E2E test runner binary
│
├── internal/
│   ├── migrator.go          # Central migrator registry (RegisterMigrator, GetMigrator, GetAllMigrators)
│   ├── pipeline/            # Pipeline orchestration (BuildConfigPipeline)
│   ├── handlers/            # Pipeline handlers
│   │   ├── pre_process.go
│   │   ├── parse.go
│   │   ├── resource_transform.go
│   │   └── formatter.go
│   ├── resources/           # Per-resource migration implementations (80+ resources)
│   ├── datasources/         # Data source migrations
│   ├── registry/            # registry.go — calls NewV4ToV5Migrator() for every package (bootstrap only)
│   ├── transform/           # Interfaces, Context, TransformResult, utilities
│   │   ├── transformer.go   # All interfaces: ResourceTransformer, PhaseOneTransformer, ResourceRenamer, etc.
│   │   ├── handler.go       # TransformationHandler interface
│   │   ├── empty_values.go  # TransformEmptyValuesToNull
│   │   ├── utils.go         # ConvertDateToRFC3339
│   │   └── hcl/             # HCL manipulation helpers
│   ├── verifydrift/         # verify-drift command implementation (embedded exemptions)
│   ├── e2e-runner/          # E2E test runner implementation
│   └── logger/              # Logging utilities
│
├── integration/
│   └── v4_to_v5/
│       ├── integration_test.go
│       └── testdata/        # Input/expected fixtures per resource
│
└── e2e/
    ├── global-drift-exemptions.yaml
    └── drift-exemptions/    # Resource-specific drift exemption configs
```

---

## Core Interfaces

All interfaces are defined in `internal/transform/transformer.go`.

### `ResourceTransformer` (required)

Every migrator must implement all four methods:

```go
type ResourceTransformer interface {
    CanHandle(resourceType string) bool
    TransformConfig(ctx *Context, block *hclwrite.Block) (*TransformResult, error)
    GetResourceType() string
    Preprocess(content string) string
}
```

- `CanHandle` — returns true if this migrator handles the given resource type string
- `TransformConfig` — performs HCL AST transformation; see return values below
- `GetResourceType` — returns the primary v4 resource type string this migrator handles
- `Preprocess` — string-level transform applied before HCL parsing (most migrators return `content` unchanged)

### `TransformResult`

```go
type TransformResult struct {
    Blocks         []*hclwrite.Block  // Output blocks to write
    RemoveOriginal bool               // Whether to remove the original block
}
```

Common patterns:
- **In-place**: `{Blocks: [modifiedBlock], RemoveOriginal: false}`
- **Split** (one-to-many): `{Blocks: newBlocks, RemoveOriginal: true}`
- **Remove** (resource gone in v5): `{Blocks: nil, RemoveOriginal: true}`

### `Context`

```go
type Context struct {
    Content       []byte
    Filename      string              // Base filename (e.g., "main.tf")
    FilePath      string              // Full path
    CFGFile       *hclwrite.File      // Parsed HCL AST (set by ParseHandler)
    CFGFiles      map[string]*hclwrite.File  // All files (for cross-file transforms)
    Diagnostics   hcl.Diagnostics
    Metadata      map[string]interface{}
    Resources     []string           // Filtered resource list (from --resources flag)
    SourceVersion string             // e.g., "v4"
    TargetVersion string             // e.g., "v5"
}
```

### Optional Interfaces

Implement these on your migrator struct to opt into additional framework features:

```go
// PhaseOneTransformer — for resources with no v5 schema.
// TransformPhaseOne is called in the first pass; it should return a removed {} block.
// Full TransformConfig is called in the second pass after state is cleaned up.
type PhaseOneTransformer interface {
    TransformPhaseOne(ctx *Context, block *hclwrite.Block) (*TransformResult, error)
}

// ResourceRenamer — enables cross-file resource type reference rewriting.
// Return all v4 names that map to the v5 name, even if only one exists.
type ResourceRenamer interface {
    GetResourceRename() (oldTypes []string, newType string)
}

// AttributeRenamer — enables cross-file attribute reference rewriting.
type AttributeRenamer interface {
    GetAttributeRenames() []AttributeRename
}

// ComputedAttributeMapper — rewrites computed attribute references across files
// when both the resource type and the attribute name change.
// Example: cloudflare_record.example.hostname → cloudflare_dns_record.example.name
type ComputedAttributeMapper interface {
    GetComputedAttributeMappings() []ComputedAttributeMapping
}
```

Supporting types:

```go
type AttributeRename struct {
    ResourceType string  // e.g., "data.cloudflare_zones"
    OldAttribute string  // e.g., "zones"
    NewAttribute string  // e.g., "result"
}

type ComputedAttributeMapping struct {
    OldResourceType string  // e.g., "cloudflare_record"
    OldAttribute    string  // e.g., "hostname"
    NewResourceType string  // e.g., "cloudflare_dns_record"
    NewAttribute    string  // e.g., "name"
}
```

### Diagnostic Severity

```go
const DiagInfo = hcl.DiagnosticSeverity(0)  // Shown only with --verbose
// hcl.DiagWarning — shown unless --quiet
// hcl.DiagError   — always shown, blocks migration of this resource
```

---

## How Migrations Work

### Migrator Registration

Each resource package exports a `NewV4ToV5Migrator()` function that creates the migrator struct and registers it:

```go
// internal/resources/dns_record/v4_to_v5.go
func NewV4ToV5Migrator() transform.ResourceTransformer {
    migrator := &V4ToV5Migrator{}
    internal.RegisterMigrator("cloudflare_record", "v4", "v5", migrator)
    return migrator
}
```

`internal/registry/registry.go` imports every resource package and calls all `NewV4ToV5Migrator()` functions from a single `RegisterAllMigrations()` function. `cmd/tf-migrate/main.go` calls `RegisterAllMigrations()` at startup.

The actual registry is `internal/migrator.go` (package `internal`):

```go
// Key format: "resourceType:sourceVersion:targetVersion"
func RegisterMigrator(sourceVersionResourceType, sourceVersion, targetVersion string, migrator transform.ResourceTransformer)
func GetMigrator(resourceType, sourceVersion, targetVersion string) transform.ResourceTransformer
func GetAllMigrators(sourceVersion, targetVersion string, resources ...string) []transform.ResourceTransformer
```

### Multiple v4 Names → One v5 Name

When multiple v4 type names map to the same v5 type, call `RegisterMigrator` multiple times with the same migrator instance:

```go
func NewV4ToV5Migrator() transform.ResourceTransformer {
    migrator := &V4ToV5Migrator{}
    internal.RegisterMigrator("cloudflare_tunnel_route", "v4", "v5", migrator)
    internal.RegisterMigrator("cloudflare_zero_trust_tunnel_route", "v4", "v5", migrator)
    return migrator
}

func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
    return []string{
        "cloudflare_tunnel_route",
        "cloudflare_zero_trust_tunnel_route",
    }, "cloudflare_zero_trust_tunnel_cloudflared_route"
}
```

Return all v4 names in `GetResourceRename()` so cross-file references from any of those names are rewritten.

### Phased Migration

Resources whose v4 type has no schema in v5 (e.g., `cloudflare_zone_settings_override`) require two passes:

**Phase 1** — detected automatically; `TransformPhaseOne` is called:
- Comments out the original resource block with `# tf-migrate: ` prefix
- Appends a `removed { lifecycle { destroy = false } }` block
- Writes the file and exits — user must commit, apply with v4 provider to drop state, then re-run

**Phase 2** — tf-migrate detects the commented blocks, prompts the user (bypassed by `--skip-phase-check`), uncomments them, and runs the full `TransformConfig` pass.

### Global Postprocessing

After all files are transformed, `applyGlobalPostprocessing` runs over every file to rewrite cross-file references. It collects:

1. `ResourceRenamer.GetResourceRename()` → updates resource type references everywhere
2. `AttributeRenamer.GetAttributeRenames()` → updates attribute references
3. `ComputedAttributeMapper.GetComputedAttributeMappings()` → updates computed attribute references (e.g., `.hostname` → `.name`)

References inside `moved {}` and `removed {}` blocks are skipped.

### Transformation Types

#### Simple attribute rename

```go
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
    tfhcl.RenameAttribute(block.Body(), "old_name", "new_name")
    return &transform.TransformResult{Blocks: []*hclwrite.Block{block}}, nil
}
```

#### Resource type rename

Handled automatically by `ResourceRenamer` + global postprocessing. Within `TransformConfig`, the block's label is already the v4 type — do not rename it manually.

#### One-to-many resource split

Return `RemoveOriginal: true` with multiple blocks. See `internal/resources/zone_setting/v4_to_v5.go` for the full example.

#### Nested block restructuring (block → attribute)

Use `tfhcl.ConvertBlocksToAttribute`, `tfhcl.ConvertSingleBlockToAttribute`, `tfhcl.ConvertBlocksToArrayAttribute`, etc.

#### Resource removed in v5

```go
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
    // Emit warning, generate removed block
    removedBlock := tfhcl.CreateRemovedBlock("cloudflare_old_type." + tfhcl.GetResourceName(block))
    return &transform.TransformResult{Blocks: []*hclwrite.Block{removedBlock}, RemoveOriginal: true}, nil
}
```

#### Cross-resource config migration

Some migrators implement `ProcessCrossResourceConfigMigration(file *hclwrite.File) error` (not part of any interface — called explicitly from the resource transform handler for resources that need it). Examples: `list_item` (merges into parent `cloudflare_list`), `zero_trust_split_tunnel` (merges into device profiles).

---

## Resource Transformers

### File structure

```
internal/resources/<resource_name>/
├── v4_to_v5.go         # Migrator implementation
├── v4_to_v5_test.go    # Unit tests
└── README.md           # Documents v4→v5 changes and examples
```

### Minimal implementation

```go
package my_resource

import (
    "github.com/cloudflare/tf-migrate/internal"
    "github.com/cloudflare/tf-migrate/internal/transform"
    tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
    "github.com/hashicorp/hcl/v2/hclwrite"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
    migrator := &V4ToV5Migrator{}
    internal.RegisterMigrator("cloudflare_my_resource", "v4", "v5", migrator)
    return migrator
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
    return resourceType == "cloudflare_my_resource"
}

func (m *V4ToV5Migrator) GetResourceType() string {
    return "cloudflare_my_resource"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
    return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
    body := block.Body()
    tfhcl.RenameAttribute(body, "old_attr", "new_attr")
    return &transform.TransformResult{Blocks: []*hclwrite.Block{block}}, nil
}
```

If the resource is renamed, also add to `registry.go` and implement `GetResourceRename()`.

---

## HCL Utilities Reference

All helpers live in `internal/transform/hcl/`. Import as `tfhcl`.

### Attributes (`attributes.go`)

| Function | Description |
|----------|-------------|
| `RenameAttribute(body, oldName, newName string) bool` | Rename attribute; also updates lifecycle ignore_changes references |
| `RemoveAttributes(body, attrNames ...string) int` | Remove one or more attributes; returns count removed |
| `SetAttribute(body, attrName string, value interface{})` | Set attribute unconditionally |
| `EnsureAttribute(body, attrName, defaultValue string)` | Set attribute only if not already present |
| `HasAttribute(body, attrName string) bool` | Check if attribute exists |
| `ExtractStringFromAttribute(attr *hclwrite.Attribute) string` | Get string value from attribute |
| `ExtractBoolFromAttribute(attr *hclwrite.Attribute) (bool, bool)` | Get bool value; second return is ok |
| `CopyAttribute(from, to *hclwrite.Body, attrName string)` | Copy attribute between bodies |
| `CopyAndRenameAttribute(from, to *hclwrite.Body, oldName, newName string) bool` | Copy with rename |
| `ApplyAttributeRenames(body *hclwrite.Body, renames map[string]string) int` | Apply a map of renames |
| `ConditionalRenameAttribute(body, oldName, newName string, condition func() bool) bool` | Rename only if condition holds |
| `MoveAttributesToNestedObject(body, nestedAttrName string, fieldNames []string) int` | Group fields into a nested object |
| `WrapMapValuesInObjects(body, attrName, wrapFieldName string) bool` | TypeMap → MapNestedAttribute pattern |
| `SortStringArrayAttribute(body, attrName string, customSort ...func(a, b string) bool)` | Sort string array attribute |

### Blocks (`blocks.go`)

| Function | Description |
|----------|-------------|
| `RenameResourceType(block *hclwrite.Block, oldType, newType string) bool` | Update the resource type label |
| `GetResourceType(block *hclwrite.Block) string` | Get resource type label |
| `GetResourceName(block *hclwrite.Block) string` | Get resource name label |
| `FindBlockByType(body, blockType string) *hclwrite.Block` | First block of given type |
| `FindBlocksByType(body, blockType string) []*hclwrite.Block` | All blocks of given type |
| `RemoveBlocksByType(body, blockType string) int` | Remove all blocks of given type |
| `ProcessBlocksOfType(body, blockType string, processor func(*hclwrite.Block) error) error` | Iterate blocks with callback |
| `HoistAttributeFromBlock(parentBody *hclwrite.Body, blockType, attrName string) bool` | Pull attribute up from nested block |
| `HoistAttributesFromBlock(parentBody *hclwrite.Body, blockType string, attrNames ...string) int` | Pull multiple attributes up |
| `ConvertBlocksToAttribute(body, blockType, attrName string, preProcess func(*hclwrite.Block))` | Convert repeated blocks to object attribute |
| `ConvertSingleBlockToAttribute(body, blockType, attrName string) bool` | Convert single block to attribute |
| `ConvertBlocksToArrayAttribute(body, blockType string, emptyIfNone bool) bool` | Convert blocks to array attribute |
| `ConvertBlocksToAttributeList(body, blockType string, preProcess func(*hclwrite.Block)) bool` | Convert blocks to list attribute |
| `ConvertDynamicBlocksToForExpression(body, targetBlockType string)` | Convert dynamic blocks to for expressions |
| `CreateMovedBlock(from, to string) *hclwrite.Block` | Generate `moved {}` block |
| `CreateRemovedBlock(from string) *hclwrite.Block` | Generate `removed { lifecycle { destroy = false } }` block |
| `CreateImportBlock(resourceType, resourceName, importID string) *hclwrite.Block` | Generate `import {}` block with string ID |
| `CreateImportBlockWithTokens(resourceType, resourceName string, idTokens hclwrite.Tokens) *hclwrite.Block` | Generate `import {}` block with expression ID |
| `CreateDerivedBlock(original *hclwrite.Block, newResourceType, newResourceName string, transform AttributeTransform) *hclwrite.Block` | Create a new block derived from an existing one |
| `AddLifecycleIgnoreChanges(body *hclwrite.Body, attrNames ...string)` | Add or merge lifecycle ignore_changes |

`AttributeTransform` struct used with `CreateDerivedBlock`:

```go
type AttributeTransform struct {
    Copy              []string            // Attribute names to copy verbatim
    Rename            map[string]string   // old → new attribute renames to apply
    Set               map[string]interface{} // Attributes to set to fixed values
    CopyMetaArguments bool                // Whether to copy depends_on, count, for_each
}
```

### Tokens (`tokens.go`)

| Function | Description |
|----------|-------------|
| `AppendWarningComment(body *hclwrite.Body, message string)` | Writes `# MIGRATION WARNING: <message>` into the body |
| `BuildResourceReference(resourceType, resourceName string) hclwrite.Tokens` | `cloudflare_foo.bar` token sequence |
| `TokensForSimpleValue(val interface{}) hclwrite.Tokens` | Tokens for a scalar value |
| `TokensForEmptyArray() hclwrite.Tokens` | Tokens for `[]` |

### Expressions (`expressions.go`)

| Function | Description |
|----------|-------------|
| `SetAttributeFromExpressionString(body *hclwrite.Body, attrName, exprStr string) error` | Set attribute to a raw expression string |
| `IsExpressionAttribute(attr *hclwrite.Attribute) bool` | True if attribute value is a non-literal expression |
| `ConvertEnabledDisabledInExpr(expr string) string` | Replaces `"enabled"`/`"disabled"` with `true`/`false` in expression strings |
| `RemoveFunctionWrapper(body *hclwrite.Body, attrName, funcName string)` | Strips a function call wrapper from an attribute |

### Arrays (`arrays.go`)

| Function | Description |
|----------|-------------|
| `ParseArrayAttribute(attr *hclwrite.Attribute) []ArrayElement` | Parse array attribute; returns nil for for-expressions |
| `MergeAttributeAndBlocksToObjectArray(body, arrayAttrName, blockType, outputAttrName, primaryField string, optionalFields []string, blocksFirst bool) bool` | Merge an existing array attribute with blocks into one object array |
| `BuildArrayFromObjects(objects []hclwrite.Tokens) hclwrite.Tokens` | Build array token sequence from object token slices |

### State transforms (`empty_values.go`)

```go
// TransformEmptyValuesToNull transforms empty string values in Terraform state JSON
// to null, but only for attributes not explicitly set in the HCL config.
// Used by resources like logpush_job, zero_trust_device_posture_rule.
func TransformEmptyValuesToNull(opts TransformEmptyValuesToNullOptions) string
```

---

## Testing System

### Test Layers

```
E2E Tests (real Cloudflare infrastructure)
  ↑ requires credentials + R2 remote state
Integration Tests (fixture files, no credentials)
  ↑ complete pipeline with testdata
Unit Tests (individual transformers)
  ↑ fast, no I/O
```

### Unit Tests

```bash
go test -race ./internal/...          # All unit tests
go test ./internal/resources/dns_record/... -v  # Single resource
```

### Integration Tests

Integration tests run the full migration pipeline against fixture files in `integration/v4_to_v5/testdata/`. No Cloudflare credentials required. The test binary is built once per test run to `tf-migrate-integration-test` in the repo root.

```bash
make test-integration
# or
go test -race ./integration/...

# Single resource (uses TEST_RESOURCE env var)
TEST_RESOURCE=dns_record go test -v -run TestSingleResource ./integration/...
```

**Testdata structure:**

```
integration/v4_to_v5/testdata/<resource>/
├── input/          # v4 .tf files fed into the migration pipeline
│   ├── main.tf
│   └── <resource>_e2e.tf   # E2E-specific variant (preferred by init over main.tf)
└── expected/       # Expected v5 output (compared after migration)
    └── main.tf
```

All resource names in testdata must use the `cftftest` prefix. Enforced by `make lint-testdata`.

### E2E Tests

E2E tests create and destroy real Cloudflare infrastructure. Use a dedicated test account — never production.

**Required environment variables:**

| Variable | Required for |
|----------|-------------|
| `CLOUDFLARE_ACCOUNT_ID` | init, run |
| `CLOUDFLARE_ZONE_ID` | init, run |
| `CLOUDFLARE_DOMAIN` | init, run |
| `CLOUDFLARE_API_KEY` | backend (v4 apply) |
| `CLOUDFLARE_EMAIL` | backend (v4 apply) |
| `CLOUDFLARE_R2_ACCESS_KEY_ID` | backend (v4 apply), clean |
| `CLOUDFLARE_R2_SECRET_ACCESS_KEY` | backend (v4 apply), clean |
| `CLOUDFLARE_CROWDSTRIKE_*` | device posture rule tests only |
| `CLOUDFLARE_BYO_IP_*` | byo_ip_prefix tests (have defaults) |

**E2E workflow:**

```
1. init    → Copy input files from testdata, prefer *_e2e.tf over *.tf
             Generate versions.tf, main.tf, terraform.tfvars
             Write R2 backend config
2. v4 apply → terraform init + apply against real Cloudflare (state in R2)
3. migrate  → Build tf-migrate binary, run migration, hoist import blocks to root
4. v5 apply → terraform apply with v5 provider against existing infrastructure
5. drift    → terraform plan; verify "No changes" (with optional exemptions)
```

**E2E runner binary commands** (binary is `./bin/e2e`):

```bash
# Full suite
./bin/e2e run --apply-exemptions

# Specific resources
./bin/e2e run --resources dns_record,zone_setting --apply-exemptions

# By phase (0, 1, or 2)
./bin/e2e run --phase 0 --apply-exemptions

# Exclude resources
./bin/e2e run --exclude byo_ip_prefix --apply-exemptions

# Individual steps
./bin/e2e init [--resources <csv>] [--phase <n>]
./bin/e2e migrate [--resources <csv>] [--phase <n>] [--target-provider-version <ver>]
./bin/e2e clean --modules dns_record,zone_setting
./bin/e2e bootstrap

# v5 provider upgrade testing
./bin/e2e v5-upgrade [--from-version <ver>] [--to-version <ver>] [--resources <csv>]
./bin/e2e v5-upgrade-clean [--from-version <ver>] [--to-version <ver>]
```

**Phase system** — resources are grouped into 3 phases (0, 1, 2) for parallelism control. Use `--phase 0,1` to run multiple phases. Defined in `internal/e2e-runner/phases.go`.

**Import annotations** — for resources that must be imported rather than created:

```hcl
# tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id = var.cloudflare_account_id
}
```

The e2e init step generates native Terraform `import {}` blocks in the root `main.tf` with `module.<name>.` prefix.

Supported variable substitutions: `${var.cloudflare_account_id}`, `${var.cloudflare_zone_id}`, `${var.cloudflare_domain}`

**Post-migration patches** — for edge cases that can't be expressed in testdata alone, create patch files:

```
integration/v4_to_v5/testdata/<resource>/postmigrate/*.patch
```

First line of each patch file = target `resourcetype.name`. Remaining lines are injected after the opening brace.

---

## Drift Exemptions System

Exemptions classify expected differences between v4 and v5 provider behavior so they don't fail the E2E drift check.

### Hierarchy

1. `e2e/global-drift-exemptions.yaml` — applies to all resources
2. `e2e/drift-exemptions/<resource>.yaml` — resource-specific; can override or disable global exemptions

Run `make sync-exemptions` to copy these into `internal/verifydrift/exemptions/` (where they are embedded at build time for the `verify-drift` command).

### Exemption schema

```yaml
version: 1

exemptions:
  - name: "unique_identifier"
    description: "Why this exemption is needed"

    # Scope filters (all optional — omitting matches everything)
    resource_types:
      - "cloudflare_zone_setting"
    resource_name_patterns:
      - 'module\.zone_setting\..*'
    attributes:
      - "ttl"

    # Match by regex pattern in plan output lines
    patterns:
      - '\(known after apply\)'
      - 'status.*->.*"active"'

    # Or use simplified exemptions (no patterns needed)
    allow_resource_creation: true
    allow_resource_destruction: true
    allow_resource_replacement: true

    enabled: true

settings:
  apply_exemptions: true         # Master toggle (default: false in global config)
  verbose_exemptions: false      # Show which exemptions matched
  warn_unused_exemptions: false  # Warn about exemptions that matched nothing
  load_resource_exemptions: true # Load resource-specific config files
```

### Key behaviors

- Resource-specific exemptions are checked **before** global exemptions
- A resource-specific exemption can disable a global one: set `enabled: false` with the same `name`
- Pattern-only exemptions (no `resource_types`) get implicitly scoped to `cloudflare_<filename>` for resource-specific files
- `allow_resource_creation/destruction/replacement` exemptions do NOT get implicit scoping

### verify-drift command

```bash
terraform plan > plan.txt
tf-migrate verify-drift --file plan.txt
```

Exit code 0 = all drift is expected or no changes. Exit code 1 = unexpected drift found. Use in CI:

```bash
terraform plan > plan.txt && tf-migrate verify-drift --file plan.txt || exit 1
```

The `verify-drift` command uses exemptions embedded at build time from `internal/verifydrift/exemptions/`. Keep them in sync with `make sync-exemptions`.

---

## Development Guide

### Setup

```bash
git clone https://github.com/cloudflare/tf-migrate
cd tf-migrate
go mod download
make build-all   # builds ./bin/tf-migrate and ./bin/e2e
```

### Adding a New Resource Transformer

**Step 1: Create the resource directory**

```bash
mkdir -p internal/resources/<resource_name>
```

Use the v5 resource name without the `cloudflare_` prefix (e.g., `dns_record`, `zone_setting`).

**Step 2: Implement `v4_to_v5.go`**

```go
package <resource_name>

import (
    "github.com/cloudflare/tf-migrate/internal"
    "github.com/cloudflare/tf-migrate/internal/transform"
    tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
    "github.com/hashicorp/hcl/v2/hclwrite"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
    migrator := &V4ToV5Migrator{}
    internal.RegisterMigrator("cloudflare_<v4_name>", "v4", "v5", migrator)
    return migrator
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
    return resourceType == "cloudflare_<v4_name>"
}

func (m *V4ToV5Migrator) GetResourceType() string { return "cloudflare_<v4_name>" }
func (m *V4ToV5Migrator) Preprocess(content string) string { return content }

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
    body := block.Body()
    // ... transform body ...
    return &transform.TransformResult{Blocks: []*hclwrite.Block{block}}, nil
}

// If the resource is renamed:
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
    return []string{"cloudflare_<v4_name>"}, "cloudflare_<v5_name>"
}
```

**Step 3: Register in `registry.go`**

Add the import and a `<package>.NewV4ToV5Migrator()` call to `internal/registry/registry.go` following the existing pattern.

**Step 4: Add integration testdata**

```
integration/v4_to_v5/testdata/<resource>/
├── input/<resource>.tf        # v4 configuration
├── input/<resource>_e2e.tf    # E2E-specific variant (if different from integration test)
└── expected/<resource>.tf     # Expected v5 output
```

All resource names must use the `cftftest` prefix.

**Step 5: Add resource README**

`internal/resources/<resource>/README.md` — document what changed, before/after examples, any manual steps.

**Step 6: Verify**

```bash
go test ./internal/resources/<resource>/... -v
make test-integration
make lint-testdata
```

### Makefile Targets

```bash
make build           # Build tf-migrate binary
make build-all       # Build tf-migrate + e2e binaries
make test            # Unit + integration tests
make test-unit       # go test -race ./internal/...
make test-integration # go test -race ./integration/...
make lint-testdata   # Enforce cftftest naming convention in testdata
make sync-exemptions # Copy e2e/ exemption YAMLs into verifydrift/exemptions/
make release-snapshot # Test GoReleaser build locally (no publish)
make clean           # Remove bin/
```

### Debugging

```bash
# Preview changes without modifying files
./bin/tf-migrate migrate --dry-run --source-version v4 --target-version v5

# Verbose output: per-file progress, rename tables, info-level diagnostics
./bin/tf-migrate migrate -v --source-version v4 --target-version v5

# Debug logging
./bin/tf-migrate migrate --log-level debug --source-version v4 --target-version v5
```

---

## Common Patterns

### Pattern 1: Attribute rename

```go
tfhcl.RenameAttribute(body, "old_name", "new_name")
```

### Pattern 2: Remove deprecated attributes

```go
tfhcl.RemoveAttributes(body, "deprecated_field_1", "deprecated_field_2")
```

### Pattern 3: Nested block → attribute

```go
// block { key = "value" } → attr = { key = "value" }
tfhcl.ConvertSingleBlockToAttribute(body, "block_name", "attr_name")
```

### Pattern 4: Repeated blocks → array attribute

```go
// Multiple include {} blocks → include = [{ ... }, { ... }]
tfhcl.ConvertBlocksToArrayAttribute(body, "include", false)
```

### Pattern 5: Resource split (one-to-many)

```go
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
    var newBlocks []*hclwrite.Block

    // Generate N new blocks from original
    for _, setting := range settings {
        newBlock := tfhcl.CreateDerivedBlock(block, "cloudflare_zone_setting", label, transform)
        importBlock := tfhcl.CreateImportBlock("cloudflare_zone_setting", label, importID)
        newBlocks = append(newBlocks, newBlock, importBlock)
    }

    // Generate removed block for original resource
    removedBlock := tfhcl.CreateRemovedBlock(from)
    newBlocks = append(newBlocks, removedBlock)

    return &transform.TransformResult{Blocks: newBlocks, RemoveOriginal: true}, nil
}
```

### Pattern 6: Manual intervention warning

When a required field can't be populated automatically:

```go
tfhcl.AppendWarningComment(body, "This resource requires manual intervention to add v5 required fields 'asn' and 'cidr'. Find values in Cloudflare Dashboard → Manage Account → IP Addresses → IP Prefixes.")
```

This writes `# MIGRATION WARNING: ...` directly into the output `.tf` file. Document the warning in `DIAGNOSTICS.md`.

### Pattern 7: Hoist attribute from nested block

```go
// Move "priority" from data {} block up to resource body
tfhcl.HoistAttributeFromBlock(body, "data", "priority")
```

### Pattern 8: Add lifecycle ignore_changes

```go
// For write-only attributes that need a placeholder value
tfhcl.AddLifecycleIgnoreChanges(body, "certificate", "private_key")
```

### Pattern 9: Generate moved block for renamed resource

```go
oldAddr := fmt.Sprintf("cloudflare_old_type.%s", tfhcl.GetResourceName(block))
newAddr := fmt.Sprintf("cloudflare_new_type.%s", tfhcl.GetResourceName(block))
movedBlock := tfhcl.CreateMovedBlock(oldAddr, newAddr)
```

Return both the transformed block and the moved block:

```go
return &transform.TransformResult{
    Blocks:         []*hclwrite.Block{block, movedBlock},
    RemoveOriginal: false,
}, nil
```

### Pattern 10: Emit a diagnostic warning

```go
ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
    Severity: hcl.DiagWarning,
    Summary:  "Action required: something needs manual attention",
    Detail:   "Detailed instructions here.",
})
```

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `internal/transform/transformer.go` | All interfaces: `ResourceTransformer`, `PhaseOneTransformer`, `ResourceRenamer`, `AttributeRenamer`, `ComputedAttributeMapper`, `MigrationProvider` |
| `internal/migrator.go` | Central registry: `RegisterMigrator`, `GetMigrator`, `GetAllMigrators` |
| `internal/registry/registry.go` | Calls all `NewV4ToV5Migrator()` functions — add new resources here |
| `internal/pipeline/pipeline.go` | `BuildConfigPipeline` — assembles the 4-handler chain |
| `internal/transform/hcl/` | All HCL manipulation helpers |
| `cmd/tf-migrate/main.go` | CLI entry point, phased migration orchestration, global postprocessing |
| `cmd/tf-migrate/preflight.go` | Pre-migration resource classification scan |
| `cmd/tf-migrate/version_check.go` | Minimum provider version check (v4.52.5) |
| `internal/verifydrift/` | `verify-drift` command implementation with embedded exemptions |
| `internal/e2e-runner/runner.go` | E2E test orchestration |
| `internal/e2e-runner/phases.go` | Phase→resource mapping |
| `internal/e2e-runner/drift.go` | Drift detection and exemption logic |
| `internal/e2e-runner/init.go` | `RunInit` — syncs testdata to e2e/tf/v4/ |
| `internal/e2e-runner/migrate.go` | Migration step in E2E workflow |
| `internal/resources/zone_setting/v4_to_v5.go` | Reference implementation: one-to-many split + PhaseOneTransformer |
| `internal/resources/argo/v4_to_v5.go` | Reference implementation: resource split with CreateDerivedBlock |
| `internal/resources/zero_trust_access_policy/v4_to_v5.go` | Reference implementation: conditional manual intervention |
| `internal/resources/dns_record/v4_to_v5.go` | Reference implementation: rename + ComputedAttributeMapper |

---

**Last Updated**: 2026-04-24
**Version**: Based on v1.0.0 GA implementation
