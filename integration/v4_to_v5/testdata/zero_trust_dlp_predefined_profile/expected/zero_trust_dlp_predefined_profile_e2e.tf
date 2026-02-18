# Variables (standard - provided by test framework)
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


# Predefined profile: "Credentials and Secrets"
# This profile already exists in the account and must be imported.
# tf-migrate:import-address=${var.cloudflare_account_id}/c8932cc4-3312-4152-8041-f3f257122dc4
resource "cloudflare_zero_trust_dlp_predefined_profile" "credentials_and_secrets" {
  account_id          = var.cloudflare_account_id
  allowed_match_count = 0





}

moved {
  from = cloudflare_dlp_profile.credentials_and_secrets
  to   = cloudflare_zero_trust_dlp_predefined_profile.credentials_and_secrets
}
