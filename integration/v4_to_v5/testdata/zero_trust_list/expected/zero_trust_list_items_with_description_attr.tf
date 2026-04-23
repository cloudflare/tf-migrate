# Integration test fixtures for Bug #001:
# items_with_description written as an attribute (not a block) must be migrated.

locals {
  tunnel_routes = [
    {
      value       = "10.0.0.0/8"
      description = "Internal network"
    },
    {
      value       = "172.16.0.0/12"
      description = "RFC 1918 block"
    },
  ]
}

# Case A: items_with_description = local.<name>  (opaque expression)
# Expected: attribute is renamed to items, expression preserved verbatim.
resource "cloudflare_zero_trust_list" "iwd_opaque_local" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Do Not Inspect Tunnels IWD Opaque"
  description = "List using items_with_description as a local reference"
  type        = "IP"
  items       = local.tunnel_routes
}

# Case B: items_with_description = [...] (inline object list with resource references)
# AND a separate items = [...] attribute.
# Expected: merged into a single items attribute; items_with_description entries first.
resource "cloudflare_zero_trust_list" "iwd_inline_with_refs" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Do Not Inspect IPs IWD Inline"
  description = "List using items_with_description as an inline object list with references"
  type        = "IP"
  items = [
    {
      value       = cloudflare_zero_trust_list.iwd_opaque_local.id
      description = "Reference to another list"
    },
    {
      value       = "8.14.199.1"
      description = null
    },
    {
      value       = "8.14.199.2"
      description = null
    }
  ]
}

# Case C: items_with_description = [...] (inline object list, static strings only)
# No separate items attr.
# Expected: items_with_description replaced by items with object entries.
resource "cloudflare_zero_trust_list" "iwd_inline_static" {
  account_id = var.cloudflare_account_id
  name       = "cftftest Do Not Inspect IPs IWD Static"
  type       = "IP"
  items = [
    {
      value       = "192.168.1.1"
      description = "Gateway"
    },
    {
      value       = "10.0.0.1"
      description = "Internal host"
    }
  ]
}
