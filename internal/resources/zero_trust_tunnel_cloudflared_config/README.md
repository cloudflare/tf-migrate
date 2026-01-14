# Zero Trust Tunnel Cloudflared Config Migration Guide (v4 → v5)

This guide explains how `cloudflare_tunnel_config` / `cloudflare_zero_trust_tunnel_cloudflared_config` resources migrate to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_tunnel_config` | `cloudflare_zero_trust_tunnel_cloudflared_config` | Renamed (deprecated) |
| Alt resource name | `cloudflare_zero_trust_tunnel_cloudflared_config` | `cloudflare_zero_trust_tunnel_cloudflared_config` | No change |
| `config` | Block | Attribute object | Syntax change |
| `ingress_rule` | Field name | `ingress` | Renamed |
| `origin_request` | Block | Attribute object | Syntax change (2 levels) |
| `access` (nested) | Block | Attribute object | Syntax change |
| Duration fields | Strings (`"30s"`) | Int64 nanoseconds | Type conversion |
| `warp_routing` | Supported | Removed | Deprecated block |
| `ip_rules` | Supported | Removed | Deprecated block |
| `bastion_mode` | Supported | Removed | Deprecated field |
| `proxy_address/port` | Supported | Removed | Deprecated fields |


---

## Migration Examples

### Example 1: Basic Tunnel Config with Ingress

**v4 Configuration:**
```hcl
resource "cloudflare_tunnel_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_tunnel.example.id

  config {
    ingress_rule {
      hostname = "app.example.com"
      service  = "http://localhost:8080"
    }

    ingress_rule {
      service = "http_status:404"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.example.id

  config = {
    ingress = [
      {
        hostname = "app.example.com"
        service  = "http://localhost:8080"
      },
      {
        service = "http_status:404"
      }
    ]
  }
}
```

**What Changed:**
- Resource type renamed from deprecated name
- `config` block → `config` attribute object
- `ingress_rule` → `ingress` (field renamed)
- Tunnel resource reference updated to match new name

---

### Example 2: With Origin Request Settings

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "origin_config" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.vpn.id

  config {
    origin_request {
      connect_timeout       = "30s"
      tls_timeout          = "10s"
      tcp_keep_alive       = "1m30s"
      no_happy_eyeballs    = false
      keep_alive_timeout   = "1m30s"
      keep_alive_connections = 1024
      http_host_header     = "example.com"
      origin_server_name   = "origin.example.com"
      ca_pool              = "/etc/ssl/certs/ca-certificates.crt"
      no_tls_verify        = false
      disable_chunked_encoding = false
    }

    ingress_rule {
      hostname = "api.example.com"
      service  = "https://localhost:8443"
    }

    ingress_rule {
      service = "http_status:404"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "origin_config" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.vpn.id

  config = {
    origin_request = {
      connect_timeout          = 30000000000  # 30s in nanoseconds
      tls_timeout             = 10000000000  # 10s in nanoseconds
      tcp_keep_alive          = 90000000000  # 1m30s in nanoseconds
      no_happy_eyeballs       = false
      keep_alive_timeout      = 90000000000  # 1m30s in nanoseconds
      keep_alive_connections  = 1024
      http_host_header        = "example.com"
      origin_server_name      = "origin.example.com"
      ca_pool                 = "/etc/ssl/certs/ca-certificates.crt"
      no_tls_verify           = false
      disable_chunked_encoding = false
    }

    ingress = [
      {
        hostname = "api.example.com"
        service  = "https://localhost:8443"
      },
      {
        service = "http_status:404"
      }
    ]
  }
}
```

**What Changed:**
- `origin_request` block → attribute object
- Duration strings converted to int64 nanoseconds:
  - `"30s"` → `30000000000`
  - `"10s"` → `10000000000`
  - `"1m30s"` → `90000000000`
- `keep_alive_connections` remains same value (int → int64)

---

### Example 3: Per-Ingress Origin Request (Deprecated Fields)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "complex" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.app.id

  config {
    warp_routing {
      enabled = true
    }

    ingress_rule {
      hostname = "ssh.example.com"
      service  = "ssh://localhost:22"

      origin_request {
        bastion_mode = true
        proxy_address = "proxy.example.com"
        proxy_port = 3128
        connect_timeout = "10s"

        ip_rules {
          prefix = "192.168.1.0/24"
          ports = [22, 80]
          allow = true
        }

        access {
          required  = true
          team_name = "example-team"
          aud_tag   = "abc123def456"
        }
      }
    }

    ingress_rule {
      service = "http_status:404"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "complex" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.app.id

  config = {
    # warp_routing removed

    ingress = [
      {
        hostname = "ssh.example.com"
        service  = "ssh://localhost:22"

        origin_request = {
          # bastion_mode removed
          # proxy_address removed
          # proxy_port removed
          connect_timeout = 10000000000  # 10s in nanoseconds
          # ip_rules removed

          access = {
            required  = true
            team_name = "example-team"
            aud_tag   = "abc123def456"
          }
        }
      },
      {
        service = "http_status:404"
      }
    ]
  }
}
```

**What Changed:**
- **Removed blocks:** `warp_routing`, `ip_rules`
- **Removed fields:** `bastion_mode`, `proxy_address`, `proxy_port`
- `access` block → attribute object (nested)
- Duration conversion applied

---

### Example 4: Multiple Ingress with Mixed Settings

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "multi" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.web.id

  config {
    origin_request {
      connect_timeout = "15s"
    }

    ingress_rule {
      hostname = "app1.example.com"
      service  = "http://localhost:3000"

      origin_request {
        connect_timeout = "5s"
        http_host_header = "app1-internal.local"
      }
    }

    ingress_rule {
      hostname = "app2.example.com"
      service  = "http://localhost:4000"

      origin_request {
        connect_timeout = "20s"
      }
    }

    ingress_rule {
      service = "http_status:404"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared_config" "multi" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.web.id

  config = {
    origin_request = {
      connect_timeout = 15000000000  # 15s in nanoseconds
    }

    ingress = [
      {
        hostname = "app1.example.com"
        service  = "http://localhost:3000"

        origin_request = {
          connect_timeout  = 5000000000  # 5s in nanoseconds
          http_host_header = "app1-internal.local"
        }
      },
      {
        hostname = "app2.example.com"
        service  = "http://localhost:4000"

        origin_request = {
          connect_timeout = 20000000000  # 20s in nanoseconds
        }
      },
      {
        service = "http_status:404"
      }
    ]
  }
}
```

**What Changed:**
- Config-level and ingress-level `origin_request` both transformed
- Each level has independent duration conversions
- Block → attribute conversions at both nesting levels

---

