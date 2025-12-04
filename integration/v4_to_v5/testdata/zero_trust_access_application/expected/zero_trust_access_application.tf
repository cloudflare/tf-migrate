variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Resource-specific variables with defaults
variable "app_prefix" {
  type    = string
  default = "test"
}

variable "enable_saas_apps" {
  type    = bool
  default = true
}

variable "policy_ids" {
  type    = list(string)
  default = ["policy-1", "policy-2", "policy-3"]
}

# Locals with common values
locals {
  name_prefix       = "cftftest"
  common_account_id = var.cloudflare_account_id
  app_domain_suffix = "cort.terraform.cfapi.net"
  common_policies   = ["default-policy-id"]
}

# ============================================================================
# Pattern Group 1: Basic Resources (Edge Cases)
# ============================================================================

# 1. Minimal resource - only required fields
resource "cloudflare_zero_trust_access_application" "minimal" {
  account_id = local.common_account_id
  name       = "${local.name_prefix} Minimal App"
  domain     = "minimal.${local.app_domain_suffix}"
  type       = "self_hosted"
}

# 2. Minimal with type specification
resource "cloudflare_zero_trust_access_application" "minimal_self_hosted" {
  account_id = local.common_account_id
  name       = "${local.name_prefix} Self Hosted"
  domain     = "self-hosted.${local.app_domain_suffix}"
  type       = "self_hosted"
}

# 3. Maximal self-hosted app - all common fields
resource "cloudflare_zero_trust_access_application" "maximal_self_hosted" {
  account_id                   = local.common_account_id
  name                         = "${local.name_prefix} Maximal Self Hosted"
  domain                       = "maximal.${local.app_domain_suffix}"
  type                         = "self_hosted"
  session_duration             = "24h"
  auto_redirect_to_identity    = false
  enable_binding_cookie        = true
  http_only_cookie_attribute   = true
  same_site_cookie_attribute   = "strict"
  custom_deny_url              = "https://deny.${local.app_domain_suffix}"
  custom_deny_message          = "Access denied - contact admin"
  custom_non_identity_deny_url = "https://login.${local.app_domain_suffix}"
  skip_interstitial            = true
  app_launcher_visible         = true
  service_auth_401_redirect    = true

  cors_headers = {
    allowed_methods   = ["GET", "POST", "PUT", "DELETE"]
    allowed_origins   = ["https://app.${local.app_domain_suffix}"]
    allowed_headers   = ["Content-Type", "Authorization"]
    allow_credentials = true
    max_age           = 3600
  }
}

# ============================================================================
# Pattern Group 2: SAAS Applications
# ============================================================================

# 4. SAML SAAS app with full configuration
resource "cloudflare_zero_trust_access_application" "saas_saml" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} SAML SAAS"
  type       = "saas"

  saas_app = {
    consumer_service_url = "https://saml.${local.app_domain_suffix}/sso/saml"
    sp_entity_id         = "saml-app-${local.name_prefix}"
    name_id_format       = "email"
    custom_attributes = [
      {
        name = "email"
        source = {
          name = "user_email"
        }
      },
      {
        name          = "department"
        friendly_name = "Department"
        source = {
          name = "department"
        }
      }
    ]
  }
}

# 5. OIDC SAAS app
resource "cloudflare_zero_trust_access_application" "saas_oidc" {
  account_id = local.common_account_id
  name       = "${local.name_prefix} OIDC SAAS"
  type       = "saas"

  saas_app = {
    auth_type        = "oidc"
    app_launcher_url = "https://oidc.${local.app_domain_suffix}/launch"
    grant_types      = ["authorization_code"]
    scopes           = ["openid", "email", "profile"]
    redirect_uris    = ["https://oidc.${local.app_domain_suffix}/callback"]
    custom_claims = [
      {
        name  = "groups"
        scope = "groups"
        source = {
          name = "user_groups"
        }
      }
    ]
    hybrid_and_implicit_options = {
      return_access_token_from_authorization_endpoint = true
      return_id_token_from_authorization_endpoint     = true
    }
    refresh_token_options = {
      lifetime = "2160h"
    }
  }
}

