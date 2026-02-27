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
resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = var.cloudflare_account_id

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

  service_mode_v2 = {
    mode = "proxy"
    port = 8080
  }
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.default
  to   = cloudflare_zero_trust_device_default_profile.default
}
