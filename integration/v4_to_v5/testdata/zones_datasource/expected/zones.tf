variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test variables for variable reference preservation tests
# These are only used to test that variable references are preserved during migration
# They won't be queried in E2E tests
variable "account_id" {
  type    = string
  default = "test-account-abc123"
}

variable "zone_name" {
  type    = string
  default = "test-zone.example.com"
}

variable "status" {
  type    = string
  default = "active"
}

# Test various zone datasource filter configurations

# First, look up the zone by ID to get its name dynamically
data "cloudflare_zone" "reference" {
  zone_id = var.cloudflare_zone_id
}

# Filter by account_id only
data "cloudflare_zones" "by_account" {
  account = {
    id = var.cloudflare_account_id
  }
}

# Filter by name only - uses dynamic zone name
data "cloudflare_zones" "by_name" {
  name = data.cloudflare_zone.reference.name
}

# Filter by status only
data "cloudflare_zones" "by_status" {
  status = "active"
}

# Filter by account_id and name
data "cloudflare_zones" "by_account_and_name" {
  account = {
    id = var.cloudflare_account_id
  }
  name = data.cloudflare_zone.reference.name
}

# Filter with all supported fields
data "cloudflare_zones" "all_supported" {
  account = {
    id = var.cloudflare_account_id
  }
  name   = data.cloudflare_zone.reference.name
  status = "active"
}

# Filter with dropped fields (lookup_type, match, paused)
# Uses partial name match so lookup_type makes sense
data "cloudflare_zones" "with_dropped_fields" {
  account = {
    id = var.cloudflare_account_id
  }
  name = "test"
}

# Variable references preserved
data "cloudflare_zones" "with_variables" {
  account = {
    id = var.account_id
  }
  name   = var.zone_name
  status = var.status
}
