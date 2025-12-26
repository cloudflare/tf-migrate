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




  items = [{
    comment = "Single IP"
    ip      = "1.1.1.1"
    }, {
    comment = "CIDR block"
    ip      = "10.0.0.0"
    }, {
    comment = "IPv6 address"
    ip      = "2001:db8::1"
    }, {
    ip = "192.168.1.1"
  }]
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

  items = [for item in var.dynamic_asn_list : {
    asn     = item.number
    comment = item.description
  }]
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


  items = concat([{ hostname = { url_hostname = "example.com" }, comment = "Static hostname" }], [for host in var.dynamic_hostnames : {
    hostname = { url_hostname = host }
  }])
}

# ========================================
# List 4: Redirect list with full options
# ========================================
resource "cloudflare_list" "redirect_list" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_redirect_list"
  kind        = "redirect"
  description = "Redirect list with full options"



  items = [{
    comment = "Full options redirect"
    redirect = {
      include_subdomains    = true
      preserve_path_suffix  = false
      preserve_query_string = true
      source_url            = "example.com/old"
      status_code           = 301
      subpath_matching      = false
      target_url            = "https://example.com/new"
    }
    }, {
    comment = "Simple redirect"
    redirect = {
      include_subdomains = false
      source_url         = "test.com/"
      status_code        = 302
      target_url         = "https://newtest.com"
    }
    }, {
    redirect = {
      include_subdomains    = true
      preserve_path_suffix  = true
      preserve_query_string = false
      source_url            = "old.example.com/"
      status_code           = 307
      subpath_matching      = true
      target_url            = "https://new.example.com"
    }
  }]
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

  items = [{
    comment = each.key
    hostname = {
      url_hostname = each.value
    }
  }]
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


  items = [{
    asn     = 65001
    comment = "Private ASN"
    }, {
    asn = 65002
  }]
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

  items = [{
    comment = "Conditional redirect"
    redirect = {
      include_subdomains = true
      source_url         = "conditional.example.com/"
      status_code        = 301
      target_url         = "https://target.example.com"
    }
  }]
}

# ========================================
# List 8: IP with lifecycle meta-arguments
# ========================================
resource "cloudflare_list" "ip_lifecycle" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_ip_lifecycle"
  kind        = "ip"
  description = "IP list with lifecycle rules"


  lifecycle {
    create_before_destroy = true
    prevent_destroy       = false
  }
  items = [{
    comment = "Protected IP"
    ip      = "198.51.100.10"
  }]
}

# ========================================
# List 9: Redirect parent (for cross-resource reference)
# ========================================
resource "cloudflare_list" "redirect_parent" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}_redirect_parent"
  kind        = "redirect"
  description = "Parent redirect list"

  items = [{
    comment = "Parent redirect"
    redirect = {
      include_subdomains = false
      source_url         = "parent.example.com/"
      status_code        = 301
      target_url         = "https://parent-target.example.com"
    }
  }]
}

# ========================================
# List 10: Hostname referencing another list
# ========================================
resource "cloudflare_list" "hostname_child" {
  account_id  = var.cloudflare_account_id
  name        = "${cloudflare_list.redirect_parent.name}_child"
  kind        = "hostname"
  description = "Child list referencing parent"

  items = [{
    comment = "Child hostname"
    hostname = {
      url_hostname = "child.example.com"
    }
  }]
}
