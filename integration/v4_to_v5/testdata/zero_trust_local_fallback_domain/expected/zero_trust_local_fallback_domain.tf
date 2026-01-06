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
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "default_single" {
  account_id = var.cloudflare_account_id

  domains = [
    {
      suffix = var.cloudflare_domain
    }
  ]
}

# Default profile - multiple domains with all fields
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "default_multi" {
  account_id = var.cloudflare_account_id

  domains = [
    {
      suffix      = "corp.${var.cloudflare_domain}"
      description = "Corporate network"
      dns_server  = ["10.0.0.1", "10.0.0.2"]
    },
    {
      suffix      = "internal.${var.cloudflare_domain}"
      description = "Internal services"
      dns_server  = ["10.1.0.1"]
    },
    {
      suffix = "local.${var.cloudflare_domain}"
    }
  ]
}

# Custom profile - with policy_id
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "custom_single" {
  account_id = var.cloudflare_account_id
  policy_id  = "test-policy-id"

  domains = [
    {
      suffix = "custom.${var.cloudflare_domain}"
    }
  ]
}

# Custom profile - multiple domains
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "custom_multi" {
  account_id = var.cloudflare_account_id
  policy_id  = "another-policy-id"

  domains = [
    {
      suffix      = "dev.${var.cloudflare_domain}"
      description = "Development environment"
      dns_server  = ["192.168.1.1"]
    },
    {
      suffix = "staging.${var.cloudflare_domain}"
    }
  ]
}

# Deprecated resource name - default profile
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "deprecated_default" {
  account_id = var.cloudflare_account_id

  domains = [
    {
      suffix      = "deprecated.${var.cloudflare_domain}"
      description = "Using deprecated resource name"
    }
  ]
}

# Deprecated resource name - custom profile
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "deprecated_custom" {
  account_id = var.cloudflare_account_id
  policy_id  = "deprecated-policy-id"

  domains = [
    {
      suffix = "old.${var.cloudflare_domain}"
    }
  ]
}
