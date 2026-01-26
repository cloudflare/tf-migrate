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










# ========================================
# Advanced Terraform Patterns for Testing
# ========================================

# Pattern 1: Variable references
variable "domain_name" {
  type    = string
  default = "cf-tf-test.com"
}

variable "record_ttl" {
  type    = number
  default = 3600
}

# Pattern 2: Local values with expressions
locals {
  name_prefix    = "cftftest"
  proxied_ttl    = 1
  tags           = ["e2e-test", "migration-test"]
  common_zone_id = var.cloudflare_zone_id

  # Complex expression
  subdomain_prefix = "cftftest"
  full_subdomain   = "${local.subdomain_prefix}.${var.domain_name}"
}

# Pattern 3: for_each with map
variable "subdomains" {
  type = map(object({
    value   = string
    proxied = bool
  }))
  default = {
    "api" = {
      value   = "192.0.2.10"
      proxied = true
    }
    "www" = {
      value   = "192.0.2.11"
      proxied = true
    }
    "static" = {
      value   = "192.0.2.12"
      proxied = false
    }
  }
}


# Pattern 4: for_each with list converted to set
variable "txt_records" {
  type = list(object({
    name  = string
    value = string
  }))
  default = [
    {
      name  = "_dmarc"
      value = "v=DMARC1; p=quarantine; rua=mailto:dmarc@cf-tf-test.com"
    },
    {
      name  = "_dkim"
      value = "v=DKIM1; k=rsa; p=MIGfMA0GCS..."
    }
  ]
}


# Pattern 5: Count-based resources
variable "backup_mx_count" {
  type    = number
  default = 3
}


# Pattern 6: Conditional resource creation
variable "enable_ipv6" {
  type    = bool
  default = true
}


# Pattern 7: Dynamic blocks (if supported)
variable "caa_records" {
  type = list(object({
    flags = number
    tag   = string
    value = string
  }))
  default = [
    {
      flags = 0
      tag   = "issue"
      value = "letsencrypt.org"
    },
    {
      flags = 0
      tag   = "issuewild"
      value = "letsencrypt.org"
    }
  ]
}






# Standard DNS records
resource "cloudflare_dns_record" "example_a" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-example"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.1"
}

moved {
  from = cloudflare_record.example_a
  to   = cloudflare_dns_record.example_a
}

resource "cloudflare_dns_record" "example_mx" {
  zone_id  = var.cloudflare_zone_id
  name     = "@"
  type     = "MX"
  priority = 10
  ttl      = 1
  content  = "mail.cf-tf-test.com"
}

moved {
  from = cloudflare_record.example_mx
  to   = cloudflare_dns_record.example_mx
}

# CAA record with data block - flags should be numeric, content renamed to value
resource "cloudflare_dns_record" "example_caa" {
  zone_id = var.cloudflare_zone_id
  name    = "@"
  type    = "CAA"
  ttl     = 1
  data = {
    flags = "0"
    tag   = "issue"
    value = "ca.cf-tf-test.com"
  }
}

moved {
  from = cloudflare_record.example_caa
  to   = cloudflare_dns_record.example_caa
}

# CAA record with data attribute map
resource "cloudflare_dns_record" "example_caa_map" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-caa-map"
  type    = "CAA"
  ttl     = 1
  data = {
    flags = "128"
    tag   = "issuewild"
    value = "ca.cf-tf-test.com"
  }
}

moved {
  from = cloudflare_record.example_caa_map
  to   = cloudflare_dns_record.example_caa_map
}

# SRV record with data block - priority should be hoisted
resource "cloudflare_dns_record" "example_srv" {
  zone_id  = var.cloudflare_zone_id
  name     = "${local.name_prefix}-service._tcp"
  type     = "SRV"
  priority = 5
  ttl      = 1
  data = {
    priority = 5
    weight   = 10
    port     = 5060
    target   = "sip.cf-tf-test.com"
  }
}

moved {
  from = cloudflare_record.example_srv
  to   = cloudflare_dns_record.example_srv
}

# URI record with data block - priority should be hoisted
resource "cloudflare_dns_record" "example_uri" {
  zone_id  = var.cloudflare_zone_id
  name     = "${local.name_prefix}-http._tcp"
  type     = "URI"
  priority = 10
  ttl      = 1
  data = {
    weight = 20
    target = "http://cf-tf-test.com/path"
  }
}

moved {
  from = cloudflare_record.example_uri
  to   = cloudflare_dns_record.example_uri
}

# Record without TTL - should add default TTL
resource "cloudflare_dns_record" "example_cname" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-www"
  type    = "CNAME"
  proxied = false
  ttl     = 1
  content = "cf-tf-test.com"
}

