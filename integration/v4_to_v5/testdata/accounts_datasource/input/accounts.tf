locals {
  name_prefix = "cftftest"
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

# Look up all accounts (no filter)
data "cloudflare_accounts" "all" {
}

# Look up accounts by name filter
data "cloudflare_accounts" "by_name" {
  name = "My Company"
}

# Look up accounts with variable reference
data "cloudflare_accounts" "with_variable" {
  name = var.account_name
}

# Use account data in a resource
variable "account_name" {
  type    = string
  default = "My Company"
}

# Reference the accounts output attribute (v4 uses "accounts")
output "first_account_id" {
  value = data.cloudflare_accounts.by_name.accounts[0].id
}

output "first_account_name" {
  value = data.cloudflare_accounts.by_name.accounts[0].name
}

output "all_account_ids" {
  value = [for a in data.cloudflare_accounts.all.accounts : a.id]
}
