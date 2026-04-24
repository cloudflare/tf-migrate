# Changelog

All notable changes to tf-migrate are documented here. For full release notes including binary download links, see the [GitHub Releases](https://github.com/cloudflare/tf-migrate/releases) page.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [v1.0.0] - 2026-04-24

First generally available release.

### Supported migration paths

- **v4 → v5**: 80+ Cloudflare Terraform Provider resource types. See the [supported resources table](README.md#supported-resources-v4--v5) for the complete list.
- **v5 → v5**: Bypass mode — generates `moved {}` blocks only, for provider-level schema moves within v5.

### Features

- Automatic `.tf` configuration transformation — resource renames, attribute changes, block restructuring, `moved {}` and `import {}` block generation
- Cross-file reference rewriting — updates resource type names, attribute references, and computed attribute references across all `.tf` files in the config directory
- Phased migration for `cloudflare_zone_settings_override` — handles state cleanup via `removed {}` blocks without requiring `terraform state rm`, compatible with Atlantis-managed workspaces
- Pre-migration preflight scan — classifies all resources before transformation, warns about unsupported types and conflicting `moved {}` blocks
- `verify-drift` command — reads `terraform plan` output and classifies each change as expected (known v4/v5 difference) or unexpected
- Dry-run mode (`--dry-run`) — previews all changes without modifying files
- Recursive directory support (`--recursive`) for module structures
- Verbose diagnostics (`--verbose`) and in-file `# MIGRATION WARNING` comments for changes requiring manual follow-up
- Automatic provider version update in `required_providers` (fetched from GitHub API, with `--target-provider-version` override for air-gapped or CI environments)
- Backup files created by default; disable with `--no-backup`
