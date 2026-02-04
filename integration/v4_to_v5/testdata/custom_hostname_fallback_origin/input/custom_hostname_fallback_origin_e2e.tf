# E2E test for custom_hostname_fallback_origin migration
# This tests the real-life singleton scenario (one fallback origin per zone)
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

# CrowdStrike variables (not used by this resource, but required by E2E framework)
variable "crowdstrike_client_id" {
  description = "CrowdStrike client ID (unused)"
  type        = string
  default     = ""
}

variable "crowdstrike_client_secret" {
  description = "CrowdStrike client secret (unused)"
  type        = string
  default     = ""
}

variable "crowdstrike_api_url" {
  description = "CrowdStrike API URL (unused)"
  type        = string
  default     = ""
}

variable "crowdstrike_customer_id" {
  description = "CrowdStrike customer ID (unused)"
  type        = string
  default     = ""
}

# Test: Single instance (singleton resource - only one per zone)
# Note: This is a SINGLETON resource - only ONE fallback origin per zone
# Note: Custom Hostname Fallback Origin requires Cloudflare for SaaS (Business+ plan)
# Note: The origin MUST be a proxied A/AAAA/CNAME DNS record within Cloudflare
resource "cloudflare_custom_hostname_fallback_origin" "e2e_test" {
  zone_id = var.cloudflare_zone_id
  origin  = "${local.name_prefix}-fallback.${var.cloudflare_domain}"
}
