# Terraform v4 to v5 Migration E2E Test - Zero Trust List
#
# This file is used by the E2E runner instead of zero_trust_list.tf.
# The main integration test file contains for-expression items patterns
# (e.g. items = [for k, v in map : k]) that are valid in v4 (list(string))
# but type-incompatible with the v5 schema (list(object({value, description}))).
# tf-migrate intentionally preserves these expressions verbatim since they are
# opaque, so applying under v5 would fail. This file uses only static values
# so the full E2E workflow (v4 apply → migrate → v5 apply → drift check) can
# run successfully. The for-expression transformation is covered by unit and
# integration tests instead.

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

resource "cloudflare_teams_list" "cftftest_ip" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest IP List"
  type        = "IP"
  description = "E2E test IP list"
  items       = ["192.168.1.1", "10.0.0.1"]
}

resource "cloudflare_teams_list" "cftftest_domain" {
  account_id = var.cloudflare_account_id
  name       = "cftftest Domain List"
  type       = "DOMAIN"

  items_with_description {
    value       = "cf-tf-test.com"
    description = "Main test domain"
  }

  items_with_description {
    value       = "api.cf-tf-test.com"
    description = "API subdomain"
  }
}
