# Load Balancer Pool Migration Guide (v4 → v5)

This guide explains how `cloudflare_load_balancer_pool` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_load_balancer_pool` | `cloudflare_load_balancer_pool` | No change |
| `origins` | Multiple blocks | Array attribute | Structure change |
| `origins.header` | Blocks | Lowercase map | Structure + case change |
| `load_shedding` | Block | Attribute object | Syntax change |
| `origin_steering` | Block | Attribute object | Syntax change |
| Dynamic origins | `dynamic "origins"` block | `for` expression | Syntax change |


---

## Migration Examples

### Example 1: Basic Pool with Origins

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer_pool" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example-pool"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
    weight  = 1
  }

  origins {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = true
    weight  = 1
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer_pool" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example-pool"

  origins = [
    {
      name    = "origin-1"
      address = "192.0.2.1"
      enabled = true
      weight  = 1
    },
    {
      name    = "origin-2"
      address = "192.0.2.2"
      enabled = true
      weight  = 1
    }
  ]
}
```

**What Changed:**
- Multiple `origins` blocks → single `origins` array attribute
- Block syntax → array of objects

---

### Example 2: Origins with Headers

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer_pool" "with_headers" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "pool-with-headers"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"

    header {
      header = "Host"
      values = ["api.example.com"]
    }

    header {
      header = "X-Custom-Header"
      values = ["value1", "value2"]
    }
  }

  origins {
    name    = "origin-2"
    address = "192.0.2.2"

    header {
      header = "Authorization"
      values = ["Bearer token123"]
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer_pool" "with_headers" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "pool-with-headers"

  origins = [
    {
      name    = "origin-1"
      address = "192.0.2.1"
      header = {
        host             = ["api.example.com"]
        x-custom-header  = ["value1", "value2"]
      }
    },
    {
      name    = "origin-2"
      address = "192.0.2.2"
      header = {
        authorization = ["Bearer token123"]
      }
    }
  ]
}
```

**What Changed:**
- Multiple `header` blocks within origins → single `header` map
- Header names converted to lowercase
- Header `header` attribute becomes map key (lowercase)
- Header `values` attribute becomes map value

---

### Example 3: Dynamic Origins (For Expression)

**v4 Configuration:**
```hcl
locals {
  backend_origins = [
    { name = "backend-1", address = "192.0.2.10" },
    { name = "backend-2", address = "192.0.2.11" },
    { name = "backend-3", address = "192.0.2.12" }
  ]
}

resource "cloudflare_load_balancer_pool" "dynamic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "dynamic-pool"

  dynamic "origins" {
    for_each = local.backend_origins
    content {
      name    = origins.value.name
      address = origins.value.address
      enabled = true
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
locals {
  backend_origins = [
    { name = "backend-1", address = "192.0.2.10" },
    { name = "backend-2", address = "192.0.2.11" },
    { name = "backend-3", address = "192.0.2.12" }
  ]
}

resource "cloudflare_load_balancer_pool" "dynamic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "dynamic-pool"

  origins = [for value in local.backend_origins : {
    name    = value.name
    address = value.address
    enabled = true
  }]
}
```

**What Changed:**
- `dynamic "origins"` block → `for` expression
- Iterator variable `origins.value` → `value`
- Inline array construction

---

### Example 4: Pool with Load Shedding and Origin Steering

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer_pool" "advanced" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "advanced-pool"
  description = "Pool with advanced features"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    weight  = 0.5
  }

  origins {
    name    = "origin-2"
    address = "192.0.2.2"
    weight  = 0.5
  }

  load_shedding {
    default_percent = 55
    default_policy  = "random"
    session_percent = 12
    session_policy  = "hash"
  }

  origin_steering {
    policy = "random"
  }

  check_regions   = ["WEU", "ENAM"]
  minimum_origins = 1
  enabled         = true
  latitude        = 37.7749
  longitude       = -122.4194
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer_pool" "advanced" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "advanced-pool"
  description = "Pool with advanced features"

  origins = [
    {
      name    = "origin-1"
      address = "192.0.2.1"
      weight  = 0.5
    },
    {
      name    = "origin-2"
      address = "192.0.2.2"
      weight  = 0.5
    }
  ]

  load_shedding = {
    default_percent = 55
    default_policy  = "random"
    session_percent = 12
    session_policy  = "hash"
  }

  origin_steering = {
    policy = "random"
  }

  check_regions   = ["WEU", "ENAM"]
  minimum_origins = 1
  enabled         = true
  latitude        = 37.7749
  longitude       = -122.4194
}
```

**What Changed:**
- `origins` blocks → array
- `load_shedding` block → attribute object
- `origin_steering` block → attribute object

---

