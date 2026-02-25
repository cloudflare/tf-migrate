variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "policy_id" {
  description = "Dynamic policy ID for testing unparseable references"
  type        = string
}

locals {
  account_id = var.cloudflare_account_id
}











/** MIGRATION_WARNING: Split tunnel "unparseable_var" has unparseable policy_id reference - manual migration required
*  # Unparseable policy_id - variable reference
*  resource "cloudflare_split_tunnel" "unparseable_var" {
*    account_id = local.account_id
*    policy_id  = var.policy_id
*    mode       = "exclude"
*  
*    tunnels {
*      address     = "192.168.1.0/24"
*      description = "Variable reference"
*    }
*  }
*/

/** MIGRATION_WARNING: Split tunnel "unparseable_expression" has unparseable policy_id reference - manual migration required
*  # Unparseable policy_id - complex expression
*  resource "cloudflare_split_tunnel" "unparseable_expression" {
*    account_id = local.account_id
*    policy_id  = element(values(cloudflare_zero_trust_device_profiles), 0).id
*    mode       = "exclude"
*  
*    tunnels {
*      address     = "192.168.2.0/24"
*      description = "Complex expression"
*    }
*  }
*/

/** MIGRATION_WARNING: Split tunnel "missing_profile" references profile "nonexistent" which was not found - manual migration required
*  # Reference to non-existent device profile
*  resource "cloudflare_split_tunnel" "missing_profile" {
*    account_id = local.account_id
*    policy_id  = cloudflare_zero_trust_device_default_profile.nonexistent.id
*    mode       = "include"
*  
*    tunnels {
*      address     = "10.20.0.0/16"
*      description = "References missing profile"
*    }
*  }
*/


# Default profile (will receive split tunnels without policy_id)
resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = local.account_id
  include = [{
    address     = "203.0.113.0/24"
    description = "Corporate VPN"
  }]
  exclude = [{
    address     = "192.168.0.0/16"
    description = "Private network"
    }, {
    address = "10.0.0.0/8"
    host    = "internal.local"
  }]
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

moved {
  from = cloudflare_zero_trust_device_profiles.default
  to   = cloudflare_zero_trust_device_default_profile.default
}

# Custom profile with single tunnel
resource "cloudflare_zero_trust_device_custom_profile" "single_tunnel" {
  account_id = local.account_id
  name       = "single_tunnel_profile"
  match      = "identity.groups == \"developers\""
  precedence = 1000
  exclude = [{
    address     = "172.16.0.0/12"
    description = "Dev environment"
  }]
}

moved {
  from = cloudflare_zero_trust_device_profiles.single_tunnel
  to   = cloudflare_zero_trust_device_custom_profile.single_tunnel
}

# Custom profile with multiple tunnels
resource "cloudflare_zero_trust_device_custom_profile" "multiple_tunnels" {
  account_id = local.account_id
  name       = "multiple_tunnels_profile"
  match      = "identity.groups == \"admins\""
  precedence = 1100
  include = [{
    address     = "10.100.0.0/16"
    description = "Admin resources"
    host        = "prod.internal"
  }]
  exclude = [{
    address     = "172.20.0.0/16"
    description = "Admin network 1"
    }, {
    address = "172.21.0.0/16"
    host    = "admin.internal"
  }]
}

moved {
  from = cloudflare_zero_trust_device_profiles.multiple_tunnels
  to   = cloudflare_zero_trust_device_custom_profile.multiple_tunnels
}
