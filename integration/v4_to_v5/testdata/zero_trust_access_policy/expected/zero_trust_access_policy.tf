# Integration Test: zero_trust_access_policy
# Comprehensive test covering all Terraform patterns (V4 format)

# Pattern Group 1: Variables & Locals
variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare Domain"
  type        = string
}

locals {
  name_prefix    = "cftftest"
  common_account = var.cloudflare_account_id
  policy_names   = ["dev", "staging", "prod"]
  enable_test    = true
  enable_demo    = false
}

# NOTE: In v5, cloudflare_zero_trust_access_policy are account-level policies.
# The migration removes zone_id and application_id fields from v4 policies.
# Application-specific policies (with application_id in v4) cannot be migrated
# as they use different API endpoints and are fundamentally different resources.

# Basic test cases
resource "cloudflare_zero_trust_access_policy" "example" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-example-policy"
  decision   = "allow"
  session_duration = "24h"

  include = [{ everyone = {} }]
  approval_groups = [{
    approvals_needed = 1
    email_addresses  = ["approver@example.com"]
  }]
}

resource "cloudflare_zero_trust_access_policy" "complex" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-complex-policy"
  decision   = "allow"
  session_duration = "24h"

  include = [{ email = { email = "user@example.com" } },
    { email = { email = "admin@example.com" } },
  { email_domain = { domain = "example.com" } }]

  exclude = [{ ip = { ip = "192.168.1.1/32" } },
  { ip = { ip = "10.0.0.0/8" } }]

  require = [{ email = { email = "required@example.com" } }]
}

# Pattern Group 2: for_each with Maps (3-5 resources)
resource "cloudflare_zero_trust_access_policy" "map_example" {
  for_each = {
    "api" = {
      decision = "allow"
    }
    "web" = {
      decision = "allow"
    }
    "admin" = {
      decision = "allow"
    }
    "readonly" = {
      decision = "allow"
    }
  }

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-map-${each.key}-policy"
  decision   = each.value.decision
  session_duration = "24h"

  include = [{ email = { email = "@example.com" } }]
}

# Pattern Group 3: for_each with Sets (3-5 items)
resource "cloudflare_zero_trust_access_policy" "set_example" {
  for_each = toset(["alpha", "beta", "gamma", "delta", "epsilon"])

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-set-${each.key}"
  decision   = "allow"
  session_duration = "24h"

  include = [{ email = { email = "@example.com" } }]
}

# Pattern Group 4: count-based Resources (at least 3)
resource "cloudflare_zero_trust_access_policy" "counted" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-counted-${count.index}"
  decision   = "allow"
  session_duration = "24h"

  include = [{ ip = { ip = "10.0.${count.index}.0/24" } }]
}

# Pattern Group 5: Conditional Creation
resource "cloudflare_zero_trust_access_policy" "conditional_enabled" {
  count = local.enable_test ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-enabled"
  decision   = "allow"
  session_duration = "24h"

  include = [{ everyone = {} }]
}

resource "cloudflare_zero_trust_access_policy" "conditional_disabled" {
  count = local.enable_demo ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-disabled"
  decision   = "deny"
  session_duration = "24h"

  include = [{ everyone = {} }]
}

# Pattern Group 6: Terraform Functions
resource "cloudflare_zero_trust_access_policy" "with_functions" {
  account_id = var.cloudflare_account_id
  name       = join("-", [local.name_prefix, "functions", "test"])
  decision   = "allow"
  session_duration = "24h"

  include = [{ email = { email = "function1@example.com" } },
  { email = { email = "function2@example.com" } }]
}

# Pattern Group 7: Lifecycle Meta-Arguments
resource "cloudflare_zero_trust_access_policy" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-test"
  decision   = "allow"
  session_duration = "24h"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [name]
  }

  include = [{ everyone = {} }]
}

resource "cloudflare_zero_trust_access_policy" "with_prevent_destroy" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-prevent-destroy"
  decision   = "allow"
  session_duration = "24h"

  lifecycle {
    prevent_destroy = false
  }

  include = [{ email = { email = "protected@example.com" } }]
}

# Pattern Group 8: Edge Cases

# Minimal resource (only required fields)
resource "cloudflare_zero_trust_access_policy" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal"
  decision   = "allow"
  session_duration = "24h"

  include = [{ everyone = {} }]
}

# Maximal resource (all optional fields populated)
resource "cloudflare_zero_trust_access_policy" "maximal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-maximal"
  decision   = "allow"
  session_duration = "24h"

  include = [{ email = { email = "maximal1@example.com" } },
    { email = { email = "maximal2@example.com" } },
    { ip = { ip = "203.0.113.0/24" } },
    { email_domain = { domain = "maximal.example.com" } },
    { geo = { country_code = "US" } },
  { geo = { country_code = "CA" } }]

  exclude = [{ email = { email = "blocked@example.com" } },
  { ip = { ip = "203.0.113.100/32" } }]

  require = [{ email = { email = "required@example.com" } }]
  approval_groups = [{
    approvals_needed = 2
    email_addresses  = ["approver1@example.com", "approver2@example.com"]
  }]
}

# Policy with common_name
resource "cloudflare_zero_trust_access_policy" "with_common_name" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-common-name"
  decision   = "allow"
  session_duration = "24h"

  include = [{ common_name = { common_name = "device1.example.com" } }]
}

# Policy with auth_method
resource "cloudflare_zero_trust_access_policy" "with_auth_method" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-auth-method"
  decision   = "allow"
  session_duration = "24h"

  include = [{ auth_method = { auth_method = "swk" } }]
}

# Policy with login_method
resource "cloudflare_zero_trust_access_policy" "with_login_method" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-login-method"
  decision   = "allow"
  session_duration = "24h"

  include = [{ login_method = { id = "otp" } },
  { login_method = { id = "warp" } }]
}

# Policy with any_valid_service_token
resource "cloudflare_zero_trust_access_policy" "with_service_token" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-service-token"
  decision   = "allow"
  session_duration = "24h"

  include = [{ any_valid_service_token = {} }]
}

# Deny policy
resource "cloudflare_zero_trust_access_policy" "deny_policy" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-deny"
  decision   = "deny"
  session_duration = "24h"

  include = [{ ip = { ip = "198.51.100.0/24" } }]
}

# Bypass policy
resource "cloudflare_zero_trust_access_policy" "bypass_policy" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bypass"
  decision   = "bypass"
  session_duration = "24h"

  include = [{ ip = { ip = "192.0.2.0/24" } }]
}
