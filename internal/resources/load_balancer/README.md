# Load Balancer Migration Guide (v4 → v5)

This guide explains how `cloudflare_load_balancer` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_load_balancer` | `cloudflare_load_balancer` | No change |
| `default_pool_ids` | `default_pool_ids = [...]` | `default_pools = [...]` | Field renamed |
| `fallback_pool_id` | `fallback_pool_id = "..."` | `fallback_pool = "..."` | Field renamed |
| `session_affinity_attributes` | Block | Attribute object | Syntax change |
| `adaptive_routing` | Block | Attribute object | Syntax change |
| `location_strategy` | Block | Attribute object | Syntax change |
| `random_steering` | Block | Attribute object | Syntax change |
| `region_pools` | Multiple blocks | Map attribute | Structure change |
| `pop_pools` | Multiple blocks | Map attribute | Structure change |
| `country_pools` | Multiple blocks | Map attribute | Structure change |


---

## Migration Examples

### Example 1: Basic Load Balancer

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer" "example" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  name             = "example-lb.example.com"
  default_pool_ids = ["17b5962d775c646f3f9725cbc7a53df4"]
  fallback_pool_id = "9290f38c5d07c2e2f4df57b1f61d4196"
  description      = "Example load balancer"
  ttl              = 30
  proxied          = true
  enabled          = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer" "example" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  name          = "example-lb.example.com"
  default_pools = ["17b5962d775c646f3f9725cbc7a53df4"]
  fallback_pool = "9290f38c5d07c2e2f4df57b1f61d4196"
  description   = "Example load balancer"
  ttl           = 30
  proxied       = true
  enabled       = true
}
```

**What Changed:**
- `default_pool_ids` → `default_pools`
- `fallback_pool_id` → `fallback_pool`

---

### Example 2: Load Balancer with Session Affinity

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer" "with_affinity" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  name             = "app-lb.example.com"
  default_pool_ids = ["pool-id-1"]
  fallback_pool_id = "pool-id-fallback"
  session_affinity = "cookie"

  session_affinity_attributes {
    samesite = "Lax"
    secure   = "Always"
    drain_duration = 60
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer" "with_affinity" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  name          = "app-lb.example.com"
  default_pools = ["pool-id-1"]
  fallback_pool = "pool-id-fallback"
  session_affinity = "cookie"

  session_affinity_attributes = {
    samesite       = "Lax"
    secure         = "Always"
    drain_duration = 60
  }
}
```

**What Changed:**
- `session_affinity_attributes { }` block → `session_affinity_attributes = { }` attribute

---

### Example 3: Regional Pool Steering

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer" "regional" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  name             = "regional-lb.example.com"
  default_pool_ids = ["default-pool-id"]
  fallback_pool_id = "fallback-pool-id"
  steering_policy  = "geo"

  region_pools {
    region   = "WNAM"
    pool_ids = ["wnam-pool-1", "wnam-pool-2"]
  }

  region_pools {
    region   = "ENAM"
    pool_ids = ["enam-pool-1"]
  }

  country_pools {
    country  = "US"
    pool_ids = ["us-pool-1", "us-pool-2"]
  }

  country_pools {
    country  = "CA"
    pool_ids = ["ca-pool-1"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer" "regional" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  name          = "regional-lb.example.com"
  default_pools = ["default-pool-id"]
  fallback_pool = "fallback-pool-id"
  steering_policy = "geo"

  region_pools = {
    "WNAM" = ["wnam-pool-1", "wnam-pool-2"]
    "ENAM" = ["enam-pool-1"]
  }

  country_pools = {
    "US" = ["us-pool-1", "us-pool-2"]
    "CA" = ["ca-pool-1"]
  }
}
```

**What Changed:**
- Multiple `region_pools` blocks → single `region_pools` map
- Multiple `country_pools` blocks → single `country_pools` map
- Block region/country becomes map key, pool_ids becomes map value

---

### Example 4: All Advanced Features

**v4 Configuration:**
```hcl
resource "cloudflare_load_balancer" "advanced" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  name             = "advanced-lb.example.com"
  default_pool_ids = ["primary-pool"]
  fallback_pool_id = "fallback-pool"

  adaptive_routing {
    failover_across_pools = true
  }

  location_strategy {
    prefer_ecs = "proximity"
    mode       = "pop"
  }

  random_steering {
    default_weight = 0.5
    pool_weights = {
      "pool-1" = 0.3
      "pool-2" = 0.7
    }
  }

  pop_pools {
    pop      = "LAX"
    pool_ids = ["lax-pool-1", "lax-pool-2"]
  }

  pop_pools {
    pop      = "SJC"
    pool_ids = ["sjc-pool-1"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_load_balancer" "advanced" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  name          = "advanced-lb.example.com"
  default_pools = ["primary-pool"]
  fallback_pool = "fallback-pool"

  adaptive_routing = {
    failover_across_pools = true
  }

  location_strategy = {
    prefer_ecs = "proximity"
    mode       = "pop"
  }

  random_steering = {
    default_weight = 0.5
    pool_weights = {
      "pool-1" = 0.3
      "pool-2" = 0.7
    }
  }

  pop_pools = {
    "LAX" = ["lax-pool-1", "lax-pool-2"]
    "SJC" = ["sjc-pool-1"]
  }
}
```

**What Changed:**
- All nested blocks converted to attribute objects
- Multiple `pop_pools` blocks → single map attribute

---

