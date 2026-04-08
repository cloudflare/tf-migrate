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











# Pattern Group 8: Edge Cases











# ============================================================
# Research team issue reproductions (TKT-002 through TKT-006)
# ============================================================










# TKT-003: application_id + precedence must be removed from policy
# (mirrors research team's app_azul_mtc_worker.tf)
# In v4, application-scoped policies had application_id + precedence.
# In v5, application_id and precedence are removed; the binding is done
# via the cloudflare_zero_trust_access_application.policies block.
# tf-migrate removes application_id and precedence with a warning.
resource "cloudflare_zero_trust_access_application" "test_app" {
  account_id                 = var.cloudflare_account_id
  name                       = "${local.name_prefix}-test-app"
  domain                     = "test.${var.cloudflare_domain}"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}


# ============================================================================
# BUGS-2006: Already-v5-named resources with block syntax not converted
# These resources already have the v5 name but nested blocks are still in
# v4 block syntax — tf-migrate must still convert them.
# ============================================================================

# Already v5-named: simple include block with email
resource "cloudflare_zero_trust_access_policy" "bugs2006_email" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bugs2006-email"
  decision   = "allow"

  include = [{ email = { email = "sara@example.com" } }]
}

# Already v5-named: multiple condition blocks
resource "cloudflare_zero_trust_access_policy" "bugs2006_multi_condition" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bugs2006-multi"
  decision   = "allow"



  include = [{ email_domain = { domain = "cloudflare.com" } }]
  exclude = [{ geo = { country_code = "CN" } },
  { geo = { country_code = "RU" } }]
  require = [{ certificate = {} }]
}

# Already v5-named: everyone boolean condition
resource "cloudflare_zero_trust_access_policy" "bugs2006_everyone" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bugs2006-everyone"
  decision   = "allow"

  include = [{ everyone = {} }]
}





# Basic test cases
resource "cloudflare_zero_trust_access_policy" "example" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-example-policy"
  decision         = "allow"
  session_duration = "24h"


  approval_groups = [{
    approvals_needed = 1
    email_addresses  = ["approver@example.com"]
  }]
  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.example
  to   = cloudflare_zero_trust_access_policy.example
}

resource "cloudflare_zero_trust_access_policy" "complex" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-complex-policy"
  decision         = "allow"
  session_duration = "24h"



  include = [{ email = { email = "user@example.com" } },
    { email = { email = "admin@example.com" } },
  { email_domain = { domain = "example.com" } }]
  exclude = [{ ip = { ip = "192.168.1.1/32" } },
  { ip = { ip = "10.0.0.0/8" } }]
  require = [{ email = { email = "required@example.com" } }]
}

moved {
  from = cloudflare_access_policy.complex
  to   = cloudflare_zero_trust_access_policy.complex
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

  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-map-${each.key}-policy"
  decision         = each.value.decision
  session_duration = "24h"

  include = [{ email = { email = "@example.com" } }]
}

moved {
  from = cloudflare_access_policy.map_example
  to   = cloudflare_zero_trust_access_policy.map_example
}

# Pattern Group 3: for_each with Sets (3-5 items)
resource "cloudflare_zero_trust_access_policy" "set_example" {
  for_each = toset(["alpha", "beta", "gamma", "delta", "epsilon"])

  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-set-${each.key}"
  decision         = "allow"
  session_duration = "24h"

  include = [{ email = { email = "@example.com" } }]
}

moved {
  from = cloudflare_access_policy.set_example
  to   = cloudflare_zero_trust_access_policy.set_example
}

# Pattern Group 4: count-based Resources (at least 3)
resource "cloudflare_zero_trust_access_policy" "counted" {
  count = 3

  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-counted-${count.index}"
  decision         = "allow"
  session_duration = "24h"

  include = [{ ip = { ip = "10.0.${count.index}.0/24" } }]
}

moved {
  from = cloudflare_access_policy.counted
  to   = cloudflare_zero_trust_access_policy.counted
}

