# tf-migrate - Cloudflare Terraform Provider Migration Tool

A CLI tool for automatically migrating Terraform configurations between different versions of the Cloudflare Terraform Provider. State migration is handled by the provider's built-in `UpgradeState`/`MoveState` mechanisms — tf-migrate only transforms config (`.tf` files).

## Overview

`tf-migrate` helps you upgrade your Terraform infrastructure code by automatically transforming:
- **Configuration files** (`.tf`) — Updates resource types, attribute names, block structures, and generates import blocks for new v5 resources
- **Provider version** — Automatically updates `required_providers` to the latest v5 version and prints instructions to regenerate the lock file

> **Note:** tf-migrate does not modify Terraform state. State is upgraded automatically by the v5 provider on first `terraform apply` via its built-in `UpgradeState`/`MoveState` support.

### Supported Migration Paths

| Source Version | Target Version |
|----------------|----------------|
| Cloudflare Provider v4 | Cloudflare Provider v5 |

### Supported Resources (v4 → v5)

<details>
<summary>80+ resources with complete config migration support (click to expand)</summary>

| Product | v4 Resource Type | v5 Resource Type | Kind |
|---------|-----------------|-----------------|------|
| **Accounts** | `cloudflare_account` | `cloudflare_account` | resource |
| | `cloudflare_account_member` | `cloudflare_account_member` | resource |
| **Addressing** | `cloudflare_byo_ip_prefix` | `cloudflare_byo_ip_prefix` | resource ⚠ |
| | `cloudflare_regional_hostname` | `cloudflare_regional_hostname` | resource |
| | `cloudflare_regional_tiered_cache` | `cloudflare_regional_tiered_cache` | resource |
| **API Shield** | `cloudflare_api_shield` | `cloudflare_api_shield` | resource |
| | `cloudflare_api_shield_operation` | `cloudflare_api_shield_operation` | resource |
| **API Tokens** | `cloudflare_api_token` | `cloudflare_api_token` | resource |
| **Argo** | `cloudflare_argo` | `cloudflare_argo_smart_routing` / `cloudflare_argo_tiered_caching` | resource |
| **Bot Management** | `cloudflare_bot_management` | `cloudflare_bot_management` | resource |
| **Cache** | `cloudflare_tiered_cache` | `cloudflare_tiered_cache` | resource |
| **Certificate Packs** | `cloudflare_certificate_pack` | `cloudflare_certificate_pack` | resource |
| **Custom Hostnames** | `cloudflare_custom_hostname` | `cloudflare_custom_hostname` | resource |
| | `cloudflare_custom_hostname_fallback_origin` | `cloudflare_custom_hostname_fallback_origin` | resource |
| **Custom Pages** | `cloudflare_custom_pages` | `cloudflare_custom_pages` | resource |
| **Custom SSL** | `cloudflare_custom_ssl` | `cloudflare_custom_ssl` | resource |
| **DNS** | `cloudflare_record` | `cloudflare_dns_record` | resource |
| | `cloudflare_zone_dnssec` | `cloudflare_zone_dnssec` | resource |
| **Healthchecks** | `cloudflare_healthcheck` | `cloudflare_healthcheck` | resource |
| **IP Access Rules** | `cloudflare_access_rule` | `cloudflare_access_rule` | resource |
| **Leaked Credentials** | `cloudflare_leaked_credential_check` | `cloudflare_leaked_credential_check` | resource |
| | `cloudflare_leaked_credential_check_rule` | `cloudflare_leaked_credential_check_rule` | resource |
| **Lists** | `cloudflare_list` | `cloudflare_list` | resource ⚠ |
| | `cloudflare_list_item` | merged into `cloudflare_list` | resource ⚠ |
| **Load Balancers** | `cloudflare_load_balancer` | `cloudflare_load_balancer` | resource |
| | `cloudflare_load_balancer_monitor` | `cloudflare_load_balancer_monitor` | resource |
| | `cloudflare_load_balancer_pool` | `cloudflare_load_balancer_pool` | resource |
| | `data.cloudflare_load_balancer_pools` | `data.cloudflare_load_balancer_pools` | data source |
| **Logpush** | `cloudflare_logpull_retention` | `cloudflare_logpull_retention` | resource |
| | `cloudflare_logpush_job` | `cloudflare_logpush_job` | resource |
| | `cloudflare_logpush_ownership_challenge` | `cloudflare_logpush_ownership_challenge` | resource |
| **Managed Transforms** | `cloudflare_managed_headers` | `cloudflare_managed_transforms` | resource |
| **mTLS** | `cloudflare_mtls_certificate` | `cloudflare_mtls_certificate` | resource |
| **Notifications** | `cloudflare_notification_policy` | `cloudflare_notification_policy` | resource |
| | `cloudflare_notification_policy_webhooks` | `cloudflare_notification_policy_webhooks` | resource |
| **Observatory** | `cloudflare_observatory_scheduled_test` | `cloudflare_observatory_scheduled_test` | resource |
| **Origin CA** | `cloudflare_origin_ca_certificate` | `cloudflare_origin_ca_certificate` | resource |
| **Origin Pulls** | `cloudflare_authenticated_origin_pulls` | `cloudflare_authenticated_origin_pulls` | resource |
| | `cloudflare_authenticated_origin_pulls_certificate` | `cloudflare_authenticated_origin_pulls_certificate` | resource |
| **Page Rules** | `cloudflare_page_rule` | `cloudflare_page_rule` | resource |
| **Pages** | `cloudflare_pages_domain` | `cloudflare_pages_domain` | resource |
| | `cloudflare_pages_project` | `cloudflare_pages_project` | resource |
| **Queues** | `cloudflare_queue` | `cloudflare_queue` | resource |
| **R2** | `cloudflare_r2_bucket` | `cloudflare_r2_bucket` | resource |
| **Rulesets** | `cloudflare_ruleset` | `cloudflare_ruleset` | resource |
| | `data.cloudflare_rulesets` | `data.cloudflare_rulesets` | data source |
| **Snippets** | `cloudflare_snippet` | `cloudflare_snippet` | resource |
| | `cloudflare_snippet_rules` | `cloudflare_snippet_rules` | resource |
| **Spectrum** | `cloudflare_spectrum_application` | `cloudflare_spectrum_application` | resource |
| **Turnstile** | `cloudflare_turnstile_widget` | `cloudflare_turnstile_widget` | resource |
| **URL Normalization** | `cloudflare_url_normalization_settings` | `cloudflare_url_normalization_settings` | resource |
| **Workers** | `cloudflare_worker_script` / `cloudflare_workers_script` | `cloudflare_workers_script` | resource |
| | `cloudflare_worker_route` / `cloudflare_workers_route` | `cloudflare_worker_route` | resource |
| | `cloudflare_worker_domain` | `cloudflare_workers_custom_domain` | resource |
| | `cloudflare_workers_kv` | `cloudflare_workers_kv` | resource |
| | `cloudflare_workers_kv_namespace` | `cloudflare_workers_kv_namespace` | resource |
| | `cloudflare_workers_for_platforms_namespace` / `cloudflare_workers_for_platforms_dispatch_namespace` | `cloudflare_workers_for_platforms_dispatch_namespace` | resource |
| **Zero Trust** | `cloudflare_access_application` / `cloudflare_zero_trust_access_application` | `cloudflare_zero_trust_access_application` | resource |
| | `cloudflare_access_group` / `cloudflare_zero_trust_access_group` | `cloudflare_zero_trust_access_group` | resource ⚠ |
| | `cloudflare_access_identity_provider` / `cloudflare_zero_trust_access_identity_provider` | `cloudflare_zero_trust_access_identity_provider` | resource |
| | `cloudflare_access_mutual_tls_certificate` / `cloudflare_zero_trust_access_mtls_certificate` | `cloudflare_zero_trust_access_mtls_certificate` | resource |
| | `cloudflare_zero_trust_access_mtls_hostname_settings` | `cloudflare_zero_trust_access_mtls_hostname_settings` | resource |
| | `cloudflare_access_policy` | `cloudflare_zero_trust_access_policy` | resource ⚠ |
| | `cloudflare_access_service_token` / `cloudflare_zero_trust_access_service_token` | `cloudflare_zero_trust_access_service_token` | resource |
| | `cloudflare_access_organization` / `cloudflare_zero_trust_access_organization` | `cloudflare_zero_trust_organization` | resource |
| | `cloudflare_device_managed_networks` / `cloudflare_zero_trust_device_managed_networks` | `cloudflare_zero_trust_device_managed_networks` | resource |
| | `cloudflare_device_posture_integration` / `cloudflare_zero_trust_device_posture_integration` | `cloudflare_zero_trust_device_posture_integration` | resource |
| | `cloudflare_device_posture_rule` / `cloudflare_zero_trust_device_posture_rule` | `cloudflare_zero_trust_device_posture_rule` | resource |
| | `cloudflare_device_settings_policy` / `cloudflare_zero_trust_device_profiles` | `cloudflare_zero_trust_device_profiles` | resource |
| | `cloudflare_device_dex_test` / `cloudflare_zero_trust_dex_test` | `cloudflare_zero_trust_dex_test` | resource |
| | `cloudflare_dlp_profile` / `cloudflare_zero_trust_dlp_profile` | `cloudflare_zero_trust_dlp_custom_profile` | resource |
| | `cloudflare_zero_trust_dlp_predefined_profile` | `cloudflare_zero_trust_dlp_predefined_profile` | resource |
| | `cloudflare_zero_trust_gateway_certificate` | `cloudflare_zero_trust_gateway_certificate` | resource |
| | `cloudflare_teams_rule` | `cloudflare_zero_trust_gateway_policy` | resource |
| | `cloudflare_teams_account` / `cloudflare_zero_trust_gateway_settings` | `cloudflare_zero_trust_gateway_settings` | resource |
| | `cloudflare_teams_list` | `cloudflare_zero_trust_list` | resource |
| | `cloudflare_fallback_domain` / `cloudflare_zero_trust_local_fallback_domain` | `cloudflare_zero_trust_local_fallback_domain` | resource |
| | `cloudflare_split_tunnel` / `cloudflare_zero_trust_split_tunnel` | `cloudflare_zero_trust_split_tunnel` | resource |
| | `cloudflare_tunnel` / `cloudflare_zero_trust_tunnel_cloudflared` | `cloudflare_zero_trust_tunnel_cloudflared` | resource |
| | `cloudflare_tunnel_config` / `cloudflare_zero_trust_tunnel_cloudflared_config` | `cloudflare_zero_trust_tunnel_cloudflared_config` | resource |
| | `cloudflare_tunnel_route` / `cloudflare_zero_trust_tunnel_route` | `cloudflare_zero_trust_tunnel_cloudflared_route` | resource |
| | `cloudflare_tunnel_virtual_network` / `cloudflare_zero_trust_tunnel_virtual_network` | `cloudflare_zero_trust_tunnel_cloudflared_virtual_network` | resource |
| **Zones** | `cloudflare_zone` | `cloudflare_zone` | resource |
| | `cloudflare_zone_settings_override` | `cloudflare_zone_setting` (one per setting) | resource |
| | `cloudflare_zone_subscription` | `cloudflare_zone_subscription` | resource |
| | `data.cloudflare_zone` | `data.cloudflare_zone` | data source |
| | `data.cloudflare_zones` | `data.cloudflare_zones` | data source |
| | `data.cloudflare_accounts` | `data.cloudflare_accounts` | data source |

