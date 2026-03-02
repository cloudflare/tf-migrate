# Comprehensive Integration Test for zero_trust_tunnel_cloudflared_virtual_network

variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

locals {
  account_id = var.cloudflare_account_id
  prefix     = "cftftest-vnet"
}









# Pattern 1: OLD v4 name - minimal
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "minimal" {
  account_id = local.account_id
  name       = "${local.prefix}-minimal"
}

moved {
  from = cloudflare_tunnel_virtual_network.minimal
  to   = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.minimal
}

# Pattern 2: NEW v4 name - complete
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "complete" {
  account_id         = local.account_id
  name               = "${local.prefix}-complete"
  is_default_network = false
  comment            = "Integration test"
}

moved {
  from = cloudflare_zero_trust_tunnel_virtual_network.complete
  to   = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.complete
}

# Pattern 4: Empty optionals (tests default handling)
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "empty_opts" {
  account_id = local.account_id
  name       = "${local.prefix}-empty"
}

moved {
  from = cloudflare_zero_trust_tunnel_virtual_network.empty_opts
  to   = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.empty_opts
}

# Pattern 5: for_each
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "foreach" {
  for_each = toset(["prod", "staging", "dev"])

  account_id = local.account_id
  name       = "${local.prefix}-${each.key}"
  comment    = "Network for ${each.key}"
}

moved {
  from = cloudflare_tunnel_virtual_network.foreach
  to   = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.foreach
}

# Pattern 6: count
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "counted" {
  count = 3

  account_id = local.account_id
  name       = "${local.prefix}-count-${count.index}"
}

moved {
  from = cloudflare_zero_trust_tunnel_virtual_network.counted
  to   = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.counted
}

# Pattern 7: Lifecycle
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "lifecycle" {
  account_id = local.account_id
  name       = "${local.prefix}-lifecycle"

  lifecycle {
    create_before_destroy = true
  }
}

moved {
  from = cloudflare_tunnel_virtual_network.lifecycle
  to   = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.lifecycle
}

# Pattern 8: Both v4 names together
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "old_name" {
  account_id = local.account_id
  name       = "${local.prefix}-old"
}

moved {
  from = cloudflare_tunnel_virtual_network.old_name
  to   = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.old_name
}

resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "new_name" {
  account_id = local.account_id
  name       = "${local.prefix}-new"
}

moved {
  from = cloudflare_zero_trust_tunnel_virtual_network.new_name
  to   = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.new_name
}
