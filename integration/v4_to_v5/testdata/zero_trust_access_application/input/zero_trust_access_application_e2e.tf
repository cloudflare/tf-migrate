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

# Minimal E2E test - basic self-hosted app
resource "cloudflare_zero_trust_access_application" "e2e_minimal" {
  account_id = var.cloudflare_account_id
  name       = "E2E Minimal App"
  domain     = var.cloudflare_domain
  type       = "self_hosted"
}

# E2E test - SAML SAAS app (no domain required)
resource "cloudflare_zero_trust_access_application" "e2e_saas_saml" {
  account_id = var.cloudflare_account_id
  name       = "E2E SAML SAAS"
  type       = "saas"

  saas_app {
    consumer_service_url = "https://${var.cloudflare_domain}/sso/saml"
    sp_entity_id         = "e2e-saml-app"
    name_id_format       = "email"

    custom_attribute {
      name = "email"
      source {
        name = "user_email"
      }
    }
  }
}
