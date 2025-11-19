variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test Case 1: Basic access service token with all fields
resource "cloudflare_zero_trust_access_service_token" "basic_token" {
  account_id                        = var.cloudflare_account_id
  name                              = "basic_token"
  duration                          = "8760h"
  min_days_for_renewal              = 30
  client_secret_version             = 2
  previous_client_secret_expires_at = "2024-12-31T23:59:59Z"
}

# Test Case 2: Legacy access service token name
resource "cloudflare_access_service_token" "basic_token_legacy" {
  account_id                        = var.cloudflare_account_id
  name                              = "basic_token_legacy"
  duration                          = "8760h"
  min_days_for_renewal              = 30
  client_secret_version             = 2
  previous_client_secret_expires_at = "2024-12-31T23:59:59Z"
}
