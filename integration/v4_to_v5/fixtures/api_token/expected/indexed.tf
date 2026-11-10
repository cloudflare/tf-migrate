data "cloudflare_api_token_permission_groups_list" "by_key" {
  for_each = toset(["zone"])
}

data "cloudflare_api_token_permission_groups_list" "by_count" {
  count = 1
}

resource "cloudflare_api_token" "cftftest_indexed" {
  for_each = toset(["zone"])
  name = "cftftest indexed ${each.key}"
  policies = [{
    permission_groups = [{ id = one([for pg in data.cloudflare_api_token_permission_groups_list.by_key[each.key].result : pg.id if pg.name == ("DNS Read") && contains(pg.scopes, "com.cloudflare.api.account.zone")]) }]
    resources = jsonencode({ "com.cloudflare.api.account.zone.*" = "*" })
    effect = "allow"
  }]
}

resource "cloudflare_api_token" "cftftest_counted" {
  count = 1
  name = "cftftest counted ${count.index}"
  policies = [{
    permission_groups = [for p in local.names : { id = one([for pg in data.cloudflare_api_token_permission_groups_list.by_count[count.index].result : pg.id if pg.name == (p) && contains(pg.scopes, "com.cloudflare.api.account.zone")]) }]
    resources = jsonencode({ "com.cloudflare.api.account.zone.*" = "*" })
    effect = "allow"
  }]
}

locals {
  indexed_dns_id = data.cloudflare_api_token_permission_groups_list.by_key["zone"].zone["DNS Read"]
}

output "counted_dns_id" {
  value = data.cloudflare_api_token_permission_groups_list.by_count[0].zone["DNS Read"]
}
