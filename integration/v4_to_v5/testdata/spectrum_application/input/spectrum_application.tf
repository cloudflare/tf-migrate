variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
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
  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/443"
  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-basic.${local.zone_name}"
  }
  origin_direct = ["tcp://203.0.113.1:443"]
}

# Test Case 2: Resource with all required fields
resource "cloudflare_spectrum_application" "with_id" {
  zone_id      = var.cloudflare_zone_id
  protocol     = "tcp/22"
  dns {
    type = "ADDRESS"
    name = "${local.name_prefix}-spectrum-ssh.${local.zone_name}"
  }
  origin_direct = ["tcp://203.0.113.2:22"]
}

# Test Case 3: Resource with edge_ips and origin_direct
resource "cloudflare_spectrum_application" "origin_dns_test" {
  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/3306"
  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-db.${local.zone_name}"
  }
  origin_direct = ["tcp://203.0.113.7:3306"]
}

# Test Case 4: Resource with edge_ips block
resource "cloudflare_spectrum_application" "edge_ips_test" {
  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/8080"
  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-app.${local.zone_name}"
  }
  edge_ips {
    type         = "dynamic"
    connectivity = "all"
  }
  origin_direct = ["tcp://203.0.113.3:8080"]
}

# Test Case 5: Resource with origin_port_range (should convert to origin_port string)
resource "cloudflare_spectrum_application" "port_range" {
  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/3306-3310"
  dns {
    type = "ADDRESS"
    name = "${local.name_prefix}-spectrum-dbrange.${local.zone_name}"
  }
  origin_port_range {
    start = 3306
    end   = 3310
  }
  origin_direct = ["tcp://203.0.113.4"]
}

# Test Case 6: Resource with origin_port attribute (should be preserved)
resource "cloudflare_spectrum_application" "port_direct" {
  zone_id       = var.cloudflare_zone_id
  protocol      = "tcp/443"
  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-secure.${local.zone_name}"
  }
  origin_port   = 8443
  origin_direct = ["tcp://203.0.113.5:443"]
}

# Test Case 7: Complex resource with all optional fields
resource "cloudflare_spectrum_application" "complex" {
  zone_id            = var.cloudflare_zone_id
  protocol           = "tcp/443"
  tls                = "flexible"
  argo_smart_routing = true
  ip_firewall        = true
  proxy_protocol     = "v1"
  traffic_type       = "direct"

  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-complex.${local.zone_name}"
  }

  edge_ips {
    type         = "dynamic"
    connectivity = "ipv4"
  }

  origin_direct = ["tcp://203.0.113.8:443"]
}

# Test Case 8: Resource with origin_port_range transformation
resource "cloudflare_spectrum_application" "dual_transform" {
  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/8080-8090"
  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-dual.${local.zone_name}"
  }
  origin_port_range {
    start = 8080
    end   = 8090
  }
  origin_direct = ["tcp://203.0.113.6"]
}

# Test Case 9: for_each with map - Multiple gaming servers
resource "cloudflare_spectrum_application" "gaming_servers" {
  for_each = {
    minecraft = { protocol = "tcp/25565", port = 25565 }
    rust      = { protocol = "tcp/28015", port = 28015 }
    ark       = { protocol = "udp/27015", port = 27015 }
  }

  zone_id  = var.cloudflare_zone_id
  protocol = each.value.protocol

  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-${each.key}.${local.zone_name}"
  }

  origin_port   = each.value.port
  origin_direct = ["tcp://203.0.113.10"]
}

# Test Case 10: for_each with set - Regional endpoints
resource "cloudflare_spectrum_application" "regional" {
  for_each = toset(["us-east", "us-west", "eu-central"])

  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/443"

  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-${each.key}.${local.zone_name}"
  }

  edge_ips {
    type         = "dynamic"
    connectivity = "all"
  }

  origin_direct = ["tcp://${each.key}.backend.internal:443"]
}

# Test Case 11: count-based resources - Load balancer backends
resource "cloudflare_spectrum_application" "lb_backend" {
  count = 3

  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/443"

  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-lb${count.index}.${local.zone_name}"
  }

  origin_direct = ["tcp://10.0.${count.index}.100:443"]
}

# Test Case 12: Conditional creation - Production only
resource "cloudflare_spectrum_application" "conditional" {
  count = var.environment == "production" ? 1 : 0

  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/443"

  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-prod.${local.zone_name}"
  }

  origin_direct = ["tcp://203.0.113.99:443"]
}

# Test Case 13: Cross-resource reference - Using zone data source
data "cloudflare_zone" "main" {
  zone_id = var.cloudflare_zone_id
}

resource "cloudflare_spectrum_application" "zone_ref" {
  zone_id  = data.cloudflare_zone.main.id
  protocol = "tcp/443"

  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-zoneref.${local.zone_name}"
  }

  origin_direct = ["tcp://203.0.113.20:443"]
}

# Test Case 14: Dynamic block usage
resource "cloudflare_spectrum_application" "dynamic_edge" {
  zone_id  = var.cloudflare_zone_id
  protocol = "tcp/443"

  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-dynamic.${local.zone_name}"
  }

  dynamic "edge_ips" {
    for_each = var.enable_edge_ips ? [1] : []
    content {
      type         = "dynamic"
      connectivity = "all"
    }
  }

  origin_direct = ["tcp://203.0.113.30"]
}

# Test Case 15: Long-form resource with all blocks
resource "cloudflare_spectrum_application" "kitchen_sink" {
  zone_id            = var.cloudflare_zone_id
  protocol           = "tcp/8000"
  tls                = "flexible"
  argo_smart_routing = true
  ip_firewall        = false
  proxy_protocol     = "v2"
  traffic_type       = "direct"

  dns {
    type = "CNAME"
    name = "${local.name_prefix}-spectrum-kitchensink.${local.zone_name}"
  }

  edge_ips {
    type         = "dynamic"
    connectivity = "all"
    ips          = []
  }

  origin_direct = ["203.0.113.9:8000"]
}

# Variables for conditional tests
variable "environment" {
  type    = string
  default = "production"
}

variable "enable_edge_ips" {
  type    = bool
  default = true
}
