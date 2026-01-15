# Comprehensive Integration Tests for workers_for_platforms_dispatch_namespace
# Target: 15-30+ resource instances covering all Terraform patterns

# Variables (no defaults - must be provided by test framework or user)
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

# Locals for test resource naming
locals {
  name_prefix = "cftftest"
}

# ==============================================================================
# Pattern 1: Basic Resources (minimal and full configurations)
# ==============================================================================

# Test 1: Minimal configuration (required fields only)
resource "cloudflare_workers_for_platforms_dispatch_namespace" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-full-namespace"
}

# Test 3: With special characters in name
resource "cloudflare_workers_for_platforms_dispatch_namespace" "special_chars" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-namespace-2024"
}

# Test 4: With hyphens and underscores
resource "cloudflare_workers_for_platforms_dispatch_namespace" "naming_variations" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-my_test-namespace"
}

# ==============================================================================
# Pattern 2-3: for_each with maps and sets
# ==============================================================================

# Test 5-9: for_each with map (5 instances)
locals {
  namespace_map = {
    production  = "${local.name_prefix}-prod-namespace"
    staging     = "${local.name_prefix}-staging-namespace"
    development = "${local.name_prefix}-dev-namespace"
    testing     = "${local.name_prefix}-test-namespace"
    demo        = "${local.name_prefix}-demo-namespace"
  }
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "for_each_map" {
  for_each   = local.namespace_map
  account_id = var.cloudflare_account_id
  name       = each.value
}

# Test 10-14: for_each with set (5 instances)
locals {
  namespace_set = toset([
    "${local.name_prefix}-api-namespace",
    "${local.name_prefix}-web-namespace",
    "${local.name_prefix}-mobile-namespace",
    "${local.name_prefix}-backend-namespace",
    "${local.name_prefix}-frontend-namespace",
  ])
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "for_each_set" {
  for_each   = local.namespace_set
  account_id = var.cloudflare_account_id
  name       = each.value
}

# ==============================================================================
# Pattern 4: count-based resources
# ==============================================================================

# Test 15-19: count with index (5 instances)
resource "cloudflare_workers_for_platforms_dispatch_namespace" "count_basic" {
  count      = 5
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-namespace-${count.index}"
}

# ==============================================================================
# Pattern 5: Conditional resource creation
# ==============================================================================

# Test 20: Conditional with count (1 instance when true)
variable "create_optional_namespace" {
  type    = bool
  default = true
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "conditional" {
  count      = var.create_optional_namespace ? 1 : 0
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-namespace"
}

# ==============================================================================
# Pattern 6: Cross-resource references
# ==============================================================================

# Test 21: Reference to another resource's attribute
resource "cloudflare_workers_for_platforms_dispatch_namespace" "primary" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-primary-namespace"
}

# Test 22: Uses primary's account_id (cross-reference)
resource "cloudflare_workers_for_platforms_dispatch_namespace" "secondary" {
  account_id = cloudflare_workers_for_platforms_dispatch_namespace.primary.account_id
  name       = "${local.name_prefix}-secondary-namespace"
}

# ==============================================================================
# Pattern 7: Lifecycle meta-arguments
# ==============================================================================

# Test 23: With create_before_destroy
resource "cloudflare_workers_for_platforms_dispatch_namespace" "lifecycle_cbd" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-cbd-namespace"

  lifecycle {
    create_before_destroy = true
  }
}

# Test 24: With ignore_changes
resource "cloudflare_workers_for_platforms_dispatch_namespace" "lifecycle_ignore" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-ignore-namespace"

  lifecycle {
    ignore_changes = [name]
  }
}

# Test 25: With prevent_destroy
resource "cloudflare_workers_for_platforms_dispatch_namespace" "lifecycle_prevent" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-prevent-namespace"

  lifecycle {
    prevent_destroy = false # Must be false for tests to clean up
  }
}

# ==============================================================================
# Pattern 8: Terraform functions
# ==============================================================================

# Test 26: Using join() function
locals {
  namespace_parts = ["${local.name_prefix}", "joined", "namespace"]
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "func_join" {
  account_id = var.cloudflare_account_id
  name       = join("-", local.namespace_parts)
}

# Test 27: Using format() function
resource "cloudflare_workers_for_platforms_dispatch_namespace" "func_format" {
  account_id = var.cloudflare_account_id
  name       = format("%s-formatted-%02d", local.name_prefix, 1)
}

# Test 28: Using lower() function
resource "cloudflare_workers_for_platforms_dispatch_namespace" "func_lower" {
  account_id = var.cloudflare_account_id
  name       = lower("${local.name_prefix}-LOWERCASE-namespace")
}

# Test 29: Using replace() function
resource "cloudflare_workers_for_platforms_dispatch_namespace" "func_replace" {
  account_id = var.cloudflare_account_id
  name       = replace("${local.name_prefix}_underscore_namespace", "_", "-")
}

# ==============================================================================
# Pattern 9: String interpolation and expressions
# ==============================================================================

# Test 30: Complex string interpolation
locals {
  environment = "production"
  region      = "us-east"
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "interpolation" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-${local.environment}-${local.region}-namespace"
}

# ==============================================================================
# Edge Cases and Special Scenarios
# ==============================================================================

# Test 31: Very long name (test length limits)
resource "cloudflare_workers_for_platforms_dispatch_namespace" "long_name" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-very-long-namespace-name-to-test-limits"
}

# Test 32: Name with numbers
resource "cloudflare_workers_for_platforms_dispatch_namespace" "with_numbers" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-namespace-12345"
}

# Test 33: Depends_on meta-argument
resource "cloudflare_workers_for_platforms_dispatch_namespace" "depends_base" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-depends-base"
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "depends_derived" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-depends-derived"

  depends_on = [cloudflare_workers_for_platforms_dispatch_namespace.depends_base]
}

# ==============================================================================
# Summary: 34 total resource instances
# ==============================================================================
# - Pattern 1 (Basic): 4 instances
# - Pattern 2 (for_each map): 5 instances
# - Pattern 3 (for_each set): 5 instances
# - Pattern 4 (count): 5 instances
# - Pattern 5 (conditional): 1 instance
# - Pattern 6 (cross-reference): 2 instances
# - Pattern 7 (lifecycle): 3 instances
# - Pattern 8 (functions): 4 instances
# - Pattern 9 (interpolation): 1 instance
# - Edge cases: 4 instances
# Total: 34 instances (exceeds 15-30 target)
