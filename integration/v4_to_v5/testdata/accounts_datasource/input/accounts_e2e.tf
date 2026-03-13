locals {
  name_prefix = "cftftest"
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

# Look up all accounts
data "cloudflare_accounts" "all" {
}

# Use the v4 "accounts" attribute in an output
output "first_account_id" {
  value = data.cloudflare_accounts.all.accounts[0].id
}

output "first_account_name" {
  value = data.cloudflare_accounts.all.accounts[0].name
}
