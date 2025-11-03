# IP list example
resource "cloudflare_list" "ip_list" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "example_ip_list"
  description = "List of IP addresses"
  kind        = "ip"

  item {
    value {
      ip = "192.0.2.1"
    }
    comment = "Example IP 1"
  }

  item {
    value {
      ip = "192.0.2.2"
    }
    comment = "Example IP 2"
  }
}

# Hostname list example
resource "cloudflare_list" "hostname_list" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example_hostname_list"
  kind       = "hostname"

  item {
    value {
      hostname {
        url_hostname = "example.com"
      }
    }
    comment = "Example hostname"
  }
}

# Redirect list with boolean strings
resource "cloudflare_list" "redirect_list" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example_redirect_list"
  kind       = "redirect"

  item {
    value {
      redirect {
        source_url            = "example.com/old"
        target_url            = "example.com/new"
        status_code           = 301
        include_subdomains    = "enabled"
        subpath_matching      = "disabled"
        preserve_query_string = "enabled"
        preserve_path_suffix  = "disabled"
      }
    }
    comment = "Example redirect"
  }
}
