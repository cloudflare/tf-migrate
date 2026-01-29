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

resource "cloudflare_zero_trust_device_profiles" "default" {
  account_id = local.account_id
  default    = true
}

resource "cloudflare_zero_trust_device_profiles" "employees" {
  account_id = local.account_id
  name       = "${local.name_prefix}_employees"
  match      = "identity.groups == \"employees\""
  precedence = 100
}

resource "cloudflare_zero_trust_device_profiles" "contractors" {
  account_id = local.account_id
  name       = "${local.name_prefix}_contractors"
  match      = "identity.email endsWith \"@contractor.example.com\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "default_exclude_private" {
  account_id = local.account_id
  mode       = "exclude"

  tunnels {
    address     = "192.168.0.0/16"
    description = "Private network - Class C"
  }

  tunnels {
    address     = "10.0.0.0/8"
    description = "Private network - Class A"
  }

  tunnels {
    address = "172.16.0.0/12"
    host    = "internal.local"
  }
}

resource "cloudflare_split_tunnel" "default_include_corporate" {
  account_id = local.account_id
  mode       = "include"

  tunnels {
    address     = "203.0.113.0/24"
    description = "Corporate VPN range"
  }

  tunnels {
    address = "198.51.100.0/24"
  }
}

resource "cloudflare_split_tunnel" "employee_exclude_dev" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.employees.id
  mode       = "exclude"

  tunnels {
    address     = "172.20.0.0/16"
    description = "Development environment"
  }

  tunnels {
    address = "172.21.0.0/16"
    host    = "dev.internal.corp"
  }
}

resource "cloudflare_split_tunnel" "employee_include_prod" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.employees.id
  mode       = "include"

  tunnels {
    address     = "10.100.0.0/16"
    description = "Production resources"
    host        = "prod.internal.corp"
  }
}

resource "cloudflare_split_tunnel" "contractor_include_limited" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.contractors.id
  mode       = "include"

  tunnels {
    address     = "10.200.0.0/16"
    description = "Contractor resources only"
  }

  tunnels {
    address = "10.201.0.0/24"
  }
}

resource "cloudflare_split_tunnel" "unparseable_policy_id" {
  account_id = local.account_id
  policy_id  = var.policy_id
  mode       = "exclude"

  tunnels {
    address     = "192.168.1.0/24"
    description = "Unparseable policy_id reference"
  }
}

resource "cloudflare_split_tunnel" "missing_profile" {
  account_id = local.account_id
  policy_id  = cloudflare_zero_trust_device_profiles.nonexistent.id
  mode       = "include"

  tunnels {
    address     = "10.20.0.0/16"
    description = "References missing profile"
  }
}

resource "cloudflare_split_tunnel" "complex_expression" {
  account_id = local.account_id
  policy_id  = element(values(cloudflare_zero_trust_device_profiles), 0).id
  mode       = "exclude"

  tunnels {
    address = "172.31.0.0/16"
  }
}
