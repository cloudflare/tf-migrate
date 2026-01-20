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
resource "cloudflare_zero_trust_tunnel_cloudflared" "minimal" {
  account_id    = var.cloudflare_account_id
  name          = "${local.name_prefix}-minimal-tunnel-config"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("test-secret-that-is-at-least-32-bytes-long")
}

# Tunnel for comprehensive config test
resource "cloudflare_zero_trust_tunnel_cloudflared" "comprehensive" {
  account_id    = var.cloudflare_account_id
  name          = "${local.name_prefix}-comprehensive-tunnel-config"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("another-secret-32-bytes-or-longer-here")
}

# Tunnel for testing deprecated resource name
resource "cloudflare_zero_trust_tunnel_cloudflared" "deprecated_name" {
  account_id    = var.cloudflare_account_id
  name          = "${local.name_prefix}-deprecated-tunnel-config"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("deprecated-tunnel-secret-32-bytes-minimum")
}

# ========================================
# Tunnel Config Resources
# ========================================

# Test 1: Minimal configuration using preferred resource name
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "minimal" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.minimal.id

  config = {
    ingress = [
      {
        service = "http_status:404"
      }
    ]
  }
}

# Test 2: Deprecated resource name (cloudflare_tunnel_config)
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "deprecated_name" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.deprecated_name.id

  config = {
    ingress = [
      {
        hostname = "app.example.com"
        service  = "http://localhost:8080"
      },
      {
        service = "http_status:404"
      }
    ]
  }
}

# Test 3: Comprehensive configuration with all transformations
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "comprehensive" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.comprehensive.id

  config = {
    ingress = [
      {
        hostname = "app.example.com"
        path     = "/api"
        service  = "http://localhost:8080"
        origin_request = {
          connect_timeout          = 15
          tls_timeout              = 5
          ca_pool                  = ""
          disable_chunked_encoding = false
          keep_alive_timeout       = 90
          no_tls_verify            = false
          origin_server_name       = ""
          proxy_type               = ""
          tcp_keep_alive           = 30
        }
      },
      {
        hostname = "api.example.com"
        service  = "http://localhost:3000"
      },
      {
        service = "http_status:404"
      }
    ]
    origin_request = {
      connect_timeout          = 30
      tls_timeout              = 10
      tcp_keep_alive           = 90
      keep_alive_timeout       = 90
      keep_alive_connections   = 100
      http_host_header         = "example.com"
      origin_server_name       = "origin.example.com"
      ca_pool                  = "/path/to/ca.pem"
      no_tls_verify            = false
      disable_chunked_encoding = false
      proxy_type               = ""
      access = {
        required  = true
        team_name = "my-team"
        aud_tag   = ["abc123"]
      }
    }
  }
}
