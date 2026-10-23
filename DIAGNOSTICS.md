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
