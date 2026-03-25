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
resource "cloudflare_dlp_profile" "aws_keys" {
  account_id          = var.cloudflare_account_id
  name                = "AWS Keys"
  type                = "predefined"
  allowed_match_count = 3

  entry {
    id      = "aws-access-key-id"
    name    = "AWS Access Key ID"
    enabled = true
  }

  entry {
    id      = "aws-secret-key-id"
    name    = "AWS Secret Access Key"
    enabled = true
  }

  entry {
    id      = "aws-session-token-id"
    name    = "AWS Session Token"
    enabled = false
  }
}

# Pattern 2: Predefined profile with zero_trust name
resource "cloudflare_zero_trust_dlp_profile" "gcp_keys" {
  account_id          = var.cloudflare_account_id
  name                = "GCP Keys"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "gcp-api-key-id"
    name    = "GCP API Key"
    enabled = true
  }
}

# Pattern 3: Predefined profile with ocr_enabled
resource "cloudflare_dlp_profile" "secrets_with_ocr" {
  account_id          = var.cloudflare_account_id
  name                = "Secrets with OCR"
  type                = "predefined"
  allowed_match_count = 5
  ocr_enabled         = true

  entry {
    id      = "ssh-private-key-id"
    name    = "SSH Private Key"
    enabled = true
  }

  entry {
    id      = "azure-client-secret-id"
    name    = "Azure Client Secret"
    enabled = false
  }
}

# Pattern 4: Predefined profile with no enabled entries
resource "cloudflare_dlp_profile" "all_disabled" {
  account_id          = var.cloudflare_account_id
  name                = "All Disabled"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "entry-1-id"
    name    = "Entry One"
    enabled = false
  }

  entry {
    id      = "entry-2-id"
    name    = "Entry Two"
    enabled = false
  }
}

# Pattern 5: Predefined profile with profile_id from import-address annotation
# tf-migrate:import-address=${var.cloudflare_account_id}/c8932cc4-3312-4152-8041-f3f257122dc4
resource "cloudflare_dlp_profile" "with_import_annotation" {
  account_id          = var.cloudflare_account_id
  name                = "Credentials and Secrets"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "d8fcfc9c-773c-405e-8426-21ecbb67ba93"
    name    = "Amazon AWS Access Key ID"
    enabled = false
  }

  entry {
    id      = "2c0e33e1-71da-40c8-aad3-32e674ad3d96"
    name    = "Amazon AWS Secret Access Key"
    enabled = false
  }
}
