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

# BYO IP Prefix variables for testing
# Set via TF_VAR_byo_ip_* environment variables or directly in terraform.tfvars
variable "byo_ip_cidr" {
  description = "BYO IP CIDR notation for the prefix"
  type        = string
  default     = "2606:54c2:2::/48"
}

variable "byo_ip_asn" {
  description = "BYO IP ASN number"
  type        = string
  default     = "13335"
}

variable "byo_ip_loa_document_id" {
  description = "BYO IP LOA document ID"
  type        = string
  default     = "c8af01b0dd8f4779980824d9a8c84136"
}

variable "byo_ip_prefix_id" {
  description = "BYO IP prefix ID (for v4 provider or migration tests)"
  type        = string
  default     = "e71bfbf3fdc740aa8b90a61e9dfffe79"
}
