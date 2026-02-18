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












resource "cloudflare_argo_smart_routing" "smart_routing_only" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}

moved {
  from = cloudflare_argo.smart_routing_only
  to   = cloudflare_argo_smart_routing.smart_routing_only
}

resource "cloudflare_argo_tiered_caching" "tiered_caching_only" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}

moved {
  from = cloudflare_argo.tiered_caching_only
  to   = cloudflare_argo_tiered_caching.tiered_caching_only
}

resource "cloudflare_argo_smart_routing" "both_attributes" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}

moved {
  from = cloudflare_argo.both_attributes
  to   = cloudflare_argo_smart_routing.both_attributes
}

resource "cloudflare_argo_tiered_caching" "both_attributes_tiered" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}

resource "cloudflare_argo_smart_routing" "neither_attribute" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

moved {
  from = cloudflare_argo.neither_attribute
  to   = cloudflare_argo_smart_routing.neither_attribute
}

resource "cloudflare_argo_smart_routing" "smart_routing_lifecycle" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
  lifecycle {
    ignore_changes = [value]
  }
}

moved {
  from = cloudflare_argo.smart_routing_lifecycle
  to   = cloudflare_argo_smart_routing.smart_routing_lifecycle
}

resource "cloudflare_argo_tiered_caching" "tiered_caching_lifecycle" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
  lifecycle {
    prevent_destroy = true
  }
}

moved {
  from = cloudflare_argo.tiered_caching_lifecycle
  to   = cloudflare_argo_tiered_caching.tiered_caching_lifecycle
}

resource "cloudflare_argo_smart_routing" "both_with_lifecycle" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
  lifecycle {
    ignore_changes = [value]
  }
}

moved {
  from = cloudflare_argo.both_with_lifecycle
  to   = cloudflare_argo_smart_routing.both_with_lifecycle
}

resource "cloudflare_argo_tiered_caching" "both_with_lifecycle_tiered" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
  lifecycle {
    ignore_changes = [value]
  }
}

resource "cloudflare_argo_smart_routing" "smart_routing_off" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

moved {
  from = cloudflare_argo.smart_routing_off
  to   = cloudflare_argo_smart_routing.smart_routing_off
}

resource "cloudflare_argo_tiered_caching" "tiered_caching_off" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

moved {
  from = cloudflare_argo.tiered_caching_off
  to   = cloudflare_argo_tiered_caching.tiered_caching_off
}

resource "cloudflare_argo_smart_routing" "with_variable" {
  zone_id = var.cloudflare_zone_id
  value   = var.enable_smart_routing
}

moved {
  from = cloudflare_argo.with_variable
  to   = cloudflare_argo_smart_routing.with_variable
}

resource "cloudflare_argo_smart_routing" "both_mixed_values" {
  zone_id = var.cloudflare_zone_id
  value   = "off"
}

moved {
  from = cloudflare_argo.both_mixed_values
  to   = cloudflare_argo_smart_routing.both_mixed_values
}

resource "cloudflare_argo_tiered_caching" "both_mixed_values_tiered" {
  zone_id = var.cloudflare_zone_id
  value   = "on"
}
