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
    dev_admin  = "dev-admin@cfapi.net"
    ops_admin  = "ops-admin@cfapi.net"
  }

  enable_analytics_team = true
  enable_audit_team     = true
}

# Basic Examples - All Status Types
resource "cloudflare_account_member" "example_accepted" {
  account_id    = var.cloudflare_account_id
  email_address = "accepted-user@cfapi.net"
  role_ids      = [local.admin_role_id]
  status        = "accepted"
}

resource "cloudflare_account_member" "example_pending" {
  account_id    = var.cloudflare_account_id
  email_address = "pending-user@cfapi.net"
  role_ids      = [local.admin_role_id]
  status        = "pending"
}

resource "cloudflare_account_member" "example_no_status" {
  account_id    = var.cloudflare_account_id
  email_address = "no-status-user@cfapi.net"
  role_ids      = [local.admin_role_id]
}

# Multiple Roles Examples
resource "cloudflare_account_member" "single_role" {
  account_id    = var.cloudflare_account_id
  email_address = "single-role@cfapi.net"
  role_ids      = [local.admin_role_id]
  status        = "accepted"
}

resource "cloudflare_account_member" "multiple_roles" {
  account_id    = var.cloudflare_account_id
  email_address = "multi-role@cfapi.net"
  role_ids = [
    local.admin_role_id,
    local.super_admin_role_id,
  ]
  status = "accepted"
}

resource "cloudflare_account_member" "three_roles" {
  account_id    = var.cloudflare_account_id
  email_address = "three-roles@cfapi.net"
  role_ids = [
    local.admin_role_id,
    local.super_admin_role_id,
    local.analytics_role_id,
  ]
  status = "accepted"
}

# Pattern 3: for_each with set (converted from list)
resource "cloudflare_account_member" "test_users" {
  for_each = toset(local.test_emails)

  account_id    = var.cloudflare_account_id
  email_address = each.value
  role_ids      = [local.admin_role_id]
  status        = "pending"
}

# Pattern 4: for_each with map (complex)
resource "cloudflare_account_member" "admin_users" {
  for_each = local.admin_emails

  account_id    = var.cloudflare_account_id
  email_address = each.value
  role_ids = [
    local.admin_role_id,
    local.super_admin_role_id,
  ]
  status = "accepted"
}

# Pattern 5: count-based resources
resource "cloudflare_account_member" "batch_users" {
  count = 3

  account_id    = var.cloudflare_account_id
  email_address = "batch-user-${count.index + 1}@cfapi.net"
  role_ids      = [local.analytics_role_id]
  status        = "pending"
}

# Pattern 6: Conditional resource creation
resource "cloudflare_account_member" "analytics_conditional" {
  count = local.enable_analytics_team ? 1 : 0

  account_id    = var.cloudflare_account_id
  email_address = "analytics-lead@cfapi.net"
  role_ids      = [local.analytics_role_id]
  status        = "accepted"
}

resource "cloudflare_account_member" "audit_conditional" {
  count = local.enable_audit_team ? 1 : 0

  account_id    = var.cloudflare_account_id
  email_address = "audit-lead@cfapi.net"
  role_ids      = [local.admin_role_id]
  status        = "accepted"
}

# Pattern 7: Lifecycle meta-arguments
resource "cloudflare_account_member" "with_lifecycle" {
  account_id    = var.cloudflare_account_id
  email_address = "lifecycle-user@cfapi.net"
  role_ids      = [local.admin_role_id]
  status        = "accepted"

  lifecycle {
    create_before_destroy = true
  }
}

# Pattern 8: Terraform functions and string interpolation
resource "cloudflare_account_member" "with_functions" {
  account_id    = var.cloudflare_account_id
  email_address = "user-${formatdate("YYYY-MM-DD", timestamp())}@cfapi.net"
  role_ids      = [local.admin_role_id]
  status        = "accepted"
}

# Cross-reference pattern (referencing other resources)
resource "cloudflare_account_member" "referenced" {
  account_id    = var.cloudflare_account_id
  email_address = "primary-admin@cfapi.net"
  role_ids      = [local.super_admin_role_id]
  status        = "accepted"
}

resource "cloudflare_account_member" "referencing" {
  account_id    = var.cloudflare_account_id
  email_address = "secondary-admin@cfapi.net"
  role_ids      = cloudflare_account_member.referenced.role_ids
  status        = "accepted"
}

# Edge case: Email with special characters
resource "cloudflare_account_member" "special_email" {
  account_id    = var.cloudflare_account_id
  email_address = "user.name+tag@cfapi.net"
  role_ids      = [local.admin_role_id]
  status        = "accepted"
}

# Edge case: Single role in array
resource "cloudflare_account_member" "single_role_array" {
  account_id    = var.cloudflare_account_id
  email_address = "single-array@cfapi.net"
  role_ids = [
    local.admin_role_id,
  ]
  status = "accepted"
}

# Field order variations (testing all field orders)
resource "cloudflare_account_member" "order_variation_1" {
  role_ids      = [local.admin_role_id]
  email_address = "order-test-1@cfapi.net"
  status        = "accepted"
  account_id    = var.cloudflare_account_id
}

resource "cloudflare_account_member" "order_variation_2" {
  status        = "accepted"
  role_ids      = [local.admin_role_id]
  account_id    = var.cloudflare_account_id
  email_address = "order-test-2@cfapi.net"
}

# Role combinations testing
resource "cloudflare_account_member" "admin_only" {
  account_id    = var.cloudflare_account_id
  email_address = "admin-only@cfapi.net"
  role_ids      = [local.admin_role_id]
  status        = "accepted"
}

resource "cloudflare_account_member" "super_admin_only" {
  account_id    = var.cloudflare_account_id
  email_address = "super-admin-only@cfapi.net"
  role_ids      = [local.super_admin_role_id]
  status        = "accepted"
}

resource "cloudflare_account_member" "analytics_only" {
  account_id    = var.cloudflare_account_id
  email_address = "analytics-only@cfapi.net"
  role_ids      = [local.analytics_role_id]
  status        = "accepted"
}
