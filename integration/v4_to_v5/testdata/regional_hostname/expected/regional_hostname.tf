variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

data "cloudflare_zone" "this" {
  zone_id = var.cloudflare_zone_id
}

locals {
  domain = data.cloudflare_zone.this.name
}

# DNS Records (required before creating regional hostnames)
resource "cloudflare_dns_record" "rh_minimal" {
  zone_id = var.cloudflare_zone_id
  name    = "rh-minimal"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.1"
}

resource "cloudflare_dns_record" "rh_timeouts" {
  zone_id = var.cloudflare_zone_id
  name    = "rh-timeouts"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.2"
}

resource "cloudflare_dns_record" "rh_create_timeout" {
  zone_id = var.cloudflare_zone_id
  name    = "rh-create-timeout"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.3"
}

resource "cloudflare_dns_record" "rh_no_timeouts" {
  zone_id = var.cloudflare_zone_id
  name    = "rh-no-timeouts"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.4"
}

# Regional hostname with no timeouts
resource "cloudflare_regional_hostname" "minimal" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "rh-minimal.${local.domain}"
  region_key = "us"

  depends_on = [cloudflare_dns_record.rh_minimal]
}

# Regional hostname with timeouts (timeouts should be removed in v5)
resource "cloudflare_regional_hostname" "with_timeouts" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "rh-timeouts.${local.domain}"
  region_key = "eu"


  depends_on = [cloudflare_dns_record.rh_timeouts]
}

# Regional hostname with only create timeout
resource "cloudflare_regional_hostname" "create_timeout" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "rh-create-timeout.${local.domain}"
  region_key = "ca"


  depends_on = [cloudflare_dns_record.rh_create_timeout]
}

resource "cloudflare_regional_hostname" "no_timeouts" {
  zone_id    = var.cloudflare_zone_id
  hostname   = "rh-no-timeouts.${local.domain}"
  region_key = "ca"

  depends_on = [cloudflare_dns_record.rh_no_timeouts]
}
