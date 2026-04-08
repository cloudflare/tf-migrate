variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

# Tunnel resource - v4 syntax with old resource type and old attribute names
resource "cloudflare_tunnel" "my_tunnel" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-my-tunnel"
  secret     = "super-secret-that-is-32-bytes-long-ok"
}

# Another tunnel for testing multiple references
resource "cloudflare_tunnel" "api_tunnel" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-api-tunnel"
  secret     = "api-secret-that-is-32-bytes-long-ok"
}
