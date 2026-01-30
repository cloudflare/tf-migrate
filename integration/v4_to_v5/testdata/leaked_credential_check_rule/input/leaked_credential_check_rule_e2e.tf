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
  username = "lookup_json_string(http.request.body.raw, \"user\")"
  password = "lookup_json_string(http.request.body.raw, \"secret\")"
}
