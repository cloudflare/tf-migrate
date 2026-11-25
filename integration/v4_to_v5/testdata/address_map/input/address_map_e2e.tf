variable "cloudflare_account_id" { type = string }

resource "cloudflare_address_map" "cftftest_empty" {
  account_id  = var.cloudflare_account_id
  description = "cftftest empty address map"
  enabled     = false
}
