# Basic zone datasource lookup by zone_id
data "cloudflare_zone" "main" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"
}

# Zone lookup with filter by name
data "cloudflare_zone" "by_name" {
  filter {
    name = "example.com"
  }
}

# Zone lookup with complex filter
data "cloudflare_zone" "filtered" {
  filter {
    account {
      id = "f037e56e89293a057740de681ac9abbe"
    }
    name      = "test.example.com"
    status    = "active"
    order     = "name"
    direction = "asc"
    match     = "all"
  }
}

# ========================================
# Advanced Terraform Patterns for Testing
# ========================================

# Pattern 1: Variable references
variable "zone_name" {
  type    = string
  default = "prod.example.com"
}

variable "account_id" {
  type    = string
  default = "f037e56e89293a057740de681ac9abbe"
}

# Pattern 2: Local values with expressions
locals {
  zone_prefix = "app"
  full_zone   = "${local.zone_prefix}.example.com"
}

# Pattern 3: Datasource with variable-driven filter
data "cloudflare_zone" "variable_driven" {
  filter {
    name = var.zone_name
    account {
      id = var.account_id
    }
  }
}

# Pattern 4: Datasource with local value reference
data "cloudflare_zone" "local_driven" {
  filter {
    name = local.full_zone
  }
}

# Pattern 5: for_each with datasources (looking up multiple zones)
variable "zone_names" {
  type = set(string)
  default = [
    "zone1.example.com",
    "zone2.example.com",
    "zone3.example.com"
  ]
}

data "cloudflare_zone" "multiple_zones" {
  for_each = var.zone_names

  filter {
    name = each.value
  }
}

# Pattern 6: Using datasource output in resource
# (This shows how zone datasource is typically used)
resource "cloudflare_dns_record" "example" {
  zone_id = data.cloudflare_zone.main.id
  name    = "www"
  content = "192.0.2.1"
  type    = "A"
  ttl     = 3600
}

# Pattern 7: Conditional datasource lookup
variable "use_zone_id" {
  type    = bool
  default = true
}

variable "zone_id_value" {
  type    = string
  default = "023e105f4ecef8ad9ca31a8372d0c353"
}

data "cloudflare_zone" "conditional" {
  count = var.use_zone_id ? 1 : 0

  zone_id = var.zone_id_value
}

# Pattern 8: Zone lookup with filter operators
data "cloudflare_zone" "with_operators" {
  filter {
    name = "starts_with:example"
  }
}

# Pattern 9: Zone lookup for subdomain
data "cloudflare_zone" "subdomain" {
  filter {
    name   = "sub.example.com"
    status = "active"
  }
}

# Pattern 10: Cross-datasource references
data "cloudflare_zone" "primary" {
  filter {
    name = "example.com"
  }
}

# Use primary zone's account in another lookup
data "cloudflare_zone" "same_account" {
  filter {
    name = "another.example.com"
    account {
      id = data.cloudflare_zone.primary.account.id
    }
  }
}

# Pattern 11: Output values from datasource
output "main_zone_id" {
  value = data.cloudflare_zone.main.id
}

output "main_zone_name" {
  value = data.cloudflare_zone.main.name
}

output "main_zone_name_servers" {
  value = data.cloudflare_zone.main.name_servers
}

output "main_zone_status" {
  value = data.cloudflare_zone.main.status
}

# Pattern 12: Using zone datasource in dynamic blocks
variable "dns_records" {
  type = list(object({
    zone_name = string
    name      = string
    content   = string
  }))
  default = [
    {
      zone_name = "example.com"
      name      = "api"
      content   = "192.0.2.10"
    },
    {
      zone_name = "example.com"
      name      = "app"
      content   = "192.0.2.11"
    }
  ]
}

# Pattern 13: Zone lookup with all filter options
data "cloudflare_zone" "comprehensive" {
  filter {
    name = "comprehensive.example.com"
    account {
      id   = var.account_id
      name = "Example Organization"
    }
    status    = "active"
    order     = "status"
    direction = "desc"
    match     = "all"
  }
}
