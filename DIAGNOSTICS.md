# Migration diagnostics

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
block has no `for_each` or no `content`, sets `labels`, or is nested in a rule
alongside static blocks of the same type. For `dynamic "rules"`, every rule is then left as
written, since converting the others would drop this one from the list.

Rewrite the block by hand as a `for` expression. Use a list for `rules`,
`categories`, `status_code_ttl` and `algorithms`, and `one([for ...])` for any
other block type. See the [ruleset migration guide](internal/resources/ruleset/README.md#example-6-dynamic-blocks-inside-a-rule).