⚠ Resources marked with this symbol require manual steps after migration. See [Manual Migration Steps](#manual-migration-steps) below.

</details>

## Installation

### Pre-built Binaries

Download the latest release from the [GitHub Releases](https://github.com/cloudflare/tf-migrate/releases) page. Binaries are available for Linux, macOS, Windows, and FreeBSD on both amd64 and arm64.

### Building from Source

```bash
git clone https://github.com/cloudflare/tf-migrate
cd tf-migrate
make
# Binary is available at ./bin/tf-migrate
```

**Requirements:** Go 1.25+, Make

## Usage

### Basic Migration

Migrate all Terraform files in the current directory:

```bash
tf-migrate migrate --source-version v4 --target-version v5
```

### Migrate a Specific Directory

```bash
tf-migrate migrate --config-dir ./terraform --source-version v4 --target-version v5
```

### Dry Run (Preview Only)

Preview changes without modifying any files:

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

### Output to a Different Directory

```bash
tf-migrate migrate \
  --config-dir ./terraform \
  --output-dir ./terraform-v5 \
  --source-version v4 \
  --target-version v5
```

### Recursive Migration (Module Structures)

```bash
tf-migrate migrate --recursive --source-version v4 --target-version v5
```

### Verbose Output

Show per-file progress, rename tables, and cross-file reference details:

```bash
tf-migrate migrate -v --source-version v4 --target-version v5
```

## What tf-migrate Does Automatically

After a successful migration, tf-migrate:

1. **Transforms all `.tf` files** — resource renames, attribute changes, block restructuring, `moved {}` blocks, `import {}` blocks
2. **Updates the provider version** in `required_providers` to the latest v5 release (fetched from GitHub, falls back to a known-good version)
3. **Prints next-step instructions** for regenerating the lock file:
   ```
   terraform init -upgrade -backend=false
   ```
4. **Handles `zone_settings_override` automatically** via phased migration — see [Phased Migration](#phased-migration-zone_settings_override) below

## Phased Migration (`zone_settings_override`)

`cloudflare_zone_settings_override` has no schema in the v5 provider. In Atlantis-managed workspaces, `terraform state rm` is typically disabled for safety, making manual state cleanup impossible.

tf-migrate handles this automatically in two phases:

### Phase 1 (first run — detected automatically)

tf-migrate detects `zone_settings_override` resources and modifies the `.tf` file in-place:
- **Comments out** each `cloudflare_zone_settings_override` resource block with a `# tf-migrate: ` prefix
- **Appends** a `removed { lifecycle { destroy = false } }` block after each commented block

Terraform only sees the `removed {}` blocks (the resource is a comment), so there is no coexistence error. The v4 provider processes the `removed {}` blocks on `terraform plan`/`apply`, dropping the state entries without touching any infrastructure.

```hcl
# tf-migrate: resource "cloudflare_zone_settings_override" "example" {
# tf-migrate:   zone_id = var.zone_id
# tf-migrate:   settings {
# tf-migrate:     always_online = "on"
# tf-migrate:   }
# tf-migrate: }

removed {
  from = cloudflare_zone_settings_override.example
  lifecycle {
    destroy = false
  }
}
```

**User workflow:**
1. Commit and push the modified `.tf` files
2. Atlantis plans and applies using the **current (v4) provider** — `removed {}` blocks drop the state entries
3. Re-run `tf-migrate migrate` in the same directory

### Phase 2 (second run — detected automatically)

tf-migrate detects the commented-out blocks and asks:

```
Did you apply the v4 config and remove the resources from state? [y/N]:
```

On confirmation:
- Uncomments the resource blocks (strips `# tf-migrate: ` prefix)
- Removes the `removed {}` blocks
- Runs the full v4→v5 migration

### No `zone_settings_override`?

If your workspace doesn't use `cloudflare_zone_settings_override`, nothing changes — single-pass migration as normal.

---

## Manual Migration Steps

tf-migrate automates the vast majority of migrations, but some resources cannot be fully migrated automatically. When manual intervention is required, tf-migrate:

1. **Prints a warning to stdout** during the migration run describing what needs to be done and where to find the required values.
2. **Leaves a `# MIGRATION WARNING:` comment** directly in the affected `.tf` file at the exact location that needs attention.

Example warning in a migrated file:

```hcl
resource "cloudflare_byo_ip_prefix" "example" {
  account_id = var.account_id
  # MIGRATION WARNING: This resource requires manual intervention to add v5
  # required fields 'asn' and 'cidr'. Find values in Cloudflare Dashboard →
  # Manage Account → IP Addresses → IP Prefixes. See migration documentation
  # for details.
}
```

Search your migrated files for `MIGRATION WARNING` to find all locations that need attention:

```bash
grep -rn "MIGRATION WARNING" ./terraform
```

For full details on what changed between v4 and v5, refer to the [Cloudflare Provider v5 Migration Guide](https://github.com/cloudflare/terraform-provider-cloudflare/blob/main/docs/guides/version-5-migration.md).

## Verifying Drift After Migration

After running `tf-migrate migrate` and switching to the v5 provider, run `terraform plan` to see what changes Terraform detects. Some changes are **expected** — known, safe differences between how v4 and v5 providers represent state. Others are **unexpected** and require attention before applying.

`tf-migrate verify-drift` reads your plan output and classifies each change.

### Workflow

```bash
# 1. Migrate your configuration
tf-migrate migrate --source-version v4 --target-version v5

# 2. Initialize the v5 provider
terraform init -upgrade -backend=false

# 3. Capture the plan output
terraform plan > plan.txt

# 4. Verify the drift
tf-migrate verify-drift --file plan.txt
```

### Example Output

**All drift is expected (exit code 0):**

```
Cloudflare Terraform Migration - Drift Verification
====================================================
Plan file:          plan.txt
Resources detected: dns_record, zone_setting

✓ Exempted Changes  (3 change(s) are expected and safe to ignore)
────────────────────────────────────────────────────
Rule: computed_value_refreshes
  "Ignore attributes that refresh to 'known after apply'"
  module.dns_record.cloudflare_dns_record.example:
    ~ ttl = (known after apply)

✓ No unexpected drift
────────────────────────────────────────────────────

====================================================
Result: ✓ MIGRATION LOOKS GOOD
  No unexpected drift detected
```

**Unexpected drift found (exit code 1):**

```
Cloudflare Terraform Migration - Drift Verification
====================================================
Plan file:          plan.txt
Resources detected: dns_record

✓ No exempted changes
────────────────────────────────────────────────────

✗ Unexpected Drift  (1 change(s) require attention)
────────────────────────────────────────────────────
  module.dns_record.cloudflare_dns_record.example:
    ~ value = "old-value" -> "new-value"

====================================================
Result: MIGRATION NEEDS ATTENTION
  1 unexpected change(s) require review
```

### Understanding the Results

| Section | Meaning |
|---------|---------|
| **Exempted Changes** | Known, safe differences between v4 and v5 providers. These do not require action — they will stabilise on the next `terraform apply`. |
| **Unexpected Drift** | Changes not accounted for by any known exemption. Review each one before applying. |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All drift is expected, or no changes detected. Safe to proceed. |
| `1` | Unexpected drift found. Review the output before running `terraform apply`. |

Use in CI pipelines:

```bash
terraform plan > plan.txt
tf-migrate verify-drift --file plan.txt || exit 1
```

---

## Migrating an Atlantis-Managed Workspace

This section covers migrating a Terraform workspace that uses [Atlantis](https://www.runatlantis.io/) for CI/CD. The key constraints are:

- **You cannot run `terraform init -upgrade` in Atlantis** — Atlantis runs its own `terraform init` as part of its workflow and does not support remote flags.
- **The remote state backend is typically unreachable from your local machine** — running `terraform init` locally will fail if the backend requires network access inside a private environment.
- **The `.terraform.lock.hcl` must be committed with v5 hashes** — Atlantis uses the lock file as-is and will not upgrade providers on its own.

### Prerequisites

- `tf-migrate` binary (see [Installation](#installation))
- Terraform CLI installed locally (same version as used in Atlantis — check `terraform_version` in `atlantis.yaml`)
- Access to the Git repository

### Step-by-step

#### 1. Run tf-migrate against your config directory

```bash
tf-migrate migrate \
  --config-dir ./tf/your-workspace \
  --source-version v4 \
  --target-version v5 \
  --no-backup
```

tf-migrate transforms all `.tf` files in-place and prints a summary of:
- Resources renamed
- Cross-file references updated
- Warnings for anything requiring manual follow-up
- Provider version updated (with lock file regeneration instructions)

Review the warnings carefully before proceeding.

#### 2. Handle `zone_settings_override` (if applicable)

If your workspace uses `cloudflare_zone_settings_override`, tf-migrate automatically enters **phased migration** (see [Phased Migration](#phased-migration-zone_settings_override)):

- On the **first run**, tf-migrate comments out the resource blocks and adds `removed {}` blocks
- Commit and push — Atlantis applies with the v4 provider, dropping the state entries
- On the **second run**, tf-migrate detects the commented blocks, prompts for confirmation, and completes the full migration

No `terraform state rm` required.

#### 3. Regenerate `.terraform.lock.hcl` locally with `-backend=false`

tf-migrate prints this instruction at the end of every migration. Because the remote backend is not reachable locally, use `-backend=false` to skip it:

```bash
cd ./tf/your-workspace
terraform init -upgrade -backend=false
```

> **Why `-backend=false`?** Without it, Terraform tries to initialise the remote backend, which fails if the backend endpoint is only reachable from within your CI environment.

#### 4. Commit everything together

```bash
git add tf/your-workspace/
git commit -m "migrate your-workspace from cloudflare provider v4 to v5"
```

The commit should include:
- All modified `.tf` files
- Updated `main.tf` with the new provider version (updated automatically by tf-migrate)
- Updated `.terraform.lock.hcl` with v5 hashes

#### 5. Push and let Atlantis plan

Atlantis will pick up the committed lock file, download the v5 provider, and run `terraform plan`. The plan output will reflect the migrated v5 config.

---

### Why the lock file matters

Atlantis uses the committed `.terraform.lock.hcl` to pin provider versions. If the lock file still references the old v4 provider, Atlantis will download and use the v4 provider even if `main.tf` specifies a v5 version — causing the entire plan to fail with schema errors like:

```
Error: Invalid resource type
  The provider cloudflare/cloudflare does not support resource type "cloudflare_dns_record".
```

This is the most common mistake when migrating an Atlantis workspace. Always regenerate and commit the lock file as part of the migration.

---

### Summary

| Step | Where | What |
|------|-------|------|
| 1. Run tf-migrate | Local | Transform `.tf` files, update provider version |
| 2. Phase 1 (if `zone_settings_override`) | Local → Git → Atlantis | Commit commented blocks, Atlantis drops state entries |
| 3. Run tf-migrate again (if phase 1 ran) | Local | Confirm apply succeeded, complete full v5 migration |
| 4. `terraform init -upgrade -backend=false` | Local | Regenerate lock file with v5 hashes |
| 5. Commit all changes | Local | `.tf` files + `main.tf` + `.terraform.lock.hcl` |
| 6. Push | Git | Atlantis picks up the v5 lock file and plans |

---

## Command Reference

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config-dir` | Current directory | Directory containing Terraform configuration files |
| `--source-version` | `v4` | Source provider version (e.g., `v4`) |
| `--target-version` | `v5` | Target provider version (e.g., `v5`) |
| `--resources` | All resources | Comma-separated list of resources to migrate |
| `--dry-run` | `false` | Preview changes without modifying files |
| `--log-level` | `warn` | Log level: `debug`, `info`, `warn`, `error`, `off` |

### `migrate` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output-dir` | In-place | Output directory for migrated files |
| `--backup` | `true` | Create backups of original files before migration |
| `--no-backup` | `false` | Skip creating backup files (alias for `--backup=false`) |
| `--recursive` | `false` | Recursively process subdirectories |
| `--skip-phase-check` | `false` | Skip the phased migration confirmation prompt and run the full migration directly (for CI/non-interactive use) |
| `-v` / `--verbose` | `false` | Show verbose output: per-file progress, rename tables, and all diagnostics |
| `-q` / `--quiet` | `false` | Suppress warnings, only show errors |

### `verify-drift` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--file` | Required | Path to `terraform plan` output file |
