variable "account_id" { type = string }
variable "cloudflare_zone_id" { type = string }

resource "cloudflare_address_map" "cftftest_populated" {
  account_id  = var.account_id
  description = "cftftest populated address map"
  enabled     = false

  ips { ip = "192.0.2.10" }
  ips { ip = "2001:db8::1" }

  memberships {
    identifier = var.cloudflare_zone_id
    kind       = "zone"
  }
  memberships {
    identifier = var.account_id
    kind       = "account"
  }
}