# Pattern Group 5: Conditional Creation
resource "cloudflare_zero_trust_access_policy" "conditional_enabled" {
  count = local.enable_test ? 1 : 0

  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-conditional-enabled"
  decision         = "allow"
  session_duration = "24h"

  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.conditional_enabled
  to   = cloudflare_zero_trust_access_policy.conditional_enabled
}

resource "cloudflare_zero_trust_access_policy" "conditional_disabled" {
  count = local.enable_demo ? 1 : 0

  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-conditional-disabled"
  decision         = "deny"
  session_duration = "24h"

  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.conditional_disabled
  to   = cloudflare_zero_trust_access_policy.conditional_disabled
}

# Pattern Group 6: Terraform Functions
resource "cloudflare_zero_trust_access_policy" "with_functions" {
  account_id       = var.cloudflare_account_id
  name             = join("-", [local.name_prefix, "functions", "test"])
  decision         = "allow"
  session_duration = "24h"

  include = [{ email = { email = "function1@example.com" } },
  { email = { email = "function2@example.com" } }]
}

moved {
  from = cloudflare_access_policy.with_functions
  to   = cloudflare_zero_trust_access_policy.with_functions
}

# Pattern Group 7: Lifecycle Meta-Arguments
resource "cloudflare_zero_trust_access_policy" "with_lifecycle" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-lifecycle-test"
  decision         = "allow"
  session_duration = "24h"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [name]
  }

  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.with_lifecycle
  to   = cloudflare_zero_trust_access_policy.with_lifecycle
}

resource "cloudflare_zero_trust_access_policy" "with_prevent_destroy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-prevent-destroy"
  decision         = "allow"
  session_duration = "24h"

  lifecycle {
    prevent_destroy = false
  }

  include = [{ email = { email = "protected@example.com" } }]
}

moved {
  from = cloudflare_access_policy.with_prevent_destroy
  to   = cloudflare_zero_trust_access_policy.with_prevent_destroy
}

# Minimal resource (only required fields)
resource "cloudflare_zero_trust_access_policy" "minimal" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-minimal"
  decision         = "allow"
  session_duration = "24h"

  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.minimal
  to   = cloudflare_zero_trust_access_policy.minimal
}

# Maximal resource (all optional fields populated)
resource "cloudflare_zero_trust_access_policy" "maximal" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-maximal"
  decision         = "allow"
  session_duration = "24h"




  approval_groups = [{
    approvals_needed = 2
    email_addresses  = ["approver1@example.com", "approver2@example.com"]
  }]
  include = [{ email = { email = "maximal1@example.com" } },
    { email = { email = "maximal2@example.com" } },
    { email_domain = { domain = "maximal.example.com" } },
    { geo = { country_code = "US" } },
    { geo = { country_code = "CA" } },
  { ip = { ip = "203.0.113.0/24" } }]
  exclude = [{ email = { email = "blocked@example.com" } },
  { ip = { ip = "203.0.113.100/32" } }]
  require = [{ email = { email = "required@example.com" } }]
}

moved {
  from = cloudflare_access_policy.maximal
  to   = cloudflare_zero_trust_access_policy.maximal
}

# Policy with common_name
resource "cloudflare_zero_trust_access_policy" "with_common_name" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-common-name"
  decision         = "allow"
  session_duration = "24h"

  include = [{ common_name = { common_name = "device1.example.com" } }]
}

moved {
  from = cloudflare_access_policy.with_common_name
  to   = cloudflare_zero_trust_access_policy.with_common_name
}

# Policy with common_names overflow array
resource "cloudflare_zero_trust_access_policy" "with_common_names" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-common-names"
  decision         = "allow"
  session_duration = "24h"

  include = [{ common_name = { common_name = "device2.example.com" } },
  { common_name = { common_name = "device3.example.com" } }]
}

moved {
  from = cloudflare_access_policy.with_common_names
  to   = cloudflare_zero_trust_access_policy.with_common_names
}

