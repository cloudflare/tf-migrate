# Terraform Provider v4 to v5 Migration Test
# Resource: cloudflare_turnstile_widget
# Comprehensive integration test covering all Terraform patterns

# Standard variables (auto-provided by E2E infrastructure)
variable "cloudflare_account_id" {
  type        = string
}

variable "cloudflare_zone_id" {
  type        = string
}

variable "cloudflare_domain" {
  type        = string
}

# Locals for DRY testing
locals {
  name_prefix = "cftftest"
  account_id  = var.cloudflare_account_id

  # Common domains for testing
  test_domains = {
    prod    = "prod.cf-tf-test.com"
    staging = "staging.cf-tf-test.com"
    dev     = "dev.cf-tf-test.com"
  }

  # Widget modes for iteration
  widget_modes = ["managed", "invisible", "non-interactive"]
}

# ==================================================
# Pattern Group 1: Basic Resources (Edge Cases)
# ==================================================

# Test Case 1: Minimal configuration (required fields only)
resource "cloudflare_turnstile_widget" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal"
  domains    = toset(["minimal.cf-tf-test.com"])
  mode       = "managed"
}

# Test Case 2: Maximal configuration (all optional fields)
resource "cloudflare_turnstile_widget" "maximal" {
  account_id     = var.cloudflare_account_id
  name           = "${local.name_prefix}-maximal"
  domains        = toset(["maximal.cf-tf-test.com", "max2.cf-tf-test.com"])
  mode           = "invisible"
  region         = "world"
  bot_fight_mode = false
  offlabel       = true
}

# Test Case 3: With explicit defaults
resource "cloudflare_turnstile_widget" "with_defaults" {
  account_id     = var.cloudflare_account_id
  name           = "${local.name_prefix}-defaults"
  domains        = toset(["defaults.cf-tf-test.com"])
  mode           = "non-interactive"
  region         = "world"
  bot_fight_mode = false
  offlabel       = false
}

# ==================================================
# Pattern Group 2: for_each with Maps
# ==================================================

# Test Case 4-6: for_each with map (3 instances)
resource "cloudflare_turnstile_widget" "foreach_map" {
  for_each = {
    "prod" = {
      domain = local.test_domains.prod
      mode   = "managed"
    }
    "staging" = {
      domain = local.test_domains.staging
      mode   = "invisible"
    }
    "dev" = {
      domain = local.test_domains.dev
      mode   = "non-interactive"
    }
  }

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-map-${each.key}"
  domains    = toset([each.value.domain])
  mode       = each.value.mode
}

# ==================================================
# Pattern Group 3: for_each with Sets
# ==================================================

# Test Case 7-10: for_each with set (4 instances)
resource "cloudflare_turnstile_widget" "foreach_set" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-set-${each.value}"
  domains    = toset(["${each.value}.cf-tf-test.com"])
  mode       = "managed"
}

# ==================================================
# Pattern Group 4: count-based Resources
# ==================================================

# Test Case 11-13: count-based (3 instances)
resource "cloudflare_turnstile_widget" "counted" {
  count = 3

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-count-${count.index}"
  domains     = toset(["count${count.index}.cf-tf-test.com"])
  mode        = local.widget_modes[count.index]
}

# ==================================================
# Pattern Group 5: Conditional Creation
# ==================================================

locals {
  enable_production  = true
  enable_development = false
}

# Test Case 14: Conditional enabled (will be created)
resource "cloudflare_turnstile_widget" "conditional_enabled" {
  count = local.enable_production ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-enabled"
  domains    = toset(["enabled.cf-tf-test.com"])
  mode       = "managed"
}

# Test Case 15: Conditional disabled (will NOT be created - no instance in state)
resource "cloudflare_turnstile_widget" "conditional_disabled" {
  count = local.enable_development ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-disabled"
  domains    = toset(["disabled.cf-tf-test.com"])
  mode       = "invisible"
}

# ==================================================
# Pattern Group 6: Terraform Functions
# ==================================================

# Test Case 16: Using join() function
resource "cloudflare_turnstile_widget" "with_join" {
  account_id  = var.cloudflare_account_id
  name        = join("-", [local.name_prefix, "join", "test"])
  domains     = toset(["join.cf-tf-test.com"])
  mode        = "managed"
}

# Test Case 17: String interpolation
resource "cloudflare_turnstile_widget" "with_interpolation" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-interp"
  domains     = toset(["interp.cf-tf-test.com"])
  mode        = "invisible"
}

# ==================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# ==================================================

# Test Case 18: Lifecycle with create_before_destroy
resource "cloudflare_turnstile_widget" "with_lifecycle_cbd" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-cbd"
  domains    = toset(["lifecycle-cbd.cf-tf-test.com"])
  mode       = "managed"

  lifecycle {
    create_before_destroy = true
  }
}

# Test Case 19: Lifecycle with ignore_changes
resource "cloudflare_turnstile_widget" "with_lifecycle_ignore" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-lifecycle-ignore"
  domains     = toset(["lifecycle-ignore.cf-tf-test.com"])
  mode        = "invisible"

  lifecycle {
    ignore_changes = [name]
  }
}

# Test Case 20: Lifecycle with prevent_destroy
resource "cloudflare_turnstile_widget" "with_lifecycle_prevent" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-prevent"
  domains    = toset(["lifecycle-prevent.cf-tf-test.com"])
  mode       = "non-interactive"

  lifecycle {
    prevent_destroy = false  # Set to false for testing purposes
  }
}

# ==================================================
# Pattern Group 8: Multiple Domains
# ==================================================

# Test Case 21: Multiple domains (2)
resource "cloudflare_turnstile_widget" "multi_domain_2" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-multi-2"
  domains    = toset(["multi1.cf-tf-test.com", "multi2.cf-tf-test.com"])
  mode       = "managed"
}

# Test Case 22: Multiple domains (5)
resource "cloudflare_turnstile_widget" "multi_domain_5" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-multi-5"
  domains = toset([
    "md1.cf-tf-test.com",
    "md2.cf-tf-test.com",
    "md3.cf-tf-test.com",
    "md4.cf-tf-test.com",
    "md5.cf-tf-test.com"
  ])
  mode = "invisible"
}

# ==================================================
# Pattern Group 9: Variable References
# ==================================================

# Test Case 23: Variable reference (toset with variable - should preserve)
resource "cloudflare_turnstile_widget" "with_var_domains" {
  account_id = local.account_id
  name       = "${local.name_prefix}-var-domains"
  domains    = toset([local.test_domains.prod])
  mode       = "managed"
}

# ==================================================
# Pattern Group 10: All Widget Modes
# ==================================================

# Test Case 24: managed mode
resource "cloudflare_turnstile_widget" "mode_managed" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-mode-managed"
  domains    = toset(["managed.cf-tf-test.com"])
  mode       = "managed"
}

# Test Case 25: invisible mode
resource "cloudflare_turnstile_widget" "mode_invisible" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-mode-invisible"
  domains    = toset(["invisible.cf-tf-test.com"])
  mode       = "invisible"
}

# Test Case 26: non-interactive mode
resource "cloudflare_turnstile_widget" "mode_noninteractive" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-mode-noninteractive"
  domains    = toset(["noninteractive.cf-tf-test.com"])
  mode       = "non-interactive"
}

# ==================================================
# Summary: Total 26 resource declarations
# Actual instances: 23 (conditional_disabled won't be created)
# ==================================================
