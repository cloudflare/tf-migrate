# Integration test for zero_trust_access_identity_provider v4 to v5 migration

variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  type        = string
  description = "Not used by this resource (account-level only), but required by e2e framework"
  default     = ""
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

variable "enable_conditional_provider" {
  type    = bool
  default = true
}

# Locals for reusable values
locals {
  base_scopes     = ["openid", "profile", "email"]
  extended_scopes = concat(local.base_scopes, ["groups"])
  name_prefix     = "cftftest"
  encoded_value   = base64encode("test-value")

  # Map for for_each with maps pattern
  google_providers = {
    "prod" = {
      name      = "${local.name_prefix} Google OAuth Production"
      client_id = "google-prod-client-id"
    }
    "staging" = {
      name      = "${local.name_prefix} Google OAuth Staging"
      client_id = "google-staging-client-id"
    }
    "dev" = {
      name      = "${local.name_prefix} Google OAuth Development"
      client_id = "google-dev-client-id"
    }
    "qa" = {
      name      = "${local.name_prefix} Google OAuth QA"
      client_id = "google-qa-client-id"
    }
  }

  # Set for for_each with sets pattern
  github_environments = toset(["production", "staging", "development"])
}














# Test 1: onetimepin (no config block in v4, but will be required in v5)
resource "cloudflare_zero_trust_access_identity_provider" "otp" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} One-Time PIN"
  type       = "onetimepin"
  config     = {}
}

moved {
  from = cloudflare_access_identity_provider.otp
  to   = cloudflare_zero_trust_access_identity_provider.otp
}

# Test 2: GitHub OAuth (basic config)
resource "cloudflare_zero_trust_access_identity_provider" "github" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} GitHub OAuth"
  type       = "github"

  config = {
    client_id     = "github-client-id"
    client_secret = "github-client-secret"
  }
}

moved {
  from = cloudflare_access_identity_provider.github
  to   = cloudflare_zero_trust_access_identity_provider.github
}

# Test 3: Azure AD with SCIM (complex config)
resource "cloudflare_zero_trust_access_identity_provider" "azure" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Azure AD SSO"
  type       = "azureAD"


  config = {
    client_id                  = "azure-client-id"
    client_secret              = "azure-client-secret"
    directory_id               = "azure-directory-uuid"
    conditional_access_enabled = false
    support_groups             = true
  }
  scim_config = {
    enabled          = true
    user_deprovision = true
    seat_deprovision = false
  }
}

moved {
  from = cloudflare_access_identity_provider.azure
  to   = cloudflare_zero_trust_access_identity_provider.azure
}

# Test 4: SAML with idp_public_cert (single string in v4, array in v5)
resource "cloudflare_zero_trust_access_identity_provider" "saml" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} SAML Provider"
  type       = "saml"

  config = {
    issuer_url       = "https://saml.example.com/issuer"
    sso_target_url   = "https://saml.example.com/sso"
    sign_request     = true
    attributes       = ["email", "username"]
    idp_public_certs = ["MIIDpDCCAoygAwIBAgIGAV2ka+55MA0GCSqGSIb3DQEBCwUAMIGSMQswCQYDVQQGEwJVUzETMBEG"]
  }
}

moved {
  from = cloudflare_access_identity_provider.saml
  to   = cloudflare_zero_trust_access_identity_provider.saml
}

# Test 5: OIDC with PKCE
resource "cloudflare_zero_trust_access_identity_provider" "oidc" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Generic OIDC"
  type       = "oidc"

  config = {
    client_id     = "oidc-client-id"
    client_secret = "oidc-client-secret"
    auth_url      = "https://oidc.example.com/authorize"
    token_url     = "https://oidc.example.com/token"
    certs_url     = "https://oidc.example.com/.well-known/jwks.json"
    scopes        = ["openid", "profile", "email"]
    pkce_enabled  = true
  }
}

moved {
  from = cloudflare_access_identity_provider.oidc
  to   = cloudflare_zero_trust_access_identity_provider.oidc
}

# Test 6-9: for_each with map (4 Google OAuth providers)
resource "cloudflare_zero_trust_access_identity_provider" "google_map" {
  for_each = local.google_providers

  account_id = var.cloudflare_account_id
  name       = each.value.name
  type       = "google"

  config = {
    client_id     = each.value.client_id
    client_secret = format("google-%s-secret", each.key)
  }
}

