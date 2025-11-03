# Zero Trust Access Identity Provider example - no transformation needed
resource "cloudflare_access_identity_provider" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Example IdP"
  type       = "onetimepin"
}

# OIDC provider
resource "cloudflare_access_identity_provider" "oidc" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "OIDC Provider"
  type       = "oidc"

  config {
    client_id     = "client-id-123"
    client_secret = "client-secret-456"
    auth_url      = "https://example.com/oauth2/authorize"
    token_url     = "https://example.com/oauth2/token"
    certs_url     = "https://example.com/oauth2/certs"
  }
}
