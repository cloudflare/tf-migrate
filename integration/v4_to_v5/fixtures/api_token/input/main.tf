data "cloudflare_api_token_permission_groups" "all" {}

resource "cloudflare_api_token" "cftftest_mixed" {
  name = "cftftest mixed permission groups"
  policy {
    effect = "allow"
    permission_groups = [
      "literal-id",
      data.cloudflare_api_token_permission_groups.all.zone["DNS Write"],
      data.cloudflare_api_token_permission_groups.all.permissions["Zone Read"],
    ]
    resources = { "com.cloudflare.api.account.zone.*" = "*" }
  }
}

resource "cloudflare_api_token" "cftftest_comprehension" {
  name = "cftftest permission group comprehension"
  policy {
    permission_groups = [for p in local.names : data.cloudflare_api_token_permission_groups.all.zone[p] if p != "skip"]
    resources = { "com.cloudflare.api.account.zone.*" = "*" }
  }
}
