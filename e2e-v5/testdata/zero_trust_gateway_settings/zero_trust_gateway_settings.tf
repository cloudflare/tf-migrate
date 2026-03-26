variable "cloudflare_account_id" {
  description = "Cloudflare account ID for E2E testing"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (not used by gateway settings, but required by test harness)"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (not used by gateway settings, but required by test harness)"
  type        = string
  default     = ""
}

# ============================================================================
# E2E Test: Gateway Settings Configuration (Basic Features Only)
# ============================================================================
# This is the ONLY test case that should be applied in E2E testing, as there
# can only be ONE gateway settings resource per account.
#
# This test case uses only basic features that don't require paid entitlements:
# - Basic boolean settings (activity_log, tls_decrypt, protocol_detection)
# - Browser isolation with defaults (false - no entitlement needed)
# - Basic blocks: block_page, body_scanning, fips
# - Deprecated blocks that create separate resources (logging, proxy)
#
# Features excluded due to entitlement requirements:
# - antivirus (requires entitlement)
# - extended_email_matching (may require entitlement)
# - certificate/custom_certificate (requires real cert IDs)
#
# Note: tls_decrypt_enabled is set to false because the Cloudflare API now
# requires a certificate to be configured when TLS decryption is enabled
# (API error 2211). fips.tls is also set to false since FIPS TLS mode
# depends on TLS decryption being active.
# ============================================================================


resource "cloudflare_zero_trust_gateway_settings" "e2e_comprehensive" {
  account_id = var.cloudflare_account_id

  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = false
    }
    protocol_detection = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled          = true
      name             = "E2E Test Block Page"
      footer_text      = "Contact IT Security"
      header_text      = "Access Denied"
      logo_path        = "https://example.com/logo.png"
      background_color = "#1a1a1a"
    }
    body_scanning = {
      inspection_mode = "deep"
    }
    fips = {
      tls = false
    }
  }
}

resource "cloudflare_zero_trust_gateway_logging" "e2e_comprehensive_logging" {
  account_id = var.cloudflare_account_id
  settings_by_rule_type = {
    dns = {
      log_all    = true
      log_blocks = false
    }
    http = {
      log_all    = true
      log_blocks = true
    }
    l4 = {
      log_all    = false
      log_blocks = true
    }
  }
  redact_pii = true
}

resource "cloudflare_zero_trust_device_settings" "e2e_comprehensive_device_settings" {
  account_id                            = var.cloudflare_account_id
  gateway_proxy_enabled                 = true
  gateway_udp_proxy_enabled             = true
  root_certificate_installation_enabled = true
  use_zt_virtual_ip                     = false
  disable_for_time                      = 600
}


variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
