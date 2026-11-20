data "cloudflare_api_token_permission_groups_list" "all" {}

resource "cloudflare_api_token" "cftftest_mixed" {
  name = "cftftest mixed permission groups"
  policies = [{
    effect = "allow"
    permission_groups = [{ id = "literal-id" }, { id = one([for pg in data.cloudflare_api_token_permission_groups_list.all.result : pg.id if pg.name == ("DNS Write") && contains(pg.scopes, "com.cloudflare.api.account.zone")]) }, { id = one([for pg in data.cloudflare_api_token_permission_groups_list.all.result : pg.id if pg.name == ("Zone Read") && contains(pg.scopes, "com.cloudflare.api.account.zone")]) }]
    resources = jsonencode({ "com.cloudflare.api.account.zone.*" = "*" })
  }]
}

resource "cloudflare_api_token" "cftftest_comprehension" {
  name = "cftftest permission group comprehension"
  policies = [{
    permission_groups = [for p in local.names : { id = one([for pg in data.cloudflare_api_token_permission_groups_list.all.result : pg.id if pg.name == (p) && contains(pg.scopes, "com.cloudflare.api.account.zone")]) } if p != "skip"]
    resources = jsonencode({ "com.cloudflare.api.account.zone.*" = "*" })
    effect = "allow"
  }]
}
