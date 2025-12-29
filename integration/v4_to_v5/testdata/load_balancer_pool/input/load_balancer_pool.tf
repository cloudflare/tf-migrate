# Comprehensive Integration Tests for cloudflare_load_balancer_pool
# This file tests v4 to v5 migration

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

# Locals for naming consistency
locals {
  name_prefix = "cftftest"
}

##########################
# BASIC PATTERNS
##########################

# 1. Basic pool with single origin (v4 uses origins block)
resource "cloudflare_load_balancer_pool" "basic" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-basic-pool"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }
}

# 2. Pool with multiple origins
resource "cloudflare_load_balancer_pool" "multi_origin" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-multi-pool"



  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
    weight  = 1
  }
  origins {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = true
    weight  = 1
  }
  origins {
    name    = "origin-3"
    address = "192.0.2.3"
    enabled = false
    weight  = 0.5
  }
}

# 3. Pool with origin headers
resource "cloudflare_load_balancer_pool" "with_headers" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-headers-pool"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }
}

# 4. Pool with load_shedding (v4 block, v5 attribute)
resource "cloudflare_load_balancer_pool" "with_load_shedding" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-shedding-pool"


  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }
  load_shedding {
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


  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }
  origin_steering {
    policy = "random"
  }
}

# 6. Pool with monitor
resource "cloudflare_load_balancer_pool" "with_monitor" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-monitored-pool"
  monitor    = "monitor-id-123"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }
}

# 7. Pool with notification settings
resource "cloudflare_load_balancer_pool" "with_notifications" {
  account_id         = var.cloudflare_account_id
  name               = "${local.name_prefix}-notified-pool"
  notification_email = "alerts@example.com"
  enabled            = true
  minimum_origins    = 1
  check_regions      = ["WEU", "ENAM"]


  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }
  origins {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = true
  }
}

# 8. for_each pattern
resource "cloudflare_load_balancer_pool" "foreach" {
  for_each = toset(["pool1", "pool2"])

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-${each.key}"

  origins {
    name    = "${each.key}-origin"
    address = "192.0.2.100"
    enabled = true
  }
}
