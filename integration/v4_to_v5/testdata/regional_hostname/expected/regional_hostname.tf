variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Pattern 1: Basic resource with required fields only
resource "cloudflare_regional_hostname" "basic" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "regional-basic.example.com"
  region_key = "eu"
}

# Pattern 2: Resource with timeouts (should be removed)
resource "cloudflare_regional_hostname" "with_timeouts" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "regional-timeouts.example.com"
  region_key = "us"

}

# Pattern 3: Wildcard hostname
resource "cloudflare_regional_hostname" "wildcard" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "*.regional.example.com"
  region_key = "ca"
}

# Pattern 4: for_each with map
locals {
  regional_hosts_map = {
    eu = {
      hostname   = "eu-regional.example.com"
      region_key = "eu"
    }
    us = {
      hostname   = "us-regional.example.com"
      region_key = "us"
    }
    au = {
      hostname   = "au-regional.example.com"
      region_key = "au"
    }
  }
}

resource "cloudflare_regional_hostname" "by_region_map" {
  for_each = local.regional_hosts_map

  zone_id    = var.cloudflare_zone_id
  hostname   = each.value.hostname
  region_key = each.value.region_key
}

# Pattern 5: for_each with set
locals {
  regional_hostnames_set = toset([
    "api-regional.example.com",
    "web-regional.example.com",
    "app-regional.example.com"
  ])
}

resource "cloudflare_regional_hostname" "from_set" {
  for_each = local.regional_hostnames_set

  zone_id    = var.cloudflare_zone_id
  hostname   = each.value
  region_key = "eu"
}

# Pattern 6: count-based resources
resource "cloudflare_regional_hostname" "counted" {
  count = 3

  zone_id    = var.cloudflare_zone_id
  hostname   = "regional-${count.index}.example.com"
  region_key = "us"
}

# Pattern 7: Conditional resource creation
variable "enable_regional" {
  type    = bool
  default = true
}

resource "cloudflare_regional_hostname" "conditional" {
  count = var.enable_regional ? 1 : 0

  zone_id    = var.cloudflare_zone_id
  hostname   = "conditional-regional.example.com"
  region_key = "in"
}

# Pattern 8: Using Terraform functions
resource "cloudflare_regional_hostname" "with_functions" {
  zone_id    = var.cloudflare_zone_id
  hostname   = lower("UPPERCASE-REGIONAL.example.com")
  region_key = "ca"
}

# Total: 1 + 1 + 1 + 3 (map) + 3 (set) + 3 (count) + 1 + 1 = 14 resources
