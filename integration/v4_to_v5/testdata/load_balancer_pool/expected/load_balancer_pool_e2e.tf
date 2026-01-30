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

  # List of backend origins for dynamic configuration
  backend_origins = [
    {
      name    = "backend-1"
      address = "192.0.2.100"
      host    = "api1.${var.cloudflare_domain}"
      weight  = 1.0
    },
    {
      name    = "backend-2"
      address = "192.0.2.101"
      host    = "api2.${var.cloudflare_domain}"
      weight  = 1.0
    },
    {
      name    = "backend-3"
      address = "192.0.2.102"
      host    = "api3.${var.cloudflare_domain}"
      weight  = 0.5
    }
  ]
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

# 4. Pool with dynamic origins and headers (v4 syntax - will be migrated to v5 for expression)
resource "cloudflare_load_balancer_pool" "e2e_dynamic_with_headers" {
  account_id      = var.cloudflare_account_id
  name            = "${local.name_prefix}-e2e-dynamic-headers-pool"
  minimum_origins = 1
  enabled         = true
  description     = "Pool with dynamic origins and headers"


  origins = [for value in local.backend_origins : {
    name    = value.name
    address = value.address
    enabled = true
    weight  = value.weight
    header  = { host = [value.host] }
  }]
  origin_steering = {
    policy = "random"
  }
}

# 5. Pool with dynamic origins without headers (v4 syntax - will be migrated to v5 for expression)
resource "cloudflare_load_balancer_pool" "e2e_dynamic_simple" {
  account_id      = var.cloudflare_account_id
  name            = "${local.name_prefix}-e2e-dynamic-simple-pool"
  minimum_origins = 1
  enabled         = true

  origins = [for value in local.backend_origins : {
    name    = value.name
    address = value.address
    enabled = true
  }]
}

# 6. Pool with static origins and headers (v4 syntax - will be migrated to v5 array)
# Note: The second origin (static-origin-2) is disabled and intentionally has no header
# configuration because the Cloudflare API does not return header values for disabled
# origins, which would cause persistent drift.
resource "cloudflare_load_balancer_pool" "e2e_static_with_headers" {
  account_id      = var.cloudflare_account_id
  name            = "${local.name_prefix}-e2e-static-headers-pool"
  minimum_origins = 1
  enabled         = true



  origins = [{
    name    = "static-origin-1"
    address = "192.0.2.200"
    enabled = true
    header  = { host = ["static1.${var.cloudflare_domain}"] }
    }, {
    name    = "static-origin-2"
    address = "192.0.2.201"
    enabled = false
  }]
  load_shedding = {
    default_percent = 75
    default_policy  = "random"
    session_percent = 50
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

output "e2e_dynamic_headers_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_dynamic_with_headers.id
}

output "e2e_dynamic_simple_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_dynamic_simple.id
}

output "e2e_static_headers_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_static_with_headers.id
}
