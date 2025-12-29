# E2E Test: cloudflare_load_balancer_pool
# Minimal e2e test that can be applied with real infrastructure
# Tests all available attributes and configurations

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

# 1. Comprehensive pool with all common attributes
resource "cloudflare_load_balancer_pool" "e2e_comprehensive" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-comprehensive"

  # Pool configuration
  enabled         = true
  minimum_origins = 1
  check_regions   = ["WEU", "ENAM"]
  description     = "Comprehensive E2E test pool with all attributes"

  # Proximity steering coordinates
  latitude  = 37.7749
  longitude = -122.4194

  # Notification settings
  notification_email = "alerts@example.com"



  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
    weight  = 0.6
    }, {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = true
    weight  = 0.4
  }]
  load_shedding = {
    default_percent = 55
    default_policy  = "random"
    session_percent = 30
    session_policy  = "hash"
  }
  origin_steering = {
    policy = "random"
  }
}

# 2. Minimal pool for baseline testing
resource "cloudflare_load_balancer_pool" "e2e_minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-minimal"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.10"
    enabled = true
  }]
}

# 3. Pool with origin steering using least_outstanding_requests policy
resource "cloudflare_load_balancer_pool" "e2e_advanced_steering" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-e2e-advanced"


  origins = [{
    name    = "origin-us-east"
    address = "192.0.2.21"
    enabled = true
    weight  = 1
    }, {
    name    = "origin-us-west"
    address = "192.0.2.20"
    enabled = true
    weight  = 1
  }]
  origin_steering = {
    policy = "least_outstanding_requests"
  }
}

##########################
# OUTPUTS
##########################

# Output pool IDs for use by load balancers
output "e2e_comprehensive_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_comprehensive.id
}

output "e2e_minimal_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_minimal.id
}

output "e2e_advanced_steering_pool_id" {
  value = cloudflare_load_balancer_pool.e2e_advanced_steering.id
}
