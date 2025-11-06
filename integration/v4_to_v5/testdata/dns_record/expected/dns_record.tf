# Standard DNS records
resource "cloudflare_dns_record" "example_a" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "example"
  type    = "A"
  proxied = true
  ttl     = 1
  content = "192.0.2.1"
}

resource "cloudflare_dns_record" "example_mx" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  name     = "@"
  type     = "MX"
  priority = 10
  ttl      = 1
  content  = "mail.example.com"
}

# CAA record with data block - flags should be numeric, content renamed to value
resource "cloudflare_dns_record" "example_caa" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "@"
  type    = "CAA"
  ttl     = 1
  data = {
    flags = "0"
    tag   = "issue"
    value = "ca.example.com"
  }
}

# CAA record with data attribute map
resource "cloudflare_dns_record" "example_caa_map" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "caa-map"
  type    = "CAA"
  data = {
    flags = 128
    tag   = "issuewild"
    value = "ca.example.com"
  }
  ttl = 1
}

# SRV record with data block - priority should be hoisted
resource "cloudflare_dns_record" "example_srv" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  name     = "_service._tcp"
  type     = "SRV"
  ttl      = 1
  priority = 5
  data = {
    priority = 5
    weight   = 10
    port     = 5060
    target   = "sip.example.com"
  }
}

# URI record with data block - priority should be hoisted
resource "cloudflare_dns_record" "example_uri" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  name     = "_http._tcp"
  type     = "URI"
  ttl      = 1
  priority = 10
  data = {
    weight = 20
    target = "http://example.com/path"
  }
}

# Record without TTL - should add default TTL
resource "cloudflare_dns_record" "example_cname" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "www"
  type    = "CNAME"
  proxied = false
  ttl     = 1
  content = "example.com"
}

# Record with existing TTL
resource "cloudflare_dns_record" "example_txt" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "@"
  type    = "TXT"
  ttl     = 3600
  content = "v=spf1 include:_spf.example.com ~all"
}

# OPENPGPKEY record - value should be renamed to content
resource "cloudflare_dns_record" "example_openpgpkey" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "user._openpgpkey"
  type    = "OPENPGPKEY"
  ttl     = 3600
  content = "mQENBFzjqGoBCADTKLKfh..."
}