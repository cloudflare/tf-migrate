# Integration test for cloudflare_d1_database migration (v4 -> v5)
# The resource name and all user-configurable attributes are identical
# between v4 and v5, so the output should match the input exactly.

# Standard variables provided by test infrastructure
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

# Locals for DRY configuration
locals {
  common_account = var.cloudflare_account_id
  name_prefix    = "cftftest"
  environment    = "test"
}

# ============================================================================
# Basic Resources with Variables and Locals
# ============================================================================

# Minimal D1 database with required fields only
resource "cloudflare_d1_database" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal-db"
  read_replication = {
    mode = "disabled"
  }
}

# D1 database using locals
resource "cloudflare_d1_database" "with_locals" {
  account_id = local.common_account
  name       = "${local.name_prefix}-locals-db"
  read_replication = {
    mode = "disabled"
  }
}

# D1 database with string interpolation
resource "cloudflare_d1_database" "with_interpolation" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-${local.environment}-db"
  read_replication = {
    mode = "disabled"
  }
}

# ============================================================================
# Lifecycle Meta-arguments
# ============================================================================

resource "cloudflare_d1_database" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-db"

  lifecycle {
    create_before_destroy = true
    prevent_destroy       = false
  }
  read_replication = {
    mode = "disabled"
  }
}

resource "cloudflare_d1_database" "ignore_changes" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-ignore-changes-db"

  lifecycle {
    ignore_changes = [name]
  }
  read_replication = {
    mode = "disabled"
  }
}

# ============================================================================
# Terraform Functions
# ============================================================================

resource "cloudflare_d1_database" "with_join" {
  account_id = var.cloudflare_account_id
  name       = join("-", [local.name_prefix, "joined", "db"])
  read_replication = {
    mode = "disabled"
  }
}

resource "cloudflare_d1_database" "with_lower" {
  account_id = var.cloudflare_account_id
  name       = lower("${local.name_prefix}-LOWERCASE-DB")
  read_replication = {
    mode = "disabled"
  }
}

resource "cloudflare_d1_database" "with_format" {
  account_id = var.cloudflare_account_id
  name       = format("%s-formatted-db-%02d", local.name_prefix, 42)
  read_replication = {
    mode = "disabled"
  }
}
