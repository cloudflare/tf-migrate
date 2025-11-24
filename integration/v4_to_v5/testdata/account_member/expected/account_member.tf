# Integration Tests for cloudflare_account_member v4 to v5 Migration
# This file contains only one test since end to end tests fail with more than one test 
# account_member is 1:1 for zone id

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

locals {
  common_account = var.cloudflare_account_id

  # Standard Cloudflare role IDs (predefined, not account-specific)
  admin_read_only_role = "e58cefd75d7adae0b761796c28815e5c"
  analytics_role       = "a4154d230e664f8b3e6e5c95a8cc812f"
  billing_role         = "a3eb64b6819c42e78c93e9cb90e6e8e2"
}

resource "cloudflare_account_member" "maximal" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "maximal-member@cfapi.net"
  roles      = [local.admin_read_only_role, local.analytics_role, local.billing_role]
}
