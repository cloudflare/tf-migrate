# Healthcheck Migration Guide (v4 → v5)

This guide explains how `cloudflare_healthcheck` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_healthcheck` | `cloudflare_healthcheck` | No change |
| HTTP/HTTPS fields | Root level | `http_config` block | Nested |
| TCP fields | Root level | `tcp_config` block | Nested |
| Header blocks | Set of blocks | Map structure | Type change |
| Numeric fields | Int | Float64 | Type conversion (state) |


---

## Migration Examples

### Example 1: HTTP Healthcheck with Headers

**v4 Configuration:**
```hcl
resource "cloudflare_healthcheck" "http" {
  zone_id             = "0da42c8d2132a9ddaf714f9e7c920711"
  name                = "http-health"
  address             = "example.com"
  type                = "HTTP"
  port                = 80
  method              = "GET"
  path                = "/health"
  expected_codes      = ["200"]
  expected_body       = "OK"
  follow_redirects    = false
  allow_insecure      = false

  header {
    header = "Host"
    values = ["example.com"]
  }

  header {
    header = "User-Agent"
    values = ["Cloudflare-Healthcheck"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_healthcheck" "http" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "http-health"
  address = "example.com"
  type    = "HTTP"

  http_config = {
    port             = 80
    method           = "GET"
    path             = "/health"
    expected_codes   = ["200"]
    expected_body    = "OK"
    follow_redirects = false
    allow_insecure   = false
    header = {
      "Host"       = ["example.com"]
      "User-Agent" = ["Cloudflare-Healthcheck"]
    }
  }
}
```

**What Changed:**
- HTTP-specific fields moved into `http_config` block
- `header` blocks converted to map structure
- Root-level fields (zone_id, name, address, type) unchanged

---

### Example 2: HTTPS Healthcheck (Minimal)

**v4 Configuration:**
```hcl
resource "cloudflare_healthcheck" "https" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "https-health"
  address = "api.example.com"
  type    = "HTTPS"
  port    = 443
  path    = "/v1/health"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_healthcheck" "https" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "https-health"
  address = "api.example.com"
  type    = "HTTPS"

  http_config = {
    port = 443
    path = "/v1/health"
  }
}
```

**What Changed:**
- `port` and `path` moved into `http_config`
- HTTPS uses `http_config` (not `https_config`)

---

### Example 3: TCP Healthcheck

**v4 Configuration:**
```hcl
resource "cloudflare_healthcheck" "tcp" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "tcp-health"
  address = "database.example.com"
  type    = "TCP"
  port    = 5432
  method  = "connection_established"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_healthcheck" "tcp" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "tcp-health"
  address = "database.example.com"
  type    = "TCP"

  tcp_config = {
    port   = 5432
    method = "connection_established"
  }
}
```

**What Changed:**
- TCP-specific fields moved into `tcp_config` block

---

### Example 4: Healthcheck Without Protocol-Specific Config

**v4 Configuration:**
```hcl
resource "cloudflare_healthcheck" "minimal" {
  zone_id             = "0da42c8d2132a9ddaf714f9e7c920711"
  name                = "minimal-health"
  address             = "example.com"
  type                = "HTTP"
  interval            = 60
  timeout             = 5
  retries             = 2
  consecutive_fails   = 3
  consecutive_successes = 2
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_healthcheck" "minimal" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  name                  = "minimal-health"
  address               = "example.com"
  type                  = "HTTP"
  interval              = 60
  timeout               = 5
  retries               = 2
  consecutive_fails     = 3
  consecutive_successes = 2
}
```

**What Changed:**
- Nothing - no protocol-specific fields means no nested config block created

---

