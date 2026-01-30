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

locals {
  account_id = var.cloudflare_account_id
}

# Custom profile with single tunnel (include mode)
resource "cloudflare_zero_trust_device_profiles" "single_tunnel" {
  account_id  = local.account_id
  name        = "single_tunnel_profile"
  description = "Profile for developers with single tunnel"
  match       = "identity.email == \"dev@example.com\""
  precedence  = 100
}

resource "cloudflare_split_tunnel" "single" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.single_tunnel.id
  mode       = "include"

  tunnels {
    address     = "203.0.113.0/24"
    description = "Corporate VPN"
  }
}

# Custom profile with multiple tunnels
resource "cloudflare_zero_trust_device_profiles" "multiple_tunnels" {
  account_id  = local.account_id
  name        = "multiple_tunnels_profile"
  description = "Profile for admins with multiple tunnels"
  match       = "identity.email == \"admin@example.com\""
  precedence  = 200
}

resource "cloudflare_split_tunnel" "multi_exclude" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.multiple_tunnels.id
  mode       = "exclude"

  tunnels {
    address     = "172.20.0.0/16"
    description = "Admin network 1"
  }

  tunnels {
    address = "172.21.0.0/16"
  }

  tunnels {
    host = "admin.internal"
  }
}
