# E2E test for cloudflare_hyperdrive_config migration (v4 -> v5)
#
# Hyperdrive validates origin connectivity at create time, so this fixture
# requires a real, publicly-reachable Postgres-compatible database.
# Set CLOUDFLARE_HYPERDRIVE_DATABASE_HOSTNAME (and optionally the other
# CLOUDFLARE_HYPERDRIVE_DATABASE_* vars) before running.

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

variable "hyperdrive_db_host" {
  description = "Publicly resolvable hostname of a Postgres-compatible database"
  type        = string
}

variable "hyperdrive_db_name" {
  description = "Database name"
  type        = string
  default     = "postgres"
}

variable "hyperdrive_db_port" {
  description = "Database port"
  type        = number
  default     = 5432
}

variable "hyperdrive_db_user" {
  description = "Database user"
  type        = string
  default     = "postgres"
}

variable "hyperdrive_db_password" {
  description = "Database password"
  type        = string
  default     = "password"
  sensitive   = true
}

locals {
  name_prefix = "cftftest"
}

resource "cloudflare_hyperdrive_config" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal-hd"

  origin = {
    database = var.hyperdrive_db_name
    host     = var.hyperdrive_db_host
    port     = var.hyperdrive_db_port
    scheme   = "postgres"
    user     = var.hyperdrive_db_user
    password = var.hyperdrive_db_password
  }

  # The Hyperdrive API always returns a default caching block even when
  # not specified. Include it explicitly to avoid perpetual drift.
  caching = {
    disabled = false
  }
}

resource "cloudflare_hyperdrive_config" "with_caching" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-caching-hd"

  origin = {
    database = var.hyperdrive_db_name
    host     = var.hyperdrive_db_host
    port     = var.hyperdrive_db_port
    scheme   = "postgres"
    user     = var.hyperdrive_db_user
    password = var.hyperdrive_db_password
  }

  caching = {
    disabled = true
  }
}
