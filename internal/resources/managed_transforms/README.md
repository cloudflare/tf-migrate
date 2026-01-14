# Managed Transforms Migration Guide (v4 → v5)

This guide explains how `cloudflare_managed_headers` resources migrate to `cloudflare_managed_transforms` in v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_managed_headers` | `cloudflare_managed_transforms` | Renamed |
| `managed_request_headers` | Multiple blocks | Array attribute | Structure change |
| `managed_response_headers` | Multiple blocks | Array attribute | Structure change |
| Empty headers | Not present | Empty array `[]` | Required fields |


---

## Migration Examples

### Example 1: Request Headers Only

**v4 Configuration:**
```hcl
resource "cloudflare_managed_headers" "request_only" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  managed_request_headers {
    id      = "add_bot_protection_headers"
    enabled = true
  }

  managed_request_headers {
    id      = "add_client_certificate_headers"
    enabled = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_managed_transforms" "request_only" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  managed_request_headers = [
    {
      id      = "add_bot_protection_headers"
      enabled = true
    },
    {
      id      = "add_client_certificate_headers"
      enabled = true
    }
  ]

  managed_response_headers = []
}
```

**What Changed:**
- Resource type: `cloudflare_managed_headers` → `cloudflare_managed_transforms`
- Multiple blocks → single array attribute
- Empty `managed_response_headers` array added (required)

---

### Example 2: Response Headers Only

**v4 Configuration:**
```hcl
resource "cloudflare_managed_headers" "response_only" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  managed_response_headers {
    id      = "remove_x-powered-by_header"
    enabled = true
  }

  managed_response_headers {
    id      = "add_security_headers"
    enabled = false
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_managed_transforms" "response_only" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  managed_request_headers = []

  managed_response_headers = [
    {
      id      = "remove_x-powered-by_header"
      enabled = true
    },
    {
      id      = "add_security_headers"
      enabled = false
    }
  ]
}
```

**What Changed:**
- Resource renamed
- Blocks → array
- Empty `managed_request_headers` array added (required)

---

### Example 3: Both Request and Response Headers

**v4 Configuration:**
```hcl
resource "cloudflare_managed_headers" "both" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  managed_request_headers {
    id      = "add_true_client_ip_headers"
    enabled = true
  }

  managed_request_headers {
    id      = "add_visitor_location_headers"
    enabled = true
  }

  managed_response_headers {
    id      = "remove_x-powered-by_header"
    enabled = true
  }

  managed_response_headers {
    id      = "add_security_headers"
    enabled = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_managed_transforms" "both" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  managed_request_headers = [
    {
      id      = "add_true_client_ip_headers"
      enabled = true
    },
    {
      id      = "add_visitor_location_headers"
      enabled = true
    }
  ]

  managed_response_headers = [
    {
      id      = "remove_x-powered-by_header"
      enabled = true
    },
    {
      id      = "add_security_headers"
      enabled = true
    }
  ]
}
```

**What Changed:**
- Resource renamed
- All blocks converted to arrays

---

### Example 4: Minimal Configuration (No Headers)

**v4 Configuration:**
```hcl
resource "cloudflare_managed_headers" "minimal" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_managed_transforms" "minimal" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  managed_request_headers  = []
  managed_response_headers = []
}
```

**What Changed:**
- Resource renamed
- Both header arrays explicitly set to empty (required in v5)

---

