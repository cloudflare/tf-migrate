# Test case 1: minimal - name only
data "cloudflare_zones" "by_name" {
  filter {
    name = "example.cfapi.net"
  }
}

# Test case 2: account_id only
data "cloudflare_zones" "by_account" {
  filter {
    account_id = "abc123def456"
  }
}

# Test case 3: all migratable fields
data "cloudflare_zones" "full" {
  filter {
    name       = "test.cfapi.net"
    account_id = "xyz789abc012"
    status     = "active"
  }
}

# Test case 4: with non-migratable fields (should be removed with warnings)
data "cloudflare_zones" "with_nonmigratable" {
  filter {
    name        = "demo.cfapi.net"
    lookup_type = "contains"
    match       = ".*\\.cfapi\\.net$"
    paused      = false
  }
}

# Test case 5: with variables
data "cloudflare_zones" "with_vars" {
  filter {
    name       = var.zone_name
    account_id = var.account_id
  }
}

# Test case 6: empty filter
data "cloudflare_zones" "all" {
  filter {
  }
}

# Test case 7: status only
data "cloudflare_zones" "active_only" {
  filter {
    status = "active"
  }
}
