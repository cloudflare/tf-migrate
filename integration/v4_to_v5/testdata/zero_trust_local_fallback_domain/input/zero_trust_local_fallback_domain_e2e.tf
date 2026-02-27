# Test comprehensive migration of zero_trust_local_fallback_domain resources
# This includes both default profile (no policy_id) and custom profile (with policy_id) variants
# Custom profile migration now works because zero_trust_device_profiles with match/precedence
# migrates to zero_trust_device_custom_profile

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

# Custom device profile for e2e testing
resource "cloudflare_zero_trust_device_profiles" "custom_e2e" {
  account_id  = var.cloudflare_account_id
  name        = "E2E Custom Profile"
  description = "Custom profile for e2e testing"
  match       = "identity.email == \"e2e@example.com\""
  precedence  = 876
}

# Custom profile - fallback domain with policy_id
resource "cloudflare_zero_trust_local_fallback_domain" "custom_e2e" {
  account_id = var.cloudflare_account_id
  policy_id  = cloudflare_zero_trust_device_profiles.custom_e2e.id

  domains {
    suffix      = "custom-e2e.${var.cloudflare_domain}"
    description = "Custom profile e2e fallback"
    dns_server  = ["192.168.1.1"]
  }
}