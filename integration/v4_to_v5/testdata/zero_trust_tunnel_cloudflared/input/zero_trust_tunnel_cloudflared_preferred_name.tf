# Cross-file .secret reference rewriting test.
#
# Tunnels defined here use the preferred v4 type name
# (cloudflare_zero_trust_tunnel_cloudflared) rather than the deprecated
# cloudflare_tunnel name.  The companion file
# zero_trust_tunnel_cloudflared_secret_refs.tf contains non-Cloudflare
# resources that reference the computed `secret` attribute of these tunnels.
#
# After migration:
#   - `secret` is renamed to `tunnel_secret` in each resource body
#   - `config_src = "local"` is added where absent
#   - No type rename and no moved block (type is already correct)
#   - Cross-file `.secret` references in the consumer file are also rewritten

# Test case 1 (the bug): resource already using the preferred v4 / v5 type name
resource "cloudflare_zero_trust_tunnel_cloudflared" "preferred_a" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-preferred-a-tunnel"
  secret     = base64encode("preferred-a-tunnel-secret-32-bytes-ok")
}

# Second preferred-name tunnel to cover the coexistence case (test case 3)
resource "cloudflare_zero_trust_tunnel_cloudflared" "preferred_b" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-preferred-b-tunnel"
  secret     = base64encode("preferred-b-tunnel-secret-32-bytes-ok")
  config_src = "cloudflare"
}