moved {
  from = cloudflare_record.example_cname
  to   = cloudflare_dns_record.example_cname
}

# Record with existing TTL
resource "cloudflare_dns_record" "example_txt" {
  zone_id = var.cloudflare_zone_id
  name    = "@"
  type    = "TXT"
  ttl     = 3600
  content = "v=spf1 include:_spf.cf-tf-test.com ~all"
}

moved {
  from = cloudflare_record.example_txt
  to   = cloudflare_dns_record.example_txt
}

# TXT record for OPENPGPKEY (not a native type in v4)
# This tests migrating custom TXT records
resource "cloudflare_dns_record" "example_openpgpkey" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-user._openpgpkey"
  type    = "TXT"
  ttl     = 3600
  content = "mQENBFzjqGoBCADTKLKfh..."
}

moved {
  from = cloudflare_record.example_openpgpkey
  to   = cloudflare_dns_record.example_openpgpkey
}

resource "cloudflare_dns_record" "subdomain_a_records" {
  for_each = var.subdomains

  zone_id = local.common_zone_id
  name    = each.key
  type    = "A"
  proxied = each.value.proxied
  ttl     = each.value.proxied ? local.proxied_ttl : var.record_ttl
  content = each.value.value
}

moved {
  from = cloudflare_record.subdomain_a_records
  to   = cloudflare_dns_record.subdomain_a_records
}

resource "cloudflare_dns_record" "security_txt_records" {
  for_each = { for idx, record in var.txt_records : record.name => record }

  zone_id = var.cloudflare_zone_id
  name    = each.value.name
  type    = "TXT"
  ttl     = var.record_ttl
  content = each.value.value
}

moved {
  from = cloudflare_record.security_txt_records
  to   = cloudflare_dns_record.security_txt_records
}

resource "cloudflare_dns_record" "backup_mx" {
  count = var.backup_mx_count

  zone_id  = var.cloudflare_zone_id
  name     = "@"
  type     = "MX"
  priority = (count.index + 2) * 10
  ttl      = var.record_ttl
  content  = "mx${count.index + 2}.${var.domain_name}"
}

moved {
  from = cloudflare_record.backup_mx
  to   = cloudflare_dns_record.backup_mx
}

resource "cloudflare_dns_record" "ipv6_aaaa" {
  count = var.enable_ipv6 ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-ipv6"
  type    = "AAAA"
  proxied = true
  ttl     = 1
  content = "2001:db8:85a3::8a2e:370:7334"
}

moved {
  from = cloudflare_record.ipv6_aaaa
  to   = cloudflare_dns_record.ipv6_aaaa
}

# Pattern 8: Resource with complex data structures
resource "cloudflare_dns_record" "dnslink" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-dnslink"
  type    = "TXT"
  ttl     = var.record_ttl
  content = "dnslink=/ipfs/QmXoypizjW3WknFiJnKLwHCnL72vedxjQkDDP1mXWo6uco"
}

moved {
  from = cloudflare_record.dnslink
  to   = cloudflare_dns_record.dnslink
}

# Pattern 9: Cross-resource references
resource "cloudflare_dns_record" "cname_to_a" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-alias"
  type    = "CNAME"
  proxied = false
  ttl     = var.record_ttl
  content = "example.${var.domain_name}"
}

moved {
  from = cloudflare_record.cname_to_a
  to   = cloudflare_dns_record.cname_to_a
}

# Pattern 10: Resource with lifecycle meta-arguments
resource "cloudflare_dns_record" "protected_record" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-protected"
  type    = "A"
  proxied = true
  ttl     = 1

  lifecycle {
    prevent_destroy       = false
    create_before_destroy = true
  }
  content = "192.0.2.99"
}

moved {
  from = cloudflare_record.protected_record
  to   = cloudflare_dns_record.protected_record
}

# Pattern 11: Using terraform expressions
resource "cloudflare_dns_record" "conditional_value" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-conditional"
  type    = var.enable_ipv6 ? "AAAA" : "A"
  proxied = true
  ttl     = 1
  content = var.enable_ipv6 ? "2001:db8::1" : "192.0.2.100"
}

moved {
  from = cloudflare_record.conditional_value
  to   = cloudflare_dns_record.conditional_value
}

# Pattern 12: Resource with tags/comments
resource "cloudflare_dns_record" "tagged_record" {
  zone_id = var.cloudflare_zone_id
  name    = "${local.name_prefix}-tagged"
  type    = "A"
  proxied = true
  ttl     = 1
  comment = "This is a test record for e2e migration testing"
  content = "192.0.2.200"
}

moved {
  from = cloudflare_record.tagged_record
  to   = cloudflare_dns_record.tagged_record
}
