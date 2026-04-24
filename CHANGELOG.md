# Changelog

All notable changes to tf-migrate are documented here. For full release notes including binary download links, see the [GitHub Releases](https://github.com/cloudflare/tf-migrate/releases) page.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [v1.0.0] - 2026-04-24

First generally available release. Includes all changes from v1.0.0-beta.1 through v1.0.0-beta.12.

### Supported Migration Paths

- **v4 → v5**: 80+ Cloudflare Terraform Provider resource types. See the [supported resources table](README.md#supported-resources-v4--v5) for the complete list.
- **v5 → v5**: Bypass mode — generates `moved {}` blocks only, for provider-level schema moves within v5.

### Highlights

- Automatic `.tf` configuration transformation — resource renames, attribute changes, block restructuring, `moved {}` / `import {}` / `removed {}` block generation
- Cross-file reference rewriting — updates resource type names, attribute references, and computed attribute references across all `.tf` files in the config directory
- Phased migration for `cloudflare_zone_settings_override` — handles state cleanup via `removed {}` blocks without requiring `terraform state rm`, compatible with Atlantis-managed workspaces
- Pre-migration preflight scan — classifies all resources before transformation, warns about unsupported types and conflicting `moved {}` blocks
- `verify-drift` command — reads `terraform plan` output and classifies each change as expected or unexpected using embedded drift exemptions
- Dry-run mode (`--dry-run`) — previews all changes without modifying files
- Recursive directory support (`--recursive`) for module structures
- Verbose diagnostics (`--verbose`) and in-file `# MIGRATION WARNING` comments for changes requiring manual follow-up
- Automatic provider version update in `required_providers` (fetched from GitHub API, with `--target-provider-version` override for air-gapped or CI environments)
- Minimum provider version enforcement — blocks migration if v4 provider is below v4.52.5
- Backup files created by default; disable with `--no-backup`
- E2E test infrastructure — full lifecycle tests against real Cloudflare infrastructure with R2-backed remote state

---

## [v1.0.0-beta.12] - 2026-04-24

### Added

- Issue templates for bug reports, feature requests, and external links ([#284](https://github.com/cloudflare/tf-migrate/pull/284))
- Semgrep OSS scanning CI workflow ([#283](https://github.com/cloudflare/tf-migrate/pull/283))
- Drift exemptions for `pages_project` build image version and `env_vars` drift ([#278](https://github.com/cloudflare/tf-migrate/pull/278))

### Fixed

- **zero_trust_local_fallback_domain**: convert dynamic `domains` blocks to for-expressions ([#282](https://github.com/cloudflare/tf-migrate/pull/282))
- **zero_trust_list**: migrate `items_with_description` written as attribute ([#281](https://github.com/cloudflare/tf-migrate/pull/281))
- **zero_trust_list**: handle comments, blank lines, for-expressions, and references in items array ([#279](https://github.com/cloudflare/tf-migrate/pull/279))
- **split_tunnel**: consolidate repeated warnings into a single actionable diagnostic ([#280](https://github.com/cloudflare/tf-migrate/pull/280))
- **ruleset**: convert `action_parameters.rules` map values from strings to lists ([#277](https://github.com/cloudflare/tf-migrate/pull/277))

---

## [v1.0.0-beta.11] - 2026-04-15

### Added

- Computed attribute rewrite support for cross-file references (e.g., `.hostname` → `.name`) ([#266](https://github.com/cloudflare/tf-migrate/pull/266))
- `cloudflare_worker_secret` migration support ([#265](https://github.com/cloudflare/tf-migrate/pull/265))
- Atlantis-friendly preflight checks — scan and classify all resources before transforming ([#263](https://github.com/cloudflare/tf-migrate/pull/263))
- `--exclude` flag for skipping specific resources; changed `--recursive` default to `false` ([#271](https://github.com/cloudflare/tf-migrate/pull/271))
- Consolidated duplicate warnings and moved informational logs behind `--verbose` flag ([#272](https://github.com/cloudflare/tf-migrate/pull/272))

### Fixed

- **api_token**: sort `permission_groups` to match v5 provider canonical ordering ([#276](https://github.com/cloudflare/tf-migrate/pull/276))
- **zero_trust_access_application**: filter type-gated attributes ([#267](https://github.com/cloudflare/tf-migrate/pull/267))
- **zero_trust_access_application**: fix cross-file reference handling ([#270](https://github.com/cloudflare/tf-migrate/pull/270))
- `session_duration` bug fix and additional drift exemptions ([#264](https://github.com/cloudflare/tf-migrate/pull/264))
- E2E test fixes and minor bug fixes ([#268](https://github.com/cloudflare/tf-migrate/pull/268), [#273](https://github.com/cloudflare/tf-migrate/pull/273))

---

## [v1.0.0-beta.10] - 2026-04-02

No changes — re-tagged from beta.9.

---

## [v1.0.0-beta.9] - 2026-04-02

### Added

- Minimum provider version check — blocks v4→v5 migration if provider is below v4.52.5 ([#260](https://github.com/cloudflare/tf-migrate/pull/260))

### Fixed

- Audit schema for edge cases and nested fields across multiple resources ([#258](https://github.com/cloudflare/tf-migrate/pull/258))
- **zero_trust_list**: handle resources already using v5 name in v4 ([#257](https://github.com/cloudflare/tf-migrate/pull/257))
- **zero_trust_access_policy**: additional test cases for v5 name handling ([#259](https://github.com/cloudflare/tf-migrate/pull/259), [#261](https://github.com/cloudflare/tf-migrate/pull/261))
- Migration UX improvements — better output formatting, clearer error messages ([#254](https://github.com/cloudflare/tf-migrate/pull/254))
- Bugs found during local testing of full migration workflow ([#253](https://github.com/cloudflare/tf-migrate/pull/253), [#256](https://github.com/cloudflare/tf-migrate/pull/256))
- E2E runner: resolve `<zone_id>` placeholder in hoisted import blocks ([#255](https://github.com/cloudflare/tf-migrate/pull/255))

---

## [v1.0.0-beta.8] - 2026-03-27

### Fixed

- **zero_trust_access_group**: fix HCL traversal of nested include/require/exclude blocks ([#252](https://github.com/cloudflare/tf-migrate/pull/252))

---

## [v1.0.0-beta.7] - 2026-03-26

### Fixed

- **worker_route**: fix script name reference in migrated config ([#250](https://github.com/cloudflare/tf-migrate/pull/250))
- CI: fix lint and test pipeline ([#251](https://github.com/cloudflare/tf-migrate/pull/251))

---

## [v1.0.0-beta.6] - 2026-03-26

### Added

- **verify-drift** subcommand — reads `terraform plan` output and classifies drift as expected or unexpected ([#236](https://github.com/cloudflare/tf-migrate/pull/236))
- Phased migration for `cloudflare_zone_settings_override` — comments out resources and generates `removed {}` blocks, eliminating the need for `terraform state rm` ([#246](https://github.com/cloudflare/tf-migrate/pull/246))
- V5 E2E test runner for post-migration validation ([#247](https://github.com/cloudflare/tf-migrate/pull/247))
- Support for multiple v4 names in `GetResourceRename` ([#231](https://github.com/cloudflare/tf-migrate/pull/231))
- Drift exemptions for identity provider `conditional_access_enabled` null handling ([#239](https://github.com/cloudflare/tf-migrate/pull/239))

### Changed

- Removed `TransformState` pipeline — state migration is now fully provider-driven via `UpgradeState`/`MoveState` ([#241](https://github.com/cloudflare/tf-migrate/pull/241))
- Replaced `StateTypeRenames` side-channel with `PerInstanceTypeRouter` interface ([#237](https://github.com/cloudflare/tf-migrate/pull/237))

### Fixed

- **access_policy**: fix migration bugs with `cloudflare_access_policy` and AOP certificate handling ([#248](https://github.com/cloudflare/tf-migrate/pull/248))
- **load_balancer, healthcheck, zero_trust_gateway_settings**: address migration coverage gaps ([#238](https://github.com/cloudflare/tf-migrate/pull/238))
- **zero_trust_gateway_certificate**: fix migration logic ([#240](https://github.com/cloudflare/tf-migrate/pull/240))
- Research team migration issues (TKT-001, TKT-004) — real-world migration bug fixes ([#249](https://github.com/cloudflare/tf-migrate/pull/249))
- Extract `profile_id` from import-address annotation for predefined DLP profiles ([#245](https://github.com/cloudflare/tf-migrate/pull/245))

---

## [v1.0.0-beta.5] - 2026-03-13

### Added

- **byo_ip_prefix**: migration support with manual intervention warnings for `asn` and `cidr` fields ([#131](https://github.com/cloudflare/tf-migrate/pull/131))

---

## [v1.0.0-beta.4] - 2026-03-13

### Added

- Drift exemptions system — classify expected v4/v5 differences to avoid false failures in E2E drift checks ([#232](https://github.com/cloudflare/tf-migrate/pull/232))
- **account**: migration support ([#230](https://github.com/cloudflare/tf-migrate/pull/230))
- **observatory_scheduled_test**: migration and state upgrader support ([#226](https://github.com/cloudflare/tf-migrate/pull/226))
- E2E runner: CLI argument improvements and drift report updates ([#233](https://github.com/cloudflare/tf-migrate/pull/233))

### Fixed

- Clean tfstate files, add state upgrader checks, fix resource type handling ([#229](https://github.com/cloudflare/tf-migrate/pull/229))

---

## [v1.0.0-beta.3] - 2026-03-10

### Added

- E2E phase 2 runner ([#228](https://github.com/cloudflare/tf-migrate/pull/228))
- **custom_ssl**: v4→v5 migration
- **workers_custom_domain**: config migration and E2E tests ([#222](https://github.com/cloudflare/tf-migrate/pull/222))

### Changed

- Converted multiple resources to use provider-side state upgraders instead of tf-migrate state transforms:
  - `pages_domain` ([#144](https://github.com/cloudflare/tf-migrate/pull/144))
  - `authenticated_origin_pulls` ([#227](https://github.com/cloudflare/tf-migrate/pull/227))
  - `custom_hostname` ([#217](https://github.com/cloudflare/tf-migrate/pull/217))
  - `zero_trust_organization` ([#216](https://github.com/cloudflare/tf-migrate/pull/216))
  - `zero_trust_tunnel_cloudflared_virtual_network` ([#207](https://github.com/cloudflare/tf-migrate/pull/207))
  - `zero_trust_gateway_certificate`, `mtls_certificate`, `zero_trust_device_posture_integration`, `zero_trust_device_profiles`, `zero_trust_local_fallback_domain`, `zero_trust_gateway_settings`
  - `account_member` ([#219](https://github.com/cloudflare/tf-migrate/pull/219)), `leaked_credential_check` ([#220](https://github.com/cloudflare/tf-migrate/pull/220)), `leaked_credential_check_rule` ([#223](https://github.com/cloudflare/tf-migrate/pull/223)), `logpush_ownership_challenge`
  - `zero_trust_split_tunnel`: removed state migration

---

## [v1.0.0-beta.2] - 2026-03-03

### Changed

- Removed API client dependency — tf-migrate no longer calls the Cloudflare API ([#215](https://github.com/cloudflare/tf-migrate/pull/215))

---

## [v1.0.0-beta.1] - 2026-03-02

Initial beta release.

### Added

- Config and state processing framework ([#1](https://github.com/cloudflare/tf-migrate/pull/1))
- Support for migrating between arbitrary provider versions ([#3](https://github.com/cloudflare/tf-migrate/pull/3))
- E2E test system with Cloudflare infrastructure lifecycle management ([#23](https://github.com/cloudflare/tf-migrate/pull/23))
- Integration test infrastructure with testdata fixtures
- CI: parallel E2E tests, post-merge-only execution ([#51](https://github.com/cloudflare/tf-migrate/pull/51), [#55](https://github.com/cloudflare/tf-migrate/pull/55))

### Resource Migrations

Initial v4→v5 migration support for:

- `cloudflare_record` → `cloudflare_dns_record` ([#4](https://github.com/cloudflare/tf-migrate/pull/4))
- `cloudflare_api_token` ([#6](https://github.com/cloudflare/tf-migrate/pull/6))
- `cloudflare_account_member` ([#9](https://github.com/cloudflare/tf-migrate/pull/9))
- `cloudflare_teams_list` → `cloudflare_zero_trust_list` ([#10](https://github.com/cloudflare/tf-migrate/pull/10))
- `cloudflare_access_service_token` → `cloudflare_zero_trust_access_service_token` ([#11](https://github.com/cloudflare/tf-migrate/pull/11))
- `cloudflare_logpull_retention` ([#12](https://github.com/cloudflare/tf-migrate/pull/12))
- `cloudflare_workers_kv_namespace` ([#14](https://github.com/cloudflare/tf-migrate/pull/14))
- `cloudflare_teams_rule` → `cloudflare_zero_trust_gateway_policy` ([#13](https://github.com/cloudflare/tf-migrate/pull/13))
- `cloudflare_dlp_profile` → `cloudflare_zero_trust_dlp_custom_profile` ([#17](https://github.com/cloudflare/tf-migrate/pull/17))
- `cloudflare_workers_kv` ([#20](https://github.com/cloudflare/tf-migrate/pull/20))
- `cloudflare_zone_dnssec` ([#8](https://github.com/cloudflare/tf-migrate/pull/8))
- `cloudflare_r2_bucket` ([#22](https://github.com/cloudflare/tf-migrate/pull/22))
- `cloudflare_notification_policy_webhooks` ([#24](https://github.com/cloudflare/tf-migrate/pull/24))
- `cloudflare_tunnel_route` → `cloudflare_zero_trust_tunnel_cloudflared_route` ([#15](https://github.com/cloudflare/tf-migrate/pull/15), [#28](https://github.com/cloudflare/tf-migrate/pull/28))
- `cloudflare_device_posture_rule` → `cloudflare_zero_trust_device_posture_rule`
- `cloudflare_zone` ([#34](https://github.com/cloudflare/tf-migrate/pull/34))
- `data.cloudflare_zone`, `data.cloudflare_zones` ([#40](https://github.com/cloudflare/tf-migrate/pull/40))
- `cloudflare_logpush_job` ([#16](https://github.com/cloudflare/tf-migrate/pull/16))
- `cloudflare_pages_project` ([#48](https://github.com/cloudflare/tf-migrate/pull/48))
- `cloudflare_healthcheck` ([#46](https://github.com/cloudflare/tf-migrate/pull/46))
- `cloudflare_managed_headers` → `cloudflare_managed_transforms` ([#31](https://github.com/cloudflare/tf-migrate/pull/31))
- Plus many more resources included in the initial release batch

---

[v1.0.0]: https://github.com/cloudflare/tf-migrate/releases/tag/v1.0.0
[v1.0.0-beta.12]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.11...v1.0.0-beta.12
[v1.0.0-beta.11]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.10...v1.0.0-beta.11
[v1.0.0-beta.10]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.9...v1.0.0-beta.10
[v1.0.0-beta.9]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.8...v1.0.0-beta.9
[v1.0.0-beta.8]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.7...v1.0.0-beta.8
[v1.0.0-beta.7]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.6...v1.0.0-beta.7
[v1.0.0-beta.6]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.5...v1.0.0-beta.6
[v1.0.0-beta.5]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.4...v1.0.0-beta.5
[v1.0.0-beta.4]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.3...v1.0.0-beta.4
[v1.0.0-beta.3]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.2...v1.0.0-beta.3
[v1.0.0-beta.2]: https://github.com/cloudflare/tf-migrate/compare/v1.0.0-beta.1...v1.0.0-beta.2
[v1.0.0-beta.1]: https://github.com/cloudflare/tf-migrate/releases/tag/v1.0.0-beta.1
