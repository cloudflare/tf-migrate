variable "account_id" { type = string }
variable "cloudflare_zone_id" { type = string }

resource "cloudflare_address_map" "cftftest_populated" {
  account_id  = var.account_id
  description = "cftftest populated address map"
  enabled     = false


  ips         = distinct(["192.0.2.10", "2001:db8::1"])
  memberships = distinct([{ identifier = var.cloudflare_zone_id, kind = "zone" }, { identifier = var.account_id, kind = "account" }])
}
