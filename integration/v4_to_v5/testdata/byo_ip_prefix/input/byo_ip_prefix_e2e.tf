# E2E-SKIP: BYO IP Prefix cannot be tested in automated E2E runs
#
# NOTE: This marker is for documentation only. To exclude this resource
# from E2E tests, use: --exclude byo_ip_prefix
#
# REASON FOR SKIP:
# ================
# We are waiting on the Addressing team to provide an IP for use 
# in testing. 
#
# TESTING COVERAGE:
# =================
# ✓ Integration tests: Verify config transformation (v4 -> v5)
#   - Location: integration/v4_to_v5/testdata/byo_ip_prefix/
#   - Tests: Field removal, warning generation, HCL structure
#
# ✓ Provider migration tests: Verify migration logic
#   - Location: cloudflare-terraform-next/internal/services/byo_ip_prefix/migration/v500/
#   - Tests: Config transformation, warning comments, state upgrades
#
# ✓ Manual testing: Can be performed with pre-existing infrastructure
#   - Requires: Existing BYO IP prefix in Cloudflare account
#   - Process: Manual migration with real resources
#
# ✗ E2E tests: Requires pre-existing infrastructure (not automatable)

# Terraform v4 to v5 Migration E2E Test - BYO IP Prefix
#
# Note: The v4 provider's Create function does not call the API to create a
# prefix. It sets the resource ID from prefix_id and reads the existing prefix.
# A real prefix ID must be provided via CLOUDFLARE_BYO_IP_PREFIX_ID.
#
# See: https://github.com/cloudflare/terraform-provider-cloudflare/blob/main/docs/resources/byo_ip_prefix.md

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
