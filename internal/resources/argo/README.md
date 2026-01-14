# Argo Migration Guide (v4 → v5)

This guide explains how `cloudflare_argo` resources migrate from v4 to v5.

## Quick Reference

| v4 Resource | v5 Resource(s) | Split Logic |
|-------------|----------------|-------------|
| `cloudflare_argo` | `cloudflare_argo_smart_routing` | If only `smart_routing` attribute exists |
| `cloudflare_argo` | `cloudflare_argo_tiered_caching` | If only `tiered_caching` attribute exists |
| `cloudflare_argo` | Both resources | If both attributes exist (requires manual import for tiered_caching) |
| `cloudflare_argo` (empty) | `cloudflare_argo_smart_routing` (default) | If neither attribute specified |


---

## Migration Examples

### Example 1: Smart Routing Only

**v4 Configuration:**
```hcl
resource "cloudflare_argo" "example" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_argo_smart_routing" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}

moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_smart_routing.example
}
```

**What Changed:**
- Resource type: `cloudflare_argo` → `cloudflare_argo_smart_routing`
- Field rename: `smart_routing` → `value`
- `moved` block automatically added for state tracking

---

### Example 2: Tiered Caching Only

**v4 Configuration:**
```hcl
resource "cloudflare_argo" "cache" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  tiered_caching = "on"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_argo_tiered_caching" "cache" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}

moved {
  from = cloudflare_argo.cache
  to   = cloudflare_argo_tiered_caching.cache
}
```

**What Changed:**
- Resource type: `cloudflare_argo` → `cloudflare_argo_tiered_caching`
- Field rename: `tiered_caching` → `value`
- `moved` block automatically added

---

### Example 3: Both Features Enabled (Requires Manual Import)

**v4 Configuration:**
```hcl
resource "cloudflare_argo" "full" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing  = "on"
  tiered_caching = "on"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_argo_smart_routing" "full" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}

moved {
  from = cloudflare_argo.full
  to   = cloudflare_argo_smart_routing.full
}

resource "cloudflare_argo_tiered_caching" "full_tiered" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
  # ⚠️ NOTE: You must manually import this resource after migration
}
```

**What Changed:**
- Single resource splits into two resources
- Smart routing keeps original name, tiered caching gets `_tiered` suffix
- Only smart routing gets `moved` block
- **Action Required:** Import the tiered_caching resource manually:
  ```bash
  terraform import cloudflare_argo_tiered_caching.full_tiered 0da42c8d2132a9ddaf714f9e7c920711
  ```

---

### Example 4: Default Behavior (Empty Configuration)

**v4 Configuration:**
```hcl
resource "cloudflare_argo" "default" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_argo_smart_routing" "default" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"  # ← Defaults to "off"
}

moved {
  from = cloudflare_argo.default
  to   = cloudflare_argo_smart_routing.default
}
```

**What Changed:**
- Defaults to smart_routing resource with `value = "off"`
- `moved` block tracks the state transition

---

### Example 5: With Lifecycle Block

**v4 Configuration:**
```hcl
resource "cloudflare_argo" "managed" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_argo_smart_routing" "managed" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"

  lifecycle {
    ignore_changes = [value]  # ← Automatically updated reference
  }
}

moved {
  from = cloudflare_argo.managed
  to   = cloudflare_argo_smart_routing.managed
}
```

**What Changed:**
- Lifecycle block references automatically updated
- `ignore_changes = [smart_routing]` → `ignore_changes = [value]`
- Both resulting resources inherit lifecycle blocks

---