# ============================================================================
# Pattern Group 3: WARP Applications with Destinations
# ============================================================================

# NOTE: WARP applications have special API behavior - they're forced to be
# named "Warp Login App" and don't respect custom configurations. Skipped for e2e tests.
# # 6. WARP app with multiple destinations
# resource "cloudflare_access_application" "warp_multi" {
#   account_id = local.common_account_id
#   name       = "${local.name_prefix} WARP Multi"
#   type       = "warp"
#
#   destinations {
#     uri  = "https://app1.internal"
#     type = "public"
#   }
#
#   destinations {
#     uri  = "tcp://10.0.0.0/24:22"
#     type = "private"
#   }
#
#   destinations {
#     uri  = "udp://192.168.1.0/24:53"
#     type = "private"
#   }
# }

# ============================================================================
# Pattern Group 4: SSH Applications with Target Criteria
# ============================================================================

# 9. SSH app (basic)
resource "cloudflare_zero_trust_access_application" "ssh_basic" {
  account_id = local.common_account_id
  name       = "${local.name_prefix} SSH Basic"
  type       = "ssh"
  domain     = "ssh-basic.${local.app_domain_suffix}"
}

# 10. SSH app (multi)
resource "cloudflare_zero_trust_access_application" "ssh_multi_criteria" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} SSH Multi"
  type       = "ssh"
  domain     = "ssh-multi.${local.app_domain_suffix}"
}

# ============================================================================
# Pattern Group 5: Apps with Landing Page Design
# ============================================================================

# NOTE: landing_page_design is not supported in v4 provider
# # 11. App with custom landing page
# resource "cloudflare_access_application" "with_landing_page" {
#   account_id = local.common_account_id
#   name       = "${local.name_prefix} Landing Page"
#   domain     = "landing.${local.app_domain_suffix}"
#   policies   = ["landing-policy"]
#
#   landing_page_design {
#     title            = "Welcome to ${local.name_prefix}"
#     message          = "Please sign in to continue"
#     logo_url         = "https://logo.${local.app_domain_suffix}/logo.png"
#     header_bg_color  = "#0051C3"
#     body_bg_color    = "#FFFFFF"
#     footer_links {
#       name = "Help Center"
#       url  = "https://help.${local.app_domain_suffix}"
#     }
#     footer_links {
#       name = "Privacy Policy"
#       url  = "https://privacy.${local.app_domain_suffix}"
#     }
#   }
# }
#
# # 12. App with minimal landing page
# resource "cloudflare_access_application" "minimal_landing" {
#   account_id = var.cloudflare_account_id
#   name       = "${local.name_prefix} Minimal Landing"
#   domain     = "minimal-landing.${local.app_domain_suffix}"
#
#   landing_page_design {
#     message = "Sign in required"
#   }
# }

# ============================================================================
# Pattern Group 6: Apps with SCIM Configuration
# ============================================================================

# 13. App with SCIM config
# NOTE: Commented out - SCIM requires special account permissions
# resource "cloudflare_access_application" "with_scim" {
#   account_id = local.common_account_id
#   name       = "${local.name_prefix} SCIM"
#   domain     = "scim.${local.app_domain_suffix}"
#
#   scim_config {
#     enabled                = true
#     remote_uri             = "https://scim.${local.app_domain_suffix}/v2"
#     idp_uid                = "email"
#     deactivate_on_delete   = true
#
#     authentication {
#       scheme   = "httpbasic"
#       user     = "scim_user"
#       password = "scim_password"
#     }
#
#     mappings {
#       schema = "urn:ietf:params:scim:schemas:core:2.0:User"
#       enabled = true
#
#       operations {
#         create = true
#         update = true
#         delete = true
#       }
#     }
#   }
# }

# ============================================================================
# Pattern Group 7: Apps with Custom Pages
# ============================================================================

# 14. App with custom pages
# NOTE: Commented out - custom_pages requires real page IDs
# resource "cloudflare_access_application" "with_custom_pages" {
#   account_id   = var.cloudflare_account_id
#   name         = "${local.name_prefix} Custom Pages"
#   domain       = "custom-pages.${local.app_domain_suffix}"
#   custom_pages = toset(["custom-forbidden-id", "custom-identity-denied-id"])
# }

