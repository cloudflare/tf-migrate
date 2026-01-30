variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Resource-specific variables with defaults
variable "enable_browser_isolation" {
  type    = bool
  default = true
}

variable "enable_tls_decrypt" {
  type    = bool
  default = true
}

variable "settings_prefix" {
  type    = string
  default = "test"
}

variable "account_configs" {
  type = map(object({
    account_id   = string
    tls_decrypt  = bool
    activity_log = bool
  }))
  default = {
    account_a = {
      account_id   = "aaaaaaaa89293a057740de681ac9aaa1"
      tls_decrypt  = true
      activity_log = true
    }
    account_b = {
      account_id   = "bbbbbbbb89293a057740de681ac9bbb2"
      tls_decrypt  = false
      activity_log = false
    }
    account_c = {
      account_id   = "cccccccc89293a057740de681ac9ccc3"
      tls_decrypt  = true
      activity_log = false
    }
  }
}

# Locals with multiple values
locals {
  name_prefix       = "cftftest"
  settings_name     = "${local.name_prefix}-${var.settings_prefix}"
  block_page_name   = "Custom Block Page - ${var.settings_prefix}"
  certificate_id    = "cert-${var.settings_prefix}-123"
  support_url       = "https://support.example.com/${var.settings_prefix}"
  common_account_id = var.cloudflare_account_id
}

# ============================================================================
# Pattern Group 1: Basic Resources (Edge Cases & Field Combinations)
# ============================================================================

# 1. Minimal configuration (only account_id)
resource "cloudflare_zero_trust_gateway_settings" "minimal" {
  account_id = var.cloudflare_account_id
  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}

# 2. Maximal configuration with ALL possible fields
resource "cloudflare_zero_trust_gateway_settings" "maximal" {
  account_id = var.cloudflare_account_id






  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = true
    }
    protocol_detection = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = true
      non_identity_enabled          = true
    }
    block_page = {
      enabled          = true
      name             = local.block_page_name
      footer_text      = "Contact IT: support@${var.settings_prefix}.com"
      header_text      = "Access Blocked - ${var.settings_prefix}"
      logo_path        = "https://example.com/${var.settings_prefix}/logo.png"
      background_color = "#FF0000"
      mailto_address   = "security@${var.settings_prefix}.com"
      mailto_subject   = "Access Request - ${var.settings_prefix}"
    }
    body_scanning = {
      inspection_mode = "deep"
    }
    fips = {
      tls = true
    }
    antivirus = {
      enabled_download_phase = true
      enabled_upload_phase   = true
      fail_closed            = true
      notification_settings = {
        enabled     = true
        support_url = local.support_url
        msg         = "File scanning in progress - ${var.settings_prefix}"
      }
    }
    extended_email_matching = {
      enabled = true
    }
    certificate = {
      id = local.certificate_id
    }
  }
}

# 3. Only flat boolean fields (no MaxItems:1 blocks)
resource "cloudflare_zero_trust_gateway_settings" "only_booleans" {
  account_id = var.cloudflare_account_id
  settings = {
    activity_log = {
      enabled = var.enable_tls_decrypt
    }
    tls_decrypt = {
      enabled = var.enable_tls_decrypt
    }
    protocol_detection = {
      enabled = false
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}

# 4. Only browser isolation fields with variable
resource "cloudflare_zero_trust_gateway_settings" "only_browser_isolation" {
  account_id = var.cloudflare_account_id
  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = var.enable_browser_isolation
      non_identity_enabled          = var.enable_browser_isolation
    }
  }
}

# 5. Only block_page (single MaxItems:1 block)
resource "cloudflare_zero_trust_gateway_settings" "only_block_page" {
  account_id = var.cloudflare_account_id

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = "Simple Block Page"
      footer_text = "Contact IT"
    }
  }
}

# 6. Only antivirus with notification_settings (nested MaxItems:1)
resource "cloudflare_zero_trust_gateway_settings" "only_antivirus" {
  account_id = var.cloudflare_account_id

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    antivirus = {
      enabled_download_phase = true
      enabled_upload_phase   = false
      fail_closed            = false
      notification_settings = {
        enabled     = true
        support_url = "https://support.example.com"
        msg         = "Scanning for threats"
      }
    }
  }
}

# 7. Multiple MaxItems:1 blocks (block_page + fips + body_scanning)
resource "cloudflare_zero_trust_gateway_settings" "multiple_blocks" {
  account_id = var.cloudflare_account_id



  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = "Multi Block Test"
      footer_text = "Multiple blocks configured"
    }
    body_scanning = {
      inspection_mode = "shallow"
    }
    fips = {
      tls = true
    }
  }
}




