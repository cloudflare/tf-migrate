# E2E test for leaked_credential_check_rule migration

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

resource "cloudflare_leaked_credential_check_rule" "basic" {
  zone_id  = var.cloudflare_zone_id
  username = "lookup_json_string(http.request.body.raw, \"v5_upgrade_user\")"
  password = "lookup_json_string(http.request.body.raw, \"v5_upgrade_secret\")"
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
