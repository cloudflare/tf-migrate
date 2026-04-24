# Contributing to tf-migrate

Thank you for your interest in contributing. This document covers how to set up your development environment, add a new resource transformer, write tests, and submit changes.

## Table of Contents

1. [Ways to Contribute](#ways-to-contribute)
2. [Development Setup](#development-setup)
3. [Project Structure](#project-structure)
4. [Adding a New Resource Transformer](#adding-a-new-resource-transformer)
5. [Testing](#testing)
6. [Code Conventions](#code-conventions)
7. [Submitting a Pull Request](#submitting-a-pull-request)
8. [Commit Message Style](#commit-message-style)

---

## Ways to Contribute

- **Bug reports** — Open a [bug report](https://github.com/cloudflare/tf-migrate/issues/new?template=bug_report.yml). Include the tf-migrate version, the input `.tf` configuration, and the actual vs expected output.
- **Feature requests** — Open a [feature request](https://github.com/cloudflare/tf-migrate/issues/new?template=feature_request.yml).
- **New resource transformers** — The most common contribution. See [Adding a New Resource Transformer](#adding-a-new-resource-transformer) below.
- **Fixes to existing transformers** — Find the relevant `internal/resources/<name>/v4_to_v5.go`, add a regression test in `v4_to_v5_test.go`, and fix the logic.
- **Documentation** — Each resource has a `README.md` at `internal/resources/<name>/README.md`. User-facing messages are catalogued in `DIAGNOSTICS.md`.

---

## Development Setup

**Prerequisites:** Go 1.25+, Make, Terraform CLI

```bash
git clone https://github.com/cloudflare/tf-migrate
cd tf-migrate
go mod download
make build-all
```

This produces two binaries in `./bin/`:
- `tf-migrate` — the migration CLI
- `e2e` — the E2E test runner

---

## Project Structure

The codebase has three main areas:

**`internal/resources/`** — one directory per Cloudflare resource, each containing `v4_to_v5.go`, `v4_to_v5_test.go`, and `README.md`. This is where the migration logic lives.

**`internal/transform/hcl/`** — shared HCL manipulation helpers used by all transformers. If you need to rename an attribute, convert a block to an attribute, or generate a `moved {}` block, there is almost certainly a helper here for it.

**`integration/v4_to_v5/testdata/`** — integration test fixtures, one directory per resource, each with `input/` (v4 config) and `expected/` (expected v5 output) subdirectories.

---

## Adding a New Resource Transformer

### 1. Create the resource directory

Create `internal/resources/<name>/` using the v5 resource name without the `cloudflare_` prefix — for example `dns_record` or `zero_trust_access_application`.

### 2. Implement the transformer

Create `v4_to_v5.go` in that directory. Look at an existing simple transformer (e.g. `internal/resources/bot_management/v4_to_v5.go`) to understand the structure. The key points:

- Export a `NewV4ToV5Migrator()` function that creates the struct and calls `internal.RegisterMigrator(...)` to register it. Do not use `init()`.
- Implement all four methods of the `ResourceTransformer` interface: `CanHandle`, `TransformConfig`, `GetResourceType`, and `Preprocess`.
- If the resource is renamed in v5, implement `GetResourceRename()` — this enables automatic cross-file reference rewriting.
- If multiple v4 names map to the same v5 name, call `RegisterMigrator` multiple times and return all v4 names from `GetResourceRename()`.
- Register the new migrator by adding a call to `NewV4ToV5Migrator()` in `internal/registry/registry.go`.

Use the helpers in `internal/transform/hcl/` for all HCL manipulation — renaming attributes, converting blocks to attributes, generating `moved {}` and `import {}` blocks, and so on. Avoid manipulating raw HCL tokens directly.

### 3. Add unit tests

Create `v4_to_v5_test.go` in the same directory. Follow the pattern in any existing test file. Cover the main transformation cases and any edge cases (dynamic blocks, expression attributes, missing optional fields).

### 4. Add integration test fixtures

Create `integration/v4_to_v5/testdata/<name>/input/` with one or more `.tf` files representing realistic v4 configurations, and `integration/v4_to_v5/testdata/<name>/expected/` with the exact v5 output tf-migrate should produce.

All resource names in fixtures must use the `cftftest` prefix — for example `resource "cloudflare_dns_record" "cftftest_example"`. This is enforced by `make lint-testdata`.

### 5. Add an E2E test file

Create `integration/v4_to_v5/testdata/<name>/<name>_e2e.tf` with the Terraform configuration the E2E runner uses to create real infrastructure. If the E2E config needs to differ from the integration test input (for example, to avoid patterns that are valid in v4 but fail to apply in v5), use this file to provide a simpler, applyable subset.

For resources that must be imported rather than created, add a `# tf-migrate:import-address=` annotation on the line before the resource block. See `e2e/README.md` for details.

### 6. Add a resource README

Create `internal/resources/<name>/README.md` documenting what changed between v4 and v5, a before/after configuration example, and any manual steps required after migration.

### 7. Verify

```bash
make test
make lint-testdata
```

---

## Testing

There are three test layers:

**Unit tests** — fast, no I/O, test individual transformer logic:

```bash
make test-unit
```

**Integration tests** — run the full migration pipeline against fixture files in `integration/v4_to_v5/testdata/`. No Cloudflare credentials required:

```bash
make test-integration

# Single resource
TEST_RESOURCE=dns_record go test -v -run TestSingleResource ./integration/...
```

**E2E tests** — create and destroy real Cloudflare infrastructure. Use a dedicated test account; never run against production. Requires `CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_ZONE_ID`, `CLOUDFLARE_DOMAIN`, `CLOUDFLARE_API_KEY`, `CLOUDFLARE_EMAIL`, and R2 credentials for remote state. See `e2e/README.md` for the full setup and command reference.

---

## Code Conventions

- **Registration** — each resource registers itself via `NewV4ToV5Migrator()` calling `internal.RegisterMigrator(...)`. Do not edit the registry map directly.
- **HCL manipulation** — use helpers from `internal/transform/hcl/`. Do not manipulate raw HCL tokens unless no helper covers the case.
- **Warning comments** — use `tfhcl.AppendWarningComment(body, message)` to write `# MIGRATION WARNING: ...` into output `.tf` files. Document every such warning in `DIAGNOSTICS.md`.
- **Diagnostics** — append to `ctx.Diagnostics` using `hcl.DiagWarning` for issues requiring user action and `hcl.DiagError` for failures. Document them in `DIAGNOSTICS.md`.
- **Testdata naming** — all resource names in `integration/` testdata must use the `cftftest` prefix. Enforced by `make lint-testdata`.
- **No real credentials in testdata** — use placeholder account and zone IDs.

---

## Submitting a Pull Request

1. Fork the repository and create a branch from `main`.
2. Make your changes and ensure `make test` and `make lint-testdata` pass.
3. Open a pull request against `main` and fill in the template — describe which resource(s) are affected and which test layers you ran.
4. Link any related issues.

All pull requests require at least one approving review before merge.

---

## Commit Message Style

Use [Conventional Commits](https://www.conventionalcommits.org/):

| Prefix | When to use |
|--------|-------------|
| `feat:` | New resource transformer or new CLI feature |
| `fix:` | Bug fix in an existing transformer or handler |
| `docs:` | Documentation changes only |
| `test:` | Adding or updating tests with no production code change |
| `refactor:` | Internal refactoring with no behaviour change |
| `ci:` | Changes to GitHub Actions workflows or scripts |
| `chore:` | Dependency updates, build system changes |

GoReleaser uses these prefixes to group entries in the GitHub release changelog. Commits prefixed with `docs:`, `test:`, or `ci:` are excluded from release notes.