# 11. Using v5 name in v4 (cloudflare_zero_trust_gateway_settings alias)
resource "cloudflare_zero_trust_gateway_settings" "v5_name_in_v4" {
  account_id = var.cloudflare_account_id


  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = false
    }
    browser_isolation = {
      url_browser_isolation_enabled = true
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = "V5 Name Test"
      footer_text = "Using v5 resource name"
    }
    fips = {
      tls = true
    }
  }
}

# ============================================================================
# Pattern Group 2: for_each with Maps
# ============================================================================

# 12-14. Resources created with for_each over account configs
resource "cloudflare_zero_trust_gateway_settings" "account_configs" {
  for_each = var.account_configs

  account_id = each.value.account_id

  settings = {
    activity_log = {
      enabled = each.value.activity_log
    }
    tls_decrypt = {
      enabled = each.value.tls_decrypt
    }
    protocol_detection = {
      enabled = false
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = "Block Page for ${each.key}"
      footer_text = "Account: ${each.key}"
    }
  }
}

# ============================================================================
# Pattern Group 3: for_each with Sets (Environment Configs)
# ============================================================================

# 15-17. Resources created with for_each over environments
resource "cloudflare_zero_trust_gateway_settings" "environment_configs" {
  for_each = toset(["dev", "staging", "prod"])

  account_id = "${each.value}${var.cloudflare_account_id}"

  settings = {
    activity_log = {
      enabled = each.value == "prod" ? true : false
    }
    tls_decrypt = {
      enabled = true
    }
    protocol_detection = {
      enabled = each.value == "prod" ? true : false
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    body_scanning = {
      inspection_mode = each.value == "prod" ? "deep" : "shallow"
    }
  }
}

# ============================================================================
# Pattern Group 4: count-based Resources
# ============================================================================

# 18-20. Resources created with count (tiered accounts)
resource "cloudflare_zero_trust_gateway_settings" "tiered_accounts" {
  count = 3

  account_id = "tier${count.index}${var.cloudflare_account_id}"


  settings = {
    activity_log = {
      enabled = count.index > 0 ? true : false
    }
    tls_decrypt = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = "Tier ${count.index} Block Page"
      footer_text = "Tier ${count.index} configuration"
    }
    fips = {
      tls = count.index == 2 ? true : false
    }
  }
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

# 21. Conditionally created (with browser isolation)
resource "cloudflare_zero_trust_gateway_settings" "conditional_enabled" {
  count = var.enable_browser_isolation ? 1 : 0

  account_id = "conditional1${var.cloudflare_account_id}"

  settings = {
    activity_log = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = true
      non_identity_enabled          = true
    }
    block_page = {
      enabled     = true
      name        = "Conditional - Enabled"
      footer_text = "Browser isolation enabled"
    }
  }
}

# 22. Conditionally not created (no browser isolation)
resource "cloudflare_zero_trust_gateway_settings" "conditional_disabled" {
  count = var.enable_browser_isolation ? 0 : 1

  account_id = "conditional2${var.cloudflare_account_id}"
  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}

# ============================================================================
# Pattern Group 6: Terraform Functions
# ============================================================================

# 23. Using join() function
resource "cloudflare_zero_trust_gateway_settings" "with_join" {
  account_id = join("-", ["join", var.cloudflare_account_id])

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = join(" - ", [local.settings_name, "Joined", "Block"])
      footer_text = "Created with join"
    }
  }
}

# 24. Using string interpolation and expressions
resource "cloudflare_zero_trust_gateway_settings" "with_interpolation" {
  account_id = var.cloudflare_account_id

  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = var.enable_tls_decrypt
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = "Block for ${var.settings_prefix} account"
      footer_text = "Account ID: ${var.cloudflare_account_id}"
      header_text = "Settings: ${local.settings_name}"
    }
  }
}

