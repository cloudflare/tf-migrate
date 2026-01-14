# Zero Trust Tunnel Cloudflared Route Migration Guide (v4 → v5)

This guide explains how `cloudflare_tunnel_route` / `cloudflare_zero_trust_tunnel_route` resources migrate to `cloudflare_zero_trust_tunnel_cloudflared_route` in v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_tunnel_route` | `cloudflare_zero_trust_tunnel_cloudflared_route` | Renamed (adds `_cloudflared`) |
| Alt resource name | `cloudflare_zero_trust_tunnel_route` | `cloudflare_zero_trust_tunnel_cloudflared_route` | Renamed (adds `_cloudflared`) |
| ID format | Network CIDR | UUID from API | Updated via API lookup |
| Fields | No changes | All preserved | None |


---

## Migration Examples

### Example 1: Basic Tunnel Route

**v4 Configuration:**
```hcl
resource "cloudflare_tunnel_route" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_tunnel.example.id
  network    = "192.0.2.0/24"
  comment    = "Production network route"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_route" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.example.id
  network    = "192.0.2.0/24"
  comment    = "Production network route"
}
```

**What Changed:**
- Resource type renamed to include `_cloudflared`
- Tunnel resource reference updated to match new name
- State ID updated from CIDR to UUID (automatic via API lookup)

---

### Example 2: IPv6 Route with Virtual Network

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_tunnel_route" "ipv6" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  tunnel_id          = cloudflare_zero_trust_tunnel.vpn.id
  network            = "2001:db8::/32"
  virtual_network_id = cloudflare_tunnel_virtual_network.vnet.id
  comment            = "IPv6 network"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_route" "ipv6" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  tunnel_id          = cloudflare_zero_trust_tunnel_cloudflared.vpn.id
  network            = "2001:db8::/32"
  virtual_network_id = cloudflare_zero_trust_tunnel_cloudflared_virtual_network.vnet.id
  comment            = "IPv6 network"
}
```

**What Changed:**
- Resource type renamed
- Referenced resource types updated

---