# Policy with connection_rules ssh structure (BUGS-2012)
resource "cloudflare_zero_trust_access_policy" "with_connection_rules" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-connection-rules"
  decision         = "allow"
  session_duration = "24h"


  connection_rules = {
    ssh = {
      usernames         = ["admin", "deploy"]
      allow_email_alias = true
    }
  }
  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.with_connection_rules
  to   = cloudflare_zero_trust_access_policy.with_connection_rules
}

# Policy with auth_method
resource "cloudflare_zero_trust_access_policy" "with_auth_method" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-auth-method"
  decision         = "allow"
  session_duration = "24h"

  include = [{ auth_method = { auth_method = "swk" } }]
}

moved {
  from = cloudflare_access_policy.with_auth_method
  to   = cloudflare_zero_trust_access_policy.with_auth_method
}

# Policy with login_method
resource "cloudflare_zero_trust_access_policy" "with_login_method" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-login-method"
  decision         = "allow"
  session_duration = "24h"

  include = [{ login_method = { id = "otp" } },
  { login_method = { id = "warp" } }]
}

moved {
  from = cloudflare_access_policy.with_login_method
  to   = cloudflare_zero_trust_access_policy.with_login_method
}

# Policy with any_valid_service_token
resource "cloudflare_zero_trust_access_policy" "with_service_token" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-service-token"
  decision         = "allow"
  session_duration = "24h"

  include = [{ any_valid_service_token = {} }]
}

moved {
  from = cloudflare_access_policy.with_service_token
  to   = cloudflare_zero_trust_access_policy.with_service_token
}

# Deny policy
resource "cloudflare_zero_trust_access_policy" "deny_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-deny"
  decision         = "deny"
  session_duration = "24h"

  include = [{ ip = { ip = "198.51.100.0/24" } }]
}

moved {
  from = cloudflare_access_policy.deny_policy
  to   = cloudflare_zero_trust_access_policy.deny_policy
}

# Bypass policy
resource "cloudflare_zero_trust_access_policy" "bypass_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-bypass"
  decision         = "bypass"
  session_duration = "24h"

  include = [{ ip = { ip = "192.0.2.0/24" } }]
}

moved {
  from = cloudflare_access_policy.bypass_policy
  to   = cloudflare_zero_trust_access_policy.bypass_policy
}

# TKT-002: include/exclude/require block → attribute list conversion
# TKT-005: email_domain from list to {domain = ...} object
resource "cloudflare_zero_trust_access_policy" "email_domain_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-email-domain"
  decision         = "allow"
  session_duration = "18h"

  include = [{ email_domain = { domain = "cloudflare.com" } }]
}

moved {
  from = cloudflare_access_policy.email_domain_policy
  to   = cloudflare_zero_trust_access_policy.email_domain_policy
}

# TKT-006: any_valid_service_token from bool to empty object {}
resource "cloudflare_zero_trust_access_policy" "any_service_token_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-any-service-token"
  decision         = "non_identity"
  session_duration = "18h"

  include = [{ any_valid_service_token = {} }]
}

moved {
  from = cloudflare_access_policy.any_service_token_policy
  to   = cloudflare_zero_trust_access_policy.any_service_token_policy
}

# TKT-006: any_valid_service_token = false should be omitted
# decision = "allow" because non_identity + email is invalid in the API
resource "cloudflare_zero_trust_access_policy" "no_service_token_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-no-service-token"
  decision         = "allow"
  session_duration = "18h"

  include = [{ email_domain = { domain = "cloudflare.com" } }]
}

moved {
  from = cloudflare_access_policy.no_service_token_policy
  to   = cloudflare_zero_trust_access_policy.no_service_token_policy
}

# TKT-004: service_token from list to {token_id = ...} object
# Also tests TKT-002 (block → list) and TKT-004 (service_token format)
resource "cloudflare_zero_trust_access_service_token" "test_token" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-service-token"
}

moved {
  from = cloudflare_access_service_token.test_token
  to   = cloudflare_zero_trust_access_service_token.test_token
}

resource "cloudflare_zero_trust_access_policy" "service_token_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-service-token-ref"
  decision         = "non_identity"
  session_duration = "18h"

  include = [{ service_token = { token_id = cloudflare_zero_trust_access_service_token.test_token.id } }]
}

