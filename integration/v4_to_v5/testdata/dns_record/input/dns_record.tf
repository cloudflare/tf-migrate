# Standard DNS records
resource "cloudflare_record" "example_a" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "example"
  value   = "192.0.2.1"
  type    = "A"
  proxied = true
}

resource "cloudflare_record" "example_mx" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "@"
  type    = "MX"
  value   = "mail.example.com"
  priority = 10
}

# CAA record with data block - flags should be numeric, content renamed to value
resource "cloudflare_record" "example_caa" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "@"
  type    = "CAA"
  data {
    flags   = "0"
    tag     = "issue"
    content = "ca.example.com"
  }
}

# CAA record with data attribute map
resource "cloudflare_record" "example_caa_map" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "caa-map"
  type    = "CAA"
  data = {
    flags   = 128
    tag     = "issuewild"
    content = "ca.example.com"
  }
}

# SRV record with data block - priority should be hoisted
resource "cloudflare_record" "example_srv" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "_service._tcp"
  type    = "SRV"
  data {
    priority = 5
    weight   = 10
    port     = 5060
    target   = "sip.example.com"
  }
}

# URI record with data block - priority should be hoisted
resource "cloudflare_record" "example_uri" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "_http._tcp"
  type    = "URI"
  data {
    priority = 10
    weight   = 20
    target   = "http://example.com/path"
  }
}

# Record without TTL - should add default TTL
resource "cloudflare_record" "example_cname" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "www"
  value   = "example.com"
  type    = "CNAME"
  proxied = false
}

# Record with existing TTL
resource "cloudflare_record" "example_txt" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "@"
  value   = "v=spf1 include:_spf.example.com ~all"
  type    = "TXT"
  ttl     = 3600
}

# OPENPGPKEY record - value should be renamed to content
resource "cloudflare_record" "example_openpgpkey" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "user._openpgpkey"
  type    = "OPENPGPKEY"
  value   = "mQENBFzjqGoBCADTKLKfh..."
  ttl     = 3600
}