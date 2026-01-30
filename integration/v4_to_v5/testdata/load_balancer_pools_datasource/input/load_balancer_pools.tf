# Load Balancer Pools datasource v4 test fixtures
# Comprehensive integration test with 20+ datasource instances

# ========================================
# Pattern 1-2: Variables & Locals
# ========================================

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain"
  type        = string
}

# Resource-specific variables with defaults
variable "pool_name_filter" {
  type    = string
  default = "prod-.*"
}

variable "enable_monitoring" {
  type    = bool
  default = true
}

variable "pool_count" {
  type    = number
  default = 3
}

locals {
  name_prefix = "cftftest"
  common_account_id = var.cloudflare_account_id

  # Complex expression
  pool_name_pattern = "${local.name_prefix}-pool-.*"
  backup_pattern    = "backup-.*"
}

# ========================================
# Basic Scenarios (from minimal test)
# ========================================

# Scenario 1: Basic datasource - no filter
data "cloudflare_load_balancer_pools" "all" {
  account_id = var.cloudflare_account_id
}

# Scenario 2: With filter block (to be removed in v5)
data "cloudflare_load_balancer_pools" "filtered" {
  account_id = var.cloudflare_account_id

  filter {
    name = "example-.*"
  }
}

# Scenario 3: With variable reference in filter
data "cloudflare_load_balancer_pools" "with_var" {
  account_id = var.cloudflare_account_id

  filter {
    name = var.pool_name_filter
  }
}

# Scenario 4: With local reference in filter
data "cloudflare_load_balancer_pools" "with_local" {
  account_id = local.common_account_id

  filter {
    name = local.pool_name_pattern
  }
}

# Scenario 5: Empty filter block
data "cloudflare_load_balancer_pools" "empty_filter" {
  account_id = var.cloudflare_account_id

  filter {
  }
}

# Scenario 6: Filter with complex expression
data "cloudflare_load_balancer_pools" "backup_pools" {
  account_id = var.cloudflare_account_id

  filter {
    name = local.backup_pattern
  }
}

# ========================================
# Pattern 3: for_each with maps
# ========================================

variable "pool_filters" {
  type = map(object({
    name_pattern = string
  }))
  default = {
    "production" = {
      name_pattern = "prod-.*"
    }
    "staging" = {
      name_pattern = "stg-.*"
    }
    "development" = {
      name_pattern = "dev-.*"
    }
    "testing" = {
      name_pattern = "test-.*"
    }
  }
}

data "cloudflare_load_balancer_pools" "by_environment" {
  for_each = var.pool_filters

  account_id = local.common_account_id

  filter {
    name = each.value.name_pattern
  }
}

# ========================================
# Pattern 4: for_each with list converted to set
# ========================================

variable "regions" {
  type = list(object({
    key     = string
    pattern = string
  }))
  default = [
    {
      key     = "us-east"
      pattern = ".*-us-east-.*"
    },
    {
      key     = "us-west"
      pattern = ".*-us-west-.*"
    },
    {
      key     = "eu-central"
      pattern = ".*-eu-central-.*"
    }
  ]
}

data "cloudflare_load_balancer_pools" "by_region" {
  for_each = { for idx, region in var.regions : region.key => region }

  account_id = var.cloudflare_account_id

  filter {
    name = each.value.pattern
  }
}

# ========================================
# Pattern 5: Count-based datasources
# ========================================

data "cloudflare_load_balancer_pools" "pool_batch" {
  count = var.pool_count

  account_id = var.cloudflare_account_id

  filter {
    name = "batch-${count.index}-.*"
  }
}

# ========================================
# Pattern 6: Conditional datasource creation
# ========================================

data "cloudflare_load_balancer_pools" "monitoring_pools" {
  count = var.enable_monitoring ? 1 : 0

  account_id = var.cloudflare_account_id

  filter {
    name = "monitor-.*"
  }
}

data "cloudflare_load_balancer_pools" "conditional_by_env" {
  count = var.enable_monitoring ? 2 : 0

  account_id = var.cloudflare_account_id

  filter {
    name = count.index == 0 ? "primary-.*" : "secondary-.*"
  }
}

# ========================================
# Pattern 7: Cross-datasource references
# ========================================

# Reference all pools first
data "cloudflare_load_balancer_pools" "primary" {
  account_id = var.cloudflare_account_id
}

# Use output from primary datasource in another context
locals {
  has_pools = length(data.cloudflare_load_balancer_pools.primary.pools) > 0
}

# ========================================
# Pattern 9: Terraform functions
# ========================================

# Using coalesce
data "cloudflare_load_balancer_pools" "with_coalesce" {
  account_id = coalesce(local.common_account_id, var.cloudflare_account_id)

  filter {
    name = coalesce(var.pool_name_filter, "default-.*")
  }
}

# Using conditional expression
data "cloudflare_load_balancer_pools" "conditional_filter" {
  account_id = var.cloudflare_account_id

  filter {
    name = var.enable_monitoring ? "monitored-.*" : "unmonitored-.*"
  }
}

# Using join/format
locals {
  filter_patterns = ["api-.*", "web-.*"]
  combined_pattern = join("|", local.filter_patterns)
}

data "cloudflare_load_balancer_pools" "combined_pattern" {
  account_id = var.cloudflare_account_id

  filter {
    name = format("%s-pool-.*", local.name_prefix)
  }
}

# ========================================
# Outputs to test pools → result rename
# ========================================

# Basic output - array access
output "all_pool_ids" {
  value       = data.cloudflare_load_balancer_pools.all.pools[*].id
  description = "All pool IDs"
}

output "all_pool_names" {
  value       = data.cloudflare_load_balancer_pools.all.pools[*].name
  description = "All pool names"
}

# Filtered output
output "filtered_pool_names" {
  value       = data.cloudflare_load_balancer_pools.filtered.pools[*].name
  description = "Filtered pool names"
}

# Complex output with nested data access
output "primary_pool_origins" {
  value = {
    for pool in data.cloudflare_load_balancer_pools.primary.pools :
    pool.id => length(lookup(pool, "origins", []))
  }
  description = "Origin count per pool"
}

# Output testing attribute access patterns
output "pool_details" {
  value = [
    for pool in data.cloudflare_load_balancer_pools.all.pools : {
      id      = pool.id
      name    = pool.name
      enabled = lookup(pool, "enabled", true)
    }
  ]
  description = "Pool details"
}
