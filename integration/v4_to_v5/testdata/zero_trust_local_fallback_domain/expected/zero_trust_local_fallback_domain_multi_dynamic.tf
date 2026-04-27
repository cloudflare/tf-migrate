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
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "cftftest_multi_dynamic_custom" {
  account_id = var.cloudflare_account_id
  policy_id  = "multi-dynamic-policy-id"



  domains = concat(
    [for value in toset(["primary.corp", "primary.internal"]) : {
      suffix = value
    }],
    [for value in toset(["static.corp"]) : {
      suffix      = value
      dns_server  = ["1.1.1.1"]
      description = "Static domains resolved via regional DNS"
    }],
    [for value in toset(["dev.corp"]) : {
      suffix      = value
      dns_server  = ["1.1.1.1"]
      description = "Dev domains resolved via regional DNS"
    }],
  )
}

moved {
  from = cloudflare_fallback_domain.cftftest_multi_dynamic_custom
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.cftftest_multi_dynamic_custom
}

# Pattern 2: Multiple dynamic "domains" blocks — default profile
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "cftftest_multi_dynamic_default" {
  account_id = var.cloudflare_account_id


  domains = concat(
    [for value in toset(["intranet", "internal"]) : {
      suffix = value
    }],
    [for value in toset(["vpn.corp"]) : {
      suffix      = value
      description = "VPN domains"
    }],
  )
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.cftftest_multi_dynamic_default
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.cftftest_multi_dynamic_default
}

# Pattern 3: Mixed static and dynamic "domains" blocks — default profile
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "cftftest_mixed_static_dynamic" {
  account_id = var.cloudflare_account_id



  domains = concat(
    [for value in toset(["dynamic.corp", "dynamic.internal"]) : {
      suffix = value
    }],
    [{
      suffix = "static-only.corp"
      }, {
      suffix      = "another-static.corp"
      description = "Another static domain"
    }],
  )
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.cftftest_mixed_static_dynamic
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.cftftest_mixed_static_dynamic
}
