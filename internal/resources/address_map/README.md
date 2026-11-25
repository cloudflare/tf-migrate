# Address map: v4 to v5

The resource keeps its Terraform address. `tf-migrate` converts repeated `ips`
blocks into a list of strings and `memberships` blocks into a list of objects
containing `identifier` and `kind`. Dynamic blocks become `for` expressions.
Omitted v4 collections become `[]`; existing v5 attributes are kept. Duplicate
entries are collapsed to match v4 set behavior.

For example, `ips { ip = var.address }` becomes `ips = [var.address]`.

Review outputs, locals, and other consumers of both collections. In v5,
`[for entry in cloudflare_address_map.example.ips : entry.ip]` can be replaced
with `cloudflare_address_map.example.ips`. The per-membership `can_delete` field
has no v5 equivalent; the map-level field describes the map itself. When
membership order matters, explicitly choose how to sort the list.

Use a provider release that includes the [address map migration and update fix](https://github.com/cloudflare/terraform-provider-cloudflare/issues/7299).
**Do not use v5.29.0 as a migration target.** The converted configuration
passes `terraform validate` on that version, but the provider cannot upgrade
v4 state with populated IPs and forces replacement on collection changes.

v4 stored both collections as sets, so their order in state can differ from
their order in the converted v5 configuration. With the fixed provider, this
can produce a one-time in-place update. If only the order differs, the provider
records the configured order in state without changing the map or its entries
in Cloudflare. Check that the plan does not replace the map and run a follow-up
plan after applying to confirm there are no further changes.
