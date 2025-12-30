# E2E Test: cloudflare_load_balancer
# Minimal e2e test that can be applied with real infrastructure
# Creates pools and load balancers in the same module to avoid dependency issues

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

resource "cloudflare_load_balancer_pool" "lb_e2e_basic" {
  account_id      = var.cloudflare_account_id
  name            = "${local.name_prefix}-lb-e2e-basic-pool"
  minimum_origins = 1
  enabled         = true

  origins = [{
    name    = "origin-1"
    address = "192.0.2.100"
    enabled = true
  }]
}

resource "cloudflare_load_balancer_pool" "lb_e2e_fallback" {
  account_id      = var.cloudflare_account_id
  name            = "${local.name_prefix}-lb-e2e-fallback-pool"
  minimum_origins = 1
  enabled         = true

  origins = [{
    name    = "origin-fallback"
    address = "192.0.2.101"
    enabled = true
  }]
}

##########################
# E2E TEST LOAD BALANCERS
##########################

# 1. Basic load balancer (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer" "e2e_basic" {
  zone_id         = var.cloudflare_zone_id
  name            = "${local.name_prefix}-e2e-basic-lb.${var.cloudflare_domain}"
  enabled         = true
  steering_policy = "off"
  ttl             = 30
  default_pools   = [cloudflare_load_balancer_pool.lb_e2e_basic.id]
  fallback_pool   = cloudflare_load_balancer_pool.lb_e2e_fallback.id
}

# 2. Load balancer with session_affinity_attributes (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer" "e2e_affinity" {
  zone_id              = var.cloudflare_zone_id
  name                 = "${local.name_prefix}-e2e-affinity-lb.${var.cloudflare_domain}"
  session_affinity     = "cookie"
  session_affinity_ttl = 3600


  enabled         = true
  steering_policy = "off"
  ttl             = 30
  default_pools   = [cloudflare_load_balancer_pool.lb_e2e_basic.id]
  fallback_pool   = cloudflare_load_balancer_pool.lb_e2e_fallback.id
  session_affinity_attributes = {
    samesite = "Lax"
    secure   = "Always"
  }
}

# 3. Load balancer with adaptive_routing (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer" "e2e_adaptive_routing" {
  zone_id         = var.cloudflare_zone_id
  name            = "${local.name_prefix}-e2e-adaptive-lb.${var.cloudflare_domain}"
  enabled         = true
  steering_policy = "off"
  ttl             = 30

  default_pools = [cloudflare_load_balancer_pool.lb_e2e_basic.id]
  fallback_pool = cloudflare_load_balancer_pool.lb_e2e_fallback.id
  adaptive_routing = {
    failover_across_pools = false
  }
}

# 4. Load balancer with location_strategy (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer" "e2e_location_strategy" {
  zone_id         = var.cloudflare_zone_id
  name            = "${local.name_prefix}-e2e-location-lb.${var.cloudflare_domain}"
  enabled         = true
  steering_policy = "off"
  ttl             = 30

  default_pools = [cloudflare_load_balancer_pool.lb_e2e_basic.id]
  fallback_pool = cloudflare_load_balancer_pool.lb_e2e_fallback.id
  location_strategy = {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
}

# 5. Load balancer with random_steering (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer" "e2e_random_steering" {
  zone_id         = var.cloudflare_zone_id
  name            = "${local.name_prefix}-e2e-random-lb.${var.cloudflare_domain}"
  enabled         = true
  steering_policy = "random"
  ttl             = 30

  default_pools = [cloudflare_load_balancer_pool.lb_e2e_basic.id]
  fallback_pool = cloudflare_load_balancer_pool.lb_e2e_fallback.id
  random_steering = {
    default_weight = 0.5
  }
}

# 6. Load balancer with all single-object attributes (v4 syntax - will be migrated to v5)
resource "cloudflare_load_balancer" "e2e_all_attributes" {
  zone_id              = var.cloudflare_zone_id
  name                 = "${local.name_prefix}-e2e-all-attrs-lb.${var.cloudflare_domain}"
  session_affinity     = "cookie"
  session_affinity_ttl = 3600
  enabled              = true
  steering_policy      = "random"
  ttl                  = 30




  default_pools = [cloudflare_load_balancer_pool.lb_e2e_basic.id]
  fallback_pool = cloudflare_load_balancer_pool.lb_e2e_fallback.id
  session_affinity_attributes = {
    samesite = "Lax"
    secure   = "Always"
  }
  adaptive_routing = {
    failover_across_pools = false
  }
  location_strategy = {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
  random_steering = {
    default_weight = 0.5
  }
}
