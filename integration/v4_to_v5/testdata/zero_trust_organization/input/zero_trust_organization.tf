variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

locals {
  name_prefix = "cftftest"
  org_names = toset(["org-alpha", "org-beta", "org-gamma"])
  org_configs = {
    dev     = { auth_domain = "${local.name_prefix}-dev.cloudflareaccess.com", session = "12h" }
    staging = { auth_domain = "${local.name_prefix}-staging.cloudflareaccess.com", session = "24h" }
    prod    = { auth_domain = "${local.name_prefix}-prod.cloudflareaccess.com", session = "8h" }
  }
}

# ===== Basic Scenarios (8) =====
resource "cloudflare_access_organization" "minimal_account" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-minimal-account.cloudflareaccess.com"
  name        = "Minimal Account Organization"
}

resource "cloudflare_access_organization" "minimal_zone" {
  zone_id     = var.cloudflare_zone_id
  auth_domain = "${local.name_prefix}-minimal-zone.cloudflareaccess.com"
  name        = "Minimal Zone Organization"
}

resource "cloudflare_access_organization" "complete" {
  account_id                         = var.cloudflare_account_id
  auth_domain                        = "${local.name_prefix}-complete.cloudflareaccess.com"
  name                               = "Complete Organization"
  is_ui_read_only                    = true
  ui_read_only_toggle_reason         = "Managed via Terraform"
  user_seat_expiration_inactive_time = "730h"
  auto_redirect_to_identity          = true
  session_duration                   = "24h"
  allow_authenticate_via_warp        = true
  warp_auth_session_duration         = "12h"

  login_design {
    background_color = "#000000"
    text_color       = "#FFFFFF"
    logo_path        = "https://assets.cf-tf-test.com/logo.png"
    header_text      = "Welcome to Our Platform"
    footer_text      = "Powered by Cloudflare Access"
  }

  custom_pages {
    forbidden       = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
    identity_denied = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
  }
}

resource "cloudflare_access_organization" "with_login_design" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-login-design.cloudflareaccess.com"
  name        = "Organization with Custom Login Design"

  login_design {
    background_color = "#1a1a2e"
    text_color       = "#eaeaea"
    logo_path        = "https://assets.cf-tf-test.com/custom-logo.png"
    header_text      = "Enterprise Portal"
    footer_text      = "© 2026 Enterprise Inc."
  }
}

resource "cloudflare_access_organization" "with_custom_pages" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-custom-pages.cloudflareaccess.com"
  name        = "Organization with Custom Pages"

  custom_pages {
    forbidden       = "cccccccc-cccc-cccc-cccc-cccccccccccc"
    identity_denied = "dddddddd-dddd-dddd-dddd-dddddddddddd"
  }
}

resource "cloudflare_access_organization" "with_both_nested" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-both-nested.cloudflareaccess.com"
  name        = "Organization with Both Nested Blocks"

  login_design {
    background_color = "#2c3e50"
    text_color       = "#ecf0f1"
  }

  custom_pages {
    forbidden = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
  }
}

resource "cloudflare_access_organization" "with_booleans_true" {
  account_id                      = var.cloudflare_account_id
  auth_domain                     = "${local.name_prefix}-booleans-true.cloudflareaccess.com"
  name                            = "Organization with Booleans True"
  is_ui_read_only                 = true
  auto_redirect_to_identity       = true
  allow_authenticate_via_warp     = true
}

resource "cloudflare_access_organization" "with_durations" {
  account_id                         = var.cloudflare_account_id
  auth_domain                        = "${local.name_prefix}-durations.cloudflareaccess.com"
  name                               = "Organization with All Duration Fields"
  session_duration                   = "48h"
  user_seat_expiration_inactive_time = "1460h"
  warp_auth_session_duration         = "2h30m"
}

# ===== v4 Resource Name Variations (2) =====
resource "cloudflare_access_organization" "deprecated_name" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-deprecated.cloudflareaccess.com"
  name        = "Using Deprecated Resource Name"
}

