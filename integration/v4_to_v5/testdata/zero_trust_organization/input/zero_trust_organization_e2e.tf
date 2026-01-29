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
# The resource is imported using Terraform's native import block!
# The import annotation below is detected by the e2e runner during init,
# which generates an import block in the root main.tf.
#
# Automated workflow (run via `e2e run`):
# 1. PREREQUISITE: Enable Zero Trust in your Cloudflare account via dashboard
# 2. E2E runner generates import block in root main.tf (from annotation below)
# 3. Terraform import block automatically imports the organization during apply
# 4. V4 apply configures the imported organization
# 5. Migration transforms v4 config to v5
# 6. V5 plan verifies no drift
# 7. V5 apply succeeds
#
# Manual workflow (for testing without E2E runner):
# 1. PREREQUISITE: Enable Zero Trust via dashboard
# 2. RUN: ./bin/e2e-runner init  # Generates main.tf with import blocks
# 3. INIT: cd e2e/tf/v4 && terraform init
# 4. APPLY: terraform apply
# 5. MIGRATE: ../../bin/tf-migrate migrate --config-dir .
# 6. UPGRADE: terraform init -upgrade
# 7. VERIFY: terraform plan  # Should show "No changes"
# 8. APPLY: terraform apply
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
# IMPORT ANNOTATION: The line below tells the E2E runner to generate an import block in root main.tf.
# The runner will generate: import { to = module.zero_trust_organization.cloudflare_access_organization.test, id = "ACCOUNT_ID" }
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
