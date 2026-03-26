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
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
}

# ========================================
# Parent Tunnel Resources
# ========================================




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



# ============================================================================
# Pattern 9: Cross-resource reference using both v4 names
# ============================================================================
# This validates that GetResourceRename() returns ALL v4 names for cross-file reference updates
# v4 name option 1: cloudflare_tunnel_config
# v4 name option 2: cloudflare_zero_trust_tunnel_cloudflared_config
# v5 name: cloudflare_zero_trust_tunnel_cloudflared_config




# Resource using v4 name option 2
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "resourcename_opt2" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.resourcename_opt2.id

  config = {
    ingress = [
      {
        service = "http_status:404"
      }
    ]
  }
}



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
          http2_origin             = false
          keep_alive_connections   = 100
          keep_alive_timeout       = 90
          no_happy_eyeballs        = false
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
      http2_origin             = false
      no_happy_eyeballs        = false
      access = {
        required  = true
        team_name = "my-team"
        aud_tag   = ["abc123"]
      }
    }
  }
}


# Parent tunnels for Pattern 9
resource "cloudflare_zero_trust_tunnel_cloudflared" "resourcename_opt1" {
  account_id    = var.cloudflare_account_id
  name          = "${local.name_prefix}-pattern9-opt1-tunnel"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("pattern9-opt1-secret-32-bytes-minimum-here")
}


resource "cloudflare_zero_trust_tunnel_cloudflared" "resourcename_opt2" {
  account_id    = var.cloudflare_account_id
  name          = "${local.name_prefix}-pattern9-opt2-tunnel"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("pattern9-opt2-secret-32-bytes-minimum-here")
}


# Resource using v4 name option 1
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "resourcename_opt1" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.resourcename_opt1.id

  config = {
    ingress = [
      {
        service = "http_status:404"
      }
    ]
  }
}


# Dependent resource that references option 1
resource "cloudflare_zero_trust_tunnel_cloudflared_route" "ref_opt1" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.resourcename_opt1.id
  network    = "192.0.2.0/24"

  depends_on = [cloudflare_zero_trust_tunnel_cloudflared_config.resourcename_opt1]
}


# Dependent resource that references option 2
resource "cloudflare_zero_trust_tunnel_cloudflared_route" "ref_opt2" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.resourcename_opt2.id
  network    = "198.51.100.0/24"

  depends_on = [cloudflare_zero_trust_tunnel_cloudflared_config.resourcename_opt2]
}


variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