# 25. Using ternary expressions
resource "cloudflare_zero_trust_gateway_settings" "with_ternary" {
  account_id = var.cloudflare_account_id

  settings = {
    activity_log = {
      enabled = var.enable_tls_decrypt ? true : false
    }
    tls_decrypt = {
      enabled = var.enable_browser_isolation ? true : false
    }
    protocol_detection = {
      enabled = var.enable_tls_decrypt && var.enable_browser_isolation ? true : false
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    antivirus = {
      enabled_download_phase = var.enable_tls_decrypt ? true : false
      enabled_upload_phase   = var.enable_browser_isolation ? true : false
      fail_closed            = false
      notification_settings = {
        enabled     = true
        support_url = "https://support.example.com"
        msg         = var.enable_tls_decrypt ? "Full scanning enabled" : "Basic scanning"
      }
    }
  }
}

# ============================================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# ============================================================================

# 26. With lifecycle - create_before_destroy
resource "cloudflare_zero_trust_gateway_settings" "with_lifecycle_create" {
  account_id = var.cloudflare_account_id


  lifecycle {
    create_before_destroy = true
  }
  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = "Protected Block Page"
      footer_text = "Lifecycle protected"
    }
  }
}

# 27. With lifecycle - ignore_changes
resource "cloudflare_zero_trust_gateway_settings" "with_lifecycle_ignore" {
  account_id = var.cloudflare_account_id


  lifecycle {
    ignore_changes = [protocol_detection_enabled]
  }
  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = true
    }
    protocol_detection = {
      enabled = false
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    fips = {
      tls = true
    }
  }
}

# 28. With lifecycle - prevent_destroy (set to false for testing)
resource "cloudflare_zero_trust_gateway_settings" "with_lifecycle_prevent" {
  account_id = var.cloudflare_account_id


  lifecycle {
    prevent_destroy = false # Set to false for testing
  }
  settings = {
    activity_log = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled     = true
      name        = "Critical Block Page"
      footer_text = "Do not destroy"
    }
  }
}

# ============================================================================
# Pattern Group 8: All MaxItems:1 Blocks in Different Combinations
# ============================================================================

# 29. Extended email matching + certificate
resource "cloudflare_zero_trust_gateway_settings" "email_and_cert" {
  account_id = var.cloudflare_account_id


  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    extended_email_matching = {
      enabled = true
    }
    certificate = {
      id = "cert-combo-123"
    }
  }
}

# 30. All scanning features (body_scanning + antivirus)
resource "cloudflare_zero_trust_gateway_settings" "all_scanning" {
  account_id = var.cloudflare_account_id



  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    body_scanning = {
      inspection_mode = "deep"
    }
    fips = {
      tls = true
    }
    antivirus = {
      enabled_download_phase = true
      enabled_upload_phase   = true
      fail_closed            = true
      notification_settings = {
        enabled     = true
        support_url = "https://support.example.com/scanning"
        msg         = "Comprehensive scanning"
      }
    }
  }
}

# Total resources:
# - Basic: 11 resources (1-11)
# - for_each map: 3 resources (12-14)
# - for_each set: 3 resources (15-17)
# - count: 3 resources (18-20)
# - conditional: 1 resource (21, 22 not created due to default var)
# - functions: 3 resources (23-25)
# - lifecycle: 3 resources (26-28)
# - combinations: 2 resources (29-30)
# Total: 29 resource instances created

resource "cloudflare_zero_trust_gateway_settings" "with_logging" {
  account_id = var.cloudflare_account_id

  settings = {
    activity_log = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}

resource "cloudflare_zero_trust_gateway_logging" "with_logging_logging" {
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

resource "cloudflare_zero_trust_gateway_settings" "with_proxy" {
  account_id = var.cloudflare_account_id

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}

resource "cloudflare_zero_trust_device_settings" "with_proxy_device_settings" {
  account_id                            = var.cloudflare_account_id
  gateway_proxy_enabled                 = true
  gateway_udp_proxy_enabled             = true
  root_certificate_installation_enabled = true
  use_zt_virtual_ip                     = false
  disable_for_time                      = 300
}

resource "cloudflare_zero_trust_gateway_settings" "with_both_deprecated" {
  account_id = var.cloudflare_account_id

  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = false
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled = true
      name    = "Deprecated Test"
    }
  }
}

resource "cloudflare_zero_trust_gateway_logging" "with_both_deprecated_logging" {
  account_id = var.cloudflare_account_id
  settings_by_rule_type = {
    dns = {
      log_all    = true
      log_blocks = false
    }
    http = {
      log_all    = true
      log_blocks = false
    }
    l4 = {
      log_all    = true
      log_blocks = false
    }
  }
  redact_pii = false
}

resource "cloudflare_zero_trust_device_settings" "with_both_deprecated_device_settings" {
  account_id                            = var.cloudflare_account_id
  gateway_proxy_enabled                 = false
  gateway_udp_proxy_enabled             = false
  root_certificate_installation_enabled = true
  use_zt_virtual_ip                     = true
  disable_for_time                      = 600
}
