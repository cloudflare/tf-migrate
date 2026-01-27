terraform {
  required_version = ">= 1.0"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }

  # Remote state backend (R2)
  # Configuration provided via backend.hcl or -backend-config flags
  # See e2e/scripts/init for backend initialization
  backend "s3" {}
}

# Uses CLOUDFLARE_API_KEY and CLOUDFLARE_EMAIL environment variables
provider "cloudflare" {}

# Common variables from environment
# Set via TF_VAR_cloudflare_account_id and TF_VAR_cloudflare_zone_id
variable "cloudflare_account_id" {
  description = "Cloudflare account ID for resources"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for DNS records"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

# CrowdStrike credentials for device posture integration tests
# Set via TF_VAR_crowdstrike_* environment variables or CLOUDFLARE_CROWDSTRIKE_* variables
variable "crowdstrike_client_id" {
  description = "CrowdStrike S2S Client ID"
  type        = string
  default     = ""
}

variable "crowdstrike_client_secret" {
  description = "CrowdStrike S2S Client Secret"
  type        = string
  sensitive   = true
  default     = ""
}

variable "crowdstrike_api_url" {
  description = "CrowdStrike API URL"
  type        = string
  default     = ""
}

variable "crowdstrike_customer_id" {
  description = "CrowdStrike Customer ID"
  type        = string
  default     = ""
}
