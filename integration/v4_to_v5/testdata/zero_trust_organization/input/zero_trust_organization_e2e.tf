# E2E Test Configuration for zero_trust_organization
#
# IMPORTANT: This resource is IMPORT-ONLY in v4 and is a SINGLETON.
# - Organizations cannot be created via Terraform
# - They are created when you enable Zero Trust in the Cloudflare dashboard
# - Each account has exactly ONE organization
# - Therefore, this config has only ONE resource
#
# E2E TEST WORKFLOW (AUTOMATED):
# ================================
#
# The E2E runner handles imports automatically via the tf-migrate:import-address annotation!
#
# Automated workflow (run via `e2e run`):
# 1. PREREQUISITE: Enable Zero Trust in your Cloudflare account via dashboard
# 2. Runner automatically imports the organization (detects annotation below)
# 3. V4 apply configures the imported organization
# 4. Migration transforms v4 config to v5
# 5. V5 plan verifies no drift
# 6. V5 apply succeeds
#
# Manual workflow (for testing without E2E runner):
# 1. PREREQUISITE: Enable Zero Trust via dashboard
# 2. IMPORT: terraform import cloudflare_access_organization.test YOUR_ACCOUNT_ID
# 3. APPLY: terraform apply
# 4. MIGRATE: tf-migrate migrate --config-dir .
# 5. UPGRADE: terraform init -upgrade
# 6. VERIFY: terraform plan  # Should show "No changes"
# 7. APPLY: terraform apply
#
# SUCCESS CRITERIA:
# - Import succeeds (automatic in E2E runner)
# - V4 apply succeeds
# - Migration transforms config correctly
# - V5 plan shows no changes
# - V5 apply succeeds
#
# NOTE: We only have ONE resource because organizations are singletons.
# We cannot test both v4 resource names (cloudflare_access_organization and
# cloudflare_zero_trust_access_organization) simultaneously because they would
# both try to manage the same underlying organization.

locals {
  name_prefix = "cftftest"
}

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

# Basic organization configuration for E2E testing
# NOTE: This is a SINGLETON resource - only one organization per account.
#
# IMPORT ANNOTATION: The line below tells the E2E runner to automatically import this resource.
# The runner will execute: terraform import module.zero_trust_organization.cloudflare_access_organization.test <ACCOUNT_ID>
# tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix} E2E Test Organization"
  auth_domain = "${local.name_prefix}-e2e.cloudflareaccess.com"

  # Test MaxItems:1 block transformation
  login_design {
    background_color = "#000000"
    text_color       = "#FFFFFF"
    logo_path        = "https://e2e-test.cf-tf-test.com/logo.png"
    header_text      = "E2E Test Portal"
    footer_text      = "E2E Testing"
  }

  # Test all optional fields
  session_duration                   = "24h"
  user_seat_expiration_inactive_time = "730h"
  warp_auth_session_duration         = "12h"
  is_ui_read_only                    = true
  # ui_read_only_toggle_reason         = "E2E Testing"
  auto_redirect_to_identity          = true
  allow_authenticate_via_warp        = true
}