# ============================================================================
# Pattern Group 8: Apps with Self Hosted Domains (Deprecated)
# ============================================================================

# 15. App with self_hosted_domains (deprecated field)
resource "cloudflare_zero_trust_access_application" "with_self_hosted_domains" {
  account_id          = local.common_account_id
  name                = "${local.name_prefix} Self Hosted Domains"
  type                = "self_hosted"
  self_hosted_domains = ["legacy1.${local.app_domain_suffix}", "legacy2.${local.app_domain_suffix}"]
}

# ============================================================================
# Pattern Group 9: Apps with Domain Type (Removed Field)
# ============================================================================

# 16. App with domain_type field (to be removed in v5)
# NOTE: v4 only supports domain_type = "public"
resource "cloudflare_zero_trust_access_application" "with_domain_type" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Domain Type"
  domain     = "domain-type.${local.app_domain_suffix}"
  type       = "self_hosted"
}

# ============================================================================
# Pattern Group 10: Meta-Arguments (Count)
# ============================================================================

# 17-19. Apps created with count
resource "cloudflare_zero_trust_access_application" "with_count" {
  count = 3

  account_id = local.common_account_id
  name       = "${local.name_prefix} Count ${count.index}"
  domain     = "count-${count.index}.${local.app_domain_suffix}"
  type       = "self_hosted"
}

# ============================================================================
# Pattern Group 11: Apps with For Each
# ============================================================================

# 20-22. Apps created with for_each
resource "cloudflare_zero_trust_access_application" "with_for_each" {
  for_each = toset(["dev", "staging", "prod"])

  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix} ${each.key}"
  domain           = "${each.key}.${local.app_domain_suffix}"
  session_duration = each.key == "prod" ? "8h" : "24h"
  type             = "self_hosted"
}

# ============================================================================
# Pattern Group 12: Complex Nested Structures
# ============================================================================

# 23. App with session_duration and cors_headers (compatible with type=self_hosted)
resource "cloudflare_zero_trust_access_application" "complex_nested" {
  account_id                = local.common_account_id
  name                      = "${local.name_prefix} Complex"
  domain                    = "complex.${local.app_domain_suffix}"
  session_duration          = "12h"
  auto_redirect_to_identity = false

  type = "self_hosted"
  cors_headers = {
    allowed_methods   = ["GET", "POST"]
    allowed_origins   = ["*"]
    allow_credentials = false
    max_age           = 7200
  }
}

# 24. App with variable references
resource "cloudflare_zero_trust_access_application" "with_var_refs" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Variable Refs"
  domain     = "var-refs.${local.app_domain_suffix}"
  type       = "self_hosted"
}

# ============================================================================
# Pattern Group 13: Edge Cases
# ============================================================================

# 25. App with empty policies array
resource "cloudflare_zero_trust_access_application" "empty_policies" {
  account_id = local.common_account_id
  name       = "${local.name_prefix} Empty Policies"
  domain     = "empty-policies.${local.app_domain_suffix}"
  type       = "self_hosted"
}

# 26. App with only required fields and null optionals
resource "cloudflare_zero_trust_access_application" "sparse" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Sparse"
  domain     = "sparse.${local.app_domain_suffix}"
  type       = "self_hosted"
}

# 27. Conditional app
resource "cloudflare_zero_trust_access_application" "conditional" {
  count = var.enable_saas_apps ? 1 : 0

  account_id = local.common_account_id
  name       = "${local.name_prefix} Conditional"
  domain     = "conditional.${local.app_domain_suffix}"
  type       = "self_hosted"
}

# ============================================================================
# Pattern Group 14: Special Characters and Escaping
# ============================================================================

# 28. App with special characters (API restricts: ,.!:@?-)
resource "cloudflare_zero_trust_access_application" "special_chars" {
  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix} Special \"Chars\" & 'Quotes'"
  domain              = "special.${local.app_domain_suffix}"
  custom_deny_message = "Access denied - contact support"
  type                = "self_hosted"
}
