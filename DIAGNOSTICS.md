# Migration diagnostics

## Manual action required: permission_groups contains unrecognized expressions

**Severity:** Warning

**Resource:** `cloudflare_api_token`

tf-migrate preserves expressions it cannot recognize as literal permission IDs
or supported permission-group datasource lookups. List elements are wrapped as
`{ id = <expression> }`, and collection expressions are mapped to ID objects.
Unsupported object comprehensions are preserved unchanged. Review the named
file and ensure every `id` evaluates to a permission group ID.

## Manual action required: permission_groups looks up a permission group by name only

**Severity:** Warning

**Resource:** `cloudflare_api_token`

The deprecated v4 `.permissions[...]` map matched by name alone. Seven names
exist at both account and zone scope (`Logs Read`, `Logs Write`, `Access: Apps
and Policies Read`/`Write`/`Revoke`, `Disable ESC Read`/`Write`), so a name-only
`one([...])` matches two groups and fails at plan time.

tf-migrate normally infers the scope from the policy's `resources` map. This
warning means it could not: the keys mix scopes, the map comes from a variable,
or a key is interpolated. The lookup was left name-only rather than guessed.
Add the scope filter to the generated expression, for example
`&& contains(pg.scopes, "com.cloudflare.api.account")` or
`&& contains(pg.scopes, "com.cloudflare.api.account.zone")`. The warning names
the file.

See the [API token migration guide](internal/resources/api_token/README.md#permission-group-data-source-references)
for the scope table.

## Unknown attribute reference: data.cloudflare_api_token_permission_groups_list

**Severity:** Warning

**Data source:** `cloudflare_api_token_permission_groups`

The renamed v5 datasource exposes `result`, not the v4 `permissions`, `zone`,
`account`, `user`, or `r2` maps. Remaining references in locals, outputs, or other
expressions require manual updates. Each warning names the reference and file.
Indexed instances (`all[0]`, `all["key"]`, or `all[each.key]`) are included in
this scan. Supported indexed lookups inside token policies are migrated
automatically; the warning applies to references left elsewhere.
Look up IDs by name in `result`, adding the original map's scope filter. The
v4 `id` checksum has no direct replacement either.

Use `one([...])` to reject ambiguous matches; it returns `null` when no entry
matches. Deprecated `.permissions[...]` lookups lack a scope, so overlapping
names may require choosing a scope explicitly.

See the [API token migration guide](internal/resources/api_token/README.md#permission-group-data-source-references)
for the scope table, examples, and supported expression forms.

## Action required: review Gateway policy precedence

**Severity:** Warning

**Resources:** `cloudflare_teams_rule`, `cloudflare_zero_trust_gateway_policy`

The v4 provider applied a hash calculation to Gateway precedence. Live API
values can differ from configuration, and policies with duplicate configured
values can have a distinct live order. tf-migrate preserves configured values
and expressions; it does not infer or automatically renumber policy order.

Before applying with v5, inspect the live order of all relevant policies and
assign unique precedence values that preserve it. Lower values run first within
each processing phase. Review allow exceptions and catch-all block policies
together. Do not substitute API values for only some policies while retaining
configured values for others. A planned precedence update can reorder policies
on apply; it is not merely a state-format change.

See the [Gateway migration guide](internal/resources/zero_trust_gateway_policy/README.md#action-required-review-gateway-policy-precedence)
for details.

## Warning: dynamic block not migrated on cloudflare_ruleset

**Severity:** Warning

**Resources:** `cloudflare_ruleset`

**Summary:** `Dynamic <type> block not migrated on cloudflare_ruleset.<name>`

v5 turns every nested block in a ruleset into an attribute, and a block cannot
appear inside an object value, so tf-migrate rewrites `dynamic` blocks as `for`
expressions. It leaves the block as written and emits this warning when the
block has no `for_each` or no `content`, sets `labels`, or sits where
tf-migrate does not convert dynamic blocks (for example `dynamic "headers"`
outside `action_parameters`). The other blocks of the same type are then left
as written too, since converting them would drop this one from the list.

Rewrite the block by hand as a `for` expression. Use a list for `rules`,
`categories`, `status_code_ttl` and `algorithms`, and `one([for ...])` for any
other block type. See the [ruleset migration guide](internal/resources/ruleset/README.md#example-6-dynamic-blocks-inside-a-rule).

## Action required: policies not converted on cloudflare_zero_trust_access_application

**Severity:** Warning

**Resources:** `cloudflare_access_application`, `cloudflare_zero_trust_access_application`

v5 expects `policies` to be a list of objects with `id` and `precedence`.
tf-migrate converts a plain list literal, such as `[a.id, b.id]`, element by
element and uses each element's position as its precedence. Any other value,
such as a variable, `concat(...)`, or a `for` expression, is left unchanged.
`terraform validate` does not reject the unchanged value, so convert it by hand
before you plan:

```hcl
policies = [for i, id in <expression> : { id = id, precedence = i + 1 }]
```

## Action required: could not resolve type on cloudflare_authenticated_origin_pulls_certificate

**Severity:** Warning

**Resource:** `cloudflare_authenticated_origin_pulls_certificate`

v5 splits this resource by `type`: per-zone certificates keep the
`cloudflare_authenticated_origin_pulls_certificate` type, and per-hostname
certificates become `cloudflare_authenticated_origin_pulls_hostname_certificate`.
When `type` is not a string literal, for example `type = var.cert_type`,
tf-migrate cannot tell which one applies, so it migrates the certificate as
per-zone. If it is per-hostname, rename the resource, add the `moved` block from
the warning, and update every reference to the new type.

## Migrated a shared file outside the config directory

**Severity:** Warning

A `.tf` file in the configuration is a symlink to a file outside the config
directory, and the migration ran in place. tf-migrate migrates each real file
once, through whichever path reaches it, so the shared file itself now holds
the v5 configuration. Every other configuration that links to it sees the
change. Before you commit, check that none of them still use the v4 provider.
If any do, copy the file into this configuration instead of linking it.
