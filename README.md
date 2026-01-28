# tf-migrate - Cloudflare Terraform Provider Migration Tool

A powerful CLI tool for automatically migrating Terraform configurations and state files between different versions of the Cloudflare Terraform Provider.

## Overview

`tf-migrate` helps you upgrade your Terraform infrastructure code by automatically transforming:
- **Configuration files** (`.tf`) - Updates resource types, attribute names, and block structures
- **State files** (`terraform.tfstate`) - Migrates resource state to match new provider schemas

Currently supports migrations:
- **v4 → v5**: Cloudflare Provider v4 to v5

## Documentation

For comprehensive documentation including architecture, testing, drift exemptions, and development guide, see **[CLAUDE.md](./CLAUDE.md)** - designed to give AI agents (and humans!) complete project context.

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

### Authentication

Some resource migrations require access to the Cloudflare API to complete the migration successfully. The tool supports two authentication methods:

**Option 1: API Token (Recommended)**
```bash
export CLOUDFLARE_API_TOKEN="your-api-token"
```

**Option 2: API Key + Email**
```bash
export CLOUDFLARE_API_KEY="your-api-key"
export CLOUDFLARE_EMAIL="your-email@example.com"
```

#### Resources Requiring Authentication

The following resources require API credentials for complete migration:

- `cloudflare_tunnel_route` → `cloudflare_zero_trust_tunnel_cloudflared_route`
  - **Why**: The v4 provider stored network CIDR as the resource ID, but v5 requires the UUID from the API. The migration queries the API to fetch the correct UUID for your tunnel routes.
  - **Without credentials**: The migration will still update resource types and attributes, but you'll need to run `terraform refresh` after migration to update the IDs.

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

### Development Tools

#### Linting Testdata

The project includes a linter to ensure all integration test resources follow naming conventions (prefixed with `cftftest`):

```bash
make lint-testdata
```

See [scripts/README.md](scripts/README.md) for more details on the testdata linter.

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
make test-integration

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

#### End-to-End Tests

E2E tests validate the complete migration workflow with real Cloudflare resources using a Go-based CLI test runner.

**What E2E Tests Do:**
1. **Init**: Sync test resources from integration testdata to `e2e/tf/v4/`
2. **V4 Apply**: Create real infrastructure using v4 provider
3. **Migrate**: Run tf-migrate to convert v4 → v5 configurations and state
4. **V5 Apply**: Apply v5 configs to verify compatibility with existing infrastructure
5. **Drift Check**: Verify v5 plan shows "No changes" (validates successful migration)

**Key Features:**
- ✅ **Credential sanitization** - Prevents API keys/secrets from leaking in logs
- ✅ **Drift detection** - Hierarchical exemptions via global + resource-specific configs
- ✅ **Colored output** - Clear success/failure indicators
- ✅ **Resource filtering** - Test specific resources: `--resources custom_pages,load_balancer_monitor`
- ✅ **88 unit tests** - Comprehensive coverage of the e2e-runner itself

**Prerequisites:**

Set required environment variables:

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

**Quick Start:**

```bash
# Build binaries and run full E2E test suite
./scripts/run-e2e-tests.sh --apply-exemptions

# Or build and run manually
make build-all
./bin/e2e-runner run --apply-exemptions
```

**CLI Commands:**

```bash
# Run full e2e test suite
./bin/e2e-runner run

# Test specific resources only
./bin/e2e-runner run --resources custom_pages,load_balancer_monitor

# Run with drift exemptions applied
./bin/e2e-runner run --apply-exemptions

# Initialize test resources from integration testdata
./bin/e2e-runner init
./bin/e2e-runner init --resources dns_record

# Run migration only
./bin/e2e-runner migrate
./bin/e2e-runner migrate --resources zero_trust_tunnel_cloudflared_route

# Bootstrap: Migrate local state to R2 remote backend (one-time setup)
./bin/e2e-runner bootstrap

# Clean up: Remove modules from remote state
./bin/e2e-runner clean --modules module.dns_record,module.load_balancer
```

**Testing the E2E Runner:**

```bash
# Run e2e-runner's own unit tests (88 tests)
make test-e2e

# Run all tests (tf-migrate + e2e-runner)
make test
```

**Drift Exemptions:**

Configure acceptable drift patterns using hierarchical YAML configuration:

- `e2e/global-drift-exemptions.yaml` - Global exemptions for all resources
- `e2e/drift-exemptions/{resource}.yaml` - Resource-specific exemptions

Use `--apply-exemptions` to apply exemptions during testing. See **[CLAUDE.md](./CLAUDE.md#drift-exemptions-system)** for complete documentation.

**Import Annotations (Import-Only Resources):**

Some Cloudflare resources cannot be created via Terraform and must be imported from existing infrastructure (e.g., `zero_trust_organization`). The E2E runner supports automatic import block generation via annotations in `_e2e.tf` files:

```hcl
# tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id  = var.cloudflare_account_id
  name        = "Test Organization"
  auth_domain = "test.cloudflareaccess.com"
}
```

**How It Works:**
1. E2E runner scans module files for `# tf-migrate:import-address=<address>` annotations during init
2. Substitutes variables: `${var.cloudflare_account_id}` → actual account ID (e.g., `abc123`)
3. Generates native Terraform import blocks in root `main.tf`:
   ```hcl
   import {
     to = module.zero_trust_organization.cloudflare_access_organization.test
     id = "abc123"
   }
   ```
4. Terraform automatically imports resources during `terraform apply`

**Why This Approach:**
- ✅ Uses native Terraform import blocks (Terraform 1.5+)
- ✅ Import blocks in root module (where they're allowed)
- ✅ Resource definitions in child modules (organized structure)
- ✅ Automatic generation from annotations (no manual import block maintenance)

**Supported Variables:**
- `${var.cloudflare_account_id}` - Account ID from environment
- `${var.cloudflare_zone_id}` - Zone ID from environment
- `${var.cloudflare_domain}` - Domain from environment

**Multiple Imports:**
Multiple resources can be annotated in different modules - all will have import blocks generated in root main.tf.

**Project Structure:**

```
cmd/
├── tf-migrate/       # Main migration binary
└── e2e-runner/       # E2E test runner binary
internal/
├── resources/        # Migration implementations
└── e2e-runner/       # E2E runner implementation (88 unit tests)
e2e/
├── global-drift-exemptions.yaml
├── drift-exemptions/
├── tf/v4/           # Test fixtures
└── migrated-v4_to_v5/  # Migration output
bin/                 # Built binaries
```

**CI/CD:**

E2E tests run automatically in GitHub Actions on push to `main` or manual workflow dispatch. See `.github/workflows/e2e-tests.yml`.

**⚠️ Important:** E2E tests create and destroy real Cloudflare resources. Always use a dedicated test account, never production infrastructure.
