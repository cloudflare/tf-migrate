resource "cloudflare_address_map" "cftftest_empty" {
  count      = 1
  account_id = var.cloudflare_account_id
  enabled    = false
}
