variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

locals {
  name_prefix    = "cftftest_list"
  common_account = var.cloudflare_account_id
}

# ========================================
# List 1: IP list with multiple items (CIDR, IPv6, static)
# ========================================
resource "cloudflare_list" "ip_list" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_ip_list"
  kind        = "ip"
  description = "IP list with various formats"

  item {
    comment = "Single IP"
    value {
      ip = "1.1.1.1"
    }
  }

  item {
    comment = "CIDR block"
    value {
      ip = "10.0.0.0/8"
    }
  }

  item {
    comment = "IPv6 address"
    value {
      ip = "2001:db8::1"
    }
  }

  item {
    value {
      ip = "192.168.1.1"
    }
  }
}

# ========================================
# List 2: ASN list with dynamic items
# ========================================
variable "dynamic_asn_list" {
  type = list(object({
    number      = number
    description = string
  }))
  default = [
    { number = 15169, description = "Google ASN" },
    { number = 13335, description = "Cloudflare ASN" },
    { number = 20940, description = "Akamai ASN" }
  ]
}

resource "cloudflare_list" "asn_list" {
  account_id  = local.common_account
  name        = "${local.name_prefix}_asn_list"
  kind        = "asn"
  description = "ASN list with dynamic items"

  dynamic "item" {
    for_each = var.dynamic_asn_list
    content {
      comment = item.value.description
      value {
        asn = item.value.number
      }
    }
  }
}

# ========================================
# List 3: Hostname list with mixed static and dynamic
# ========================================
variable "dynamic_hostnames" {
  type    = list(string)
  default = ["app1.example.com", "app2.example.com"]
}

resource "cloudflare_list" "hostname_list" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_hostname_list"
  kind        = "hostname"
  description = "Hostname list with mixed items"

  item {
    comment = "Static hostname"
    value {
      hostname {
        url_hostname = "example.com"
      }
    }
  }

  dynamic "item" {
    for_each = var.dynamic_hostnames
    iterator = host
    content {
      value {
        hostname {
          url_hostname = host.value
        }
      }
    }
  }
}

# ========================================
# List 4: Redirect list with full options
# ========================================
resource "cloudflare_list" "redirect_list" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_redirect_list"
  kind        = "redirect"
  description = "Redirect list with full options"

  item {
    comment = "Full options redirect"
    value {
      redirect {
        source_url            = "example.com/old"
        target_url            = "https://example.com/new"
        include_subdomains    = "enabled"
        subpath_matching      = "disabled"
        preserve_query_string = "enabled"
        preserve_path_suffix  = "disabled"
        status_code           = 301
      }
    }
  }

  item {
    comment = "Simple redirect"
    value {
      redirect {
        source_url         = "test.com"
        target_url         = "https://newtest.com"
        include_subdomains = "disabled"
        status_code        = 302
      }
    }
  }

  item {
    value {
      redirect {
        source_url            = "old.example.com"
        target_url            = "https://new.example.com"
        include_subdomains    = "enabled"
        subpath_matching      = "enabled"
        preserve_query_string = "disabled"
        preserve_path_suffix  = "enabled"
        status_code           = 307
      }
    }
  }
}

# ========================================
# List 5: Hostname with for_each (single item map)
# ========================================
variable "hostname_map" {
  type = map(string)
  default = {
    "api" = "api.example.com"
  }
}

resource "cloudflare_list" "hostname_foreach" {
  for_each = var.hostname_map

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_hostname_${each.key}"
  kind        = "hostname"
  description = "Hostname list via for_each"

  item {
    comment = each.key
    value {
      hostname {
        url_hostname = each.value
      }
    }
  }
}

# ========================================
# List 6: ASN with count = 1
# ========================================
resource "cloudflare_list" "asn_count" {
  count = 1

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_asn_count"
  kind        = "asn"
  description = "ASN list with count"

  item {
    comment = "Private ASN"
    value {
      asn = 65001
    }
  }

  item {
    value {
      asn = 65002
    }
  }
}

# ========================================
# List 7: Redirect conditional
# ========================================
variable "enable_redirect" {
  type    = bool
  default = true
}

resource "cloudflare_list" "redirect_conditional" {
  count = var.enable_redirect ? 1 : 0

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_redirect_cond"
  kind        = "redirect"
  description = "Conditional redirect list"

  item {
    comment = "Conditional redirect"
    value {
      redirect {
        source_url         = "conditional.example.com"
        target_url         = "https://target.example.com"
        include_subdomains = "enabled"
        status_code        = 301
      }
    }
  }
}

# ========================================
# List 8: IP with lifecycle meta-arguments
# ========================================
resource "cloudflare_list" "ip_lifecycle" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_ip_lifecycle"
  kind        = "ip"
  description = "IP list with lifecycle rules"

  item {
    comment = "Protected IP"
    value {
      ip = "198.51.100.10"
    }
  }

  lifecycle {
    create_before_destroy = true
    prevent_destroy       = false
  }
}

# ========================================
# List 9: Redirect parent (for cross-resource reference)
# ========================================
resource "cloudflare_list" "redirect_parent" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_redirect_parent"
  kind        = "redirect"
  description = "Parent redirect list"

  item {
    comment = "Parent redirect"
    value {
      redirect {
        source_url         = "parent.example.com"
        target_url         = "https://parent-target.example.com"
        include_subdomains = "disabled"
        status_code        = 301
      }
    }
  }
}

# ========================================
# List 10: Hostname referencing another list
# ========================================
resource "cloudflare_list" "hostname_child" {
  account_id  = var.cloudflare_account_id
  name        = "${cloudflare_list.redirect_parent.name}_child"
  kind        = "hostname"
  description = "Child list referencing parent"

  item {
    comment = "Child hostname"
    value {
      hostname {
        url_hostname = "child.example.com"
      }
    }
  }
}
