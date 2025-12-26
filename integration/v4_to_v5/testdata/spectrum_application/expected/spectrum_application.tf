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

variable "spectrum_origin_ip" {
  description = "Origin IP address for Spectrum applications"
  type        = string
  default     = "54.205.230.94" # httpbin.org
}

# Look up zone to get the domain name
data "cloudflare_zone" "this" {
  zone_id = var.cloudflare_zone_id
}

locals {
  name_prefix = "cftftest"
  zone_name   = data.cloudflare_zone.this.name
}

# Test Case 1: Basic resource with required fields only
resource "cloudflare_spectrum_application" "basic" {
  zone_id       = var.cloudflare_zone_id
  protocol      = "tcp/443"
  origin_direct = ["tcp://${var.spectrum_origin_ip}:443"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-basic.${local.zone_name}"
  }
}

# Test Case 2: Resource with all required fields
resource "cloudflare_spectrum_application" "with_id" {
  zone_id       = var.cloudflare_zone_id
  protocol      = "tcp/22"
  origin_direct = ["tcp://${var.spectrum_origin_ip}:22"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-ssh.${local.zone_name}"
  }
}


# Test Case 3: Resource with origin_direct
resource "cloudflare_spectrum_application" "origin_dns_test" {
  zone_id       = var.cloudflare_zone_id
  protocol      = "tcp/80"
  origin_direct = ["tcp://${var.spectrum_origin_ip}:80"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-http.${local.zone_name}"
  }
}

# Test Case 4: Resource with edge_ips block
resource "cloudflare_spectrum_application" "edge_ips_test" {
  zone_id       = var.cloudflare_zone_id
  protocol      = "tcp/80"
  origin_direct = ["tcp://${var.spectrum_origin_ip}:80"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-app.${local.zone_name}"
  }
  edge_ips = {
    type         = "dynamic"
    connectivity = "all"
  }
}

# Test Case 5: Complex resource with all optional fields
resource "cloudflare_spectrum_application" "complex" {
  zone_id            = var.cloudflare_zone_id
  protocol           = "tcp/443"
  tls                = "flexible"
  argo_smart_routing = true
  ip_firewall        = true
  proxy_protocol     = "v1"
  traffic_type       = "direct"



  origin_direct = ["tcp://${var.spectrum_origin_ip}:443"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-complex.${local.zone_name}"
  }
  edge_ips = {
    type         = "dynamic"
    connectivity = "ipv4"
  }
}

# Test Case 6: Conditional creation - Production only
resource "cloudflare_spectrum_application" "conditional" {
  count = var.environment == "production" ? 1 : 0

  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/443"


  origin_direct = ["tcp://${var.spectrum_origin_ip}:443"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-prod.${local.zone_name}"
  }
}

# Test Case 7: Cross-resource reference - Using zone data source
data "cloudflare_zone" "main" {
  zone_id = var.cloudflare_zone_id
}

resource "cloudflare_spectrum_application" "zone_ref" {
  zone_id  = data.cloudflare_zone.main.id
  protocol = "tcp/443"


  origin_direct = ["tcp://${var.spectrum_origin_ip}:443"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-zoneref.${local.zone_name}"
  }
}

# Test Case 8: Long-form resource with all blocks
resource "cloudflare_spectrum_application" "kitchen_sink" {
  zone_id            = var.cloudflare_zone_id
  protocol           = "tcp/80"
  tls                = "flexible"
  argo_smart_routing = true
  ip_firewall        = false
  proxy_protocol     = "v2"
  traffic_type       = "direct"



  origin_direct = ["tcp://${var.spectrum_origin_ip}:80"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-kitchensink.${local.zone_name}"
  }
  edge_ips = {
    type         = "dynamic"
    connectivity = "all"
    ips          = []
  }
}

# Variables for conditional tests
variable "environment" {
  type    = string
  default = "production"
}
