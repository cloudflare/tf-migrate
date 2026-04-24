# Integration test for the tunnel_token DiagWarning (ticket 004).
#
# A reference to <tunnel_resource>.<name>.tunnel_token is NOT a valid v4 or v5
# attribute. tf-migrate must NOT rewrite it (there is no safe replacement), and
# must emit a DiagWarning with guidance.
#
# The expected output file for this fixture is identical to the input — the
# tunnel_token reference is left unchanged and the warning appears in the
# diagnostic output, not in the .tf file.

resource "vault_generic_secret" "tunnel_token_ref" {
  path = "secret/tunnels/token"

  data_json = jsonencode({
    # tunnel_token is not a valid attribute — tf-migrate should warn, not rewrite
    token = cloudflare_zero_trust_tunnel_cloudflared.minimal.tunnel_token
  })
}
