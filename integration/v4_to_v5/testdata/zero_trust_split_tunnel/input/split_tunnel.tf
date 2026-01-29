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

# Default profile (will receive split tunnels without policy_id)
resource "cloudflare_zero_trust_device_profiles" "default" {
  account_id = local.account_id
  default    = true
}

# Split tunnel embedded in default profile (no policy_id)
resource "cloudflare_split_tunnel" "default_exclude" {
  account_id = local.account_id
  mode       = "exclude"

  tunnels {
    address     = "192.168.0.0/16"
    description = "Private network"
  }

  tunnels {
    address = "10.0.0.0/8"
    host    = "internal.local"
  }
}

resource "cloudflare_split_tunnel" "default_include" {
  account_id = local.account_id
  mode       = "include"

  tunnels {
    address     = "203.0.113.0/24"
    description = "Corporate VPN"
  }
}

# Custom profile with single tunnel
resource "cloudflare_zero_trust_device_profiles" "single_tunnel" {
  account_id = local.account_id
  name       = "single_tunnel_profile"
  match      = "identity.groups == \"developers\""
  precedence = 100
}

resource "cloudflare_split_tunnel" "single" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.single_tunnel.id
  mode       = "exclude"

  tunnels {
    address     = "172.16.0.0/12"
    description = "Dev environment"
  }
}

# Custom profile with multiple tunnels
resource "cloudflare_zero_trust_device_profiles" "multiple_tunnels" {
  account_id = local.account_id
  name       = "multiple_tunnels_profile"
  match      = "identity.groups == \"admins\""
  precedence = 200
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
    host    = "admin.internal"
  }
}

resource "cloudflare_split_tunnel" "multi_include" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.multiple_tunnels.id
  mode       = "include"

  tunnels {
    address     = "10.100.0.0/16"
    description = "Admin resources"
    host        = "prod.internal"
  }
}

# Unparseable policy_id - variable reference
resource "cloudflare_split_tunnel" "unparseable_var" {
  account_id = local.account_id
  policy_id  = var.policy_id
  mode       = "exclude"

  tunnels {
    address     = "192.168.1.0/24"
    description = "Variable reference"
  }
}

# Unparseable policy_id - complex expression
resource "cloudflare_split_tunnel" "unparseable_expression" {
  account_id = local.account_id
  policy_id  = element(values(cloudflare_zero_trust_device_profiles), 0).id
  mode       = "exclude"

  tunnels {
    address     = "192.168.2.0/24"
    description = "Complex expression"
  }
}

# Reference to non-existent device profile
resource "cloudflare_split_tunnel" "missing_profile" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.nonexistent.id
  mode       = "include"

  tunnels {
    address     = "10.20.0.0/16"
    description = "References missing profile"
  }
}
