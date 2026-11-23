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

The provider upgrades Terraform state. The converted configuration matches the
v5.29.0 schema, but **v5.29.0 cannot upgrade populated v4 IP state**. It also
forces replacement when either collection changes. Before applying, use a
provider release that includes the [address map migration and update fix](https://github.com/cloudflare/terraform-provider-cloudflare/issues/7299)
and confirm the plan does not replace the map.
