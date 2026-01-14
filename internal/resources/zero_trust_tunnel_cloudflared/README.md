# Zero Trust Tunnel Cloudflared Migration Guide (v4 → v5)

This guide explains how `cloudflare_tunnel` / `cloudflare_zero_trust_tunnel` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_tunnel` | `cloudflare_zero_trust_tunnel_cloudflared` | Renamed (adds `_cloudflared`) |
| Alt resource name | `cloudflare_zero_trust_tunnel` | `cloudflare_zero_trust_tunnel_cloudflared` | Renamed (adds `_cloudflared`) |
| Field | `secret` | `tunnel_secret` | Renamed |
| Removed fields | `cname`, `tunnel_token` | - | Computed fields removed |


---

## Migration Examples

### Example 1: Basic Tunnel

**v4 Configuration:**
```hcl
resource "cloudflare_tunnel" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-tunnel"
  secret     = base64encode(random_bytes.tunnel_secret.result)
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared" "example" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "my-tunnel"
  tunnel_secret = base64encode(random_bytes.tunnel_secret.result)
}
```

**What Changed:**
- Resource type: `cloudflare_tunnel` → `cloudflare_zero_trust_tunnel_cloudflared`
- Field: `secret` → `tunnel_secret`

---

### Example 2: With Config Secret

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_tunnel" "app_tunnel" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "app-tunnel"
  secret      = var.tunnel_secret
  config_src  = "cloudflare"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared" "app_tunnel" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "app-tunnel"
  tunnel_secret = var.tunnel_secret
  config_src    = "cloudflare"
}
```

---

