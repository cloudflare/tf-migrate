# Test migration of zero_trust_local_fallback_domain resources with dynamic "domains" blocks
# These are common patterns seen in real-world configurations (ticket #002)

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

# Pattern 1: dynamic "domains" block with toset([...]) - default profile
# The most common real-world pattern: avoids repeating individual domains blocks
resource "cloudflare_zero_trust_local_fallback_domain" "dynamic_default" {
  account_id = var.cloudflare_account_id

  dynamic "domains" {
    for_each = toset(["intranet", "internal", "corp", "local", "localhost"])
    content {
      suffix = domains.value
    }
  }
}

# Pattern 2: dynamic "domains" block with toset([...]) - custom profile
resource "cloudflare_zero_trust_local_fallback_domain" "dynamic_custom" {
  account_id = var.cloudflare_account_id
  policy_id  = "custom-policy-id"

  dynamic "domains" {
    for_each = toset(["corp", "internal"])
    content {
      suffix = domains.value
    }
  }
}

# Pattern 3: deprecated resource name with dynamic "domains" block
resource "cloudflare_fallback_domain" "dynamic_deprecated" {
  account_id = var.cloudflare_account_id

  dynamic "domains" {
    for_each = toset(["old-corp", "old-internal"])
    content {
      suffix = domains.value
    }
  }
}
