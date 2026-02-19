# Comprehensive test cases for cloudflare_argo v4 to v5 migration
# Covers all transformation scenarios from v4_to_v5.go and unit tests

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

variable "enable_smart_routing" {
  description = "Variable for testing variable references"
  type        = string
  default     = "on"
}

# Scenario 1: Smart routing only
resource "cloudflare_argo" "smart_routing_only" {
  zone_id       = var.cloudflare_zone_id
  smart_routing = "on"
}

# Scenario 2: Tiered caching only
resource "cloudflare_argo" "tiered_caching_only" {
  zone_id        = var.cloudflare_zone_id
  tiered_caching = "on"
}

# Scenario 3: Both attributes (creates both resources with naming conflict resolution)
resource "cloudflare_argo" "both_attributes" {
  zone_id        = var.cloudflare_zone_id
  smart_routing  = "on"
  tiered_caching = "on"
}

# Scenario 4: Neither attribute (defaults to smart_routing with value off)
resource "cloudflare_argo" "neither_attribute" {
  zone_id = var.cloudflare_zone_id
}

# Scenario 5: Smart routing with lifecycle block
resource "cloudflare_argo" "smart_routing_lifecycle" {
  zone_id       = var.cloudflare_zone_id
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing]
  }
}

# Scenario 6: Tiered caching with lifecycle block
resource "cloudflare_argo" "tiered_caching_lifecycle" {
  zone_id        = var.cloudflare_zone_id
  tiered_caching = "on"

  lifecycle {
    prevent_destroy = true
  }
}

# Scenario 7: Both attributes with lifecycle block
resource "cloudflare_argo" "both_with_lifecycle" {
  zone_id        = var.cloudflare_zone_id
  smart_routing  = "on"
  tiered_caching = "on"

  lifecycle {
    ignore_changes = [smart_routing, tiered_caching]
  }
}

# Scenario 8: Smart routing explicitly off
resource "cloudflare_argo" "smart_routing_off" {
  zone_id       = var.cloudflare_zone_id
  smart_routing = "off"
}

# Scenario 9: Tiered caching explicitly off
resource "cloudflare_argo" "tiered_caching_off" {
  zone_id        = var.cloudflare_zone_id
  tiered_caching = "off"
}

# Scenario 10: With variable reference for attribute
resource "cloudflare_argo" "with_variable" {
  zone_id       = var.cloudflare_zone_id
  smart_routing = var.enable_smart_routing
}

# Scenario 11: Both attributes with mixed values
resource "cloudflare_argo" "both_mixed_values" {
  zone_id        = var.cloudflare_zone_id
  smart_routing  = "off"
  tiered_caching = "on"
}
