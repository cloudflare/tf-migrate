# E2E tests are not applicable for cloudflare_split_tunnel.
#
# cloudflare_split_tunnel does not exist in v5 — it is dissolved into
# cloudflare_zero_trust_device_custom_profile and cloudflare_zero_trust_device_default_profile.
# The migration requires a manual `terraform state rm` step for each split tunnel entry,
# which cannot be automated by the e2e runner. No resources are created here.
