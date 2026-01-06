# Test comprehensive migration of zero_trust_local_fallback_domain resources
# This includes default profile (no policy_id) but not custom profile (with policy_id) variants
# TODO Add custom_profile e2e tests once zero_trust_device_profiles is migrated and we can use those policy_ids

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