resource "cloudflare_zero_trust_access_organization" "current_name" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-current.cloudflareaccess.com"
  name        = "Using Current Resource Name"
}

# ===== Partial Field Combinations (8) =====
resource "cloudflare_access_organization" "partial_login_1" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-partial-login-1.cloudflareaccess.com"
  name        = "Partial Login Design - Colors Only"

  login_design {
    background_color = "#16213e"
    text_color       = "#f1f1f1"
  }
}

resource "cloudflare_access_organization" "partial_login_2" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-partial-login-2.cloudflareaccess.com"
  name        = "Partial Login Design - Logo Only"

  login_design {
    logo_path = "https://assets.cf-tf-test.com/simple-logo.png"
  }
}

resource "cloudflare_access_organization" "partial_custom_1" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-partial-custom-1.cloudflareaccess.com"
  name        = "Partial Custom Pages - Identity Denied Only"

  custom_pages {
    identity_denied = "11111111-1111-1111-1111-111111111111"
  }
}

resource "cloudflare_access_organization" "partial_custom_2" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-partial-custom-2.cloudflareaccess.com"
  name        = "Partial Custom Pages - Forbidden Only"

  custom_pages {
    forbidden = "22222222-2222-2222-2222-222222222222"
  }
}

resource "cloudflare_access_organization" "only_session" {
  account_id       = var.cloudflare_account_id
  auth_domain      = "${local.name_prefix}-only-session.cloudflareaccess.com"
  name             = "Organization with Only Session Duration"
  session_duration = "6h"
}

resource "cloudflare_access_organization" "only_seat_expiration" {
  account_id                         = var.cloudflare_account_id
  auth_domain                        = "${local.name_prefix}-only-seat.cloudflareaccess.com"
  name                               = "Organization with Only Seat Expiration"
  user_seat_expiration_inactive_time = "730h"
}

resource "cloudflare_access_organization" "only_warp_duration" {
  account_id                 = var.cloudflare_account_id
  auth_domain                = "${local.name_prefix}-only-warp.cloudflareaccess.com"
  name                       = "Organization with Only WARP Duration"
  warp_auth_session_duration = "1h30m"
}

resource "cloudflare_access_organization" "only_readonly" {
  account_id                 = var.cloudflare_account_id
  auth_domain                = "${local.name_prefix}-only-readonly.cloudflareaccess.com"
  name                       = "Organization with Only Read-Only Flag"
  is_ui_read_only            = true
  ui_read_only_toggle_reason = "Locked down for compliance"
}

# ===== Edge Cases (5) =====
resource "cloudflare_access_organization" "special_chars" {
  account_id                 = var.cloudflare_account_id
  auth_domain                = "${local.name_prefix}-special.cloudflareaccess.com"
  name                       = "Org with Special: @#$% & Chars!"
  ui_read_only_toggle_reason = "Locked: Deployment (2026-01-20 @ 14:00 UTC)"

  login_design {
    header_text = "Welcome! Let's get started..."
    footer_text = "Questions? Email: support@cf-tf-test.com"
  }
}

resource "cloudflare_access_organization" "long_strings" {
  account_id                 = var.cloudflare_account_id
  auth_domain                = "${local.name_prefix}-long.cloudflareaccess.com"
  name                       = "Organization with Very Long Name That Contains Many Characters"
  ui_read_only_toggle_reason = "This organization is locked because of maintenance"

  login_design {
    header_text = "This is a very long header text about the organization"
    footer_text = "This footer contains legal disclaimers and copyright notices"
  }
}

resource "cloudflare_access_organization" "duration_formats" {
  account_id                         = var.cloudflare_account_id
  auth_domain                        = "${local.name_prefix}-duration-formats.cloudflareaccess.com"
  name                               = "Organization with Various Duration Formats"
  session_duration                   = "2h45m"
  user_seat_expiration_inactive_time = "730h"
  warp_auth_session_duration         = "90m"
}

