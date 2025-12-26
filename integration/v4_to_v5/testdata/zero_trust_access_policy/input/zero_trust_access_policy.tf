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

locals {
  name_prefix    = "cftftest"
  common_account = var.cloudflare_account_id
  policy_names   = ["dev", "staging", "prod"]
  enable_test    = true
  enable_demo    = false
}

# Create test application (required for policies)
# Using SaaS type which doesn't require domain validation
resource "cloudflare_access_application" "test_app" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-test-app"
  type       = "saas"

  saas_app {
    consumer_service_url = "https://example.com/saml/consume"
    name_id_format       = "email"
    sp_entity_id         = "example-sp-entity"
  }
}

# Basic test cases
resource "cloudflare_access_policy" "example" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 1
  name           = "${local.name_prefix}-example-policy"
  decision       = "allow"

  approval_group {
    approvals_needed = 1
    email_addresses  = ["approver@example.com"]
  }

  include {
    everyone = true
  }
}

resource "cloudflare_access_policy" "complex" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 2
  name           = "${local.name_prefix}-complex-policy"
  decision       = "allow"

  include {
    email        = ["user@example.com", "admin@example.com"]
    email_domain = ["example.com"]
  }

  exclude {
    ip = ["192.168.1.1", "10.0.0.0/8"]
  }

  require {
    email = ["required@example.com"]
  }
}

# Pattern Group 2: for_each with Maps (3-5 resources)
resource "cloudflare_access_policy" "map_example" {
  for_each = {
    "api" = {
      decision   = "allow"
      precedence = 10
    }
    "web" = {
      decision   = "allow"
      precedence = 11
    }
    "admin" = {
      decision   = "allow"
      precedence = 12
    }
    "readonly" = {
      decision   = "allow"
      precedence = 13
    }
  }

  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = each.value.precedence
  name           = "${local.name_prefix}-map-${each.key}-policy"
  decision       = each.value.decision

  include {
    email = ["@example.com"]
  }
}

# Pattern Group 3: for_each with Sets (3-5 items)
resource "cloudflare_access_policy" "set_example" {
  for_each = {
    "alpha" = {
      precedence = 20
    }
    "beta" = {
      precedence = 21
    }
    "gamma" = {
      precedence = 22
    }
    "delta" = {
      precedence = 23
    }
    "epsilon" = {
      precedence = 24
    }
  }

  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = each.value.precedence
  name           = "${local.name_prefix}-set-${each.key}"
  decision       = "allow"

  include {
    email = ["@example.com"]
  }
}

# Pattern Group 4: count-based Resources (at least 3)
resource "cloudflare_access_policy" "counted" {
  count = 3

  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 30 + count.index
  name           = "${local.name_prefix}-counted-${count.index}"
  decision       = "allow"

  include {
    ip = ["10.0.${count.index}.0/24"]
  }
}

# Pattern Group 5: Conditional Creation
resource "cloudflare_access_policy" "conditional_enabled" {
  count = local.enable_test ? 1 : 0

  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 40
  name           = "${local.name_prefix}-conditional-enabled"
  decision       = "allow"

  include {
    everyone = true
  }
}

resource "cloudflare_access_policy" "conditional_disabled" {
  count = local.enable_demo ? 1 : 0

  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 41
  name           = "${local.name_prefix}-conditional-disabled"
  decision       = "deny"

  include {
    everyone = true
  }
}

# Pattern Group 6: Terraform Functions
resource "cloudflare_access_policy" "with_functions" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 50
  name           = join("-", [local.name_prefix, "functions", "test"])
  decision       = "allow"

  include {
    email = ["function1@example.com", "function2@example.com"]
  }
}

# Pattern Group 7: Lifecycle Meta-Arguments
resource "cloudflare_access_policy" "with_lifecycle" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 60
  name           = "${local.name_prefix}-lifecycle-test"
  decision       = "allow"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [name]
  }

  include {
    everyone = true
  }
}

resource "cloudflare_access_policy" "with_prevent_destroy" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 61
  name           = "${local.name_prefix}-prevent-destroy"
  decision       = "allow"

  lifecycle {
    prevent_destroy = false
  }

  include {
    email = ["protected@example.com"]
  }
}

# Pattern Group 8: Edge Cases

# Minimal resource (only required fields)
resource "cloudflare_access_policy" "minimal" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 70
  name           = "${local.name_prefix}-minimal"
  decision       = "allow"

  include {
    everyone = true
  }
}

# Maximal resource (all optional fields populated)
resource "cloudflare_access_policy" "maximal" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 71
  name           = "${local.name_prefix}-maximal"
  decision       = "allow"

  approval_group {
    approvals_needed = 2
    email_addresses  = ["approver1@example.com", "approver2@example.com"]
  }

  include {
    email        = ["maximal1@example.com", "maximal2@example.com"]
    email_domain = ["maximal.example.com"]
    geo          = ["US", "CA"]
    ip           = ["203.0.113.0/24"]
  }

  exclude {
    email = ["blocked@example.com"]
    ip    = ["203.0.113.100"]
  }

  require {
    email = ["required@example.com"]
  }
}

# Policy with common_name
resource "cloudflare_access_policy" "with_common_name" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 72
  name           = "${local.name_prefix}-common-name"
  decision       = "allow"

  include {
    common_name = "device1.example.com"
  }
}

# Policy with auth_method
resource "cloudflare_access_policy" "with_auth_method" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 73
  name           = "${local.name_prefix}-auth-method"
  decision       = "allow"

  include {
    auth_method = "swk"
  }
}

# Policy with login_method
resource "cloudflare_access_policy" "with_login_method" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 74
  name           = "${local.name_prefix}-login-method"
  decision       = "allow"

  include {
    login_method = ["otp", "warp"]
  }
}

# Policy with any_valid_service_token
resource "cloudflare_access_policy" "with_service_token" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 75
  name           = "${local.name_prefix}-service-token"
  decision       = "allow"

  include {
    any_valid_service_token = true
  }
}

# Deny policy
resource "cloudflare_access_policy" "deny_policy" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 80
  name           = "${local.name_prefix}-deny"
  decision       = "deny"

  include {
    ip = ["198.51.100.0/24"]
  }
}

# Bypass policy
resource "cloudflare_access_policy" "bypass_policy" {
  account_id     = var.cloudflare_account_id
  application_id = cloudflare_access_application.test_app.id
  precedence     = 81
  name           = "${local.name_prefix}-bypass"
  decision       = "bypass"

  include {
    ip = ["192.0.2.0/24"]
  }
}
