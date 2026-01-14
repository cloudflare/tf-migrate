# Zero Trust Tunnel Cloudflared Virtual Network Migration Guide (v4 → v5)

This guide explains how `cloudflare_tunnel_virtual_network` / `cloudflare_zero_trust_tunnel_virtual_network` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_tunnel_virtual_network` | `cloudflare_zero_trust_tunnel_cloudflared_virtual_network` | Renamed (adds `_cloudflared`) |
| Alt resource name | `cloudflare_zero_trust_tunnel_virtual_network` | `cloudflare_zero_trust_tunnel_cloudflared_virtual_network` | Renamed (adds `_cloudflared`) |
| State defaults | - | `comment = ""`, `is_default_network = false` | Added if missing |


---

## Migration Examples

### Example 1: Basic Virtual Network

**v4 Configuration:**
```hcl
resource "cloudflare_tunnel_virtual_network" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "production-vnet"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "production-vnet"
}
```

**What Changed:**
- Resource type: `cloudflare_tunnel_virtual_network` → `cloudflare_zero_trust_tunnel_cloudflared_virtual_network`
- State gets defaults: `comment = ""`, `is_default_network = false`

---

### Example 2: With Comment and Default Flag

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_tunnel_virtual_network" "default_vnet" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name               = "default-vnet"
  comment            = "Default virtual network"
  is_default_network = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_virtual_network" "default_vnet" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name               = "default-vnet"
  comment            = "Default virtual network"
  is_default_network = true
}
```

**What Changed:**
- Resource type renamed
- Values preserved (no defaults needed since fields are set)

---

