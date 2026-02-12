# tf-migrate - Claude Context Guide

This document provides comprehensive context about the tf-migrate project for AI agents starting with empty context.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [How Migrations Work](#how-migrations-work)
4. [Resource Transformers](#resource-transformers)
5. [Testing System](#testing-system)
6. [Drift Exemptions System](#drift-exemptions-system)
7. [Development Guide](#development-guide)
8. [Common Patterns](#common-patterns)

---

## Project Overview

**tf-migrate** is a CLI tool for automatically migrating Terraform configurations and state files between different versions of the Cloudflare Terraform Provider.

### What It Does

- **Transforms `.tf` configuration files** - Updates resource types, attribute names, and block structures
- **Migrates `terraform.tfstate` files** - Updates state to match new provider schemas
- **Handles complex transformations** - One-to-many resource splits, nested block restructuring, API-based migrations

### Current Support

- **v4 → v5**: Cloudflare Provider v4 to v5 (60+ resource types)
- Future: v5 → v6, etc.

### Key Use Cases

1. **Automated provider upgrades** - Bulk migrate large Terraform codebases
2. **API-aware migrations** - Fetch data from Cloudflare API when needed (e.g., tunnel route UUIDs)
3. **State file migration** - Update state in place or to new file
4. **Testing infrastructure** - Validate migrations with real resources via E2E tests

### Technology Stack

- **Language**: Go 1.25+
- **HCL Parsing**: `github.com/hashicorp/hcl/v2`
- **State Manipulation**: `github.com/tidwall/gjson`, `github.com/tidwall/sjson`
- **Cloudflare API**: `github.com/cloudflare/cloudflare-go/v6`
- **CLI Framework**: `github.com/spf13/cobra`

---

## Architecture

### Design Patterns

**Chain of Responsibility Pattern** - Transformations flow through a pipeline of handlers:

```
Input → Preprocess → Parse → Transform → Format → Output
```

### Core Components

```
tf-migrate/
├── cmd/
│   ├── tf-migrate/          # Main migration CLI binary
│   └── e2e-runner/          # E2E test runner CLI binary
│
├── internal/
│   ├── pipeline/            # Pipeline orchestration
│   │   └── pipeline.go      # BuildConfigPipeline, BuildStatePipeline
│   │
│   ├── handlers/            # Pipeline handlers (chain of responsibility)
│   │   ├── preprocess.go    # Resource filtering, validation
│   │   ├── parse.go         # HCL parsing
│   │   ├── transform.go     # Resource transformation orchestration
│   │   ├── format.go        # HCL formatting
│   │   └── state.go         # State file transformation
│   │
│   ├── resources/           # Per-resource migration implementations
│   │   ├── dns_record/      # cloudflare_dns_record v4→v5
│   │   ├── zone_setting/    # cloudflare_zone_settings_override v4→v5
│   │   ├── bot_management/  # cloudflare_bot_management v4→v5
│   │   └── ...              # 60+ resources
│   │
│   ├── datasources/         # Data source migrations
│   │   ├── zone/
│   │   └── ...
│   │
│   ├── registry/            # Resource transformer registry
│   │   └── registry.go      # Register, GetTransformer
│   │
│   ├── transform/           # Transformation interfaces
│   │   ├── interfaces.go    # ResourceTransformer, StateTransformer
│   │   ├── context.go       # Transformation context
│   │   └── result.go        # Transformation result
│   │
│   ├── e2e-runner/          # E2E test runner implementation
│   │   ├── runner.go        # Test orchestration
│   │   ├── drift.go         # Drift detection & exemptions
│   │   ├── init.go          # Test initialization
│   │   └── migrate.go       # Migration step
│   │
│   └── logger/              # Logging utilities
│
├── integration/             # Integration tests
│   ├── v4_to_v5/           # v4→v5 migration tests
│   │   ├── integration_test.go
│   │   └── testdata/        # Test fixtures per resource
│   │       ├── dns_record/
│   │       ├── zone_setting/
│   │       └── ...
│   └── test_runner.go       # Shared test infrastructure
│
├── e2e/                     # End-to-end tests
│   ├── tf/v4/              # v4 test resources (generated)
│   ├── migrated-v4_to_v5/  # Migration output (generated)
│   ├── global-drift-exemptions.yaml    # Global drift exemptions
│   ├── drift-exemptions/    # Resource-specific drift exemptions
│   │   ├── zone_setting.yaml
│   │   ├── zone_dnssec.yaml
│   │   └── bot_management.yaml
│   └── DRIFT-EXEMPTIONS.md  # Drift exemptions docs (deprecated, see CLAUDE.md)
│
└── scripts/                 # Helper scripts
    ├── run-e2e-tests.sh    # E2E test runner
    └── ...
```

### Pipeline Flow

#### Config File Pipeline

```
1. Preprocess Handler
   ↓ (filters resources, validates)
2. Parse Handler
   ↓ (HCL → AST)
3. Resource Transform Handler
   ↓ (registry.GetTransformer → resource.TransformHCL)
4. Formatter Handler
   ↓ (AST → formatted HCL)
Output: Migrated .tf file
```

#### State File Pipeline

```
1. Preprocess Handler
   ↓ (filters resources, validates)
2. State Transform Handler
   ↓ (registry.GetTransformer → resource.TransformState)
3. State Formatter Handler
   ↓ (format JSON)
Output: Migrated .tfstate file
```

### Resource Transformer Registry

All resource transformers register themselves in `init()`:

```go
// internal/resources/dns_record/v4_to_v5.go
func init() {
    registry.Register(registry.ResourceEntry{
        Version:      "v4_to_v5",
        ResourceType: "cloudflare_dns_record",
        Transformer:  &DNSRecordTransformer{},
    })
}
```

The registry provides:
- **GetTransformer(version, resourceType)** - Lookup transformer
- **ListResources(version)** - List all available resources
- **IsRegistered(version, resourceType)** - Check if resource supported

---

## How Migrations Work

### Transformation Types

#### 1. Simple Attribute Rename

```hcl
# v4
resource "cloudflare_dns_record" "example" {
  zone_id = "abc123"
  name    = "example.com"
  type    = "A"
  value   = "192.0.2.1"
  proxied = true
}

# v5 (no changes needed for dns_record)
resource "cloudflare_dns_record" "example" {
  zone_id = "abc123"
  name    = "example.com"
  type    = "A"
  content = "192.0.2.1"  # value → content
  proxied = true
}
```

#### 2. Resource Type Rename

```hcl
# v4
resource "cloudflare_access_application" "example" {
  # ...
}

# v5
resource "cloudflare_zero_trust_access_application" "example" {
  # ...
}
```

#### 3. One-to-Many Resource Split

**Most complex transformation pattern**

```hcl
# v4 - ONE resource
resource "cloudflare_zone_settings_override" "example" {
  zone_id = "abc123"
  settings {
    tls_1_3 = "on"
    minify {
      css = "on"
      js  = "on"
    }
  }
}

# v5 - MULTIPLE resources
resource "cloudflare_zone_setting" "example_tls_1_3" {
  zone_id    = "abc123"
  setting_id = "tls_1_3"
  value      = "on"
}

resource "cloudflare_zone_setting" "example_minify" {
  zone_id    = "abc123"
  setting_id = "minify"
  value = {
    css = "on"
    js  = "on"
  }
}
```

#### 4. Nested Block Restructuring

```hcl
# v4
resource "cloudflare_access_policy" "example" {
  application_id = "abc123"
  include {
    email = ["user@example.com"]
  }
  exclude {
    email_domain = ["contractor.com"]
  }
}

# v5
resource "cloudflare_zero_trust_access_policy" "example" {
  application_id = "abc123"
  include = [{
    email = ["user@example.com"]
  }]
  exclude = [{
    email_domain = ["contractor.com"]
  }]
}
```

#### 5. API-Based Migration

Some resources require API calls to complete migration:

```hcl
# v4
resource "cloudflare_tunnel_route" "example" {
  account_id = "abc123"
  tunnel_id  = "def456"
  network    = "10.0.0.0/16"  # v4 used this as the ID
}

# v5
resource "cloudflare_zero_trust_tunnel_cloudflared_route" "example" {
  account_id = "abc123"
  tunnel_id  = "def456"
  network    = "10.0.0.0/16"
  # ID must be UUID from API (not the network CIDR)
}
```

The transformer calls the Cloudflare API to fetch the UUID:

```go
func (t *TunnelRouteTransformer) TransformState(ctx *transform.Context) (*transform.Result, error) {
    // Fetch tunnel routes from API
    routes, err := ctx.CloudflareClient.ListTunnelRoutes(...)

    // Find matching route by network
    for _, route := range routes {
        if route.Network == network {
            // Update state ID with API UUID
            ctx.State.SetID(route.ID)
        }
    }
}
```

### State Migration

State files are JSON:

```json
{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "type": "cloudflare_dns_record",
      "name": "example",
      "instances": [
        {
          "attributes": {
            "zone_id": "abc123",
            "name": "example.com",
            "value": "192.0.2.1"
          }
        }
      ]
    }
  ]
}
```

State transformers update:
- `type` field (resource type rename)
- `instances[].attributes` (attribute migrations)
- `instances[].schema_version` (provider schema version)

---

## Resource Transformers

Each resource has a `v4_to_v5.go` file implementing:

```go
type ResourceTransformer interface {
    // Transform HCL configuration
    TransformHCL(ctx *transform.Context) (*transform.Result, error)

    // Transform state file
    TransformState(ctx *transform.Context) (*transform.Result, error)
}
```

### Example: DNS Record Transformer

```go
// internal/resources/dns_record/v4_to_v5.go
type DNSRecordTransformer struct{}

func (t *DNSRecordTransformer) TransformHCL(ctx *transform.Context) (*transform.Result, error) {
    // No changes needed for dns_record in v4→v5
    return &transform.Result{
        Content: ctx.OriginalContent,
    }, nil
}

func (t *DNSRecordTransformer) TransformState(ctx *transform.Context) (*transform.Result, error) {
    // Update schema version
    state := ctx.State
    state.SetSchemaVersion(1)

    return &transform.Result{
        Content: state.Bytes(),
    }, nil
}
```

### Common Transformation Utilities

**HCL Utilities** (`internal/transform/hcl_utils.go`):
- `RenameAttribute(block, old, new)` - Rename attribute
- `RenameBlock(block, old, new)` - Rename block type
- `RemoveAttribute(block, name)` - Remove attribute
- `GetAttribute(block, name)` - Get attribute value
- `SetAttribute(block, name, value)` - Set attribute value

**State Utilities** (`internal/transform/state_utils.go`):
- `GetAttribute(path)` - GJSON path query
- `SetAttribute(path, value)` - SJSON path update
- `DeleteAttribute(path)` - Remove attribute
- `SetResourceType(type)` - Change resource type

### Resource-Specific Documentation

Each resource has a README.md explaining:
- Changes from v4 to v5
- Configuration examples
- State migration notes
- Special cases

Example: `internal/resources/zone_setting/README.md`

---

## Testing System

### Test Layers

```
┌─────────────────────────────────────────┐
│          E2E Tests (Real API)           │  ← Full workflow with real Cloudflare resources
│  ./scripts/run-e2e-tests.sh            │
└─────────────────────────────────────────┘
                  ↑
┌─────────────────────────────────────────┐
│      Integration Tests (Fixtures)       │  ← Complete migration workflow with test data
│  make test-integration                  │
└─────────────────────────────────────────┘
                  ↑
┌─────────────────────────────────────────┐
│          Unit Tests (Go)                │  ← Individual component testing
│  go test ./...                          │
└─────────────────────────────────────────┘
```

### 1. Unit Tests

Test individual transformers:

```bash
# Run all unit tests
go test ./...

# Test specific resource
go test ./internal/resources/dns_record -v

# With coverage
go test ./... -cover
```

### 2. Integration Tests

Located in `integration/v4_to_v5/testdata/`:

```
testdata/
├── dns_record/
│   ├── main.tf           # v4 configuration
│   ├── expected.tf       # Expected v5 output
│   ├── state.json        # v4 state
│   └── expected_state.json  # Expected v5 state
├── zone_setting/
│   └── ...
└── bot_management/
    └── ...
```

Run integration tests:

```bash
# All v4→v5 integration tests
make test-integration

# Specific resource
go test -v -run TestV4ToV5Migration/DNSRecord

# Single resource with environment variable
TEST_RESOURCE=dns_record go test -v -run TestSingleResource

# Keep temp directory for debugging
KEEP_TEMP=true TEST_RESOURCE=dns_record go test -v -run TestSingleResource
```

### 3. E2E Tests

**Most comprehensive testing** - Uses real Cloudflare infrastructure.

#### E2E Workflow

```
1. Init      → Copy integration testdata to e2e/tf/v4/
2. V4 Apply  → Create real resources with v4 provider
3. Migrate   → Run tf-migrate (config + state)
4. V5 Apply  → Apply v5 configs to existing infrastructure
5. Drift     → Verify v5 plan shows "No changes"
```

#### Prerequisites

```bash
export CLOUDFLARE_ACCOUNT_ID="your-account-id"
export CLOUDFLARE_ZONE_ID="your-zone-id"
export CLOUDFLARE_DOMAIN="your-test-domain.com"

# Authentication (choose one)
export CLOUDFLARE_API_TOKEN="your-api-token"  # Recommended
# OR
export CLOUDFLARE_EMAIL="your-email@example.com"
export CLOUDFLARE_API_KEY="your-api-key"
```

#### Run E2E Tests

```bash
# Build and run full suite
./scripts/run-e2e-tests.sh --apply-exemptions

# Or manually
make build-all
./bin/e2e-runner run --apply-exemptions

# Test specific resources
./bin/e2e-runner run --resources dns_record,zone_setting

# Individual commands
./bin/e2e-runner init
./bin/e2e-runner migrate
./bin/e2e-runner clean --modules module.dns_record
```

#### E2E Runner Features

- ✅ **Credential sanitization** - Prevents secrets in logs
- ✅ **Drift detection** - Validates successful migration
- ✅ **Resource filtering** - Test specific resources
- ✅ **Colored output** - Clear success/failure indicators
- ✅ **88 unit tests** - E2E runner itself is well-tested

#### Import Annotations

Some resources cannot be created, only imported (e.g., `zero_trust_organization`).

Use annotations in module files:

```hcl
# integration/v4_to_v5/testdata/zero_trust_organization/_e2e.tf

# tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id  = var.cloudflare_account_id
  name        = "Test Organization"
  auth_domain = "test.cloudflareaccess.com"
}
```

E2E runner generates import blocks:

```hcl
# e2e/tf/v4/main.tf (auto-generated)

import {
  to = module.zero_trust_organization.cloudflare_access_organization.test
  id = var.cloudflare_account_id
}
```

---

## Drift Exemptions System

The drift exemptions system allows you to define acceptable differences between v4 and v5 provider behavior during E2E testing, preventing false positives from failing CI while still catching real migration bugs.

### Hierarchical Configuration

Exemptions are organized in two levels:

1. **Global exemptions** (`e2e/global-drift-exemptions.yaml`) - Apply to all resources
2. **Resource-specific exemptions** (`e2e/drift-exemptions/{resource}.yaml`) - Override global settings

### File Locations

```
e2e/
├── global-drift-exemptions.yaml    # Global exemptions (all resources)
└── drift-exemptions/
    ├── zone_setting.yaml           # Zone setting specific
    ├── zone_dnssec.yaml            # Zone DNSSEC specific
    ├── bot_management.yaml         # Bot management specific
    └── {resource}.yaml             # Any resource can have one
```

### Global Exemptions

**File**: `e2e/global-drift-exemptions.yaml`

```yaml
version: 1

exemptions:
  # Standard computed field exemptions
  - name: "computed_value_refreshes"
    description: "Ignore attributes that refresh to 'known after apply'"
    patterns:
      - '\(known after apply\)'
      - '= \{\} -> null'
    enabled: true

settings:
  apply_exemptions: true
  verbose_exemptions: false
  warn_unused_exemptions: false
  load_resource_exemptions: true
```

### Resource-Specific Exemptions

**File**: `e2e/drift-exemptions/zone_setting.yaml`

```yaml
version: 1

exemptions:
  # One-to-many transformation: v4 zone_settings_override -> multiple v5 zone_setting
  - name: "allow_zone_setting_creation"
    description: "Zone settings are created during v4->v5 migration"
    allow_resource_creation: true
    enabled: true

  # Plan-specific features
  - name: "unsupported_plan_features"
    description: "Features not available on free/pro plans"
    patterns:
      - "0rtt"
      - "tls_1_3"
    enabled: true

  # Disable global exemption for this resource
  - name: "computed_value_refreshes"
    enabled: false

settings:
  apply_exemptions: true
  verbose_exemptions: false
```

### Exemption Schema

#### Required Fields

- `name` - Unique identifier for the exemption
- `description` - Human-readable explanation (what and why)
- `enabled` - Boolean to enable/disable the exemption

#### Scope Filters (optional)

- `resource_types` - List of resource types (e.g., `["cloudflare_zone"]`)
- `resource_name_patterns` - Regex patterns for resource names (e.g., `["module\\.zone_setting\\..*"]`)
- `attributes` - List of attribute names to match

#### Pattern Matching

**Regex Patterns:**
```yaml
patterns:
  - 'status.*->.*"active"'  # Status changes to active
  - '\(known after apply\)' # Computed fields
  - '- email = .* -> null'  # Deletions
```

**Simplified Patterns (recommended):**
```yaml
# Allow entire resource to be created
allow_resource_creation: true

# Allow entire resource to be destroyed
allow_resource_destruction: true

# Allow entire resource to be replaced
allow_resource_replacement: true
```

### Usage Examples

#### Example 1: Allow Resource Creation

When migrating from a single v4 resource to multiple v5 resources:

```yaml
exemptions:
  - name: "allow_zone_setting_creation"
    description: "v4 zone_settings_override splits into multiple v5 zone_setting"
    allow_resource_creation: true
    resource_types:
      - "cloudflare_zone_setting"
    enabled: true
```

#### Example 2: Resource-Specific Computed Fields

Only exempt computed fields for test resources:

```yaml
exemptions:
  - name: "test_policy_computed"
    description: "Computed fields for test policies only"
    resource_name_patterns:
      - 'module\.zero_trust_access_policy\..*\.test'
    patterns:
      - 'precedence.*\(known after apply\)'
    enabled: true
```

#### Example 3: Override Global Exemption

Disable a global exemption for a specific resource:

```yaml
exemptions:
  # Disable global timestamp exemption - we want to catch drift here
  - name: "computed_value_refreshes"
    enabled: false
```

#### Example 4: Instance-Specific Pattern

```yaml
exemptions:
  - name: "test_resources_only"
    description: "Only for test resources"
    resource_name_patterns:
      - 'module\.test_.*'
      - 'module\.staging\..*'
    patterns:
      - "some-pattern"
    enabled: true
```

### Running E2E Tests with Exemptions

```bash
# With exemptions (default in CI)
./scripts/run-e2e-tests.sh --apply-exemptions

# Without exemptions (strict mode)
./scripts/run-e2e-tests.sh

# For specific resources
./scripts/run-e2e-tests.sh --apply-exemptions --resources zone_setting,bot_management
```

### Creating Resource-Specific Exemptions

```bash
# Create new resource-specific exemption
cat > e2e/drift-exemptions/my_resource.yaml <<EOF
version: 1
exemptions:
  - name: "my_exemption"
    description: "Why this is needed"
    allow_resource_creation: true
    enabled: true
settings:
  apply_exemptions: true
EOF

# Test
./scripts/run-e2e-tests.sh --apply-exemptions --resources my_resource
```

### Settings Reference

#### Global Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `apply_exemptions` | `false` | Master toggle for exemptions |
| `verbose_exemptions` | `false` | Show which exemptions matched |
| `warn_unused_exemptions` | `false` | Warn about unused exemptions |
| `load_resource_exemptions` | `true` | Load resource-specific configs |

#### For Debugging

```yaml
settings:
  verbose_exemptions: true        # See what's being exempted
  warn_unused_exemptions: true    # Find stale exemptions
```

### Best Practices

#### 1. Be Specific

```yaml
# ❌ Too broad
patterns:
  - ".*"

# ✅ Specific
patterns:
  - 'status.*->.*"active"'
resource_types:
  - "cloudflare_zone_dnssec"
```

#### 2. Use Simplified Patterns

```yaml
# ❌ Complex
patterns:
  - "will be created"
  - "\\+ resource"
  - "\\+ id"
  # ... many more

# ✅ Simple
allow_resource_creation: true
```

#### 3. Document Exemptions

```yaml
# ✅ Well documented
- name: "zone_setting_migration_drift"
  description: "v4 zone_settings_override splits into multiple v5 zone_setting resources during migration"
  allow_resource_creation: true
```

#### 4. Use Resource-Specific Configs

Keep global config clean - put resource-specific exemptions in their own files.

#### 5. Enable Warnings

```yaml
settings:
  warn_unused_exemptions: true
```

### Troubleshooting

#### Exemption Not Matching

1. Enable verbose mode:
```yaml
settings:
  verbose_exemptions: true
```

2. Check regex:
```bash
echo "your-text" | grep -E "your-pattern"
```

3. Remove filters:
```yaml
# Comment out to match all resources
# resource_types: ["cloudflare_zone"]
```

#### Too Many False Positives

Create resource-specific exemption:
```bash
cat > e2e/drift-exemptions/my_resource.yaml <<EOF
version: 1
exemptions:
  - name: "my_specific_exemption"
    description: "Special case for my_resource"
    patterns:
      - "specific-pattern"
    enabled: true
settings:
  apply_exemptions: true
EOF
```

### Output Format

E2E test output shows which exemptions were applied:

```
Step 4: Verifying stable state (v5 plan after apply)
✓ No drift detected

Drift Exemptions Applied:
  Global exemptions:
    - computed_value_refreshes: 12 matches (from global-drift-exemptions.yaml)

  Resource-specific exemptions:
    - allow_zone_setting_creation: 8 matches (from e2e/drift-exemptions/zone_setting.yaml)
    - unsupported_plan_features: 2 matches (from e2e/drift-exemptions/zone_setting.yaml)
```

---

## Development Guide

### Setting Up Development Environment

```bash
# Clone repository
git clone <repository-url>
cd tf-migrate

# Install dependencies
go mod download

# Build binaries
make build-all

# Run tests
make test
```

### Adding a New Resource Transformer

#### Step 1: Create Resource Directory

```bash
mkdir -p internal/resources/my_resource
```

#### Step 2: Implement Transformer

```go
// internal/resources/my_resource/v4_to_v5.go
package my_resource

import (
    "github.com/cloudflare/tf-migrate/internal/registry"
    "github.com/cloudflare/tf-migrate/internal/transform"
)

type MyResourceTransformer struct{}

func init() {
    registry.Register(registry.ResourceEntry{
        Version:      "v4_to_v5",
        ResourceType: "cloudflare_my_resource",
        Transformer:  &MyResourceTransformer{},
    })
}

func (t *MyResourceTransformer) TransformHCL(ctx *transform.Context) (*transform.Result, error) {
    // Transform HCL configuration
    // Use transform.RenameAttribute, transform.RenameBlock, etc.

    return &transform.Result{
        Content: ctx.OriginalContent,
    }, nil
}

func (t *MyResourceTransformer) TransformState(ctx *transform.Context) (*transform.Result, error) {
    // Transform state file
    // Use ctx.State.SetAttribute, ctx.State.SetResourceType, etc.

    return &transform.Result{
        Content: ctx.State.Bytes(),
    }, nil
}
```

#### Step 3: Add Tests

```go
// internal/resources/my_resource/v4_to_v5_test.go
package my_resource

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/cloudflare/tf-migrate/internal/transform"
)

func TestMyResourceTransformer_TransformHCL(t *testing.T) {
    input := `
resource "cloudflare_my_resource" "test" {
  name = "example"
}
`

    transformer := &MyResourceTransformer{}
    ctx := &transform.Context{
        OriginalContent: []byte(input),
    }

    result, err := transformer.TransformHCL(ctx)
    assert.NoError(t, err)
    assert.Contains(t, string(result.Content), "cloudflare_my_resource")
}
```

#### Step 4: Add Integration Test Data

```bash
mkdir -p integration/v4_to_v5/testdata/my_resource
```

Create test fixtures:

```hcl
# integration/v4_to_v5/testdata/my_resource/main.tf
resource "cloudflare_my_resource" "test" {
  name = "example"
}
```

```hcl
# integration/v4_to_v5/testdata/my_resource/expected.tf
resource "cloudflare_my_resource" "test" {
  name = "example"
}
```

#### Step 5: Add Documentation

```markdown
# internal/resources/my_resource/README.md

# My Resource Migration Guide (v4 → v5)

## Changes

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_my_resource` | `cloudflare_my_resource` | No change |
| `name` attribute | Required | Required | No change |

## Examples

...
```

#### Step 6: Run Tests

```bash
# Unit tests
go test ./internal/resources/my_resource -v

# Integration tests
TEST_RESOURCE=my_resource go test -v -run TestSingleResource

# E2E tests (if applicable)
./scripts/run-e2e-tests.sh --resources my_resource --apply-exemptions
```

### Makefile Targets

```bash
make build           # Build tf-migrate binary
make build-all       # Build both tf-migrate and e2e-runner
make test            # Run all tests
make test-unit       # Run unit tests only
make test-integration # Run integration tests
make test-e2e        # Run e2e-runner unit tests
make lint            # Run linter
make lint-testdata   # Lint testdata naming conventions
make clean           # Clean build artifacts
```

### Debugging Tips

#### Enable Debug Logging

```bash
./bin/tf-migrate migrate --log-level debug --dry-run
```

#### Inspect Intermediate Results

```bash
# Keep temp files in integration tests
KEEP_TEMP=true TEST_RESOURCE=my_resource go test -v -run TestSingleResource

# Check generated files
ls -la /tmp/tf-migrate-test-*
```

#### Debug E2E Tests

```bash
# Enable verbose drift exemptions
# Edit e2e/global-drift-exemptions.yaml:
settings:
  verbose_exemptions: true

# Run with debug output
./bin/e2e-runner run --resources my_resource --apply-exemptions
```

---

## Common Patterns

### Pattern 1: Simple Attribute Rename

```go
func (t *MyTransformer) TransformHCL(ctx *transform.Context) (*transform.Result, error) {
    for _, block := range ctx.File.Body.Blocks {
        if block.Type == "resource" {
            transform.RenameAttribute(block.Body, "old_name", "new_name")
        }
    }
    return &transform.Result{Content: ctx.OriginalContent}, nil
}
```

### Pattern 2: Resource Type Rename

```go
func (t *MyTransformer) TransformHCL(ctx *transform.Context) (*transform.Result, error) {
    for _, block := range ctx.File.Body.Blocks {
        if block.Type == "resource" && block.Labels[0] == "cloudflare_old_name" {
            block.Labels[0] = "cloudflare_new_name"
        }
    }
    return &transform.Result{Content: ctx.OriginalContent}, nil
}
```

### Pattern 3: Nested Block to Attribute

```go
// v4: nested block
// block {
//   nested {
//     value = "foo"
//   }
// }

// v5: attribute
// nested = { value = "foo" }

func (t *MyTransformer) TransformHCL(ctx *transform.Context) (*transform.Result, error) {
    // Use HCL traversal to restructure
    // See internal/resources/zero_trust_access_policy for examples
}
```

### Pattern 4: One-to-Many Split

```go
// See internal/resources/zone_setting/v4_to_v5.go for full example

func (t *ZoneSettingTransformer) TransformHCL(ctx *transform.Context) (*transform.Result, error) {
    // 1. Parse original resource
    // 2. Extract settings from settings block
    // 3. Generate N new resources (one per setting)
    // 4. Copy meta-arguments to all resources
    // 5. Return combined content
}
```

### Pattern 5: API-Based Migration

```go
func (t *TunnelRouteTransformer) TransformState(ctx *transform.Context) (*transform.Result, error) {
    // 1. Get resource attributes
    accountID := ctx.State.GetAttribute("account_id")
    tunnelID := ctx.State.GetAttribute("tunnel_id")
    network := ctx.State.GetAttribute("network")

    // 2. Call Cloudflare API
    routes, err := ctx.CloudflareClient.ListTunnelRoutes(ctx, accountID, tunnelID)
    if err != nil {
        return nil, err
    }

    // 3. Find matching route
    for _, route := range routes {
        if route.Network == network {
            // 4. Update state with API data
            ctx.State.SetAttribute("id", route.ID)
            break
        }
    }

    return &transform.Result{Content: ctx.State.Bytes()}, nil
}
```

### Pattern 6: Conditional Transformation

```go
func (t *MyTransformer) TransformHCL(ctx *transform.Context) (*transform.Result, error) {
    for _, block := range ctx.File.Body.Blocks {
        if block.Type == "resource" {
            // Check if attribute exists
            if attr := transform.GetAttribute(block.Body, "old_attr"); attr != nil {
                // Only transform if attribute is present
                transform.RenameAttribute(block.Body, "old_attr", "new_attr")
            }
        }
    }
    return &transform.Result{Content: ctx.OriginalContent}, nil
}
```

---

## Additional Resources

### Key Files to Understand

1. **`internal/pipeline/pipeline.go`** - Pipeline orchestration
2. **`internal/registry/registry.go`** - Resource transformer registration
3. **`internal/transform/interfaces.go`** - Core interfaces
4. **`internal/e2e-runner/runner.go`** - E2E test orchestration
5. **`internal/e2e-runner/drift.go`** - Drift detection & exemptions
6. **`internal/resources/zone_setting/v4_to_v5.go`** - Complex one-to-many example
7. **`internal/resources/zero_trust_tunnel_cloudflared_route/v4_to_v5.go`** - API-based migration example

### External Documentation

- [Terraform HCL2 Specification](https://github.com/hashicorp/hcl/tree/main/hclsyntax)
- [Cloudflare Terraform Provider v5 Docs](https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs)
- [Cloudflare API Docs](https://developers.cloudflare.com/api/)

### Getting Help

- Check resource-specific README.md files
- Review integration test fixtures for examples
- Use `--dry-run` to preview changes
- Enable debug logging: `--log-level debug`
- Review E2E test output for drift exemptions

---

## Quick Command Reference

```bash
# Build
make build-all

# Test
go test ./...                              # All unit tests
make test-integration                      # Integration tests
./scripts/run-e2e-tests.sh --apply-exemptions  # E2E tests

# Run migration
./bin/tf-migrate migrate --source-version v4 --target-version v5
./bin/tf-migrate migrate --dry-run  # Preview only

# E2E runner
./bin/e2e-runner run --apply-exemptions
./bin/e2e-runner run --resources dns_record,zone_setting
./bin/e2e-runner init
./bin/e2e-runner migrate

# Debug
./bin/tf-migrate migrate --log-level debug --dry-run
KEEP_TEMP=true TEST_RESOURCE=dns_record go test -v -run TestSingleResource
```

---

**Last Updated**: 2026-02-11
**Version**: Based on v4→v5 migration implementation
