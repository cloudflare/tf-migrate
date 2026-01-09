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
# ============================================================================

resource "cloudflare_teams_account" "e2e_comprehensive" {
  account_id = var.cloudflare_account_id

  # Flat boolean fields -> nested structures in v5
  activity_log_enabled       = true
  tls_decrypt_enabled        = true
  protocol_detection_enabled = true

  # Browser isolation fields set to false (no entitlement needed for false values)
  # These still test the migration logic for combining into browser_isolation block
  url_browser_isolation_enabled          = false
  non_identity_browser_isolation_enabled = false

  # Basic MaxItems:1 blocks -> nested under settings in v5
  block_page {
    enabled          = true
    name             = "E2E Test Block Page"
    footer_text      = "Contact IT Security"
    header_text      = "Access Denied"
    logo_path        = "https://example.com/logo.png"
    background_color = "#1a1a1a"
  }

  body_scanning {
    inspection_mode = "deep"
  }

  fips {
    tls = true
  }

  # Deprecated blocks - these will create separate resources in v5
  # logging -> cloudflare_zero_trust_gateway_logging
  logging {
    settings_by_rule_type {
      dns {
        log_all    = true
        log_blocks = false
      }
      http {
        log_all    = true
        log_blocks = true
      }
      l4 {
        log_all    = false
        log_blocks = true
      }
    }
    redact_pii = true
  }

  # proxy -> cloudflare_zero_trust_device_settings
  proxy {
    tcp              = true
    udp              = true
    root_ca          = true
    virtual_ip       = false
    disable_for_time = 600
  }
}
