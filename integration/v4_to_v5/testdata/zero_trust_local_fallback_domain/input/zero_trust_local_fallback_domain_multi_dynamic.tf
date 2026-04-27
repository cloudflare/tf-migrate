# Test migration of zero_trust_local_fallback_domain with MULTIPLE dynamic "domains" blocks.
# Regression test for https://github.com/cloudflare/tf-migrate/issues/288
#
# Previously, only the last dynamic block survived (silent data loss).
# After the fix, all dynamic blocks are merged via concat().

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

# Pattern 1: Multiple dynamic "domains" blocks — custom profile (issue #288 exact scenario)
resource "cloudflare_fallback_domain" "cftftest_multi_dynamic_custom" {
  account_id = var.cloudflare_account_id
  policy_id  = "multi-dynamic-policy-id"

  dynamic "domains" {
    for_each = toset(["primary.corp", "primary.internal"])
    content {
      suffix = domains.value
    }
  }

  dynamic "domains" {
    for_each = toset(["static.corp"])
    content {
      suffix      = domains.value
      dns_server  = ["1.1.1.1"]
      description = "Static domains resolved via regional DNS"
    }
  }

  dynamic "domains" {
    for_each = toset(["dev.corp"])
    content {
      suffix      = domains.value
      dns_server  = ["1.1.1.1"]
      description = "Dev domains resolved via regional DNS"
    }
  }
}

# Pattern 2: Multiple dynamic "domains" blocks — default profile
resource "cloudflare_zero_trust_local_fallback_domain" "cftftest_multi_dynamic_default" {
  account_id = var.cloudflare_account_id

  dynamic "domains" {
    for_each = toset(["intranet", "internal"])
    content {
      suffix = domains.value
    }
  }

  dynamic "domains" {
    for_each = toset(["vpn.corp"])
    content {
      suffix      = domains.value
      description = "VPN domains"
    }
  }
}

# Pattern 3: Mixed static and dynamic "domains" blocks — default profile
resource "cloudflare_zero_trust_local_fallback_domain" "cftftest_mixed_static_dynamic" {
  account_id = var.cloudflare_account_id

  domains {
    suffix = "static-only.corp"
  }

  dynamic "domains" {
    for_each = toset(["dynamic.corp", "dynamic.internal"])
    content {
      suffix = domains.value
    }
  }

  domains {
    suffix      = "another-static.corp"
    description = "Another static domain"
  }
}
