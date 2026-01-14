# Spectrum Application Migration Guide (v4 → v5)

This guide explains how `cloudflare_spectrum_application` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_spectrum_application` | `cloudflare_spectrum_application` | No change |
| `dns` | Block (MaxItems:1) | Attribute object | Syntax change |
| `origin_dns` | Block (MaxItems:1) | Attribute object | Syntax change |
| `edge_ips` | Block (MaxItems:1) | Attribute object | Syntax change |
| `origin_port_range` | Block with `start`/`end` | `origin_port` string | Structure change + format |


---

## Migration Examples

### Example 1: Basic Spectrum Application

**v4 Configuration:**
```hcl
resource "cloudflare_spectrum_application" "ssh" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/22"

  dns {
    type = "CNAME"
    name = "ssh.example.com"
  }

  origin_dns {
    name = "origin-ssh.example.com"
  }

  origin_port_range {
    start = 22
    end   = 22
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_spectrum_application" "ssh" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/22"

  dns = {
    type = "CNAME"
    name = "ssh.example.com"
  }

  origin_dns = {
    name = "origin-ssh.example.com"
  }

  origin_port = "22-22"
}
```

**What Changed:**
- `dns { }` block → `dns = { }` attribute
- `origin_dns { }` block → `origin_dns = { }` attribute
- `origin_port_range { start = 22 end = 22 }` → `origin_port = "22-22"`

---

### Example 2: Port Range Application

**v4 Configuration:**
```hcl
resource "cloudflare_spectrum_application" "mysql" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/3306"

  dns {
    type = "CNAME"
    name = "db.example.com"
  }

  origin_dns {
    name = "origin-db.example.com"
  }

  origin_port_range {
    start = 3306
    end   = 3310
  }

  ip_firewall = true
  proxy_protocol = "v1"
  tls = "full"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_spectrum_application" "mysql" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/3306"

  dns = {
    type = "CNAME"
    name = "db.example.com"
  }

  origin_dns = {
    name = "origin-db.example.com"
  }

  origin_port = "3306-3310"

  ip_firewall    = true
  proxy_protocol = "v1"
  tls            = "full"
}
```

**What Changed:**
- Port range block converted to string format: `"start-end"`

---

### Example 3: With Edge IPs

**v4 Configuration:**
```hcl
resource "cloudflare_spectrum_application" "with_edge_ips" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/443"

  dns {
    type = "CNAME"
    name = "secure.example.com"
  }

  origin_dns {
    name = "origin.example.com"
  }

  origin_port_range {
    start = 443
    end   = 443
  }

  edge_ips {
    type         = "dynamic"
    connectivity = "all"
  }

  argo_smart_routing = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_spectrum_application" "with_edge_ips" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/443"

  dns = {
    type = "CNAME"
    name = "secure.example.com"
  }

  origin_dns = {
    name = "origin.example.com"
  }

  origin_port = "443-443"

  edge_ips = {
    type         = "dynamic"
    connectivity = "all"
  }

  argo_smart_routing = true
}
```

**What Changed:**
- `edge_ips { }` block → `edge_ips = { }` attribute

---

### Example 4: Origin Direct with IP

**v4 Configuration:**
```hcl
resource "cloudflare_spectrum_application" "direct_ip" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/5432"

  dns {
    type = "CNAME"
    name = "postgres.example.com"
  }

  origin_direct = ["192.0.2.1"]

  origin_port_range {
    start = 5432
    end   = 5432
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_spectrum_application" "direct_ip" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/5432"

  dns = {
    type = "CNAME"
    name = "postgres.example.com"
  }

  origin_direct = ["192.0.2.1"]

  origin_port = "5432-5432"
}
```

**What Changed:**
- Block conversions applied
- `origin_direct` array preserved as-is

---

