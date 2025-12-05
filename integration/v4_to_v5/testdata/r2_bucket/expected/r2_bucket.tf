# ============================================================================
# Standard Variables (Auto-provided by test infrastructure)
# ============================================================================
variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# ============================================================================
# Pattern Group 1: Locals for Common Values
# ============================================================================
locals {
  common_account = var.cloudflare_account_id
  name_prefix    = "cftftest"
  locations = {
    "wnam" = "WNAM"
    "enam" = "ENAM"
    "weur" = "WEUR"
    "eeur" = "EEUR"
    "apac" = "APAC"
    "oc"   = "OC"
  }
  enable_feature = true
  enable_test    = false
}

# ============================================================================
# Pattern Group 2: Basic Resource Configurations
# ============================================================================

# Test Case 1: Minimal bucket with only required fields
resource "cloudflare_r2_bucket" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal-bucket"
}

# Test Case 2: Bucket with all locations tested
resource "cloudflare_r2_bucket" "location_wnam" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bucket-wnam"
  location   = "WNAM"
}

resource "cloudflare_r2_bucket" "location_enam" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bucket-enam"
  location   = "ENAM"
}

resource "cloudflare_r2_bucket" "location_weur" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bucket-weur"
  location   = "WEUR"
}

resource "cloudflare_r2_bucket" "location_eeur" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bucket-eeur"
  location   = "EEUR"
}

resource "cloudflare_r2_bucket" "location_apac" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bucket-apac"
  location   = "APAC"
}

resource "cloudflare_r2_bucket" "location_oc" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bucket-oc"
  location   = "OC"
}

# ============================================================================
# Pattern Group 3: for_each with Maps
# ============================================================================

resource "cloudflare_r2_bucket" "map_example" {
  for_each = {
    "data" = {
      name     = "${local.name_prefix}-data-bucket"
      location = "WNAM"
    }
    "logs" = {
      name     = "${local.name_prefix}-logs-bucket"
      location = "ENAM"
    }
    "backups" = {
      name     = "${local.name_prefix}-backups-bucket"
      location = "WEUR"
    }
  }

  account_id = var.cloudflare_account_id
  name       = each.value.name
  location   = each.value.location
}

# ============================================================================
# Pattern Group 4: for_each with Sets
# ============================================================================

resource "cloudflare_r2_bucket" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta",
  ])

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-set-${each.value}-bucket"
}

# ============================================================================
# Pattern Group 5: count-based Resources
# ============================================================================

resource "cloudflare_r2_bucket" "counted" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-counted-bucket-${count.index}"
  location   = count.index == 0 ? "WNAM" : count.index == 1 ? "EEUR" : "APAC"
}

# ============================================================================
# Pattern Group 6: Conditional Creation
# ============================================================================

resource "cloudflare_r2_bucket" "conditional_enabled" {
  count = local.enable_feature ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-enabled-bucket"
  location   = "WNAM"
}

resource "cloudflare_r2_bucket" "conditional_disabled" {
  count = local.enable_test ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-disabled-bucket"
  location   = "EEUR"
}

# ============================================================================
# Pattern Group 7: Terraform Functions
# ============================================================================

resource "cloudflare_r2_bucket" "with_functions" {
  account_id = var.cloudflare_account_id
  name       = join("-", [local.name_prefix, "function", "test"])
  location   = lookup(local.locations, "wnam", "WNAM")
}

resource "cloudflare_r2_bucket" "with_interpolation" {
  account_id = local.common_account
  name       = "${local.name_prefix}-interpolated-bucket"
  location   = local.locations["eeur"]
}

# ============================================================================
# Pattern Group 8: Lifecycle Meta-Arguments
# ============================================================================

resource "cloudflare_r2_bucket" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-test-bucket"
  location   = "WNAM"

  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_r2_bucket" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-prevent-destroy-bucket"
  location   = "EEUR"

  lifecycle {
    prevent_destroy = false
  }
}

# ============================================================================
# Pattern Group 9: Edge Cases
# ============================================================================

# Special characters in bucket names (hyphens and numbers)
resource "cloudflare_r2_bucket" "special_chars" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bucket-with-dashes-and-numbers-123"
}

# Long bucket name (testing near max length - 63 chars max)
resource "cloudflare_r2_bucket" "long_name" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-very-long-bucket-name-for-testing-migration-lim"
}

# Bucket name with all allowed characters (lowercase, numbers, hyphens)
resource "cloudflare_r2_bucket" "all_chars" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bucket-name-with-all-valid-chars-123-test"
  location   = "WNAM"
}
