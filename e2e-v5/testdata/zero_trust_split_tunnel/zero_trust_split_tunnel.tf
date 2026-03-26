# E2E tests are not applicable for cloudflare_split_tunnel.
#
# cloudflare_split_tunnel does not exist in v5 — it is dissolved into
# cloudflare_zero_trust_device_custom_profile and cloudflare_zero_trust_device_default_profile.
# The migration requires a manual `terraform state rm` step for each split tunnel entry,
# which cannot be automated by the e2e runner. No resources are created here.


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

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
