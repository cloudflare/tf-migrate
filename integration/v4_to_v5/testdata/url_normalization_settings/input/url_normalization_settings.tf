# Terraform v4 to v5 Migration Integration Tests
# Resource: cloudflare_url_normalization_settings
#
# This file contains comprehensive test cases covering all Terraform patterns
# Target: 20+ resource instances

# Variables (auto-provided by test infrastructure)
# - var.cloudflare_account_id
# - var.cloudflare_zone_id

# Locals for test organization
locals {
  common_zone = var.cloudflare_zone_id
  test_zones = toset([
    "zone1-abc123",
    "zone2-def456",
    "zone3-ghi789",
  ])
  zone_configs = {
    "prod"    = { zone_id = "prod-zone-123", type = "cloudflare", scope = "both" }
    "staging" = { zone_id = "stag-zone-456", type = "rfc3986", scope = "incoming" }
    "dev"     = { zone_id = "dev-zone-789", type = "cloudflare", scope = "none" }
  }
}

# Pattern 1: Basic resource with variable reference
resource "cloudflare_url_normalization_settings" "basic" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"
}

# Pattern 2: All type and scope combinations
resource "cloudflare_url_normalization_settings" "cloudflare_incoming" {
  zone_id = "test-zone-001"
  type    = "cloudflare"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "cloudflare_both" {
  zone_id = "test-zone-002"
  type    = "cloudflare"
  scope   = "both"
}

resource "cloudflare_url_normalization_settings" "cloudflare_none" {
  zone_id = "test-zone-003"
  type    = "cloudflare"
  scope   = "none"
}

resource "cloudflare_url_normalization_settings" "rfc3986_incoming" {
  zone_id = "test-zone-004"
  type    = "rfc3986"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "rfc3986_both" {
  zone_id = "test-zone-005"
  type    = "rfc3986"
  scope   = "both"
}

resource "cloudflare_url_normalization_settings" "rfc3986_none" {
  zone_id = "test-zone-006"
  type    = "rfc3986"
  scope   = "none"
}

# Pattern 3: for_each with map (3 resources)
resource "cloudflare_url_normalization_settings" "foreach_map" {
  for_each = local.zone_configs

  zone_id = each.value.zone_id
  type    = each.value.type
  scope   = each.value.scope
}

# Pattern 4: for_each with set using toset() (3 resources)
resource "cloudflare_url_normalization_settings" "foreach_set" {
  for_each = local.test_zones

  zone_id = each.value
  type    = "cloudflare"
  scope   = "incoming"
}

# Pattern 5: count-based resources (3 resources)
resource "cloudflare_url_normalization_settings" "count_based" {
  count = 3

  zone_id = "count-zone-${count.index}"
  type    = count.index == 0 ? "cloudflare" : "rfc3986"
  scope   = count.index < 2 ? "incoming" : "both"
}

# Pattern 6: Conditional creation (ternary operators)
resource "cloudflare_url_normalization_settings" "conditional" {
  count = var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"
}

# Pattern 7: Using locals for values
resource "cloudflare_url_normalization_settings" "with_local" {
  zone_id = local.common_zone
  type    = "cloudflare"
  scope   = "both"
}

# Pattern 8: Lifecycle meta-arguments
resource "cloudflare_url_normalization_settings" "with_lifecycle" {
  zone_id = "lifecycle-zone-001"
  type    = "cloudflare"
  scope   = "incoming"

  lifecycle {
    prevent_destroy = true
  }
}

resource "cloudflare_url_normalization_settings" "with_create_before_destroy" {
  zone_id = "lifecycle-zone-002"
  type    = "rfc3986"
  scope   = "both"

  lifecycle {
    create_before_destroy = true
  }
}

# Pattern 9: String interpolation
resource "cloudflare_url_normalization_settings" "with_interpolation" {
  zone_id = "${var.cloudflare_zone_id}-suffix"
  type    = "cloudflare"
  scope   = "incoming"
}

# Pattern 10: Terraform functions
resource "cloudflare_url_normalization_settings" "with_function" {
  zone_id = lower("ZONE-ABC123")
  type    = upper("cloudflare")
  scope   = lower("INCOMING")
}

# Pattern 11: depends_on meta-argument
resource "cloudflare_url_normalization_settings" "with_depends" {
  zone_id = "depends-zone-001"
  type    = "cloudflare"
  scope   = "incoming"

  depends_on = [
    cloudflare_url_normalization_settings.basic
  ]
}

# Pattern 12: Complex for_each with inline object
resource "cloudflare_url_normalization_settings" "complex_foreach" {
  for_each = {
    "web" = {
      zone = "web-zone-123"
      norm = "cloudflare"
      when = "both"
    }
    "api" = {
      zone = "api-zone-456"
      norm = "rfc3986"
      when = "incoming"
    }
  }

  zone_id = each.value.zone
  type    = each.value.norm
  scope   = each.value.when
}

# Total resources:
# - 1 basic
# - 6 all combinations
# - 3 foreach_map
# - 3 foreach_set
# - 3 count_based
# - 1 conditional
# - 1 with_local
# - 2 lifecycle variations
# - 1 with_interpolation
# - 1 with_function
# - 1 with_depends
# - 2 complex_foreach
# = 25 resource declarations (22+ actual instances after conditionals)
