# E2E test for zero_trust_device_default_profile migration
# This is a simplified version for actual API testing (singleton resource)

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (not used by this account-scoped resource)"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (not used by this account-scoped resource)"
  type        = string
}

# Single default profile test with all v4-valid fields populated
# Note: This is a singleton resource - only one default profile exists per account
resource "cloudflare_zero_trust_device_profiles" "default" {
  account_id  = var.cloudflare_account_id
  name        = "Default Device Profile"
  description = "Default device profile for E2E test"
  default     = true

  # All v4-supported optional fields
  allow_mode_switch     = true
  allow_updates         = true
  allowed_to_leave      = false
  auto_connect          = 0
  captive_portal        = 180
  switch_locked         = false
  disable_auto_fallback = false
  exclude_office_ips    = false
  support_url           = "https://support.example.com"
  tunnel_protocol       = "wireguard"

  # Service mode configuration (tests v4→v5 transformation)
  service_mode_v2_mode = "proxy"
  service_mode_v2_port = 8080
}
