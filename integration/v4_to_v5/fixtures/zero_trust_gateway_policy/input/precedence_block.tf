locals {
  cftftest_gateway_block_precedence = 20000
}

resource "cloudflare_teams_rule" "cftftest_catch_all_block" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Block documentation network"
  action      = "block"
  filters     = ["l4"]
  traffic     = "net.dst.ip in {192.0.2.0/24}"
  precedence  = local.cftftest_gateway_block_precedence
}
