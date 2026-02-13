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

# ============================================================================
# Pattern Group 1: Variables & Locals
# ============================================================================

locals {
  common_account = var.cloudflare_account_id
  name_prefix                        = "cftftest"
  default_schedule = "24h"
  enable_firewall_rules = true
  enable_test_rules = false
}

# ============================================================================
# Pattern Group 2: for_each with Maps (5 resources)
# ============================================================================

resource "cloudflare_device_posture_rule" "map_example" {
  for_each = {
    "prod" = {
      account_id  = var.cloudflare_account_id
      name = "${local.name_prefix}-prod-posture-rule"
      type        = "os_version"
      schedule    = "24h"
      platform    = "linux"
      version     = "20.4.0"
    }
    "staging" = {
      account_id  = var.cloudflare_account_id
      name = "${local.name_prefix}-staging-posture-rule"
      type        = "os_version"
      schedule    = "12h"
      platform    = "windows"
      version     = "10.0.0"
    }
    "dev" = {
      account_id  = var.cloudflare_account_id
      name = "${local.name_prefix}-dev-posture-rule"
      type        = "os_version"
      schedule    = "6h"
      platform    = "mac"
      version     = "12.0.0"
    }
    "qa" = {
      account_id  = var.cloudflare_account_id
      name = "${local.name_prefix}-qa-posture-rule"
      type        = "os_version"
      schedule    = "12h"
      platform    = "linux"
      version     = "1.0.0"
    }
    "perf" = {
      account_id  = var.cloudflare_account_id
      name = "${local.name_prefix}-perf-posture-rule"
      type        = "os_version"
      schedule    = "24h"
      platform    = "windows"
      version     = "11.0.0"
    }
  }

  account_id = each.value.account_id
  name       = each.value.name
  type       = each.value.type
  schedule   = each.value.schedule

  match {
    platform = each.value.platform
  }

  input {
    version  = each.value.version
    operator = ">="
  }
}

# ============================================================================
# Pattern Group 3: for_each with Sets (4 items)
# ============================================================================

resource "cloudflare_device_posture_rule" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-set-${each.value}-rule"
  type       = "firewall"
  schedule   = "5m"

  match {
    platform = "linux"
  }

  input {
    enabled = true
  }
}

# ============================================================================
# Pattern Group 4: count-based Resources (3 instances)
# ============================================================================

resource "cloudflare_device_posture_rule" "counted" {
  count = 3

  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix}-counted-rule-${count.index}"
  type        = "os_version"
  schedule    = "24h"
  description = "This is posture rule number ${count.index}"

  match {
    platform = "linux"
  }

  input {
    version  = "1.${count.index}.0"
    operator = "=="
  }
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

resource "cloudflare_device_posture_rule" "conditional_enabled" {
  count = local.enable_firewall_rules ? 1 : 0

  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix}-conditional-firewall-enabled"
  type        = "firewall"
  schedule    = "12h"
  description = "Conditionally enabled firewall rule"

  match {
    platform = "windows"
  }

  input {
    enabled = true
  }
}

resource "cloudflare_device_posture_rule" "conditional_disabled" {
  count = local.enable_test_rules ? 1 : 0

  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix}-conditional-test-disabled"
  type        = "os_version"
  schedule    = "6h"
  description = "Conditionally disabled test rule"

  match {
    platform = "linux"
  }

  input {
    version  = "1.0.0"
    operator = ">"
  }
}

# ============================================================================
# Pattern Group 7: Terraform Functions
# ============================================================================

resource "cloudflare_device_posture_rule" "with_functions" {
  account_id = local.common_account

  # join() function
  name = join("-", [local.name_prefix, "function", "example"])

  type        = "os_version"
  schedule    = local.default_schedule
  description = "Rule for account ${var.cloudflare_account_id}"

  match {
    platform = "linux"
  }

  input {
    version  = "22.4.0"
    operator = ">="
  }
}

resource "cloudflare_device_posture_rule" "with_interpolation" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix}-rule-for-account-${var.cloudflare_account_id}"
  type        = "firewall"
  schedule    = "5m"
  description = "Interpolated rule name"

  match {
    platform = "windows"
  }

  input {
    enabled = true
  }
}

