# Minimal load_balancer testdata for v4 to v5 migration
# This file uses v4 schema which has major breaking changes in v5

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
  description = "Domain for testing"
}

# Locals for naming consistency
locals {
  name_prefix = "cftftest"
}

# Note: Load balancer resources require load_balancer_pool resources
# These are not included here as they would need their own v4 to v5 migration
# For now, this file only tests the load_balancer resource schema changes

# 1. Basic load balancer (v4 schema)
# v4 uses: default_pool_ids, fallback_pool_id
# v5 uses: default_pools, fallback_pool
resource "cloudflare_load_balancer" "basic" {
  zone_id       = var.cloudflare_zone_id
  name          = "${local.name_prefix}-basic-lb.${var.cloudflare_domain}"
  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
}

# 2. Load balancer with session_affinity_attributes (v4 block, v5 map)
resource "cloudflare_load_balancer" "with_affinity" {
  zone_id              = var.cloudflare_zone_id
  name                 = "${local.name_prefix}-affinity-lb.${var.cloudflare_domain}"
  session_affinity     = "cookie"
  session_affinity_ttl = 3600

  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  session_affinity_attributes = {
    samesite = "Lax"
    secure   = "Always"
  }
}

# 3. Load balancer with region_pools (v4 blocks, v5 map)
resource "cloudflare_load_balancer" "with_region_pools" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-region-lb.${var.cloudflare_domain}"


  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  region_pools = {
    "WNAM" = ["pool-id-1"]
    "ENAM" = ["pool-id-2"]
  }
}

# 4. Load balancer with adaptive_routing (v4 block, v5 single object)
resource "cloudflare_load_balancer" "with_adaptive_routing" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-adaptive-lb.${var.cloudflare_domain}"

  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  adaptive_routing = {
    failover_across_pools = false
  }
}

# 5. Load balancer with location_strategy (v4 block, v5 single object)
resource "cloudflare_load_balancer" "with_location_strategy" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-location-lb.${var.cloudflare_domain}"

  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  location_strategy = {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
}

# 6. Load balancer with random_steering (v4 block, v5 single object)
resource "cloudflare_load_balancer" "with_random_steering" {
  zone_id         = var.cloudflare_zone_id
  name            = "${local.name_prefix}-random-lb.${var.cloudflare_domain}"
  steering_policy = "random"

  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  random_steering = {
    default_weight = 0.5
    pool_weights = {
      "pool-id-1" = 0.7
    }
  }
}

# 7. Load balancer with all single-object attributes (comprehensive test)
resource "cloudflare_load_balancer" "with_all_attributes" {
  zone_id              = var.cloudflare_zone_id
  name                 = "${local.name_prefix}-all-attrs-lb.${var.cloudflare_domain}"
  session_affinity     = "cookie"
  session_affinity_ttl = 3600
  steering_policy      = "random"




  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
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

# 8. Load balancer with rules - fixed_response
# v4: rules { fixed_response { ... } }
# v5: rules = [{ fixed_response = { ... } }]
resource "cloudflare_load_balancer" "with_rules_fixed_response" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-rules-fixed-lb.${var.cloudflare_domain}"

  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  rules = [
    {
      name      = "return 200"
      condition = "dns.qry.type == 28"
      fixed_response = {
        message_body = "hello"
        status_code  = 200
        content_type = "html"
        location     = "www.example.com"
      }
    }
  ]
}

# 9. Load balancer with rules - overrides containing block→attribute sub-blocks
# v4: rules { overrides { session_affinity_attributes { } adaptive_routing { } ... } }
# v5: rules = [{ overrides = { session_affinity_attributes = { } adaptive_routing = { } ... } }]
resource "cloudflare_load_balancer" "with_rules_overrides" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-rules-overrides-lb.${var.cloudflare_domain}"

  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  rules = [
    {
      name      = "override rule"
      condition = "dns.qry.type == 28"
      overrides = {
        steering_policy = "geo"
        session_affinity_attributes = {
          samesite               = "Auto"
          secure                 = "Auto"
          zero_downtime_failover = "sticky"
        }
        adaptive_routing = {
          failover_across_pools = true
        }
        location_strategy = {
          prefer_ecs = "always"
          mode       = "resolver_ip"
        }
        random_steering = {
          default_weight = 0.2
        }
        region_pools = {
          "ENAM" = ["pool-id-1"]
        }
      }
    }
  ]
}

# 10. Load balancer with multiple rules (mixed overrides and fixed_response)
resource "cloudflare_load_balancer" "with_multiple_rules" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-multi-rules-lb.${var.cloudflare_domain}"


  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  rules = [
    {
      name      = "geo rule"
      condition = "dns.qry.type == 28"
      priority  = 1
      overrides = {
        steering_policy = "geo"
        region_pools = {
          "WNAM" = ["pool-id-1"]
          "ENAM" = ["pool-id-2"]
        }
      }
    },
    {
      name      = "fallback rule"
      condition = "dns.qry.type == 1"
      priority  = 2
      fixed_response = {
        message_body = "not found"
        status_code  = 404
      }
    }
  ]
}
