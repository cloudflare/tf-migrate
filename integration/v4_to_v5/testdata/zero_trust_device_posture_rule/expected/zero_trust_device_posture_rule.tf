variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# ============================================================================
# Pattern Group 1: Variables & Locals
# ============================================================================

locals {
  common_account        = var.cloudflare_account_id
  name_prefix = "cftftest"
  default_schedule      = "24h"
  enable_firewall_rules = true
  enable_test_rules     = false
}

# ============================================================================
# Pattern Group 2: for_each with Maps (5 resources)
# ============================================================================

resource "cloudflare_zero_trust_device_posture_rule" "map_example" {
  for_each = {
    "prod" = {
      account_id = var.cloudflare_account_id
      name = "cftftest-prod-posture-rule"
      type       = "os_version"
      schedule   = "24h"
      platform   = "linux"
      version    = "20.4.0"
    }
    "staging" = {
      account_id = var.cloudflare_account_id
      name = "cftftest-staging-posture-rule"
      type       = "firewall"
      schedule   = "12h"
      platform   = "windows"
      version    = "10.0.0"
    }
    "dev" = {
      account_id = var.cloudflare_account_id
      name = "cftftest-dev-posture-rule"
      type       = "os_version"
      schedule   = "6h"
      platform   = "mac"
      version    = "12.0.0"
    }
    "qa" = {
      account_id = var.cloudflare_account_id
      name = "cftftest-qa-posture-rule"
      type       = "firewall"
      schedule   = "12h"
      platform   = "linux"
      version    = "1.0.0"
    }
    "perf" = {
      account_id = var.cloudflare_account_id
      name = "cftftest-perf-posture-rule"
      type       = "os_version"
      schedule   = "24h"
      platform   = "windows"
      version    = "11.0.0"
    }
  }

  account_id = each.value.account_id
  name       = each.value.name
  type       = each.value.type
  schedule   = each.value.schedule


  input = {
    version  = each.value.version
    operator = ">="
  }
  match = [
    {
      platform = each.value.platform
    }
  ]
}

# ============================================================================
# Pattern Group 3: for_each with Sets (4 items)
# ============================================================================

resource "cloudflare_zero_trust_device_posture_rule" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  account_id = var.cloudflare_account_id
  name = "cftftest-set-${each.value}-rule"
  type       = "firewall"
  schedule   = "5m"


  input = {
    enabled = true
  }
  match = [{
    platform = "linux"
  }]
}

# ============================================================================
# Pattern Group 4: count-based Resources (3 instances)
# ============================================================================

resource "cloudflare_zero_trust_device_posture_rule" "counted" {
  count = 3

  account_id  = var.cloudflare_account_id
  name = "cftftest-counted-rule-${count.index}"
  type        = "os_version"
  schedule    = "24h"
  description = "This is posture rule number ${count.index}"


  input = {
    version  = "1.${count.index}.0"
    operator = "=="
  }
  match = [{
    platform = "linux"
  }]
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

resource "cloudflare_zero_trust_device_posture_rule" "conditional_enabled" {
  count = local.enable_firewall_rules ? 1 : 0

  account_id  = var.cloudflare_account_id
  name = "cftftest-conditional-firewall-enabled"
  type        = "firewall"
  schedule    = "12h"
  description = "Conditionally enabled firewall rule"


  input = {
    enabled = true
  }
  match = [{
    platform = "windows"
  }]
}

resource "cloudflare_zero_trust_device_posture_rule" "conditional_disabled" {
  count = local.enable_test_rules ? 1 : 0

  account_id  = var.cloudflare_account_id
  name = "cftftest-conditional-test-disabled"
  type        = "os_version"
  schedule    = "6h"
  description = "Conditionally disabled test rule"


  input = {
    version  = "1.0.0"
    operator = ">"
  }
  match = [{
    platform = "linux"
  }]
}

# ============================================================================
# Pattern Group 7: Terraform Functions
# ============================================================================

resource "cloudflare_zero_trust_device_posture_rule" "with_functions" {
  account_id = local.common_account

  # join() function
  name = join("-", [local.name_prefix, "function", "example"])

  type        = "os_version"
  schedule    = local.default_schedule
  description = "Rule for account ${var.cloudflare_account_id}"


  input = {
    version  = "22.4.0"
    operator = ">="
  }
  match = [{
    platform = "linux"
  }]
}

resource "cloudflare_zero_trust_device_posture_rule" "with_interpolation" {
  account_id  = var.cloudflare_account_id
  name = "cftftest-rule-for-account-${var.cloudflare_account_id}"
  type        = "firewall"
  schedule    = "5m"
  description = "Interpolated rule name"


  input = {
    enabled = true
  }
  match = [{
    platform = "windows"
  }]
}

# ============================================================================
# Pattern Group 8: Lifecycle Meta-Arguments
# ============================================================================

resource "cloudflare_zero_trust_device_posture_rule" "with_lifecycle" {
  account_id  = var.cloudflare_account_id
  name = "cftftest-lifecycle-test-rule"
  type        = "os_version"
  schedule    = "24h"
  description = "Rule with lifecycle arguments"



  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
  input = {
    version  = "20.4.0"
    operator = ">="
  }
  match = [{
    platform = "linux"
  }]
}

resource "cloudflare_zero_trust_device_posture_rule" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  name = "cftftest-prevent-destroy-rule"
  type       = "firewall"
  schedule   = "12h"



  lifecycle {
    prevent_destroy = false # Set to false for testing
  }
  input = {
    enabled = true
  }
  match = [{
    platform = "windows"
  }]
}

