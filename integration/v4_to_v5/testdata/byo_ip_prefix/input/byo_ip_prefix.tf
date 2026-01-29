# Comprehensive Integration Tests for byo_ip_prefix Migration (v4 → v5)
# This file covers all v4 schema attributes and Terraform patterns
# Target: 15-30+ resource instances

# ============================================================================
# PATTERN 1-2: Variables & Locals
# ============================================================================

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
  # No default - must be provided
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
  # No default - must be provided
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

variable "prefix_base" {
  description = "Base prefix ID for testing"
  type        = string
  default     = "prefix-test"
}

variable "enable_optional" {
  description = "Whether to create optional resources"
  type        = bool
  default     = true
}

variable "advertisement_status" {
  description = "Advertisement status for prefixes"
  type        = string
  default     = "on"
}

locals {
  name_prefix = "tf-acc-test"
  environment = "integration-test"
  description_template = "BYO IP Prefix for ${local.environment}"

  prefix_map = {
    prod    = "prefix-prod-001"
    staging = "prefix-staging-001"
    dev     = "prefix-dev-001"
    test    = "prefix-test-001"
  }

  prefix_list = [
    "prefix-list-001",
    "prefix-list-002",
    "prefix-list-003"
  ]
}

# ============================================================================
# PATTERN 1: Basic Resources (3 instances)
# ============================================================================

# Instance 1: Minimal resource (only required fields)
resource "cloudflare_byo_ip_prefix" "minimal" {
  account_id = var.cloudflare_account_id
  prefix_id  = "prefix-minimal-001"
}

# Instance 2: Full resource with all optional fields
resource "cloudflare_byo_ip_prefix" "full" {
  account_id    = var.cloudflare_account_id
  prefix_id     = "prefix-full-001"
  description   = "Full BYO IP prefix with all fields"
  advertisement = "on"
}

# Instance 3: Resource with variables
resource "cloudflare_byo_ip_prefix" "with_variables" {
  account_id    = var.cloudflare_account_id
  prefix_id     = var.prefix_base
  description   = local.description_template
  advertisement = var.advertisement_status
}

# ============================================================================
# PATTERN 3: for_each with Maps (4 instances)
# ============================================================================

resource "cloudflare_byo_ip_prefix" "from_map" {
  for_each = local.prefix_map

  account_id    = var.cloudflare_account_id
  prefix_id     = each.value
  description   = "Prefix for ${each.key} environment"
  advertisement = each.key == "prod" ? "on" : "off"
}

# ============================================================================
# PATTERN 4: for_each with Sets (3 instances)
# ============================================================================

resource "cloudflare_byo_ip_prefix" "from_set" {
  for_each = toset(local.prefix_list)

  account_id  = var.cloudflare_account_id
  prefix_id   = each.value
  description = "Prefix from set: ${each.value}"
}

# ============================================================================
# PATTERN 5: count-based Resources (3 instances)
# ============================================================================

resource "cloudflare_byo_ip_prefix" "with_count" {
  count = 3

  account_id    = var.cloudflare_account_id
  prefix_id     = "prefix-count-${count.index}"
  description   = "Prefix number ${count.index + 1}"
  advertisement = count.index % 2 == 0 ? "on" : "off"
}

# ============================================================================
# PATTERN 6: Conditional Resource Creation (2 instances)
# ============================================================================

resource "cloudflare_byo_ip_prefix" "conditional" {
  count = var.enable_optional ? 1 : 0

  account_id    = var.cloudflare_account_id
  prefix_id     = "prefix-conditional-001"
  description   = "Conditional prefix"
  advertisement = "off"
}

resource "cloudflare_byo_ip_prefix" "conditional_ternary" {
  count = var.enable_optional ? 2 : 0

  account_id  = var.cloudflare_account_id
  prefix_id   = "prefix-conditional-${count.index + 1}"
  description = var.enable_optional ? "Enabled prefix ${count.index}" : null
}

# ============================================================================
# PATTERN 7: Cross-resource References (1 instance)
# ============================================================================

# Note: byo_ip_prefix is a standalone resource, so cross-references are limited
# But we can reference other instances
resource "cloudflare_byo_ip_prefix" "reference" {
  account_id  = cloudflare_byo_ip_prefix.minimal.account_id
  prefix_id   = "prefix-reference-001"
  description = "References minimal: ${cloudflare_byo_ip_prefix.minimal.prefix_id}"
}

# ============================================================================
# PATTERN 8: Lifecycle Meta-arguments (2 instances)
# ============================================================================

resource "cloudflare_byo_ip_prefix" "with_lifecycle" {
  account_id    = var.cloudflare_account_id
  prefix_id     = "prefix-lifecycle-001"
  description   = "Prefix with lifecycle rules"
  advertisement = "on"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
}

resource "cloudflare_byo_ip_prefix" "prevent_destroy" {
  account_id  = var.cloudflare_account_id
  prefix_id   = "prefix-prevent-001"
  description = "Protected prefix"

  lifecycle {
    prevent_destroy = true
  }
}

# ============================================================================
# PATTERN 9: Terraform Functions (3 instances)
# ============================================================================

resource "cloudflare_byo_ip_prefix" "with_join" {
  account_id  = var.cloudflare_account_id
  prefix_id   = "prefix-join-001"
  description = join(" - ", ["BYO IP", local.environment, "test"])
}

resource "cloudflare_byo_ip_prefix" "with_format" {
  account_id    = var.cloudflare_account_id
  prefix_id     = "prefix-format-001"
  description   = format("Prefix for %s environment", local.environment)
  advertisement = "on"
}

resource "cloudflare_byo_ip_prefix" "with_interpolation" {
  account_id  = var.cloudflare_account_id
  prefix_id   = "${local.name_prefix}-prefix-001"
  description = "Prefix managed by ${local.name_prefix} for ${local.environment}"
}

# ============================================================================
# EDGE CASES (2 instances)
# ============================================================================

# Edge case 1: Empty string description (vs null)
resource "cloudflare_byo_ip_prefix" "empty_description" {
  account_id  = var.cloudflare_account_id
  prefix_id   = "prefix-empty-desc-001"
  description = ""
}

# Edge case 2: Advertisement off
resource "cloudflare_byo_ip_prefix" "advertisement_off" {
  account_id    = var.cloudflare_account_id
  prefix_id     = "prefix-adv-off-001"
  description   = "Prefix with advertisement disabled"
  advertisement = "off"
}

# ============================================================================
# INSTANCE COUNT SUMMARY
# ============================================================================
# Basic: 3
# for_each maps: 4 (prod, staging, dev, test)
# for_each sets: 3
# count-based: 3
# conditional: 3 (1 + 2 when enabled)
# cross-reference: 1
# lifecycle: 2
# functions: 3
# edge cases: 2
# TOTAL: 24 instances (exceeds 15-30 target)
# ============================================================================
