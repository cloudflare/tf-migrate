resource "cloudflare_list" "ip_list" {
  account_id  = "abc123"
  name        = "ip_list"
  kind        = "ip"
  description = "List of IP addresses"



  items = [{
    comment = "First IP"
    ip      = "1.1.1.1"
    }, {
    comment = "Second IP"
    ip      = "1.1.1.2"
    }, {
    ip = "1.1.1.3"
  }]
}

resource "cloudflare_list" "asn_list" {
  account_id = "abc123"
  name       = "asn_list"
  kind       = "asn"


  items = [{
    asn     = 15169
    comment = "Google ASN"
    }, {
    asn = 13335
  }]
}

resource "cloudflare_list" "hostname_list" {
  account_id = "abc123"
  name       = "hostname_list"
  kind       = "hostname"


  items = [{
    comment = "Example hostname"
    hostname = {
      url_hostname = "example.com"
    }
    }, {
    hostname = {
      url_hostname = "test.example.com"
    }
  }]
}

resource "cloudflare_list" "redirect_list" {
  account_id = "abc123"
  name       = "redirect_list"
  kind       = "redirect"


  items = [{
    comment = "Main redirect"
    redirect = {
      include_subdomains    = true
      preserve_path_suffix  = false
      preserve_query_string = true
      source_url            = "example.com/old"
      status_code           = 301
      subpath_matching      = false
      target_url            = "example.com/new"
    }
    }, {
    redirect = {
      include_subdomains = false
      source_url         = "test.com"
      target_url         = "newtest.com"
    }
  }]
}

resource "cloudflare_list" "empty_list" {
  account_id  = "abc123"
  name        = "empty_list"
  kind        = "ip"
  description = "Empty list"
}
