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
resource "cloudflare_dlp_profile" "credentials_and_secrets" {
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

  entry {
    id      = "6c6579e4-d832-42d5-905c-8e53340930f2"
    name    = "Google GCP API Key"
    enabled = false
  }

  entry {
    id      = "4e92c006-3802-4dff-bbe1-8e1513b1c92a"
    name    = "Microsoft Azure Client Secret"
    enabled = false
  }

  entry {
    id      = "5c713294-2375-4904-abcf-e4a15be4d592"
    name    = "SSH Private Key"
    enabled = false
  }
}
