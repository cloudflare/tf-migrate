# Worker Route Migration Guide (v4 → v5)

This guide explains how `cloudflare_worker_route` / `cloudflare_workers_route` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name (singular) | `cloudflare_worker_route` | `cloudflare_workers_route` | Normalized to plural |
| Resource name (plural) | `cloudflare_workers_route` | `cloudflare_workers_route` | No change |
| Field | `script_name` | `script` | Renamed |


---

## Migration Examples

### Example 1: Singular Resource Name

**v4 Configuration:**
```hcl
resource "cloudflare_worker_route" "example" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern     = "example.com/*"
  script_name = "my-worker"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_route" "example" {  # ← Plural
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "example.com/*"
  script  = "my-worker"  # ← script_name → script
}
```

**What Changed:**
- Resource type: `cloudflare_worker_route` → `cloudflare_workers_route` (plural)
- `script_name` → `script`

---

### Example 2: Plural Resource Name (Already Correct)

**v4 Configuration:**
```hcl
resource "cloudflare_workers_route" "example" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern     = "api.example.com/*"
  script_name = "api-worker"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_route" "example" {  # ← Already plural
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "api.example.com/*"
  script  = "api-worker"  # ← script_name → script
}
```

**What Changed:**
- Resource type unchanged (already plural)
- `script_name` → `script`

---

### Example 3: Route Without Script

**v4 Configuration:**
```hcl
resource "cloudflare_worker_route" "catchall" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "*.example.com/*"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_route" "catchall" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "*.example.com/*"
}
```

**What Changed:**
- Resource type normalized to plural
- No script field (was optional, remains optional)

---

