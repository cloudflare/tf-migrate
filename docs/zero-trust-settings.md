# Preserve Zero Trust settings during v4 → v5 migration

The v4 organization and Gateway schemas cannot represent some live settings
exposed by v5. Refreshing v4 state cannot recover them. Preserve them once by
capturing API responses before migration and copying the values into explicit
v5 HCL before applying.

This workflow does not change the provider's omission/default behavior, add
`ignore_changes`, or modify Terraform state. Afterwards the values are ordinary
configuration: edit or remove them using the provider's existing semantics.

## 1. Capture the current API values

For each account, save the relevant GET responses before migration:

```bash
curl --fail-with-body --silent --show-error \
  --header "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/access/organizations" \
  > organization.json

curl --fail-with-body --silent --show-error \
  --header "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/gateway/configuration" \
  > gateway.json

# Include this response when migrating a Gateway proxy block to device settings.
curl --fail-with-body --silent --show-error \
  --header "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/devices/settings" \
  > device.json
```

Combine responses into a JSON object keyed by the **local v5 resource addresses**
that the HCL migration will produce. For example, if the old organization and
Gateway resources are both named `example`, and Gateway contains a `proxy` block:

```bash
jq -n --slurpfile organization organization.json \
  --slurpfile gateway gateway.json --slurpfile device device.json '{
    "cloudflare_zero_trust_organization.example": $organization[0],
    "cloudflare_zero_trust_gateway_settings.example": $gateway[0],
    "cloudflare_zero_trust_device_settings.example_device_settings": $device[0]
  }' > zero-trust-settings.json
```

Include only resources managed by that module. Check the migrated device
resource name: tf-migrate derives it from the original Gateway resource name.
Both full successful API response envelopes and bare `result` objects are accepted.

## 2. Migrate HCL and restore the captured settings

```bash
tf-migrate migrate
tf-migrate --dry-run preserve-zero-trust-settings --snapshot zero-trust-settings.json
tf-migrate preserve-zero-trust-settings --snapshot zero-trust-settings.json
```

Use `--config-dir` for another module directory. The restoration command reads
that directory's `.tf` files, validates all snapshot entries before writing, and
changes only files requiring additions. It can be rerun: existing explicit HCL
wins, including `false`, `null`, empty lists, and expressions in individual fields.

The copied fields are:

| Resource | Fields |
| --- | --- |
| `cloudflare_zero_trust_organization` | `deny_unmatched_requests`, `deny_unmatched_requests_exempted_zone_names`, `mfa_config`, `mfa_piv_key_requirements`, `mfa_required_for_all_apps` |
| `cloudflare_zero_trust_gateway_settings` | `settings.host_selector`, `settings.sandbox`, `settings.inspection`, `settings.max_ttl_secs`, `settings.block_page.mode`, `settings.block_page.include_context`, `settings.block_page.suppress_footer`, `settings.block_page.target_uri` |
| `cloudflare_zero_trust_device_settings` | `external_emergency_signal_enabled`, `external_emergency_signal_url`, `external_emergency_signal_fingerprint`, `external_emergency_signal_interval` |

Absent or null API values are not synthesized. Unrelated response fields such as
timestamps, account IDs, and read-only fields are not copied.

Only fields the v4 schema could not express are copied. A nested field such as
`settings.block_page.mode` is added to `block_page` when that object is already
configured, preserving the comments, expressions, and field values already there,
and `block_page` is created when it is absent. In that second case it holds only
the fields listed above: live values for fields v4 could express, such as
`block_page.name`, are not restored, because configuration that omits them is a
divergence that predates migration. Set those explicitly if the live values
should be kept.

For modules, run restoration within each source module directory with local
addresses. A resource using `count` or `for_each` may represent multiple accounts:
restore its per-instance settings explicitly rather than copying one account's
response to every instance. Likewise, if Gateway `settings` or a nested object
such as `block_page` is supplied by a variable, local, or other expression, add
the fields listed above to that source object. The command reports these cases
without writing any files.

## 3. Verify before applying v5

Follow the provider migration guide for provider versions and resource moves.
Review `terraform plan` before the first v5 apply: the restored settings should
remain at their captured values unless the HCL explicitly requests a change.
After applying, check the follow-up plan and the service's settings.

State upgrading alone is insufficient: if a v5 Optional field is missing from
HCL, a refreshed live value in state can still be planned for removal.
