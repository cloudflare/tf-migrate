# Test comprehensive migration of zero_trust_local_fallback_domain resources
# This includes both default profile (no policy_id) and custom profile (with policy_id) variants

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

# Default profile - no policy_id
resource "cloudflare_zero_trust_local_fallback_domain" "default_single" {
  account_id = var.cloudflare_account_id

  domains {
    suffix = var.cloudflare_domain
  }
}

# Default profile - multiple domains with all fields
resource "cloudflare_zero_trust_local_fallback_domain" "default_multi" {
  account_id = var.cloudflare_account_id

  domains {
    suffix      = "corp.${var.cloudflare_domain}"
    description = "Corporate network"
    dns_server  = ["10.0.0.1", "10.0.0.2"]
  }
  domains {
    suffix      = "internal.${var.cloudflare_domain}"
    description = "Internal services"
    dns_server  = ["10.1.0.1"]
  }
  domains {
    suffix = "local.${var.cloudflare_domain}"
  }
}

# Custom profile - with policy_id
resource "cloudflare_zero_trust_local_fallback_domain" "custom_single" {
  account_id = var.cloudflare_account_id
  policy_id  = "test-policy-id"

  domains {
    suffix = "custom.${var.cloudflare_domain}"
  }
}

# Custom profile - multiple domains
resource "cloudflare_zero_trust_local_fallback_domain" "custom_multi" {
  account_id = var.cloudflare_account_id
  policy_id  = "another-policy-id"

  domains {
    suffix      = "dev.${var.cloudflare_domain}"
    description = "Development environment"
    dns_server  = ["192.168.1.1"]
  }
  domains {
    suffix = "staging.${var.cloudflare_domain}"
  }
}

# Deprecated resource name - default profile
resource "cloudflare_fallback_domain" "deprecated_default" {
  account_id = var.cloudflare_account_id

  domains {
    suffix      = "deprecated.${var.cloudflare_domain}"
    description = "Using deprecated resource name"
  }
}

# Deprecated resource name - custom profile
resource "cloudflare_fallback_domain" "deprecated_custom" {
  account_id = var.cloudflare_account_id
  policy_id  = "deprecated-policy-id"

  domains {
    suffix = "old.${var.cloudflare_domain}"
  }
}

# ============================================================================
# Pattern Group 9: Cross-File References (Resource Rename Test)
# ============================================================================

# Pattern 9 tests that cross-file references are updated when resource names change.
# The migrator renames:
#   cloudflare_zero_trust_local_fallback_domain -> cloudflare_zero_trust_device_default_profile_local_domain_fallback (default)
#   cloudflare_zero_trust_local_fallback_domain -> cloudflare_zero_trust_device_custom_profile_local_domain_fallback (custom)
#   cloudflare_fallback_domain -> cloudflare_zero_trust_device_default_profile_local_domain_fallback (default)
#   cloudflare_fallback_domain -> cloudflare_zero_trust_device_custom_profile_local_domain_fallback (custom)
# We create dependent resources (device profiles) that reference these fallback domains via depends_on.
# After migration, the references must be updated to use the new resource names.

# Using old v4 name (cloudflare_fallback_domain) - default profile
resource "cloudflare_fallback_domain" "ref_source_old_default" {
  account_id = var.cloudflare_account_id

  domains {
    suffix      = "ref-old-default.${var.cloudflare_domain}"
    description = "Referenced by device profile (old name, default)"
  }
}

# Using new v4 name (cloudflare_zero_trust_local_fallback_domain) - default profile
resource "cloudflare_zero_trust_local_fallback_domain" "ref_source_new_default" {
  account_id = var.cloudflare_account_id

  domains {
    suffix      = "ref-new-default.${var.cloudflare_domain}"
    description = "Referenced by device profile (new name, default)"
  }
}

# Dependent resources that reference the above fallback domains
# Using realistic resources that would depend on fallback domains

# Device profile depending on old-name fallback domain
resource "cloudflare_zero_trust_device_profiles" "depends_on_old_fallback" {
  account_id  = var.cloudflare_account_id
  name        = "Default Profile - Old Fallback Domain"
  description = "Profile depending on old-name fallback domain"
  default     = true

  allow_mode_switch = true
  auto_connect      = 30

  depends_on = [cloudflare_fallback_domain.ref_source_old_default]
}

# Device profile depending on new-name fallback domain
resource "cloudflare_zero_trust_device_profiles" "depends_on_new_fallback" {
  account_id  = var.cloudflare_account_id
  name        = "Default Profile - New Fallback Domain"
  description = "Profile depending on new-name fallback domain"
  default     = true

  allow_mode_switch = false
  auto_connect      = 15

  depends_on = [cloudflare_zero_trust_local_fallback_domain.ref_source_new_default]
}
