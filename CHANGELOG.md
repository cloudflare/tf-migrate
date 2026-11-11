# Changelog

All notable changes to tf-migrate are documented here. For full release notes including binary download links, see the [GitHub Releases](https://github.com/cloudflare/tf-migrate/releases) page.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [v1.2.0] - 2026-11-10

### Added

- `preserve-zero-trust-settings` command: copies Zero Trust settings that exist only in the API, not in v4 config or state, from a saved API snapshot into the migrated v5 config. Covers organization MFA/PIV and deny-unmatched, Gateway host selection, sandbox and emergency signals, Gateway `block_page` `mode` / `include_context` / `suppress_footer`, and DLP predefined profile `confidence_threshold` / `ai_context_enabled`. Existing HCL always wins; the command does not call the Cloudflare API or touch state ([#336](https://github.com/cloudflare/tf-migrate/pull/336), [#340](https://github.com/cloudflare/tf-migrate/pull/340), [#344](https://github.com/cloudflare/tf-migrate/pull/344))

### Fixed

- **ruleset**: `dynamic` blocks at any depth in a rule, including `action_parameters`, `ratelimit`, `cache_key`, `query_string`, `categories` and `headers`, are converted to `for` expressions with the same rewrites as static blocks. Static and dynamic blocks of one type are all kept, in source order ([#349](https://github.com/cloudflare/tf-migrate/pull/349), [#351](https://github.com/cloudflare/tf-migrate/pull/351))
- **managed_transforms**, **notification_policy**: `dynamic` blocks were left as v4 syntax or written out as an invalid `dynamic = {}` attribute; they are now converted to `for` expressions, or reported with a warning when that can't be done faithfully ([#346](https://github.com/cloudflare/tf-migrate/pull/346))
- **zone_setting**: `prevent_destroy` and `create_before_destroy` are kept when `lifecycle` also has `ignore_changes` ([#346](https://github.com/cloudflare/tf-migrate/pull/346))
- **zero_trust_access_identity_provider**: with several `config` blocks, the first (the one v4 actually used) is kept instead of the last ([#346](https://github.com/cloudflare/tf-migrate/pull/346))
- **ip_ranges** data source: references to `ipv4_cidr_blocks` / `ipv6_cidr_blocks` are rewritten to `ipv4_cidrs` / `ipv6_cidrs` ([#346](https://github.com/cloudflare/tf-migrate/pull/346))
- Values that were dropped silently now produce a warning: `zone.plan`, the `universal_ssl` zone setting, `logpush_job` `kind = "instant-logs"`, notification integrations without an `id`, zone settings under `count`, and `tiered_cache` with an expression `cache_type` ([#346](https://github.com/cloudflare/tf-migrate/pull/346))
- **tiered_cache**: an `import {}` block is generated for the derived `cloudflare_argo_tiered_caching` resource, which was planned as a create ([#346](https://github.com/cloudflare/tf-migrate/pull/346))
- **zone** data source: `.zone_id` references were rewritten to the nonexistent `.name_id`; they now become `.id`. Configs already migrated with v1.1.0 or v1.1.1 should replace `.name_id` with `.id` ([#350](https://github.com/cloudflare/tf-migrate/pull/350))
- **zero_trust_tunnel_cloudflared**: no longer writes `config_src = "local"` when it wasn't set in v4. It forced a replacement of tunnels managed with `cloudflare_tunnel_config` ([#343](https://github.com/cloudflare/tf-migrate/pull/343))
- **zero_trust_dlp_predefined_profile**: `enabled_entries` is kept for expression, conditional and `dynamic "entry"` forms ([#344](https://github.com/cloudflare/tf-migrate/pull/344))
- **zero_trust_device_profiles**, **spectrum_application**: expressions in device profile `precedence` and Spectrum `origin_port_range` bounds are no longer flattened to a literal taken from their first token. Device profiles with expression `precedence` had all collapsed to one value ([#341](https://github.com/cloudflare/tf-migrate/pull/341))
- **zone_dnssec**: writes `status = "active"` when v4 had none, so migrated configs don't plan `status -> null` ([#342](https://github.com/cloudflare/tf-migrate/pull/342))
- **page_rule**: expressions in `cache_ttl_by_status` `codes` / `ttl` are preserved ([#339](https://github.com/cloudflare/tf-migrate/pull/339))
- String values are no longer cut off at `%` or `$`, for example percent-encoded URLs in `zero_trust_list` items ([#338](https://github.com/cloudflare/tf-migrate/pull/338))
- **api_token**: `cloudflare_api_token_permission_groups` references are migrated to `cloudflare_api_token_permission_groups_list` instead of being replaced with map keys, and the data source is no longer deleted ([#314](https://github.com/cloudflare/tf-migrate/pull/314))
- **zero_trust_access_policy**: both v4 names (`cloudflare_access_policy`, `cloudflare_zero_trust_access_policy`) with `application_id` follow the manual migration path. Preflight and the final summary list each affected policy under ACTION REQUIRED ([#335](https://github.com/cloudflare/tf-migrate/pull/335))
- **zero_trust_gateway_policy**: warns to review policy precedence before the first v5 apply ([#333](https://github.com/cloudflare/tf-migrate/pull/333))
- Provider version check recognizes OpenTofu lock files (`registry.opentofu.org/cloudflare/cloudflare`) ([#348](https://github.com/cloudflare/tf-migrate/pull/348))
- `verify-drift` reports resource types instead of module names, detects resources in the root module and nested modules, and detects `must be replaced` changes and addresses with quoted keys ([#328](https://github.com/cloudflare/tf-migrate/pull/328), [#347](https://github.com/cloudflare/tf-migrate/pull/347))

---

## [v1.1.1] - 2026-10-22

### Changed

- Built with Go 1.27.3, which distrusts a set of intermediate certificate authorities that have been revoked for CA compromise. The distrust list is compiled into the binary, so affected chains are rejected regardless of the host's system trust store ([#334](https://github.com/cloudflare/tf-migrate/pull/334))

### Added

- **workers_secret**: migrate `cloudflare_workers_secret` and the deprecated `cloudflare_worker_secret` to `secret_text` bindings on `cloudflare_workers_script`, since v5 has no standalone secret resource ([#320](https://github.com/cloudflare/tf-migrate/pull/320))

### Fixed

- **zero_trust_access_group**: single-element lists in `azure.id`, `gsuite.email` and `okta.name` are unwrapped to the scalar strings v5 expects, instead of failing apply with `string required, but have tuple` ([#321](https://github.com/cloudflare/tf-migrate/pull/321))
- **ruleset**: `exclude = ["*"]` in `cache_key.custom_key.query_string` becomes `{ all = true }` ([#327](https://github.com/cloudflare/tf-migrate/pull/327))

---

## [v1.1.0] - 2026-07-14

### Added

- **hyperdrive_config**: v4 to v5 migration support ([#317](https://github.com/cloudflare/tf-migrate/pull/317))
- **d1_database**: v4 to v5 migration support, adding `read_replication = { mode = "disabled" }` so the first v5 apply doesn't fail ([#315](https://github.com/cloudflare/tf-migrate/pull/315), [#319](https://github.com/cloudflare/tf-migrate/pull/319))

### Fixed

- **zone**: cross-file `cloudflare_zone.<name>.zone` references are rewritten to `.name` ([#295](https://github.com/cloudflare/tf-migrate/pull/295))
- **zero_trust_access_policy**: `include` / `exclude` / `require` conditions that use a variable or function call become a `for` expression instead of invalid HCL ([#313](https://github.com/cloudflare/tf-migrate/pull/313))
- **zero_trust_access_policy**: warns when `zone_id` is removed and no `account_id` is set, and when an application-scoped policy is removed without inline `policies` on the application, which would detach it on apply ([#309](https://github.com/cloudflare/tf-migrate/pull/309), [#310](https://github.com/cloudflare/tf-migrate/pull/310))
- **zero_trust_access_application**: `http_only_cookie_attribute` stays a bool instead of becoming the string `"false"`, which Terraform treats as true ([#308](https://github.com/cloudflare/tf-migrate/pull/308))
- **logpush_ownership_challenge**: references to `ownership_challenge_filename` are rewritten to `filename` ([#312](https://github.com/cloudflare/tf-migrate/pull/312))
- **zone_settings_override**: phase 1 no longer reports success while leaving the file unchanged when settings have aligned inline comments ([#311](https://github.com/cloudflare/tf-migrate/pull/311))

---

## [v1.0.1] - 2026-05-01

### Fixed

- Multiple `dynamic` blocks of the same type, or static and dynamic blocks mixed, are all kept and joined with `concat()` instead of only the last surviving ([#291](https://github.com/cloudflare/tf-migrate/pull/291))
- **zero_trust_split_tunnel**, **zero_trust_local_fallback_domain**: the newer v4 type names are handled, and multiple dynamic blocks no longer lose data ([#292](https://github.com/cloudflare/tf-migrate/pull/292))
- Preflight scan shows the correct v5 target type for resources that map to one of two types, such as `zero_trust_device_profiles` ([#293](https://github.com/cloudflare/tf-migrate/pull/293))

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
