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

data "cloudflare_zone" "this" {
  zone_id = var.cloudflare_zone_id
}

locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
  domain = data.cloudflare_zone.this.name
}





# Regional hostname with no timeouts
resource "cloudflare_regional_hostname" "minimal" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "${local.name_prefix}-rh-minimal.${local.domain}"
  region_key = "us"

  depends_on = [cloudflare_dns_record.rh_minimal]
}

# Regional hostname with timeouts (timeouts should be removed in v5)
resource "cloudflare_regional_hostname" "with_timeouts" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "${local.name_prefix}-rh-timeouts.${local.domain}"
  region_key = "eu"


  depends_on = [cloudflare_dns_record.rh_timeouts]
}

# Regional hostname with only create timeout
resource "cloudflare_regional_hostname" "create_timeout" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "${local.name_prefix}-rh-create-timeout.${local.domain}"
  region_key = "ca"


  depends_on = [cloudflare_dns_record.rh_create_timeout]
}

resource "cloudflare_regional_hostname" "no_timeouts" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "${local.name_prefix}-rh-no-timeouts.${local.domain}"
  region_key = "ca"

  depends_on = [cloudflare_dns_record.rh_no_timeouts]
}

# DNS Records (required before creating regional hostnames)
resource "cloudflare_dns_record" "rh_minimal" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-rh-minimal"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.1"
}


resource "cloudflare_dns_record" "rh_timeouts" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-rh-timeouts"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.2"
}


resource "cloudflare_dns_record" "rh_create_timeout" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-rh-create-timeout"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.3"
}


resource "cloudflare_dns_record" "rh_no_timeouts" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-rh-no-timeouts"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.4"
}


variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
