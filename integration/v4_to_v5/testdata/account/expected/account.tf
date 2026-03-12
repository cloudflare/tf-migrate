# Integration Test for account v4 → v5 Migration
# Tests enforce_twofactor wrapping into settings block

# Pattern 1: Variables (no defaults - must be provided)
variable "cloudflare_account_id" {
  description = "Cloudflare account ID for testing"
  type        = string
}

# Pattern 2: Locals
locals {
  enforce_2fa = true
}

# Basic: no enforce_twofactor (nothing to transform)
resource "cloudflare_account" "minimal" {
  name = "Minimal Account"
}

# Basic: enforce_twofactor = true
resource "cloudflare_account" "with_2fa_true" {
  name = "2FA Enabled Account"
  settings = {
    enforce_twofactor = true
  }
}

# Basic: enforce_twofactor = false
resource "cloudflare_account" "with_2fa_false" {
  name = "2FA Disabled Account"
  settings = {
    enforce_twofactor = false
  }
}

# Full config with type
resource "cloudflare_account" "full_config" {
  name = "Full Config Account"
  type = "standard"
  settings = {
    enforce_twofactor = true
  }
}

# Variable reference for enforce_twofactor
resource "cloudflare_account" "with_variable" {
  name = "Variable Account"
  settings = {
    enforce_twofactor = local.enforce_2fa
  }
}

# Field order variation
resource "cloudflare_account" "order_variation" {
  name = "Order Test Account"
  type = "standard"
  settings = {
    enforce_twofactor = true
  }
}

# Lifecycle meta-arguments
resource "cloudflare_account" "with_lifecycle" {
  name = "Lifecycle Account"

  lifecycle {
    ignore_changes = [name]
  }
  settings = {
    enforce_twofactor = true
  }
}
