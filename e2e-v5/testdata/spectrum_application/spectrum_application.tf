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

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}

variable "spectrum_origin_ip" {
  description = "Origin IP address for Spectrum applications"
  type        = string
  default     = "54.205.230.94" # httpbin.org
}

data "cloudflare_zone" "this" {
  zone_id = var.cloudflare_zone_id
}

locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
  zone_name   = data.cloudflare_zone.this.name
}

# Basic - IPv6 only to avoid exhausting IPv4 quota
resource "cloudflare_spectrum_application" "basic" {
  zone_id       = var.cloudflare_zone_id
  protocol      = "tcp/443"
  origin_direct = ["tcp://${var.spectrum_origin_ip}:443"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-basic.${local.zone_name}"
  }
  edge_ips = {
    type         = "dynamic"
    connectivity = "ipv6"
  }
}

# With all optional fields - IPv6 only
resource "cloudflare_spectrum_application" "full" {
  zone_id            = var.cloudflare_zone_id
  protocol           = "tcp/22"
  tls                = "flexible"
  argo_smart_routing = true
  ip_firewall        = true
  proxy_protocol     = "v1"
  traffic_type       = "direct"
  origin_direct      = ["tcp://${var.spectrum_origin_ip}:22"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-full.${local.zone_name}"
  }
  edge_ips = {
    type         = "dynamic"
    connectivity = "ipv6"
  }
}

# Cross-resource reference - IPv6 only
resource "cloudflare_spectrum_application" "zone_ref" {
  zone_id       = data.cloudflare_zone.this.id
  protocol      = "tcp/8443"
  origin_direct = ["tcp://${var.spectrum_origin_ip}:8443"]
  dns = {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-ref.${local.zone_name}"
  }
  edge_ips = {
    type         = "dynamic"
    connectivity = "ipv6"
  }
}
