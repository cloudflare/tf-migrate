# Comprehensive Integration Tests for cloudflare_load_balancer_pool
# This file tests v4 to v5 migration including:
# - Static origins blocks → origins attribute
# - Dynamic origins blocks → for expressions
# - load_shedding blocks → load_shedding attribute
# - origin_steering blocks → origin_steering attribute

# Variables for DRY configuration
variable "cloudflare_account_id" {
  type    = string
  default = "f037e56e89293a057740de681ac9abbe"
}

variable "cloudflare_zone_id" {
  type    = string
  default = "0da42c8d2132a9ddaf714f9e7c920711"
}

variable "cloudflare_domain" {
  type        = string
  description = "Domain for testing (not used by this module but accepted for consistency)"
}

# Locals for naming consistency and dynamic origin configs
locals {
  name_prefix = "cftftest"

  # Simple list of origin configs for dynamic block
  origin_configs = [
    {
      name    = "origin-1"
      address = "192.0.2.1"
      enabled = true
    },
    {
      name    = "origin-2"
      address = "192.0.2.2"
      enabled = true
    },
    {
      name    = "origin-3"
      address = "192.0.2.3"
      enabled = false
    }
  ]

  # Origin configs with weight for advanced testing
  weighted_origins = [
    {
      name    = "weighted-1"
      address = "192.0.2.10"
      weight  = 0.8
    },
    {
      name    = "weighted-2"
      address = "192.0.2.11"
      weight  = 0.2
    }
  ]
}

##########################
# BASIC PATTERNS
##########################

# 1. Basic pool with single origin (v4 uses origins block)
resource "cloudflare_load_balancer_pool" "basic" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-basic-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }]
}

# 2. Pool with multiple origins
resource "cloudflare_load_balancer_pool" "multi_origin" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-multi-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
    weight  = 1
    }, {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = true
    weight  = 1
    }, {
    name    = "origin-3"
    address = "192.0.2.3"
    enabled = false
    weight  = 0.5
  }]
}

# 3. Pool with origin headers
resource "cloudflare_load_balancer_pool" "with_headers" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-headers-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }]
}

# 4. Pool with load_shedding (v4 block, v5 attribute)
resource "cloudflare_load_balancer_pool" "with_load_shedding" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-shedding-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }]
  load_shedding = {
    default_percent = 50
    default_policy  = "random"
    session_percent = 25
    session_policy  = "hash"
  }
}

# 5. Pool with origin_steering (v4 block, v5 attribute)
resource "cloudflare_load_balancer_pool" "with_steering" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-steering-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }]
  origin_steering = {
    policy = "random"
  }
}

# 6. Pool with for_each pattern
resource "cloudflare_load_balancer_pool" "foreach" {
  for_each = toset(["pool1", "pool2"])

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-${each.key}"

  origins = [{
    name    = "${each.key}-origin"
    address = "192.0.2.100"
    enabled = true
  }]
}

##########################
# DYNAMIC BLOCK PATTERNS
##########################

# 7. Pool with dynamic origins block using local list
resource "cloudflare_load_balancer_pool" "dynamic_basic" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-dynamic-basic"

  origins = [for origins in local.origin_configs : {
    name    = origins.name
    address = origins.address
    enabled = origins.enabled
  }]
}

# 8. Pool with dynamic origins and static load_shedding block
resource "cloudflare_load_balancer_pool" "dynamic_with_load_shedding" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-dynamic-shedding"


  origins = [for origins in local.origin_configs : {
    name    = origins.name
    address = origins.address
    enabled = origins.enabled
  }]
  load_shedding = {
    default_percent = 50
    default_policy  = "random"
  }
}

# 9. Pool with dynamic origins including weight attribute
resource "cloudflare_load_balancer_pool" "dynamic_weighted" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-dynamic-weighted"

  origins = [for origins in local.weighted_origins : {
    name    = origins.name
    address = origins.address
    weight  = origins.weight
    enabled = true
  }]
}

# 10. Pool with dynamic origins using custom iterator name
resource "cloudflare_load_balancer_pool" "dynamic_custom_iterator" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-dynamic-iterator"

  origins = [for origin_item in local.origin_configs : {
    name    = origin_item.name
    address = origin_item.address
    enabled = origin_item.enabled
  }]
}
