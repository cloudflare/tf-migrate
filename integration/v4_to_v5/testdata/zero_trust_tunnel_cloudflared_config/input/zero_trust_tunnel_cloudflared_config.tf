variable "cloudflare_account_id" {
  type = string
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
  name_prefix = "cftftest"
}

# ========================================
# Parent Tunnel Resources
# ========================================

# Tunnel for minimal config test
resource "cloudflare_tunnel" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-minimal-tunnel-config"
  secret     = base64encode("test-secret-that-is-at-least-32-bytes-long")
  config_src = "cloudflare"
}

# Tunnel for comprehensive config test
resource "cloudflare_tunnel" "comprehensive" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-comprehensive-tunnel-config"
  secret     = base64encode("another-secret-32-bytes-or-longer-here")
  config_src = "cloudflare"
}

# Tunnel for testing deprecated resource name
resource "cloudflare_tunnel" "deprecated_name" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-deprecated-tunnel-config"
  secret     = base64encode("deprecated-tunnel-secret-32-bytes-minimum")
  config_src = "cloudflare"
}

# ========================================
# Tunnel Config Resources
# ========================================

# Test 1: Minimal configuration using preferred resource name
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "minimal" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.minimal.id

  config {
    ingress_rule {
      service = "http_status:404"
    }
  }
}

# Test 2: Deprecated resource name (cloudflare_tunnel_config)
resource "cloudflare_tunnel_config" "deprecated_name" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.deprecated_name.id

  config {
    ingress_rule {
      hostname = "app.example.com"
      service  = "http://localhost:8080"
    }
    ingress_rule {
      service = "http_status:404"
    }
  }
}

# Test 3: Comprehensive configuration with all transformations
resource "cloudflare_tunnel_config" "comprehensive" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.comprehensive.id

  config {
    # warp_routing will be removed in v5
    warp_routing {
      enabled = true
    }

    # Config-level origin_request
    origin_request {
      connect_timeout          = "30s"
      tls_timeout              = "10s"
      tcp_keep_alive           = "1m30s"
      keep_alive_timeout       = "1m30s"
      keep_alive_connections   = 100
      http_host_header         = "example.com"
      origin_server_name       = "origin.example.com"
      ca_pool                  = "/path/to/ca.pem"
      no_tls_verify            = false
      disable_chunked_encoding = false
      # Deprecated fields to be removed
      bastion_mode  = true
      proxy_address = "127.0.0.1"
      proxy_port    = 8080
      proxy_type    = ""
      # ip_rules block will be removed
      ip_rules {
        prefix = "192.0.2.0/24"
        ports  = [80, 443]
        allow  = true
      }

      # access block (MaxItems:1 array -> object)
      access {
        required  = true
        team_name = "my-team"
        aud_tag   = ["abc123"]
      }
    }

    ingress_rule {
      hostname = "app.example.com"
      service  = "http://localhost:8080"
      path     = "/api"

      # Ingress-level origin_request
      origin_request {
        connect_timeout = "15s"
        tls_timeout     = "5s"
        # ip_rules to be removed
        ip_rules {
          prefix = "198.51.100.0/24"
          allow  = false
        }
        # access block
        access {
          required = false
        }
      }
    }

    ingress_rule {
      hostname = "api.example.com"
      service  = "http://localhost:3000"
    }

    ingress_rule {
      service = "http_status:404"
    }
  }
}

# ============================================================================
# Pattern 9: Cross-resource reference using both v4 names
# ============================================================================
# This validates that GetResourceRename() returns ALL v4 names for cross-file reference updates
# v4 name option 1: cloudflare_tunnel_config
# v4 name option 2: cloudflare_zero_trust_tunnel_cloudflared_config
# v5 name: cloudflare_zero_trust_tunnel_cloudflared_config

# Parent tunnels for Pattern 9
resource "cloudflare_tunnel" "resourcename_opt1" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-pattern9-opt1-tunnel"
  secret     = base64encode("pattern9-opt1-secret-32-bytes-minimum-here")
  config_src = "cloudflare"
}

resource "cloudflare_tunnel" "resourcename_opt2" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-pattern9-opt2-tunnel"
  secret     = base64encode("pattern9-opt2-secret-32-bytes-minimum-here")
  config_src = "cloudflare"
}

# Resource using v4 name option 1
resource "cloudflare_tunnel_config" "resourcename_opt1" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.resourcename_opt1.id

  config {
    ingress_rule {
      service = "http_status:404"
    }
  }
}

# Resource using v4 name option 2
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "resourcename_opt2" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.resourcename_opt2.id

  config {
    ingress_rule {
      service = "http_status:404"
    }
  }
}

# Dependent resource that references option 1
resource "cloudflare_tunnel_route" "ref_opt1" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.resourcename_opt1.id
  network    = "192.0.2.0/24"

  depends_on = [cloudflare_tunnel_config.resourcename_opt1]
}

# Dependent resource that references option 2
resource "cloudflare_tunnel_route" "ref_opt2" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.resourcename_opt2.id
  network    = "198.51.100.0/24"

  depends_on = [cloudflare_zero_trust_tunnel_cloudflared_config.resourcename_opt2]
}
