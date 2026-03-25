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

variable "domain_name" {
  type    = string
  default = "cf-tf-test.com"
}

variable "record_ttl" {
  type    = number
  default = 3600
}

variable "backup_mx_count" {
  type    = number
  default = 3
}

variable "enable_ipv6" {
  type    = bool
  default = true
}

locals {
  name_prefix    = "v5-upgrade-${replace(var.from_version, ".", "-")}"
  proxied_ttl    = 1
  common_zone_id = var.cloudflare_zone_id
}

# A record
resource "cloudflare_dns_record" "example_a" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-example"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.1"
}

# MX record - use name_prefix to avoid collision with v4->v5 @ MX records
resource "cloudflare_dns_record" "example_mx" {
  zone_id  = var.cloudflare_zone_id
  name     = "${local.name_prefix}-mail"
  type     = "MX"
  priority = 10
  ttl      = 3600
  content  = "${local.name_prefix}-mail.cf-tf-test.com"
}

# CAA record
resource "cloudflare_dns_record" "example_caa" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-caa"
  type    = "CAA"
  ttl     = 3600
  data = {
    flags = "0"
    tag   = "issue"
    value = "ca.cf-tf-test.com"
  }
}

# CAA record with data attribute map
resource "cloudflare_dns_record" "example_caa_map" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-caa-map"
  type    = "CAA"
  ttl     = 3600
  data = {
    flags = "128"
    tag   = "issuewild"
    value = "ca.cf-tf-test.com"
  }
}

# SRV record
resource "cloudflare_dns_record" "example_srv" {
  zone_id  = var.cloudflare_zone_id
  name     = "${local.name_prefix}-service._tcp"
  type     = "SRV"
  priority = 5
  ttl      = 3600
  data = {
    priority = 5
    weight   = 10
    port     = 5060
    target   = "sip.cf-tf-test.com"
  }
}

# URI record
resource "cloudflare_dns_record" "example_uri" {
  zone_id  = var.cloudflare_zone_id
  name     = "${local.name_prefix}-http._tcp"
  type     = "URI"
  priority = 10
  ttl      = 3600
  data = {
    weight = 20
    target = "http://cf-tf-test.com/path"
  }
}

# CNAME record
resource "cloudflare_dns_record" "example_cname" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-www"
  type    = "CNAME"
  proxied = false
  ttl     = 3600
  content = "cf-tf-test.com"
}

# TXT record - use name_prefix to avoid collision
resource "cloudflare_dns_record" "example_txt" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-spf"
  type    = "TXT"
  ttl     = 3600
  content = "v=spf1 include:${local.name_prefix}._spf.cf-tf-test.com ~all"
}

# TXT record - OPENPGPKEY style
resource "cloudflare_dns_record" "example_openpgpkey" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-user._openpgpkey"
  type    = "TXT"
  ttl     = 3600
  content = "mQENBFzjqGoBCADTKLKfh..."
}

# for_each with map - prefix keys through name_prefix
resource "cloudflare_dns_record" "subdomain_a_records" {
  for_each = {
    "${local.name_prefix}-api"    = { value = "192.0.2.10", proxied = true }
    "${local.name_prefix}-www2"   = { value = "192.0.2.11", proxied = true }
    "${local.name_prefix}-static" = { value = "192.0.2.12", proxied = false }
  }

  zone_id = local.common_zone_id
  name    = each.key
  type    = "A"
  proxied = each.value.proxied
  ttl     = each.value.proxied ? local.proxied_ttl : var.record_ttl
  content = each.value.value
}

# for_each TXT records - use name_prefix in names
resource "cloudflare_dns_record" "security_txt_records" {
  for_each = {
    "${local.name_prefix}-dmarc" = "v=DMARC1; p=quarantine; rua=mailto:dmarc@cf-tf-test.com"
    "${local.name_prefix}-dkim"  = "v=DKIM1; k=rsa; p=MIGfMA0GCS..."
  }

  zone_id = var.cloudflare_zone_id
  name    = each.key
  type    = "TXT"
  ttl     = var.record_ttl
  content = each.value
}

# count-based MX records - use name_prefix
resource "cloudflare_dns_record" "backup_mx" {
  count = var.backup_mx_count

  zone_id  = var.cloudflare_zone_id
  name     = "${local.name_prefix}-mx"
  type     = "MX"
  priority = (count.index + 2) * 10
  ttl      = var.record_ttl
  content  = "${local.name_prefix}-mx${count.index + 2}.${var.domain_name}"
}

# AAAA record
resource "cloudflare_dns_record" "ipv6_aaaa" {
  count = var.enable_ipv6 ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-ipv6"
  type    = "AAAA"
  proxied = true
  ttl     = 1
  content = "2001:db8:85a3::8a2e:370:7334"
}

# TXT with complex content
resource "cloudflare_dns_record" "dnslink" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-dnslink"
  type    = "TXT"
  ttl     = var.record_ttl
  content = "dnslink=/ipfs/QmXoypizjW3WknFiJnKLwHCnL72vedxjQkDDP1mXWo6uco"
}

# CNAME cross-reference
resource "cloudflare_dns_record" "cname_to_a" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-alias"
  type    = "CNAME"
  proxied = false
  ttl     = var.record_ttl
  content = "example.${var.domain_name}"
}

# lifecycle
resource "cloudflare_dns_record" "protected_record" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-protected"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.99"

  lifecycle {
    prevent_destroy       = false
    create_before_destroy = true
  }
}

# conditional type
resource "cloudflare_dns_record" "conditional_value" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-conditional"
  type    = var.enable_ipv6 ? "AAAA" : "A"
  proxied = true
  ttl     = 1
  content = var.enable_ipv6 ? "2001:db8::1" : "192.0.2.100"
}

# with comment
resource "cloudflare_dns_record" "tagged_record" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-tagged"
  type    = "A"
  proxied = true
  ttl     = 1
  comment = "This is a test record for e2e migration testing"
  content = "192.0.2.200"
}
