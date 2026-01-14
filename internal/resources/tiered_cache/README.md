# Tiered Cache Migration Guide (v4 → v5)

This guide explains how `cloudflare_tiered_cache` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_tiered_cache` | `cloudflare_tiered_cache` + `cloudflare_argo_tiered_caching` | Resource splitting |
| `cache_type` | "smart", "generic", "off" | Removed | Field removed |
| New field | - | `value` ("on" or "off") | Added |
| Resource count | 1 resource | 2 resources | Split |


---

## Migration Examples

### Example 1: Smart Tiered Caching

**v4 Configuration:**
```hcl
resource "cloudflare_tiered_cache" "smart" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  cache_type = "smart"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_tiered_cache" "smart" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}

resource "cloudflare_argo_tiered_caching" "smart" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
```

**What Changed:**
- ONE v4 resource → TWO v5 resources
- `cache_type = "smart"` → `tiered_cache.value = "on"` + `argo_tiered_caching.value = "on"`
- Second resource (`cloudflare_argo_tiered_caching`) automatically created

---

### Example 2: Generic Tiered Caching

**v4 Configuration:**
```hcl
resource "cloudflare_tiered_cache" "generic" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  cache_type = "generic"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_tiered_cache" "generic" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"
}

resource "cloudflare_argo_tiered_caching" "generic" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
```

**What Changed:**
- `cache_type = "generic"` → `tiered_cache.value = "off"` + `argo_tiered_caching.value = "on"`
- Generic caching disables tiered cache but enables Argo tiered caching

---

### Example 3: Disabled Tiered Caching

**v4 Configuration:**
```hcl
resource "cloudflare_tiered_cache" "disabled" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  cache_type = "off"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_tiered_cache" "disabled" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"
}

resource "cloudflare_argo_tiered_caching" "disabled" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"
}
```

**What Changed:**
- `cache_type = "off"` → both resources set to `value = "off"`

---

### Example 4: With Lifecycle Meta-Arguments

**v4 Configuration:**
```hcl
resource "cloudflare_tiered_cache" "protected" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  cache_type = "smart"

  lifecycle {
    prevent_destroy = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_tiered_cache" "protected" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"

  lifecycle {
    prevent_destroy = true
  }
}

resource "cloudflare_argo_tiered_caching" "protected" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"

  lifecycle {
    prevent_destroy = true
  }
}
```

**What Changed:**
- Lifecycle blocks copied to both new resources

---

