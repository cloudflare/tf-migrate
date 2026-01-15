# Comprehensive Integration Test for account_member v4 → v5 Migration
# Target: 20-25 resource instances covering all patterns

# Pattern 1: Variables (no defaults - must be provided)
variable "cloudflare_account_id" {
  description = "Cloudflare account ID for testing"
  type        = string
}

# Pattern 2: Locals with resource-specific configuration
locals {
  admin_role_id       = "e58cefd75d7adae0b761796c28815e5c"
  super_admin_role_id = "a4154d230e664f8b3e6e5c95a8cc812f"
  analytics_role_id   = "3030687196b94b638145a3953da2b699"

  test_emails = [
    "test-user-1@cfapi.net",
    "test-user-2@cfapi.net",
    "test-user-3@cfapi.net",
  ]

  admin_emails = {
    dev_admin = "dev-admin@cfapi.net"
    ops_admin = "ops-admin@cfapi.net"
  }

  enable_analytics_team = true
  enable_audit_team     = true
}

# Basic Examples - All Status Types
resource "cloudflare_account_member" "example_accepted" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "accepted-user@cfapi.net"
  roles      = [local.admin_role_id]
}

resource "cloudflare_account_member" "example_pending" {
  account_id = var.cloudflare_account_id
  status     = "pending"
  email      = "pending-user@cfapi.net"
  roles      = [local.admin_role_id]
}

resource "cloudflare_account_member" "example_no_status" {
  account_id = var.cloudflare_account_id
  email      = "no-status-user@cfapi.net"
  roles      = [local.admin_role_id]
}

# Multiple Roles Examples
resource "cloudflare_account_member" "single_role" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "single-role@cfapi.net"
  roles      = [local.admin_role_id]
}

resource "cloudflare_account_member" "multiple_roles" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "multi-role@cfapi.net"
  roles = [
    local.admin_role_id,
    local.super_admin_role_id,
  ]
}

resource "cloudflare_account_member" "three_roles" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "three-roles@cfapi.net"
  roles = [
    local.admin_role_id,
    local.super_admin_role_id,
    local.analytics_role_id,
  ]
}

# Pattern 3: for_each with set (converted from list)
resource "cloudflare_account_member" "test_users" {
  for_each = toset(local.test_emails)

  account_id = var.cloudflare_account_id
  status     = "pending"
  email      = each.value
  roles      = [local.admin_role_id]
}

# Pattern 4: for_each with map (complex)
resource "cloudflare_account_member" "admin_users" {
  for_each = local.admin_emails

  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = each.value
  roles = [
    local.admin_role_id,
    local.super_admin_role_id,
  ]
}

# Pattern 5: count-based resources
resource "cloudflare_account_member" "batch_users" {
  count = 3

  account_id = var.cloudflare_account_id
  status     = "pending"
  email      = "batch-user-${count.index + 1}@cfapi.net"
  roles      = [local.analytics_role_id]
}

# Pattern 6: Conditional resource creation
resource "cloudflare_account_member" "analytics_conditional" {
  count = local.enable_analytics_team ? 1 : 0

  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "analytics-lead@cfapi.net"
  roles      = [local.analytics_role_id]
}

resource "cloudflare_account_member" "audit_conditional" {
  count = local.enable_audit_team ? 1 : 0

  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "audit-lead@cfapi.net"
  roles      = [local.admin_role_id]
}

# Pattern 7: Lifecycle meta-arguments
resource "cloudflare_account_member" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  status     = "accepted"

  lifecycle {
    create_before_destroy = true
  }
  email = "lifecycle-user@cfapi.net"
  roles = [local.admin_role_id]
}

# Pattern 8: Terraform functions and string interpolation
resource "cloudflare_account_member" "with_functions" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "user-${formatdate("YYYY-MM-DD", timestamp())}@cfapi.net"
  roles      = [local.admin_role_id]
}

# Cross-reference pattern (referencing other resources)
resource "cloudflare_account_member" "referenced" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "primary-admin@cfapi.net"
  roles      = [local.super_admin_role_id]
}

resource "cloudflare_account_member" "referencing" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "secondary-admin@cfapi.net"
  roles      = cloudflare_account_member.referenced.role_ids
}

# Edge case: Email with special characters
resource "cloudflare_account_member" "special_email" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "user.name+tag@cfapi.net"
  roles      = [local.admin_role_id]
}

# Edge case: Single role in array
resource "cloudflare_account_member" "single_role_array" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "single-array@cfapi.net"
  roles = [
    local.admin_role_id,
  ]
}

# Field order variations (testing all field orders)
resource "cloudflare_account_member" "order_variation_1" {
  status     = "accepted"
  account_id = var.cloudflare_account_id
  email      = "order-test-1@cfapi.net"
  roles      = [local.admin_role_id]
}

resource "cloudflare_account_member" "order_variation_2" {
  status     = "accepted"
  account_id = var.cloudflare_account_id
  email      = "order-test-2@cfapi.net"
  roles      = [local.admin_role_id]
}

# Role combinations testing
resource "cloudflare_account_member" "admin_only" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "admin-only@cfapi.net"
  roles      = [local.admin_role_id]
}

resource "cloudflare_account_member" "super_admin_only" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "super-admin-only@cfapi.net"
  roles      = [local.super_admin_role_id]
}

resource "cloudflare_account_member" "analytics_only" {
  account_id = var.cloudflare_account_id
  status     = "accepted"
  email      = "analytics-only@cfapi.net"
  roles      = [local.analytics_role_id]
}
