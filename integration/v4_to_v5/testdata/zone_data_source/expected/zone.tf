# Test Case 1: Zone lookup by zone_id only
data "cloudflare_zone" "by_id" {
  zone_id = "abc123def456"
}

# Test Case 2: Zone lookup by zone_id with account_id (should remove account_id)
data "cloudflare_zone" "by_id_with_account" {
  zone_id = "xyz789uvw012"
}

# Test Case 3: Zone lookup by name only
data "cloudflare_zone" "by_name" {
  filter = {
    name = "example.com"
  }
}

# Test Case 4: Zone lookup by name with account_id
data "cloudflare_zone" "by_name_with_account" {
  filter = {
    name = "test.example.com"
    account = {
      id = "f037e56e89293a057740de681ac9abbe"
    }
  }
}

# Test Case 5: Zone lookup by account_id only
data "cloudflare_zone" "by_account" {
  filter = {
    account = {
      id = "f037e56e89293a057740de681ac9abbe"
    }
  }
}

# Test Case 6: Zone lookup by name with variable
data "cloudflare_zone" "with_variable" {
  filter = {
    name = var.zone_name
  }
}

# Test Case 7: Zone lookup by name with account_id variable
data "cloudflare_zone" "with_variables" {
  filter = {
    name = var.zone_name
    account = {
      id = var.account_id
    }
  }
}
