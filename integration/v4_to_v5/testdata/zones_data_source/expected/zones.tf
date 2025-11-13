# Test case 1: minimal - name only
data "cloudflare_zones" "by_name" {
  name = "example.cfapi.net"
}

# Test case 2: account_id only
data "cloudflare_zones" "by_account" {
  account = {
    id = "abc123def456"
  }
}

# Test case 3: all migratable fields
data "cloudflare_zones" "full" {
  name   = "test.cfapi.net"
  status = "active"
  account = {
    id = "xyz789abc012"
  }
}

# Test case 4: with non-migratable fields (should be removed with warnings)
data "cloudflare_zones" "with_nonmigratable" {
  name = "demo.cfapi.net"
}

# Test case 5: with variables
data "cloudflare_zones" "with_vars" {
  name = var.zone_name
  account = {
    id = var.account_id
  }
}

# Test case 6: empty filter
data "cloudflare_zones" "all" {
}

# Test case 7: status only
data "cloudflare_zones" "active_only" {
  status = "active"
}
