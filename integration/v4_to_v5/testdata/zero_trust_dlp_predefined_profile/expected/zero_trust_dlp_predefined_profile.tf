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





# Pattern 1: Basic predefined profile with entries
resource "cloudflare_zero_trust_dlp_predefined_profile" "aws_keys" {
  account_id          = var.cloudflare_account_id
  name                = "AWS Keys"
  allowed_match_count = 3



  enabled_entries = ["aws-access-key-id", "aws-secret-key-id"]
}

moved {
  from = cloudflare_dlp_profile.aws_keys
  to   = cloudflare_zero_trust_dlp_predefined_profile.aws_keys
}

# Pattern 2: Predefined profile with zero_trust name
resource "cloudflare_zero_trust_dlp_predefined_profile" "gcp_keys" {
  account_id          = var.cloudflare_account_id
  name                = "GCP Keys"
  allowed_match_count = 0

  enabled_entries = ["gcp-api-key-id"]
}

moved {
  from = cloudflare_zero_trust_dlp_profile.gcp_keys
  to   = cloudflare_zero_trust_dlp_predefined_profile.gcp_keys
}

# Pattern 3: Predefined profile with ocr_enabled
resource "cloudflare_zero_trust_dlp_predefined_profile" "secrets_with_ocr" {
  account_id          = var.cloudflare_account_id
  name                = "Secrets with OCR"
  allowed_match_count = 5
  ocr_enabled         = true


  enabled_entries = ["ssh-private-key-id"]
}

moved {
  from = cloudflare_dlp_profile.secrets_with_ocr
  to   = cloudflare_zero_trust_dlp_predefined_profile.secrets_with_ocr
}

# Pattern 4: Predefined profile with no enabled entries
resource "cloudflare_zero_trust_dlp_predefined_profile" "all_disabled" {
  account_id          = var.cloudflare_account_id
  name                = "All Disabled"
  allowed_match_count = 0


}

moved {
  from = cloudflare_dlp_profile.all_disabled
  to   = cloudflare_zero_trust_dlp_predefined_profile.all_disabled
}
