# Consumer resources that reference the computed `secret` attribute of tunnel
# resources across files.  Uses vault_generic_secret (a non-Cloudflare resource)
# to simulate the real-world pattern described in the bug report.
#
# All `.secret` references below must be rewritten to `.tunnel_secret` by the
# cross-file postprocessing pass regardless of which v4 type name was used for
# the tunnel resource.

# Test case 1 (the bug): reference to a tunnel that already uses the preferred
# v4 name cloudflare_zero_trust_tunnel_cloudflared.
# Before the fix this reference was NOT rewritten because
# GetComputedAttributeMappings only listed cloudflare_tunnel as OldResourceType.
resource "vault_generic_secret" "preferred_a_secret" {
  path = "secret/tunnels/preferred-a"

  data_json = jsonencode({
    tunnel_secret = cloudflare_zero_trust_tunnel_cloudflared.preferred_a.secret
  })
}

# Test case 2 (existing behaviour): reference to a tunnel using the deprecated
# cloudflare_tunnel name — must still be rewritten correctly.
resource "vault_generic_secret" "from_old_name_tunnel" {
  path = "secret/tunnels/minimal"

  data_json = jsonencode({
    tunnel_secret = cloudflare_tunnel.minimal.secret
  })
}

# Test case 3 (coexistence): both v4 type names referenced in the same file
resource "vault_generic_secret" "combined" {
  path = "secret/tunnels/combined"

  data_json = jsonencode({
    old_secret       = cloudflare_tunnel.minimal.secret
    preferred_secret = cloudflare_zero_trust_tunnel_cloudflared.preferred_a.secret
  })
}
