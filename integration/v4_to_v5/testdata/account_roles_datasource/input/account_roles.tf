variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

# Look up all account roles - this is a datasource test
data "cloudflare_account_roles" "all" {
  account_id = var.cloudflare_account_id
}

# Build a lookup map by role name (v4: .roles)
locals {
  account_member_roles = {
    for role in data.cloudflare_account_roles.all.roles :
    role.name => role
  }
}

# Direct attribute references (v4: .roles)
output "all_role_ids" {
  value = [for r in data.cloudflare_account_roles.all.roles : r.id]
}

output "all_role_names" {
  value = [for r in data.cloudflare_account_roles.all.roles : r.name]
}

# Output the role map to verify the lookup works
output "admin_role_id" {
  value = local.account_member_roles["Administrator"].id
}
