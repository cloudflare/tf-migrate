# Integration test for cloudflare_custom_hostname_fallback_origin v4 to v5 migration
# This resource has a simple schema with only zone_id and origin as user-provided fields

locals {
  name_prefix = "cftftest"
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
}

# Variables (no defaults as per pattern)
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

# Test 1: Basic resource with all required fields
resource "cloudflare_custom_hostname_fallback_origin" "basic" {
  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-fallback.cf-tf-test.com"
}

# Test 2: Resource with variable reference
resource "cloudflare_custom_hostname_fallback_origin" "with_variable" {
  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-backup.cf-tf-test.com"
}

# Test 3: Resource with explicit string values (no interpolation)
resource "cloudflare_custom_hostname_fallback_origin" "explicit" {
  zone_id = "0123456789abcdef0123456789abcdef"
  origin  = "${local.name_prefix}-explicit.cf-tf-test.com"
}

# Test 4: Using for_each with a set
resource "cloudflare_custom_hostname_fallback_origin" "for_each_set" {
  for_each = toset(["primary", "secondary", "tertiary"])

  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-${each.value}.cf-tf-test.com"
}

# Test 5: Using for_each with a map
resource "cloudflare_custom_hostname_fallback_origin" "for_each_map" {
  for_each = {
    prod    = "production.cf-tf-test.com"
    staging = "staging.cf-tf-test.com"
    dev     = "development.cf-tf-test.com"
  }

  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-${each.value}"
}

# Test 6: Using count
resource "cloudflare_custom_hostname_fallback_origin" "count" {
  count = 3

  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-count-${count.index}.cf-tf-test.com"
}

# Test 7: Conditional creation
resource "cloudflare_custom_hostname_fallback_origin" "conditional" {
  count = var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-conditional.cf-tf-test.com"
}

# Test 8: Using string functions
resource "cloudflare_custom_hostname_fallback_origin" "with_functions" {
  zone_id = var.cloudflare_zone_id
  origin  = lower("${local.name_prefix}-FUNCTIONS.cf-tf-test.com")
}

# Test 9: Using join() function
resource "cloudflare_custom_hostname_fallback_origin" "with_join" {
  zone_id = var.cloudflare_zone_id
  origin  = join(".", [local.name_prefix, "joined", "cf-tf-test", "com"])
}

# Test 10: Lifecycle meta-arguments
resource "cloudflare_custom_hostname_fallback_origin" "with_lifecycle" {
  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-lifecycle.cf-tf-test.com"

  lifecycle {
    create_before_destroy = true
  }
}

# Test 11: Using depends_on
resource "cloudflare_custom_hostname_fallback_origin" "with_depends_on" {
  depends_on = [cloudflare_custom_hostname_fallback_origin.basic]

  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-depends.cf-tf-test.com"
}

# Test 12: Minimal configuration (just required fields)
resource "cloudflare_custom_hostname_fallback_origin" "minimal" {
  zone_id = "fedcba9876543210fedcba9876543210"
  origin  = "${local.name_prefix}-minimal.cf-tf-test.com"
}

# Test 13: Using terraform expressions
resource "cloudflare_custom_hostname_fallback_origin" "expression" {
  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-${true ? "active" : "inactive"}.cf-tf-test.com"
}

# Test 14: Resource with complex origin format
resource "cloudflare_custom_hostname_fallback_origin" "complex_origin" {
  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-api-v2.cf-tf-test.com"
}

# Test 15: Multiple zones scenario
resource "cloudflare_custom_hostname_fallback_origin" "zone_1" {
  zone_id = "zone1zone1zone1zone1zone1zone1zo"
  origin  = "${local.name_prefix}-zone1.cf-tf-test.com"
}

resource "cloudflare_custom_hostname_fallback_origin" "zone_2" {
  zone_id = "zone2zone2zone2zone2zone2zone2zo"
  origin  = "${local.name_prefix}-zone2.cf-tf-test.com"
}

# Total: 20+ resource instances covering:
# - Basic resources
# - Variables and locals
# - for_each with sets and maps
# - count
# - Conditional creation
# - String functions
# - Lifecycle meta-arguments
# - depends_on
# - Multiple zones
