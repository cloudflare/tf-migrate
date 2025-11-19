variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Standard DNS records
resource "cloudflare_record" "example_a" {
  zone_id = var.cloudflare_zone_id
  name    = "example"
  value   = "192.0.2.1"
  type    = "A"
  proxied = true
}

resource "cloudflare_record" "example_mx" {
  zone_id = var.cloudflare_zone_id
  name    = "@"
  type    = "MX"
  value   = "mail.example.com"
  priority = 10
}

# CAA record with data block - flags should be numeric, content renamed to value
resource "cloudflare_record" "example_caa" {
  zone_id = var.cloudflare_zone_id
  name    = "@"
  type    = "CAA"
  data {
    flags = "0"
    tag   = "issue"
    value = "ca.example.com"
  }
}

# CAA record with data attribute map
resource "cloudflare_record" "example_caa_map" {
  zone_id = var.cloudflare_zone_id
  name    = "caa-map"
  type    = "CAA"
  data {
    flags   = "128"
    tag     = "issuewild"
    value   = "ca.example.com"
  }
}

# SRV record with data block - priority should be hoisted
resource "cloudflare_record" "example_srv" {
  zone_id  = var.cloudflare_zone_id
  name     = "_service._tcp"
  type     = "SRV"
  priority = 5
  data {
    priority = 5
    weight   = 10
    port     = 5060
    target   = "sip.example.com"
  }
}

# URI record with data block - priority should be hoisted
resource "cloudflare_record" "example_uri" {
  zone_id  = var.cloudflare_zone_id
  name     = "_http._tcp"
  type     = "URI"
  priority = 10
  data {
    weight = 20
    target = "http://example.com/path"
  }
}

# Record without TTL - should add default TTL
resource "cloudflare_record" "example_cname" {
  zone_id = var.cloudflare_zone_id
  name    = "www-e2e-test"
  value   = "example.com"
  type    = "CNAME"
  proxied = false
}

# Record with existing TTL
resource "cloudflare_record" "example_txt" {
  zone_id = var.cloudflare_zone_id
  name    = "@"
  value   = "v=spf1 include:_spf.example.com ~all"
  type    = "TXT"
  ttl     = 3600
}

# TXT record for OPENPGPKEY (not a native type in v4)
# This tests migrating custom TXT records
resource "cloudflare_record" "example_openpgpkey" {
  zone_id = var.cloudflare_zone_id
  name    = "user._openpgpkey"
  type    = "TXT"
  value   = "mQENBFzjqGoBCADTKLKfh..."
  ttl     = 3600
}

# ========================================
# Advanced Terraform Patterns for Testing
# ========================================

# Pattern 1: Variable references
variable "domain_name" {
  type    = string
  default = "example.com"
}

variable "record_ttl" {
  type    = number
  default = 3600
}

# Pattern 2: Local values with expressions
locals {
  proxied_ttl = 1
  tags        = ["e2e-test", "migration-test"]
  common_zone_id = var.cloudflare_zone_id

  # Complex expression
  subdomain_prefix = "test"
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

resource "cloudflare_record" "subdomain_a_records" {
  for_each = var.subdomains

  zone_id = local.common_zone_id
  name    = each.key
  value   = each.value.value
  type    = "A"
  proxied = each.value.proxied
  ttl     = each.value.proxied ? local.proxied_ttl : var.record_ttl
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
      value = "v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com"
    },
    {
      name  = "_dkim"
      value = "v=DKIM1; k=rsa; p=MIGfMA0GCS..."
    }
  ]
}

resource "cloudflare_record" "security_txt_records" {
  for_each = { for idx, record in var.txt_records : record.name => record }

  zone_id = var.cloudflare_zone_id
  name    = each.value.name
  value   = each.value.value
  type    = "TXT"
  ttl     = var.record_ttl
}

# Pattern 5: Count-based resources
variable "backup_mx_count" {
  type    = number
  default = 3
}

resource "cloudflare_record" "backup_mx" {
  count = var.backup_mx_count

  zone_id  = var.cloudflare_zone_id
  name     = "@"
  type     = "MX"
  value    = "mx${count.index + 2}.${var.domain_name}"
  priority = (count.index + 2) * 10
  ttl      = var.record_ttl
}

# Pattern 6: Conditional resource creation
variable "enable_ipv6" {
  type    = bool
  default = true
}

resource "cloudflare_record" "ipv6_aaaa" {
  count = var.enable_ipv6 ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = "ipv6"
  value   = "2001:db8:85a3::8a2e:370:7334"
  type    = "AAAA"
  proxied = true
  ttl     = 1
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

# Pattern 8: Resource with complex data structures
resource "cloudflare_record" "dnslink" {
  zone_id = var.cloudflare_zone_id
  name    = "_dnslink"
  type    = "TXT"
  value   = "dnslink=/ipfs/QmXoypizjW3WknFiJnKLwHCnL72vedxjQkDDP1mXWo6uco"
  ttl     = var.record_ttl
}

# Pattern 9: Cross-resource references
resource "cloudflare_record" "cname_to_a" {
  zone_id = var.cloudflare_zone_id
  name    = "alias"
  value   = "example.${var.domain_name}"
  type    = "CNAME"
  proxied = false
  ttl     = var.record_ttl
}

# Pattern 10: Resource with lifecycle meta-arguments
resource "cloudflare_record" "protected_record" {
  zone_id = var.cloudflare_zone_id
  name    = "protected"
  value   = "192.0.2.99"
  type    = "A"
  proxied = true
  ttl     = 1

  lifecycle {
    prevent_destroy = false
    create_before_destroy = true
  }
}

# Pattern 11: Using terraform expressions
resource "cloudflare_record" "conditional_value" {
  zone_id = var.cloudflare_zone_id
  name    = "conditional"
  value   = var.enable_ipv6 ? "2001:db8::1" : "192.0.2.100"
  type    = var.enable_ipv6 ? "AAAA" : "A"
  proxied = true
  ttl     = 1
}

# Pattern 12: Resource with tags/comments
resource "cloudflare_record" "tagged_record" {
  zone_id = var.cloudflare_zone_id
  name    = "tagged"
  value   = "192.0.2.200"
  type    = "A"
  proxied = true
  ttl     = 1
  comment = "This is a test record for e2e migration testing"
}