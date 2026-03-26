# Migration Issues — Research Team (MR !7756)

Issues discovered when the research team migrated their Terraform config
at `tf/research-team/` from Cloudflare Provider v4 → v5.

## Summary

| Ticket | Resource | Issue | Severity |
|--------|----------|-------|----------|
| [TKT-001](TKT-001-authenticated-origin-pulls-cert-id-reference.md) | `cloudflare_authenticated_origin_pulls` | `cert_id` still references `cloudflare_authenticated_origin_pulls_certificate` instead of `cloudflare_authenticated_origin_pulls_hostname_certificate` when using `for_each` | High |
| [TKT-002](TKT-002-zero-trust-access-policy-include-block-vs-attribute.md) | `cloudflare_zero_trust_access_policy` | `include`/`exclude`/`require` remain as blocks instead of being converted to `include = [{ ... }]` attribute list syntax | Critical — causes plan failure |
| [TKT-003](TKT-003-zero-trust-access-policy-application-id-precedence.md) | `cloudflare_zero_trust_access_policy` | `application_id` and `precedence` not removed from policy; not moved to linked `cloudflare_zero_trust_access_application.policies = [...]` | Critical — causes state upgrade failure |
| [TKT-004](TKT-004-zero-trust-access-policy-service-token-format.md) | `cloudflare_zero_trust_access_policy` | `service_token = [id]` list not converted to `service_token = { token_id = id }` object | High |
| [TKT-005](TKT-005-zero-trust-access-policy-email-domain-format.md) | `cloudflare_zero_trust_access_policy` | `email_domain = ["domain"]` list not converted to `email_domain = { domain = "domain" }` | High |
| [TKT-006](TKT-006-zero-trust-access-policy-any-valid-service-token-format.md) | `cloudflare_zero_trust_access_policy` | `any_valid_service_token = true` not converted to `any_valid_service_token = {}` | High |

## Plan Error

The plan error from Atlantis (`plan-error-1.log`) shows:

```
Error: Unable to Read Previously Saved State for UpgradeResourceState
  AttributeName("connection_rules"): invalid JSON, expected "{", got "["
```

This is caused by TKT-002/TKT-003: the `include` blocks remain as HCL blocks,
and the state upgrade handler in the v5 provider tries to deserialize JSON but
finds a list `[` where it expects an object `{`.

Additionally:
```
Warning: Failed to decode resource from state
Error decoding "cloudflare_zero_trust_access_policy.allow_any_service_token"
from prior state: unsupported attribute "application_id"
```

This is TKT-003: `application_id` should have been removed from the policy.

## What the research team did manually

Per the commit message in `727e0f99c`:
1. Fixed `cert_id` references manually with sed (TKT-001)
2. Manually fixed `include` options for `cloudflare_zero_trust_access_policy` resources
3. Used `cloudflare_zero_trust_access_application` to bind policies (TKT-003)
4. Used the latest `main` branch of tf-migrate instead of tagged version because
   v1.0.0-beta.5 cleared `service_token` entirely (TKT-004)

## E2E Tests Added

Test cases reproducing all issues have been added to:
- `integration/v4_to_v5/testdata/authenticated_origin_pulls/` — TKT-001
- `integration/v4_to_v5/testdata/zero_trust_access_policy/` — TKT-002 through TKT-006

The expected output files reflect the **current (buggy) migration output**
so these tests will fail once the bugs are fixed, reminding us to update
the expected output to the correct v5 syntax.