resource "cloudflare_access_organization" "color_formats" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-colors.cloudflareaccess.com"
  name        = "Organization with Various Color Formats"

  login_design {
    background_color = "#abc"
    text_color       = "#FFFFFF"
  }
}

resource "cloudflare_access_organization" "booleans_false" {
  account_id                      = var.cloudflare_account_id
  auth_domain                     = "${local.name_prefix}-booleans-false.cloudflareaccess.com"
  name                            = "Organization with Booleans False"
  is_ui_read_only                 = false
  auto_redirect_to_identity       = false
  allow_authenticate_via_warp     = false
}

# ===== Terraform Patterns (8) =====
resource "cloudflare_access_organization" "foreach_set" {
  for_each        = local.org_names
  account_id      = var.cloudflare_account_id
  auth_domain     = "${local.name_prefix}-${each.value}.cloudflareaccess.com"
  name            = "Organization ${each.value}"
  session_duration = "24h"
}

resource "cloudflare_zero_trust_access_organization" "foreach_map" {
  for_each         = local.org_configs
  account_id       = var.cloudflare_account_id
  auth_domain      = each.value.auth_domain
  name             = "Environment: ${each.key}"
  session_duration = each.value.session
}

resource "cloudflare_access_organization" "with_count" {
  count                           = 2
  account_id                      = var.cloudflare_account_id
  auth_domain                     = "${local.name_prefix}-count-${count.index}.cloudflareaccess.com"
  name                            = "Count-based Organization ${count.index}"
  allow_authenticate_via_warp     = count.index == 0
}

resource "cloudflare_access_organization" "with_locals" {
  account_id       = var.cloudflare_account_id
  auth_domain      = "${local.name_prefix}-locals.cloudflareaccess.com"
  name             = "${local.name_prefix} Organization with Locals"

  login_design {
    header_text = "Welcome to ${local.name_prefix}"
  }
}

resource "cloudflare_access_organization" "conditional" {
  account_id                = var.cloudflare_account_id
  auth_domain               = "${local.name_prefix}-conditional.cloudflareaccess.com"
  name                      = "Organization with Conditional"
  is_ui_read_only           = true
  auto_redirect_to_identity = true
  session_duration          = true ? "12h" : "24h"
}

# ===== Meta-Arguments (3) =====
resource "cloudflare_access_organization" "with_lifecycle" {
  account_id       = var.cloudflare_account_id
  auth_domain      = "${local.name_prefix}-lifecycle.cloudflareaccess.com"
  name             = "Organization with Lifecycle"
  session_duration = "24h"

  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_access_organization" "with_depends" {
  account_id  = var.cloudflare_account_id
  auth_domain = "${local.name_prefix}-depends.cloudflareaccess.com"
  name        = "Organization with depends_on"

  depends_on = [cloudflare_access_organization.minimal_account]
}

resource "cloudflare_zero_trust_access_organization" "zone_comprehensive" {
  zone_id                            = var.cloudflare_zone_id
  auth_domain                        = "${local.name_prefix}-zone-comprehensive.cloudflareaccess.com"
  name                               = "Comprehensive Zone-Scoped Organization"
  is_ui_read_only                    = true
  ui_read_only_toggle_reason         = "Zone-level configuration"
  auto_redirect_to_identity          = true
  session_duration                   = "12h"
  allow_authenticate_via_warp        = true
  warp_auth_session_duration         = "6h"

  login_design {
    background_color = "#1e1e1e"
    text_color       = "#ffffff"
    logo_path        = "https://zone.cf-tf-test.com/logo.png"
    header_text      = "Zone Portal"
    footer_text      = "Zone-level Access"
  }

  custom_pages {
    forbidden       = "33333333-3333-3333-3333-333333333333"
    identity_denied = "44444444-4444-4444-4444-444444444444"
  }
}
