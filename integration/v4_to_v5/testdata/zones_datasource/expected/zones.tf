# Test various zone datasource filter configurations

# Filter by account_id only
data "cloudflare_zones" "by_account" {
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
}

# Filter by name only
data "cloudflare_zones" "by_name" {
  name = "example.com"
}

# Filter by status only
data "cloudflare_zones" "by_status" {
  status = "active"
}

# Filter by account_id and name
data "cloudflare_zones" "by_account_and_name" {
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
  name = "example.com"
}

# Filter with all supported fields
data "cloudflare_zones" "all_supported" {
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
  name   = "example.com"
  status = "active"
}

# Filter with dropped fields (lookup_type, match, paused)
data "cloudflare_zones" "with_dropped_fields" {
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
  name = "example"
}

# Variable references preserved
data "cloudflare_zones" "with_variables" {
  account = {
    id = var.account_id
  }
  name   = var.zone_name
  status = var.status
}