moved {
  from = cloudflare_access_policy.service_token_policy
  to   = cloudflare_zero_trust_access_policy.service_token_policy
}

# TKT-004: multiple service tokens in include
resource "cloudflare_zero_trust_access_service_token" "test_token_2" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-service-token-2"
}

moved {
  from = cloudflare_access_service_token.test_token_2
  to   = cloudflare_zero_trust_access_service_token.test_token_2
}

resource "cloudflare_zero_trust_access_policy" "multi_service_token_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-multi-service-token"
  decision         = "non_identity"
  session_duration = "18h"

  include = [{ service_token = { token_id = cloudflare_zero_trust_access_service_token.test_token.id } },
  { service_token = { token_id = cloudflare_zero_trust_access_service_token.test_token_2.id } }]
}

moved {
  from = cloudflare_access_policy.multi_service_token_policy
  to   = cloudflare_zero_trust_access_policy.multi_service_token_policy
}

# TKT-005: multiple email domains — each becomes a separate include entry
resource "cloudflare_zero_trust_access_policy" "multi_email_domain_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-multi-email-domain"
  decision         = "allow"
  session_duration = "18h"

  include = [{ email_domain = { domain = "cloudflare.com" } },
  { email_domain = { domain = "example.com" } }]
}

moved {
  from = cloudflare_access_policy.multi_email_domain_policy
  to   = cloudflare_zero_trust_access_policy.multi_email_domain_policy
}

# TKT-002 + TKT-004 + TKT-005 + TKT-006: Combined real-world policy
# (mirrors research team's actual access_policies.tf)
resource "cloudflare_zero_trust_access_policy" "combined_research_team_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-combined"
  decision         = "non_identity"
  session_duration = "18h"

  include = [{ service_token = { token_id = cloudflare_zero_trust_access_service_token.test_token.id } },
  { service_token = { token_id = cloudflare_zero_trust_access_service_token.test_token_2.id } }]
}

moved {
  from = cloudflare_access_policy.combined_research_team_policy
  to   = cloudflare_zero_trust_access_policy.combined_research_team_policy
}

removed {
  from = cloudflare_access_policy.app_scoped_policy
  lifecycle {
    destroy = false
  }
}

# BUGS-2007: nested and list selector migrations
resource "cloudflare_zero_trust_access_policy" "bugs2007_lists" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bugs2007-lists"
  decision   = "allow"

  include = [{ device_posture = { integration_uid = "posture-1" } },
    { email_list = { id = "email-list-1" } },
  { ip_list = { id = "ip-list-1" } }]
}

moved {
  from = cloudflare_access_policy.bugs2007_lists
  to   = cloudflare_zero_trust_access_policy.bugs2007_lists
}

resource "cloudflare_zero_trust_access_policy" "bugs2007_azure" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bugs2007-azure"
  decision   = "allow"

  include = [{ azure_ad = { id = "group-1"
    identity_provider_id = "idp-1" } },
    { azure_ad = { id = "group-2"
  identity_provider_id = "idp-1" } }]
}

moved {
  from = cloudflare_access_policy.bugs2007_azure
  to   = cloudflare_zero_trust_access_policy.bugs2007_azure
}

resource "cloudflare_zero_trust_access_policy" "bugs2007_saml" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bugs2007-saml"
  decision   = "allow"

  include = [{ saml = { attribute_name = "group"
    attribute_value = "engineering"
  identity_provider_id = "idp-saml" } }]
}

moved {
  from = cloudflare_access_policy.bugs2007_saml
  to   = cloudflare_zero_trust_access_policy.bugs2007_saml
}

resource "cloudflare_zero_trust_access_policy" "bugs2007_auth_context" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-bugs2007-auth-context"
  decision   = "allow"

  include = [{ auth_context = { id = "ctx-id"
    ac_id = "ctx-ac-id"
  identity_provider_id = "idp-auth" } }]
}

moved {
  from = cloudflare_access_policy.bugs2007_auth_context
  to   = cloudflare_zero_trust_access_policy.bugs2007_auth_context
}
