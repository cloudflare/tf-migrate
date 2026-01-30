# Comprehensive integration test for cloudflare_rulesets datasource migration
# This file tests various scenarios for v4 to v5 migration

# Variables (no defaults - integration tests should provide these)
variable "cloudflare_account_id" {
  type        = string
  description = "Cloudflare account ID"
}

variable "cloudflare_zone_id" {
  type        = string
  description = "Cloudflare zone ID"
}

variable "cloudflare_domain" {
  type        = string
  description = "Domain for testing"
}

# Scenario 1: Basic datasource with account_id only
data "cloudflare_rulesets" "account_all" {
  account_id = var.cloudflare_account_id
}

# Scenario 2: Basic datasource with zone_id only
data "cloudflare_rulesets" "zone_all" {
  zone_id = var.cloudflare_zone_id
}

# Scenario 3: Datasource with filter block (simple filter)
data "cloudflare_rulesets" "filtered_by_name" {
  account_id = var.cloudflare_account_id
  filter {
    name = "my-custom-ruleset"
  }
}

# Scenario 4: Datasource with filter block (complex filter - all fields)
data "cloudflare_rulesets" "filtered_complex" {
  account_id = var.cloudflare_account_id
  filter {
    id      = "abc123def456"
    name    = "production-ruleset"
    kind    = "custom"
    phase   = "http_request_firewall_managed"
    version = "5"
  }
}

# Scenario 5: Datasource with include_rules only
data "cloudflare_rulesets" "with_rules" {
  account_id    = var.cloudflare_account_id
  include_rules = true
}

# Scenario 6: Datasource with both filter and include_rules
data "cloudflare_rulesets" "filtered_with_rules" {
  zone_id       = var.cloudflare_zone_id
  include_rules = true
  filter {
    kind  = "managed"
    phase = "http_request_firewall_custom"
  }
}

# Scenario 7: Datasource with include_rules = false
data "cloudflare_rulesets" "without_rules" {
  account_id    = var.cloudflare_account_id
  include_rules = false
}

# Scenario 8: Multiple filters testing different phases
data "cloudflare_rulesets" "firewall_managed" {
  account_id = var.cloudflare_account_id
  filter {
    phase = "http_request_firewall_managed"
  }
}

data "cloudflare_rulesets" "firewall_custom" {
  account_id = var.cloudflare_account_id
  filter {
    phase = "http_request_firewall_custom"
  }
}

data "cloudflare_rulesets" "rate_limit" {
  zone_id = var.cloudflare_zone_id
  filter {
    phase = "http_ratelimit"
  }
}

# Scenario 9: Cross-resource references (datasource output used elsewhere)
output "all_account_rulesets" {
  value       = data.cloudflare_rulesets.account_all.rulesets
  description = "All rulesets for the account"
}

output "filtered_rulesets" {
  value       = data.cloudflare_rulesets.filtered_by_name.rulesets
  description = "Filtered rulesets (filter will be removed in v5)"
}

output "ruleset_count" {
  value       = length(data.cloudflare_rulesets.account_all.rulesets)
  description = "Number of rulesets in account"
}

# Scenario 10: Conditional datasource usage
locals {
  use_account_scope = true
}

data "cloudflare_rulesets" "conditional" {
  count      = local.use_account_scope ? 1 : 0
  account_id = var.cloudflare_account_id
  filter {
    kind = "custom"
  }
}

# Scenario 11: Datasource with for_each (using a set)
locals {
  phases_to_check = toset([
    "http_request_firewall_managed",
    "http_request_firewall_custom",
    "http_ratelimit"
  ])
}

data "cloudflare_rulesets" "by_phase" {
  for_each   = local.phases_to_check
  account_id = var.cloudflare_account_id
  filter {
    phase = each.value
  }
}

# Scenario 12: Datasource with for_each (using a map)
locals {
  scope_config = {
    account = var.cloudflare_account_id
    zone    = var.cloudflare_zone_id
  }
}

data "cloudflare_rulesets" "by_scope" {
  for_each = local.scope_config
  # Using dynamic attribute selection
  account_id = each.key == "account" ? each.value : null
  zone_id    = each.key == "zone" ? each.value : null
}
