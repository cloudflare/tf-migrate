# tf-migrate - Cloudflare Terraform Provider Migration Tool

A powerful CLI tool for automatically migrating Terraform configurations and state files between different versions of the Cloudflare Terraform Provider.

## Overview

`tf-migrate` helps you upgrade your Terraform infrastructure code by automatically transforming:
- **Configuration files** (`.tf`) - Updates resource types, attribute names, and block structures
- **State files** (`terraform.tfstate`) - Migrates resource state to match new provider schemas

Currently supports migrations:
- **v4 → v5**: Cloudflare Provider v4 to v5

## Installation

### Building from Source

```bash
# Clone the repository
git clone <repository-url>
cd tf-migrate

# Build the binary
make

# The binary will be available as ./tf-migrate
```

### Requirements
- Go 1.25 or later
- Make
- Terraform (for testing migrated configurations)

## Usage

### Basic Migration

Migrate all Terraform files in the current directory:

```bash
tf-migrate migrate --source-version v4 --target-version v5
```

### Migrate Specific Directory

```bash
tf-migrate migrate --config-dir ./terraform --source-version v4 --target-version v5
```

### Include State File Migration

```bash
tf-migrate migrate \
  --config-dir ./terraform \
  --state-file terraform.tfstate \
  --source-version v4 \
  --target-version v5
```

### Dry Run Mode

Preview changes without modifying files:

```bash
tf-migrate migrate --dry-run --source-version v4 --target-version v5
```

### Migrate Specific Resources Only

```bash
tf-migrate migrate \
  --resources dns_record,zero_trust_list \
  --source-version v4 \
  --target-version v5
```

### Output to Different Directory

```bash
tf-migrate migrate \
  --config-dir ./terraform \
  --output-dir ./terraform-v5 \
  --state-file terraform.tfstate \
  --output-state terraform-v5.tfstate \
  --source-version v4 \
  --target-version v5
```

## Command Reference

### Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config-dir` | Directory containing Terraform configuration files | Current directory |
| `--state-file` | Path to Terraform state file | None |
| `--source-version` | Source provider version (e.g., v4) | Required |
| `--target-version` | Target provider version (e.g., v5) | Required |
| `--resources` | Comma-separated list of resources to migrate | All resources |
| `--dry-run` | Preview changes without modifying files | false |
| `--log-level` | Set log level (debug, info, warn, error, off) | warn |

### Migrate Command Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--output-dir` | Output directory for migrated configuration files | In-place |
| `--output-state` | Output path for migrated state file | In-place |
| `--backup` | Create backup of original files before migration | true |

### Running Tests

#### Unit Tests

Run all unit tests:
```bash
go test ./...
```

Run tests for a specific package:
```bash
go test ./internal/resources/dns_record -v
```

Run with coverage:
```bash
go test ./... -cover
```

#### Integration Tests

Integration tests verify the complete migration workflow using real configuration and state files.

```bash
# Run all v4 to v5 integration tests
cd integration/v4_to_v5
go test -v

# Run tests for a specific resource
go test -v -run TestV4ToV5Migration/DNSRecord

# Test a single resource using environment variable
TEST_RESOURCE=dns_record go test -v -run TestSingleResource

# Run with detailed diff output (set KEEP_TEMP to see test directory)
KEEP_TEMP=true TEST_RESOURCE=dns_record go test -v -run TestSingleResource
```

##### Test Organization

Integration tests are organized by version migration path:
- `integration/test_runner.go` - Shared test runner used by all version tests
- `integration/v4_to_v5/` - Tests for v4 to v5 migrations
  - `integration_test.go` - Test suite specific to v4→v5
  - `testdata/` - Test fixtures for each resource
- Future: `integration/v5_to_v6/` - Tests for v5 to v6 migrations
  - `integration_test.go` - Test suite specific to v5→v6
  - `testdata/` - Test fixtures for each resource

Each version migration has its own test suite with explicit migration registration, while sharing the common test runner infrastructure.