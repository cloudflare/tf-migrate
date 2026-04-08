variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}



# Tunnel resource - v4 syntax with old resource type and old attribute names
resource "cloudflare_zero_trust_tunnel_cloudflared" "my_tunnel" {
  account_id    = var.cloudflare_account_id
  name          = "cftftest-my-tunnel"
  tunnel_secret = "super-secret-that-is-32-bytes-long-ok"
  config_src    = "local"
}

moved {
  from = cloudflare_tunnel.my_tunnel
  to   = cloudflare_zero_trust_tunnel_cloudflared.my_tunnel
}

# Another tunnel for testing multiple references
resource "cloudflare_zero_trust_tunnel_cloudflared" "api_tunnel" {
  account_id    = var.cloudflare_account_id
  name          = "cftftest-api-tunnel"
  tunnel_secret = "api-secret-that-is-32-bytes-long-ok"
  config_src    = "local"
}

moved {
  from = cloudflare_tunnel.api_tunnel
  to   = cloudflare_zero_trust_tunnel_cloudflared.api_tunnel
}
