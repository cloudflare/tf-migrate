# E2E Test: cloudflare_load_balancer_pool
# Minimal e2e test that can be applied with real infrastructure

variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_domain" {
  type        = string
  description = "Domain for testing"
}

locals {
  name_prefix = "cftftest"
}

##########################
# E2E TEST POOLS
##########################

# 1. Basic pool with single origin (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer_pool" "e2e_basic" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-basic-pool"


  minimum_origins = 1
  enabled         = true
  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }]
}

# 2. Pool with multiple origins (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer_pool" "e2e_multi" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-multi-pool"



  minimum_origins = 1
  enabled         = true
  origins = [{
    name    = "origin-1"
    address = "192.0.2.10"
    enabled = true
    weight  = 1
    }, {
    name    = "origin-2"
    address = "192.0.2.11"
    enabled = true
    weight  = 1
  }]
}

# 3. Pool with load_shedding (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer_pool" "e2e_shedding" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-shedding-pool"



  minimum_origins = 1
  enabled         = true
  origins = [{
    name    = "origin-1"
    address = "192.0.2.20"
    enabled = true
  }]
  load_shedding = {
    default_percent = 55
    default_policy  = "random"
    session_percent = 30
    session_policy  = "hash"
  }
}

# Output pool IDs for use by load balancers
output "e2e_basic_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_basic.id
}

output "e2e_multi_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_multi.id
}

output "e2e_shedding_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_shedding.id
}