# ============================================================================
# Pattern Group 8: Lifecycle Meta-Arguments
# ============================================================================

resource "cloudflare_device_posture_rule" "with_lifecycle" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix}-lifecycle-test-rule"
  type        = "os_version"
  schedule    = "24h"
  description = "Rule with lifecycle arguments"

  match {
    platform = "linux"
  }

  input {
    version  = "20.4.0"
    operator = ">="
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
}

resource "cloudflare_device_posture_rule" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-prevent-destroy-rule"
  type       = "firewall"
  schedule   = "12h"

  match {
    platform = "windows"
  }

  input {
    enabled = true
  }

  lifecycle {
    prevent_destroy = false  # Set to false for testing
  }
}

# ============================================================================
# Pattern Group 9: Edge Cases
# ============================================================================

# Minimal resource (only required fields)
resource "cloudflare_device_posture_rule" "minimal" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-minimal-rule"
  type       = "firewall"
  schedule   = "5m"

  match {
    platform = "linux"
  }

  input {
    enabled = true
  }
}

# Maximal resource (all fields populated)
resource "cloudflare_device_posture_rule" "maximal" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix}-maximal-rule"
  type        = "os_version"
  description = "All fields populated"
  schedule    = "24h"
  expiration  = "25h"

  match {
    platform = "linux"
  }

  input {
    version            = "22.4.0"
    operator           = ">="
    os_distro_name     = "ubuntu"
    os_distro_revision = "22.4.0"
    os_version_extra   = "(LTS)"
  }
}

# Resource with empty/null optional fields
resource "cloudflare_device_posture_rule" "with_nulls" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix}-with-nulls"
  type        = "firewall"
  description = null
  expiration  = null
  schedule    = "5m"

  match {
    platform = "windows"
  }

  input {
    enabled = true
  }
}

# ============================================================================
# Original Test Cases (Comprehensive Coverage)
# ============================================================================

# Test case 1: Basic os_version rule with input and match
resource "cloudflare_device_posture_rule" "basic" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix}-tf-test-posture-basic"
  type        = "os_version"
  description = "Device posture rule for corporate devices."
  schedule    = "24h"
  expiration  = "25h"

  match {
    platform = "linux"
  }

  input {
    version            = "1.0.0"
    operator           = "<"
    os_distro_name     = "ubuntu"
    os_distro_revision = "1.0.0"
    os_version_extra   = "(a)"
  }
}

# Test case 2: Firewall rule with enabled input
resource "cloudflare_device_posture_rule" "firewall" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-tf-test-firewall"
  type       = "firewall"
  schedule   = "5m"

  match {
    platform = "windows"
  }

  input {
    enabled = true
  }
}

# Test case 3: Disk encryption with check_disks (Set->List conversion)
resource "cloudflare_device_posture_rule" "disk_encryption" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-tf-test-disk"
  type       = "disk_encryption"
  schedule   = "5m"

  match {
    platform = "windows"
  }

  input {
    check_disks = ["C:", "D:"]
    require_all = true
  }
}

# Test case 4: Multiple platforms (multiple match blocks)
resource "cloudflare_device_posture_rule" "multi_platform" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-tf-test-multi"
  type       = "firewall"
  schedule   = "5m"

  match {
    platform = "windows"
  }

  match {
    platform = "mac"
  }

  match {
    platform = "linux"
  }

  input {
    enabled = true
  }
}

# Test case 5: Application rule with path and running (removed attribute)
resource "cloudflare_device_posture_rule" "application" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-tf-test-application"
  type       = "application"
  schedule   = "30m"

  match {
    platform = "linux"
  }

  input {
    path    = "/usr/bin/security-app"
    sha256  = "abc123def456"
    running = true
  }
}

# Test case 6: Domain joined rule
resource "cloudflare_device_posture_rule" "domain_joined" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix}-tf-test-domain-joined"
  type       = "domain_joined"
  schedule   = "5m"

  match {
    platform = "windows"
  }

  input {
    domain = "example.com"
  }
}
