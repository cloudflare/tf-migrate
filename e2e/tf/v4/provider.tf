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
