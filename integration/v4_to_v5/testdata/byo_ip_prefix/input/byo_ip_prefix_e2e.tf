# Terraform v4 to v5 Migration E2E Test - BYO IP Prefix
#
# IMPORTANT: BYO IP prefixes CANNOT be created via Terraform.
# They must be pre-created via the Cloudflare API and imported.
#
# This E2E test file is designed to work with the e2e-runner,
# which will create a prefix using the API before importing it.
#
# See: https://github.com/cloudflare/terraform-provider-cloudflare/blob/main/docs/resources/byo_ip_prefix.md
#
# import annotation for e2e-runner:
# @import cloudflare_byo_ip_prefix.test ${CLOUDFLARE_ACCOUNT_ID}/${CLOUDFLARE_BYO_IP_PREFIX_ID}

# Variables passed from root module
variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "byo_ip_cidr" {
  description = "BYO IP CIDR (e.g., 2606:54c2:3::/48)"
  type        = string
}

variable "byo_ip_asn" {
  description = "BYO IP ASN (e.g., 13335)"
  type        = string
}

variable "byo_ip_loa_document_id" {
  description = "LOA Document ID"
  type        = string
}

variable "byo_ip_prefix_id" {
  description = "Pre-existing BYO IP prefix ID"
  type        = string
}

# Test instance: BYO IP prefix
# This resource will be imported, not created
resource "cloudflare_byo_ip_prefix" "test" {
  account_id    = var.cloudflare_account_id
  prefix_id     = var.byo_ip_prefix_id
  description   = "E2E migration test prefix"
  advertisement = "on"
}
