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









# Pattern 9: Cross-resource reference using both v4 names
# This validates that GetResourceRename() returns ALL v4 names for cross-file reference updates
# v4 resource name option 1: cloudflare_zero_trust_tunnel_virtual_network
# v4 resource name option 2: cloudflare_tunnel_virtual_network




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

# Create a tunnel to reference in the route
resource "cloudflare_zero_trust_tunnel_cloudflared" "for_route_test" {
  account_id    = local.account_id
  name          = "${local.prefix}-route-test-tunnel"
  tunnel_secret = base64encode("test-secret-that-is-at-least-32-bytes-long")
  config_src    = "local"
}

moved {
  from = cloudflare_tunnel.for_route_test
  to   = cloudflare_zero_trust_tunnel_cloudflared.for_route_test
}

# Route using option 1 v4 name that references the virtual network
# cloudflare_zero_trust_tunnel_virtual_network (the "option 1" v4 name)
resource "cloudflare_zero_trust_tunnel_cloudflared_route" "test_opt1_v4_name_ref" {
  account_id         = local.account_id
  tunnel_id          = cloudflare_zero_trust_tunnel_cloudflared.for_route_test.id
  network            = "10.254.0.0/16"
  virtual_network_id = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.new_name.id
  comment            = "VNet: ${cloudflare_zero_trust_tunnel_cloudflared_virtual_network.new_name.name}"
}

moved {
  from = cloudflare_zero_trust_tunnel_route.test_opt1_v4_name_ref
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.test_opt1_v4_name_ref
}

# Route using option 2 v4 name that references the virtual network
# cloudflare_tunnel_virtual_network (the "option 2" v4 name)
resource "cloudflare_zero_trust_tunnel_cloudflared_route" "test_opt2_v4_name_ref" {
  account_id         = local.account_id
  tunnel_id          = cloudflare_zero_trust_tunnel_cloudflared.for_route_test.id
  network            = "10.255.0.0/16"
  virtual_network_id = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.old_name.id
  comment            = "VNet: ${cloudflare_zero_trust_tunnel_cloudflared_virtual_network.old_name.name}"
}

moved {
  from = cloudflare_tunnel_route.test_opt2_v4_name_ref
  to   = cloudflare_zero_trust_tunnel_cloudflared_route.test_opt2_v4_name_ref
}
