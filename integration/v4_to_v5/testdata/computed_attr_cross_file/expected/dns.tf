variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# DNS record that references tunnel's computed cname attribute (v4 syntax)
resource "cloudflare_dns_record" "tunnel_record" {
  zone_id = var.cloudflare_zone_id
  name    = "cftftest-tunnel"
  type    = "CNAME"
  content = cloudflare_zero_trust_tunnel_cloudflared.my_tunnel.name
  ttl     = 300
}

# DNS record that references tunnel's computed secret attribute (v4 syntax)
resource "cloudflare_dns_record" "api_record" {
  zone_id = var.cloudflare_zone_id
  name    = "cftftest-api"
  type    = "CNAME"
  content = cloudflare_zero_trust_tunnel_cloudflared.api_tunnel.name
  ttl     = 300
}

# Example of referencing .cname in a local value
locals {
  tunnel_cname = cloudflare_zero_trust_tunnel_cloudflared.my_tunnel.name
}

# Example of referencing .cname in output
output "tunnel_cname" {
  value = cloudflare_zero_trust_tunnel_cloudflared.my_tunnel.name
}
