resource "cloudflare_list" "ip_list" {
  account_id  = "abc123"
  name        = "ip_list"
  kind        = "ip"
  description = "List of IP addresses"

  item {
    comment = "First IP"
    value {
      ip = "1.1.1.1"
    }
  }

  item {
    comment = "Second IP"
    value {
      ip = "1.1.1.2"
    }
  }

  item {
    value {
      ip = "1.1.1.3"
    }
  }
}

resource "cloudflare_list" "asn_list" {
  account_id = "abc123"
  name       = "asn_list"
  kind       = "asn"

  item {
    comment = "Google ASN"
    value {
      asn = 15169
    }
  }

  item {
    value {
      asn = 13335
    }
  }
}

resource "cloudflare_list" "hostname_list" {
  account_id = "abc123"
  name       = "hostname_list"
  kind       = "hostname"

  item {
    comment = "Example hostname"
    value {
      hostname {
        url_hostname = "example.com"
      }
    }
  }

  item {
    value {
      hostname {
        url_hostname = "test.example.com"
      }
    }
  }
}

resource "cloudflare_list" "redirect_list" {
  account_id = "abc123"
  name       = "redirect_list"
  kind       = "redirect"

  item {
    comment = "Main redirect"
    value {
      redirect {
        source_url            = "example.com/old"
        target_url            = "example.com/new"
        include_subdomains    = "enabled"
        subpath_matching      = "disabled"
        preserve_query_string = "enabled"
        preserve_path_suffix  = "disabled"
        status_code           = 301
      }
    }
  }

  item {
    value {
      redirect {
        source_url         = "test.com"
        target_url         = "newtest.com"
        include_subdomains = "disabled"
      }
    }
  }
}

resource "cloudflare_list" "empty_list" {
  account_id  = "abc123"
  name        = "empty_list"
  kind        = "ip"
  description = "Empty list"
}
