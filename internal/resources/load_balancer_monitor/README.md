# Load Balancer Monitor Migration Guide (v4 → v5)

This guide explains how `cloudflare_load_balancer_monitor` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_load_balancer_monitor` | `cloudflare_load_balancer_monitor` | No change |
| `header` blocks | Multiple blocks | Map attribute | Structure change |
| Default values | Not in state | Added to state | Prevents drift |
| Numeric fields | Int | Int64 (float64 in state) | Type conversion |


---

## Migration Examples

### Example 1: Basic HTTP Monitor

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer_monitor" "http" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "http"
  path       = "/health"
  interval   = 60
  timeout    = 5
  retries    = 2
  method     = "GET"
  port       = 80
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer_monitor" "http" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "http"
  path       = "/health"
  interval   = 60
  timeout    = 5
  retries    = 2
  method     = "GET"
  port       = 80
}
```

**What Changed:**
- Configuration unchanged (numeric fields remain the same)
- State adds default values internally

---

### Example 2: HTTPS Monitor with Headers

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer_monitor" "https" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  type           = "https"
  path           = "/api/health"
  interval       = 45
  timeout        = 7
  retries        = 3
  method         = "GET"
  port           = 443
  expected_codes = "200,201"
  expected_body  = "healthy"

  header {
    header = "Host"
    values = ["api.example.com"]
  }

  header {
    header = "Authorization"
    values = ["Bearer token123"]
  }

  header {
    header = "X-Custom-Header"
    values = ["value1", "value2"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer_monitor" "https" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  type           = "https"
  path           = "/api/health"
  interval       = 45
  timeout        = 7
  retries        = 3
  method         = "GET"
  port           = 443
  expected_codes = "200,201"
  expected_body  = "healthy"

  header = {
    "Host"            = ["api.example.com"]
    "Authorization"   = ["Bearer token123"]
    "X-Custom-Header" = ["value1", "value2"]
  }
}
```

**What Changed:**
- Multiple `header` blocks → single `header` map attribute
- Header name becomes map key, values array becomes map value

---

### Example 3: TCP Monitor with Consecutive Checks

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer_monitor" "tcp" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  type              = "tcp"
  port              = 3306
  interval          = 30
  timeout           = 5
  retries           = 2
  consecutive_up    = 3
  consecutive_down  = 2
  description       = "MySQL health check"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer_monitor" "tcp" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  type              = "tcp"
  port              = 3306
  interval          = 30
  timeout           = 5
  retries           = 2
  consecutive_up    = 3
  consecutive_down  = 2
  description       = "MySQL health check"
}
```

**What Changed:**
- Configuration unchanged
- State handles consecutive_up/down properly

---

### Example 4: HTTPS with Follow Redirects

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer_monitor" "redirect" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  type              = "https"
  path              = "/redirect-test"
  interval          = 60
  timeout           = 10
  retries           = 2
  method            = "GET"
  port              = 443
  follow_redirects  = true
  allow_insecure    = false
  probe_zone        = "example.com"

  header {
    header = "User-Agent"
    values = ["Cloudflare-Traffic-Manager/1.0"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer_monitor" "redirect" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  type              = "https"
  path              = "/redirect-test"
  interval          = 60
  timeout           = 10
  retries           = 2
  method            = "GET"
  port              = 443
  follow_redirects  = true
  allow_insecure    = false
  probe_zone        = "example.com"

  header = {
    "User-Agent" = ["Cloudflare-Traffic-Manager/1.0"]
  }
}
```

---

