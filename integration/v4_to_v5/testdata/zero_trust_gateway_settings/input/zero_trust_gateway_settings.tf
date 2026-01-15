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
resource "cloudflare_teams_account" "minimal" {
  account_id = var.cloudflare_account_id
}

# 2. Maximal configuration with ALL possible fields
resource "cloudflare_teams_account" "maximal" {
  account_id                             = var.cloudflare_account_id
  activity_log_enabled                   = true
  tls_decrypt_enabled                    = true
  protocol_detection_enabled             = true
  url_browser_isolation_enabled          = true
  non_identity_browser_isolation_enabled = true

  block_page {
    enabled          = true
    name             = local.block_page_name
    footer_text      = "Contact IT: support@${var.settings_prefix}.com"
    header_text      = "Access Blocked - ${var.settings_prefix}"
    logo_path        = "https://example.com/${var.settings_prefix}/logo.png"
    background_color = "#FF0000"
    mailto_address   = "security@${var.settings_prefix}.com"
    mailto_subject   = "Access Request - ${var.settings_prefix}"
  }

  body_scanning {
    inspection_mode = "deep"
  }

  fips {
    tls = true
  }

  antivirus {
    enabled_download_phase = true
    enabled_upload_phase   = true
    fail_closed            = true

    notification_settings {
      enabled     = true
      message     = "File scanning in progress - ${var.settings_prefix}"
      support_url = local.support_url
    }
  }

  extended_email_matching {
    enabled = true
  }

  certificate {
    id = local.certificate_id
  }
}

# 3. Only flat boolean fields (no MaxItems:1 blocks)
resource "cloudflare_teams_account" "only_booleans" {
  account_id                 = var.cloudflare_account_id
  activity_log_enabled       = var.enable_tls_decrypt
  tls_decrypt_enabled        = var.enable_tls_decrypt
  protocol_detection_enabled = false
}

# 4. Only browser isolation fields with variable
resource "cloudflare_teams_account" "only_browser_isolation" {
  account_id                             = var.cloudflare_account_id
  url_browser_isolation_enabled          = var.enable_browser_isolation
  non_identity_browser_isolation_enabled = var.enable_browser_isolation
}

# 5. Only block_page (single MaxItems:1 block)
resource "cloudflare_teams_account" "only_block_page" {
  account_id = var.cloudflare_account_id

  block_page {
    enabled     = true
    name        = "Simple Block Page"
    footer_text = "Contact IT"
  }
}

# 6. Only antivirus with notification_settings (nested MaxItems:1)
resource "cloudflare_teams_account" "only_antivirus" {
  account_id = var.cloudflare_account_id

  antivirus {
    enabled_download_phase = true
    enabled_upload_phase   = false
    fail_closed            = false

    notification_settings {
      enabled     = true
      message     = "Scanning for threats"
      support_url = "https://support.example.com"
    }
  }
}

# 7. Multiple MaxItems:1 blocks (block_page + fips + body_scanning)
resource "cloudflare_teams_account" "multiple_blocks" {
  account_id = var.cloudflare_account_id

  block_page {
    enabled     = true
    name        = "Multi Block Test"
    footer_text = "Multiple blocks configured"
  }

  fips {
    tls = true
  }

  body_scanning {
    inspection_mode = "shallow"
  }
}

# 8. With deprecated logging block (will create new resource)
resource "cloudflare_teams_account" "with_logging" {
  account_id           = var.cloudflare_account_id
  activity_log_enabled = true

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
}

# 9. With deprecated proxy block (will create new resource)
resource "cloudflare_teams_account" "with_proxy" {
  account_id = var.cloudflare_account_id

  proxy {
    tcp              = true
    udp              = true
    root_ca          = true
    virtual_ip       = false
    disable_for_time = 300
  }
}

# 10. With both deprecated blocks (logging + proxy)
resource "cloudflare_teams_account" "with_both_deprecated" {
  account_id           = var.cloudflare_account_id
  activity_log_enabled = true
  tls_decrypt_enabled  = false

  block_page {
    enabled = true
    name    = "Deprecated Test"
  }

  logging {
    settings_by_rule_type {
      dns {
        log_all    = true
        log_blocks = false
      }
      http {
        log_all    = true
        log_blocks = false
      }
      l4 {
        log_all    = true
        log_blocks = false
      }
    }
    redact_pii = false
  }

  proxy {
    tcp              = false
    udp              = false
    root_ca          = true
    virtual_ip       = true
    disable_for_time = 600
  }
}

# 11. Using v5 name in v4 (cloudflare_zero_trust_gateway_settings alias)
resource "cloudflare_zero_trust_gateway_settings" "v5_name_in_v4" {
  account_id                             = var.cloudflare_account_id
  activity_log_enabled                   = true
  tls_decrypt_enabled                    = false
  url_browser_isolation_enabled          = true
  non_identity_browser_isolation_enabled = false

  block_page {
    enabled     = true
    name        = "V5 Name Test"
    footer_text = "Using v5 resource name"
  }

  fips {
    tls = true
  }
}

# ============================================================================
# Pattern Group 2: for_each with Maps
# ============================================================================

# 12-14. Resources created with for_each over account configs
resource "cloudflare_teams_account" "account_configs" {
  for_each = var.account_configs

  account_id                 = each.value.account_id
  activity_log_enabled       = each.value.activity_log
  tls_decrypt_enabled        = each.value.tls_decrypt
  protocol_detection_enabled = false

  block_page {
    enabled     = true
    name        = "Block Page for ${each.key}"
    footer_text = "Account: ${each.key}"
  }
}

# ============================================================================
# Pattern Group 3: for_each with Sets (Environment Configs)
# ============================================================================

