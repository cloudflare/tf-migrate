# Test various zone datasource filter configurations

# Filter by account_id only
data "cloudflare_zones" "by_account" {
  filter {
    account_id = "f037e56e89293a057740de681ac9abbe"
  }
}

# Filter by name only
data "cloudflare_zones" "by_name" {
  filter {
    name = "example.com"
  }
}

# Filter by status only
data "cloudflare_zones" "by_status" {
  filter {
    status = "active"
  }
}

# Filter by account_id and name
data "cloudflare_zones" "by_account_and_name" {
  filter {
    account_id = "f037e56e89293a057740de681ac9abbe"
    name       = "example.com"
  }
}

# Filter with all supported fields
data "cloudflare_zones" "all_supported" {
  filter {
    account_id = "f037e56e89293a057740de681ac9abbe"
    name       = "example.com"
    status     = "active"
  }
}

# Filter with dropped fields (lookup_type, match, paused)
data "cloudflare_zones" "with_dropped_fields" {
  filter {
    account_id  = "f037e56e89293a057740de681ac9abbe"
    name        = "example"
    lookup_type = "contains"
    match       = "^example-"
    paused      = false
  }
}

# Variable references preserved
data "cloudflare_zones" "with_variables" {
  filter {
    account_id = var.account_id
    name       = var.zone_name
    status     = var.status
  }
}
