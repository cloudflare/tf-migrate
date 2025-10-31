# Test Case 1: Basic access service token with all fields
resource "cloudflare_zero_trust_access_service_token" "basic_token" {
  account_id                        = "f037e56e89293a057740de681ac9abbe"
  name                              = "basic_token"
  duration                          = "8760h"
  client_secret_version             = 2
  previous_client_secret_expires_at = "2024-12-31T23:59:59Z"
}

# Test Case 2: Legacy access service token name
resource "cloudflare_zero_trust_access_service_token" "basic_token_legacy" {
  account_id                        = "f037e56e89293a057740de681ac9abbe"
  name                              = "basic_token_legacy"
  duration                          = "8760h"
  client_secret_version             = 2
  previous_client_secret_expires_at = "2024-12-31T23:59:59Z"
}