# 15-17. Resources created with for_each over environments
resource "cloudflare_teams_account" "environment_configs" {
  for_each = toset(["dev", "staging", "prod"])

  account_id                 = "${each.value}${var.cloudflare_account_id}"
  activity_log_enabled       = each.value == "prod" ? true : false
  tls_decrypt_enabled        = true
  protocol_detection_enabled = each.value == "prod" ? true : false

  body_scanning {
    inspection_mode = each.value == "prod" ? "deep" : "shallow"
  }
}

# ============================================================================
# Pattern Group 4: count-based Resources
# ============================================================================

# 18-20. Resources created with count (tiered accounts)
resource "cloudflare_teams_account" "tiered_accounts" {
  count = 3

  account_id           = "tier${count.index}${var.cloudflare_account_id}"
  activity_log_enabled = count.index > 0 ? true : false
  tls_decrypt_enabled  = true

  block_page {
    enabled     = true
    name        = "Tier ${count.index} Block Page"
    footer_text = "Tier ${count.index} configuration"
  }

  fips {
    tls = count.index == 2 ? true : false
  }
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

# 21. Conditionally created (with browser isolation)
resource "cloudflare_teams_account" "conditional_enabled" {
  count = var.enable_browser_isolation ? 1 : 0

  account_id                             = "conditional1${var.cloudflare_account_id}"
  url_browser_isolation_enabled          = true
  non_identity_browser_isolation_enabled = true
  activity_log_enabled                   = true

  block_page {
    enabled     = true
    name        = "Conditional - Enabled"
    footer_text = "Browser isolation enabled"
  }
}

# 22. Conditionally not created (no browser isolation)
resource "cloudflare_teams_account" "conditional_disabled" {
  count = var.enable_browser_isolation ? 0 : 1

  account_id           = "conditional2${var.cloudflare_account_id}"
  activity_log_enabled = true
  tls_decrypt_enabled  = true
}

# ============================================================================
# Pattern Group 6: Terraform Functions
# ============================================================================

# 23. Using join() function
resource "cloudflare_teams_account" "with_join" {
  account_id = join("-", ["join", var.cloudflare_account_id])

  block_page {
    enabled     = true
    name        = join(" - ", [local.settings_name, "Joined", "Block"])
    footer_text = "Created with join"
  }
}

# 24. Using string interpolation and expressions
resource "cloudflare_teams_account" "with_interpolation" {
  account_id           = var.cloudflare_account_id
  activity_log_enabled = true
  tls_decrypt_enabled  = var.enable_tls_decrypt

  block_page {
    enabled     = true
    name        = "Block for ${var.settings_prefix} account"
    footer_text = "Account ID: ${var.cloudflare_account_id}"
    header_text = "Settings: ${local.settings_name}"
  }
}

# 25. Using ternary expressions
resource "cloudflare_teams_account" "with_ternary" {
  account_id                 = var.cloudflare_account_id
  activity_log_enabled       = var.enable_tls_decrypt ? true : false
  tls_decrypt_enabled        = var.enable_browser_isolation ? true : false
  protocol_detection_enabled = var.enable_tls_decrypt && var.enable_browser_isolation ? true : false

  antivirus {
    enabled_download_phase = var.enable_tls_decrypt ? true : false
    enabled_upload_phase   = var.enable_browser_isolation ? true : false
    fail_closed            = false

    notification_settings {
      enabled     = true
      message     = var.enable_tls_decrypt ? "Full scanning enabled" : "Basic scanning"
      support_url = "https://support.example.com"
    }
  }
}

# ============================================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# ============================================================================

# 26. With lifecycle - create_before_destroy
resource "cloudflare_teams_account" "with_lifecycle_create" {
  account_id           = var.cloudflare_account_id
  activity_log_enabled = true
  tls_decrypt_enabled  = true

  block_page {
    enabled     = true
    name        = "Protected Block Page"
    footer_text = "Lifecycle protected"
  }

  lifecycle {
    create_before_destroy = true
  }
}

# 27. With lifecycle - ignore_changes
resource "cloudflare_teams_account" "with_lifecycle_ignore" {
  account_id                 = var.cloudflare_account_id
  activity_log_enabled       = true
  tls_decrypt_enabled        = true
  protocol_detection_enabled = false

  fips {
    tls = true
  }

  lifecycle {
    ignore_changes = [protocol_detection_enabled]
  }
}

# 28. With lifecycle - prevent_destroy (set to false for testing)
resource "cloudflare_teams_account" "with_lifecycle_prevent" {
  account_id           = var.cloudflare_account_id
  activity_log_enabled = true

  block_page {
    enabled     = true
    name        = "Critical Block Page"
    footer_text = "Do not destroy"
  }

  lifecycle {
    prevent_destroy = false # Set to false for testing
  }
}

# ============================================================================
# Pattern Group 8: All MaxItems:1 Blocks in Different Combinations
# ============================================================================

# 29. Extended email matching + certificate
resource "cloudflare_teams_account" "email_and_cert" {
  account_id = var.cloudflare_account_id

  extended_email_matching {
    enabled = true
  }

  certificate {
    id = "cert-combo-123"
  }
}

# 30. All scanning features (body_scanning + antivirus)
resource "cloudflare_teams_account" "all_scanning" {
  account_id = var.cloudflare_account_id

  body_scanning {
    inspection_mode = "deep"
  }

  antivirus {
    enabled_download_phase = true
    enabled_upload_phase   = true
    fail_closed            = true

    notification_settings {
      enabled     = true
      message     = "Comprehensive scanning"
      support_url = "https://support.example.com/scanning"
    }
  }

  fips {
    tls = true
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
