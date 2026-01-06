# Terraform v4 to v5 Migration Integration Test - Certificate Pack
# This file contains 20+ certificate pack instances using various Terraform patterns

# Variables without defaults (passed at runtime)
variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_domain" {
  type = string
}

# Locals and computed values
locals {
  name_prefix = "cftftest"
  zones = {
    primary   = var.cloudflare_zone_id
    secondary = var.cloudflare_zone_id
  }

  validation_methods = ["txt", "http"]

  certificate_configs = {
    lets-encrypt-90d = {
      authority     = "lets_encrypt"
      validity_days = 90
      branding      = false
    }
    google-90d = {
      authority     = "google"
      validity_days = 90
      branding      = true
    }
    ssl-com-30d = {
      authority     = "ssl_com"
      validity_days = 30
      branding      = false
    }
  }
}

# Pattern 1: Simple resource with all required fields
resource "cloudflare_certificate_pack" "basic" {
  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = [var.cloudflare_domain, "*.${var.cloudflare_domain}"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
}

# Pattern 2: Resource with wait_for_active_status (v4 only - will be removed)
resource "cloudflare_certificate_pack" "with_wait" {
  zone_id                = var.cloudflare_zone_id
  type                   = "advanced"
  hosts                  = ["test.${var.cloudflare_domain}"]
  validation_method      = "http"
  validity_days          = 90
  certificate_authority  = "google"
  wait_for_active_status = false
}

# Pattern 3: Resource with cloudflare_branding
resource "cloudflare_certificate_pack" "with_branding" {
  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = ["branded.${var.cloudflare_domain}", "*.branded.${var.cloudflare_domain}"]
  validation_method     = "txt"
  validity_days         = 30
  certificate_authority = "ssl_com"
  cloudflare_branding   = true
}

# Pattern 4: for_each with maps (3 resources)
resource "cloudflare_certificate_pack" "by_config" {
  for_each = local.certificate_configs

  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = ["${each.key}.${var.cloudflare_domain}"]
  validation_method     = "txt"
  validity_days         = each.value.validity_days
  certificate_authority = each.value.authority
  cloudflare_branding   = each.value.branding
  wait_for_active_status = false
}

# Pattern 5: for_each with sets (2 resources)
resource "cloudflare_certificate_pack" "by_validation" {
  for_each = toset(local.validation_methods)

  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  # HTTP validation doesn't support wildcards, TXT does
  hosts                 = each.value == "http" ? ["${each.value}.${var.cloudflare_domain}"] : ["${each.value}.${var.cloudflare_domain}", "*.${each.value}.${var.cloudflare_domain}"]
  validation_method     = each.value
  validity_days         = 90
  certificate_authority = "lets_encrypt"
}

# Pattern 6: count-based resources (4 resources)
resource "cloudflare_certificate_pack" "count_based" {
  count = 4

  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = ["count-${count.index}.${var.cloudflare_domain}"]
  validation_method     = element(["txt", "http", "txt", "txt"], count.index)
  validity_days         = element([90, 90, 30, 14], count.index)
  certificate_authority = element(["lets_encrypt", "google", "ssl_com", "google"], count.index)
}

# Pattern 7: Conditional resource creation (ternary)
resource "cloudflare_certificate_pack" "conditional" {
  count = length(var.cloudflare_zone_id) > 0 ? 1 : 0

  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = ["conditional.${var.cloudflare_domain}"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
  wait_for_active_status = false
}

# Pattern 8: Cross-resource references (depends on zone variable)
resource "cloudflare_certificate_pack" "with_reference" {
  zone_id               = local.zones.primary
  type                  = "advanced"
  hosts                 = ["ref.${var.cloudflare_domain}"]
  validation_method     = "http"
  validity_days         = 90
  certificate_authority = "google"
  cloudflare_branding   = false
}

# Pattern 9: Lifecycle meta-arguments
resource "cloudflare_certificate_pack" "with_lifecycle" {
  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = ["lifecycle.${var.cloudflare_domain}"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"

  lifecycle {
    create_before_destroy = true
    prevent_destroy       = false
  }
}

# Pattern 10: Terraform functions (join, length, etc.)
resource "cloudflare_certificate_pack" "with_functions" {
  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = [for i in range(2) : "${local.name_prefix}-${i}.${var.cloudflare_domain}"]
  validation_method     = length(local.validation_methods) > 0 ? local.validation_methods[0] : "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
}

# Pattern 11: Complex for_each with filtering (2 resources after filtering)
resource "cloudflare_certificate_pack" "filtered" {
  for_each = {
    for k, v in local.certificate_configs :
    k => v
    if v.validity_days >= 90
  }

  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = ["filtered-${each.key}.${var.cloudflare_domain}"]
  validation_method     = "txt"
  validity_days         = each.value.validity_days
  certificate_authority = each.value.authority
}

# Pattern 12: Dynamic expressions with coalescence
resource "cloudflare_certificate_pack" "with_coalesce" {
  zone_id               = var.cloudflare_zone_id
  type                  = "advanced"
  hosts                 = ["coalesce.${var.cloudflare_domain}"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
  cloudflare_branding   = coalesce(false, true)
}

# Total: 22 certificate pack resources
# - 3 simple resources (basic, with_wait, with_branding)
# - 3 for_each with maps
# - 3 for_each with sets
# - 4 count-based resources
# - 1 conditional
# - 1 with cross-resource reference
# - 1 with lifecycle
# - 1 with terraform functions
# - 2 filtered for_each
# - 1 with coalesce
# = 20+ total instances
