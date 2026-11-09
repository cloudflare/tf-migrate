data "cloudflare_api_token_permission_groups" "by_key" {
  for_each = toset(["zone"])
}

data "cloudflare_api_token_permission_groups" "by_count" {
  count = 1
}

resource "cloudflare_api_token" "cftftest_indexed" {
  for_each = toset(["zone"])
  name = "cftftest indexed ${each.key}"
  policy {
    permission_groups = [data.cloudflare_api_token_permission_groups.by_key[each.key].zone["DNS Read"]]
    resources = { "com.cloudflare.api.account.zone.*" = "*" }
  }
}

resource "cloudflare_api_token" "cftftest_counted" {
  count = 1
  name = "cftftest counted ${count.index}"
  policy {
    permission_groups = [for p in local.names : data.cloudflare_api_token_permission_groups.by_count[count.index].zone[p]]
    resources = { "com.cloudflare.api.account.zone.*" = "*" }
  }
}

locals {
  indexed_dns_id = data.cloudflare_api_token_permission_groups.by_key["zone"].zone["DNS Read"]
}

output "counted_dns_id" {
  value = data.cloudflare_api_token_permission_groups.by_count[0].zone["DNS Read"]
}
