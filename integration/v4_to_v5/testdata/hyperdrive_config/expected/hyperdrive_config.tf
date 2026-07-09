# Integration test for cloudflare_hyperdrive_config migration (v4 -> v5)
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

# Minimal Hyperdrive config with required fields only
resource "cloudflare_hyperdrive_config" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal-hd"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }
}

# Hyperdrive config using locals
resource "cloudflare_hyperdrive_config" "with_locals" {
  account_id = local.common_account
  name       = "${local.name_prefix}-locals-hd"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }
}

# ============================================================================
# With Caching Configuration
# ============================================================================

resource "cloudflare_hyperdrive_config" "with_caching" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-caching-hd"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }

  caching = {
    disabled               = false
    max_age                = 300
    stale_while_revalidate = 60
  }
}

resource "cloudflare_hyperdrive_config" "caching_disabled" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-nocache-hd"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }

  caching = {
    disabled = true
  }
}

# ============================================================================
# With Access Origin
# ============================================================================

resource "cloudflare_hyperdrive_config" "with_access" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-access-hd"

  origin = {
    database             = "mydb"
    host                 = "db.internal.example.com"
    scheme               = "postgres"
    user                 = "admin"
    password             = "secret"
    access_client_id     = "client-id-abc.access"
    access_client_secret = "client-secret-xyz"
  }
}

# ============================================================================
# Lifecycle Meta-arguments
# ============================================================================

resource "cloudflare_hyperdrive_config" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-hd"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }

  lifecycle {
    create_before_destroy = true
    prevent_destroy       = false
  }
}

# ============================================================================
# MySQL Scheme
# ============================================================================

resource "cloudflare_hyperdrive_config" "mysql" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-mysql-hd"

  origin = {
    database = "mydb"
    host     = "mysql.example.com"
    port     = 3306
    scheme   = "mysql"
    user     = "root"
    password = "secret"
  }
}
