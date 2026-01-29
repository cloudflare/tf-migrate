variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "policy_id" {
  description = "Dynamic policy ID"
  type        = string
}

locals {
  name_prefix = "cftftest_split_tunnel"
  account_id  = var.cloudflare_account_id
}

resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = local.account_id
  include = [{
    address     = "203.0.113.0/24"
    description = "Corporate VPN range"
    }, {
    address = "198.51.100.0/24"
  }]
  exclude = [{
    address     = "192.168.0.0/16"
    description = "Private network - Class C"
    }, {
    address     = "10.0.0.0/8"
    description = "Private network - Class A"
    }, {
    address = "172.16.0.0/12"
    host    = "internal.local"
  }]
  register_interface_ip_with_dns = true
  sccm_vpn_boundary_support      = false
}

resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = local.account_id
  name       = "${local.name_prefix}_employees"
  match      = "identity.groups == \"employees\""
  precedence = 1000
  include = [{
    address     = "10.100.0.0/16"
    description = "Production resources"
    host        = "prod.internal.corp"
  }]
  exclude = [{
    address     = "172.20.0.0/16"
    description = "Development environment"
    }, {
    address = "172.21.0.0/16"
    host    = "dev.internal.corp"
  }]
}

resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = local.account_id
  name       = "${local.name_prefix}_contractors"
  match      = "identity.email endsWith \"@contractor.example.com\""
  precedence = 1100
  include = [{
    address     = "10.200.0.0/16"
    description = "Contractor resources only"
    }, {
    address = "10.201.0.0/24"
  }]
}








/** MIGRATION_WARNING: Split tunnel "unparseable_policy_id" has unparseable policy_id reference - manual migration required
*  resource "cloudflare_split_tunnel" "unparseable_policy_id" {
*    account_id = local.account_id
*    policy_id  = var.policy_id
*    mode       = "exclude"
*  
*    tunnels {
*      address     = "192.168.1.0/24"
*      description = "Unparseable policy_id reference"
*    }
*  }
*/

/** MIGRATION_WARNING: Split tunnel "complex_expression" has unparseable policy_id reference - manual migration required
*  resource "cloudflare_split_tunnel" "complex_expression" {
*    account_id = local.account_id
*    policy_id  = element(values(cloudflare_zero_trust_device_profiles), 0).id
*    mode       = "exclude"
*  
*    tunnels {
*      address = "172.31.0.0/16"
*    }
*  }
*/

/** MIGRATION_WARNING: Split tunnel "missing_profile" references profile "nonexistent" which was not found - manual migration required
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

