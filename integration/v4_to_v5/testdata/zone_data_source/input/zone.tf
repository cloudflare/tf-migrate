# Test Case 1: Zone lookup by zone_id only
data "cloudflare_zone" "by_id" {
  zone_id = "abc123def456"
}

# Test Case 2: Zone lookup by zone_id with account_id (should remove account_id)
data "cloudflare_zone" "by_id_with_account" {
  zone_id    = "xyz789uvw012"
  account_id = "f037e56e89293a057740de681ac9abbe"
}

# Test Case 3: Zone lookup by name only
data "cloudflare_zone" "by_name" {
  name = "example.com"
}

# Test Case 4: Zone lookup by name with account_id
data "cloudflare_zone" "by_name_with_account" {
  name       = "test.example.com"
  account_id = "f037e56e89293a057740de681ac9abbe"
}

# Test Case 5: Zone lookup by account_id only
data "cloudflare_zone" "by_account" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}

# Test Case 6: Zone lookup by name with variable
data "cloudflare_zone" "with_variable" {
  name = var.zone_name
}

# Test Case 7: Zone lookup by name with account_id variable
data "cloudflare_zone" "with_variables" {
  name       = var.zone_name
  account_id = var.account_id
}
