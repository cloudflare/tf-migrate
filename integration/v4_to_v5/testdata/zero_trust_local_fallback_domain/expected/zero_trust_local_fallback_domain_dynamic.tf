# Test migration of zero_trust_local_fallback_domain resources with dynamic "domains" blocks
# These are common patterns seen in real-world configurations (ticket #002)

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}




# Pattern 1: dynamic "domains" block with toset([...]) - default profile
# The most common real-world pattern: avoids repeating individual domains blocks
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "dynamic_default" {
  account_id = var.cloudflare_account_id

  domains = [for value in toset(["intranet", "internal", "corp", "local", "localhost"]) : {
    suffix = value
  }]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.dynamic_default
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.dynamic_default
}

# Pattern 2: dynamic "domains" block with toset([...]) - custom profile
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "dynamic_custom" {
  account_id = var.cloudflare_account_id
  policy_id  = "custom-policy-id"

  domains = [for value in toset(["corp", "internal"]) : {
    suffix = value
  }]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.dynamic_custom
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.dynamic_custom
}

# Pattern 3: deprecated resource name with dynamic "domains" block
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "dynamic_deprecated" {
  account_id = var.cloudflare_account_id

  domains = [for value in toset(["old-corp", "old-internal"]) : {
    suffix = value
  }]
}

moved {
  from = cloudflare_fallback_domain.dynamic_deprecated
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.dynamic_deprecated
}
