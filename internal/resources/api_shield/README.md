# API Shield Migration Guide (v4 → v5)

This guide explains how `cloudflare_api_shield` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change Type |
|--------|----|----|-------------|
| Resource name | `cloudflare_api_shield` | `cloudflare_api_shield` | No change |
| `auth_id_characteristics` | Nested blocks | Array attribute | Syntax change |
| Fields | `zone_id`, `type`, `name` | `zone_id`, `type`, `name` | No change |


---

## Migration Examples

### Example 1: Single Authentication Characteristic

**v4 Configuration:**
```hcl
resource "cloudflare_api_shield" "header_auth" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_api_shield" "header_auth" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  auth_id_characteristics = [{
    type = "header"
    name = "authorization"
  }]
}
```

**What Changed:**
- Block syntax `auth_id_characteristics { }` → Array syntax `auth_id_characteristics = [{ }]`
- Field names and values remain identical

---

### Example 2: Multiple Authentication Characteristics

**v4 Configuration:**
```hcl
resource "cloudflare_api_shield" "multi_auth" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "session_id"
  }

  auth_id_characteristics {
    type = "header"
    name = "x-api-key"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_api_shield" "multi_auth" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  auth_id_characteristics = [
    {
      type = "header"
      name = "authorization"
    },
    {
      type = "cookie"
      name = "session_id"
    },
    {
      type = "header"
      name = "x-api-key"
    }
  ]
}
```

**What Changed:**
- Multiple blocks → Array with comma-separated objects
- Order of characteristics is preserved
- All field values remain unchanged

---

### Example 3: No Authentication Characteristics

**v4 Configuration:**
```hcl
resource "cloudflare_api_shield" "minimal" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_api_shield" "minimal" {
  zone_id                 = "0da42c8d2132a9ddaf714f9e7c920711"
  auth_id_characteristics = []
}
```

**What Changed:**
- Empty array `[]` is explicitly added for schema consistency
- Zone ID remains unchanged

---

