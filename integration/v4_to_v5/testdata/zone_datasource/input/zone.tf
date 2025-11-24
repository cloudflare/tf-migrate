# Zone datasource v4 test fixtures
# Testing all lookup scenarios

# Scenario 1: zone_id lookup (no changes)
data "cloudflare_zone" "by_id" {
  zone_id = "abc123def456"
}

# Scenario 2: name only lookup
data "cloudflare_zone" "by_name" {
  name = "example.com"
}

# Scenario 3: account_id only lookup
data "cloudflare_zone" "by_account" {
  account_id = "account789"
}

# Scenario 4: name + account_id lookup
data "cloudflare_zone" "by_name_and_account" {
  name       = "example.org"
  account_id = "account456"
}

# Scenario 5: with variable references
data "cloudflare_zone" "with_variables" {
  name       = var.zone_name
  account_id = var.account_id
}

# Scenario 6: another zone_id lookup
data "cloudflare_zone" "another_by_id" {
  zone_id = "xyz789abc123"
}

# Scenario 7: name with local reference
data "cloudflare_zone" "from_local" {
  name       = local.zone_domain
  account_id = local.cf_account_id
}