moved {
  from = cloudflare_access_identity_provider.google_map
  to   = cloudflare_zero_trust_access_identity_provider.google_map
}

# Test 10-12: for_each with set (3 GitHub providers)
resource "cloudflare_zero_trust_access_identity_provider" "github_set" {
  for_each = local.github_environments

  account_id = var.cloudflare_account_id
  name       = format("%s GitHub %s", local.name_prefix, title(each.key))
  type       = "github"

  config = {
    client_id     = join("-", ["github", each.key, "client"])
    client_secret = join("-", ["github", each.key, "secret"])
  }
}

moved {
  from = cloudflare_access_identity_provider.github_set
  to   = cloudflare_zero_trust_access_identity_provider.github_set
}

# Test 13-15: count-based resources (3 GitHub Enterprise providers)
resource "cloudflare_zero_trust_access_identity_provider" "github_enterprise" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = format("%s GitHub Enterprise %d", local.name_prefix, count.index + 1)
  type       = "github"

  config = {
    client_id     = format("ghe-client-%d", count.index)
    client_secret = format("ghe-secret-%d", count.index)
  }
}

moved {
  from = cloudflare_access_identity_provider.github_enterprise
  to   = cloudflare_zero_trust_access_identity_provider.github_enterprise
}

# Test 16: conditional resource creation (count with ternary)
resource "cloudflare_zero_trust_access_identity_provider" "conditional" {
  count = var.enable_conditional_provider ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Conditional Provider"
  type       = "github"

  config = {
    client_id     = "conditional-client-id"
    client_secret = "conditional-secret"
  }
}

moved {
  from = cloudflare_access_identity_provider.conditional
  to   = cloudflare_zero_trust_access_identity_provider.conditional
}

# Test 17: cross-resource reference (references the otp provider)
resource "cloudflare_zero_trust_access_identity_provider" "with_reference" {
  account_id = var.cloudflare_account_id
  name       = format("%s Referenced Provider - %s", local.name_prefix, cloudflare_zero_trust_access_identity_provider.otp.id)
  type       = "github"
  config = {
    client_id     = "ref-client-id"
    client_secret = "ref-client-secret"
  }
}

moved {
  from = cloudflare_access_identity_provider.with_reference
  to   = cloudflare_zero_trust_access_identity_provider.with_reference
}

# Test 18: using Terraform functions (base64encode, concat)
resource "cloudflare_zero_trust_access_identity_provider" "with_functions" {
  account_id = var.cloudflare_account_id
  name       = format("%s Provider with Functions", local.name_prefix)
  type       = "oidc"

  config = {
    client_id     = format("client-%s", local.encoded_value)
    client_secret = base64encode("secret-value")
    auth_url      = "https://auth.example.com/authorize"
    token_url     = "https://auth.example.com/token"
    certs_url     = "https://auth.example.com/.well-known/jwks.json"
    scopes        = local.extended_scopes
  }
}

moved {
  from = cloudflare_access_identity_provider.with_functions
  to   = cloudflare_zero_trust_access_identity_provider.with_functions
}

# Test 19: with lifecycle meta-arguments
resource "cloudflare_zero_trust_access_identity_provider" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Provider with Lifecycle"
  type       = "github"


  lifecycle {
    create_before_destroy = true
    ignore_changes        = [name]
  }
  config = {
    client_id     = "lifecycle-client-id"
    client_secret = "lifecycle-secret"
  }
}

moved {
  from = cloudflare_access_identity_provider.with_lifecycle
  to   = cloudflare_zero_trust_access_identity_provider.with_lifecycle
}

# Test 20: Azure AD with all optional fields
resource "cloudflare_zero_trust_access_identity_provider" "azure_full" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Azure AD Full Configuration"
  type       = "azureAD"


  config = {
    client_id                  = "azure-full-client-id"
    client_secret              = "azure-full-client-secret"
    directory_id               = "azure-full-directory-uuid"
    conditional_access_enabled = false
    support_groups             = false
  }
  scim_config = {
    enabled          = true
    user_deprovision = true
    seat_deprovision = true
  }
}

moved {
  from = cloudflare_access_identity_provider.azure_full
  to   = cloudflare_zero_trust_access_identity_provider.azure_full
}
