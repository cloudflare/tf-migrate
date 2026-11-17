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

data "cloudflare_dcv_delegation" "basic" {
  zone_id = var.cloudflare_zone_id
}

# id is renamed to uuid in v5
output "dcv_delegation_id" {
  value = data.cloudflare_dcv_delegation.basic.uuid
}

# zone_id is unchanged in v5
output "dcv_delegation_zone_id" {
  value = data.cloudflare_dcv_delegation.basic.zone_id
}

# id inside an interpolation, the usual CNAME target
output "dcv_delegation_cname_target" {
  value = "${data.cloudflare_dcv_delegation.basic.uuid}.dcv.cloudflare.com"
}