# ============================================================================
# Pattern Group 9: Edge Cases
# ============================================================================

# Minimal resource (only required fields)
resource "cloudflare_zero_trust_device_posture_rule" "minimal" {
  account_id = var.cloudflare_account_id
  name = "cftftest-minimal-rule"
  type       = "firewall"
  schedule   = "5m"


  input = {
    enabled = true
  }
  match = [{
    platform = "linux"
  }]
}

# Maximal resource (all fields populated)
resource "cloudflare_zero_trust_device_posture_rule" "maximal" {
  account_id  = var.cloudflare_account_id
  name = "cftftest-maximal-rule"
  type        = "os_version"
  description = "All fields populated"
  schedule    = "24h"
  expiration  = "24h"


  input = {
    version            = "22.4.0"
    operator           = ">="
    os_distro_name     = "ubuntu"
    os_distro_revision = "22.4.0"
    os_version_extra   = "(LTS)"
  }
  match = [{
    platform = "linux"
  }]
}

# Resource with empty/null optional fields
resource "cloudflare_zero_trust_device_posture_rule" "with_nulls" {
  account_id  = var.cloudflare_account_id
  name = "cftftest-with-nulls"
  type        = "firewall"
  description = null
  expiration  = null
  schedule    = "5m"


  input = {
    enabled = true
  }
  match = [{
    platform = "windows"
  }]
}

# ============================================================================
# Original Test Cases (Comprehensive Coverage)
# ============================================================================

# Test case 1: Basic os_version rule with input and match
resource "cloudflare_zero_trust_device_posture_rule" "basic" {
  account_id  = var.cloudflare_account_id
  name = "cftftest-tf-test-posture-basic"
  type        = "os_version"
  description = "Device posture rule for corporate devices."
  schedule    = "24h"
  expiration  = "24h"


  input = {
    version            = "1.0.0"
    operator           = "<"
    os_distro_name     = "ubuntu"
    os_distro_revision = "1.0.0"
    os_version_extra   = "(a)"
  }
  match = [{
    platform = "linux"
  }]
}

# Test case 2: Firewall rule with enabled input
resource "cloudflare_zero_trust_device_posture_rule" "firewall" {
  account_id = var.cloudflare_account_id
  name = "cftftest-tf-test-firewall"
  type       = "firewall"
  schedule   = "5m"


  input = {
    enabled = true
  }
  match = [{
    platform = "windows"
  }]
}

# Test case 3: Disk encryption with check_disks (Set->List conversion)
resource "cloudflare_zero_trust_device_posture_rule" "disk_encryption" {
  account_id = var.cloudflare_account_id
  name = "cftftest-tf-test-disk"
  type       = "disk_encryption"
  schedule   = "5m"


  input = {
    check_disks = ["C:", "D:"]
    require_all = true
  }
  match = [{
    platform = "windows"
  }]
}

# Test case 4: Multiple platforms (multiple match blocks)
resource "cloudflare_zero_trust_device_posture_rule" "multi_platform" {
  account_id = var.cloudflare_account_id
  name = "cftftest-tf-test-multi"
  type       = "firewall"
  schedule   = "5m"




  input = {
    enabled = true
  }
  match = [{
    platform = "windows"
    }, {
    platform = "mac"
    }, {
    platform = "linux"
  }]
}

# Test case 5: Application rule with path and running (removed attribute)
resource "cloudflare_zero_trust_device_posture_rule" "application" {
  account_id = var.cloudflare_account_id
  name = "cftftest-tf-test-application"
  type       = "application"
  schedule   = "30m"


  input = {
    path   = "/usr/bin/security-app"
    sha256 = "abc123def456"
  }
  match = [{
    platform = "linux"
  }]
}
