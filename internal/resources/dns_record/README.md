# DNS Record Migration Guide (v4 → v5)

This guide explains how `cloudflare_record` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_record` | `cloudflare_dns_record` | Renamed |
| Simple records (`value`) | `value` | `content` | Renamed |
| CAA `data.content` | `content` | `value` | Renamed |
| Priority (MX, URI) | `data.priority` | `priority` (root) | Hoisted |
| Priority (SRV) | `data.priority` | Both root & data | Duplicated |
| `ttl` | Optional | Required (defaults to 1) | Required |
| Data blocks | `data { }` | `data = { }` | Syntax change |


---

## Migration Examples

### Example 1: Simple A Record

**v4 Configuration:**
```hcl
resource "cloudflare_record" "www" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "www"
  type    = "A"
  value   = "192.0.2.1"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_dns_record" "www" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "www"
  type    = "A"
  content = "192.0.2.1"  # ← value renamed to content
  ttl     = 1            # ← Added (required)
}
```

**What Changed:**
- Resource type: `cloudflare_record` → `cloudflare_dns_record`
- `value` → `content`
- `ttl` added with default value

---

### Example 2: MX Record with Priority Hoisting

**v4 Configuration:**
```hcl
resource "cloudflare_record" "mx" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "@"
  type    = "MX"
  data {
    priority = 10
    target   = "mail.example.com"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_dns_record" "mx" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  name     = "@"
  type     = "MX"
  ttl      = 1
  priority = 10  # ← Hoisted from data.priority
  data = {       # ← Block syntax changed to attribute
    target = "mail.example.com"  # priority removed from data
  }
}
```

**What Changed:**
- `data` block → `data` attribute (`data { }` → `data = { }`)
- `priority` moved from `data.priority` to root level
- `priority` removed from data object
- State includes generated `content` field: `"10 mail.example.com"`

---

### Example 3: CAA Record with Data Transformations

**v4 Configuration:**
```hcl
resource "cloudflare_record" "caa" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "example.com"
  type    = "CAA"
  data {
    flags   = 0
    tag     = "issue"
    content = "letsencrypt.org"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_dns_record" "caa" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "example.com"
  type    = "CAA"
  ttl     = 1
  data = {
    flags = 0
    tag   = "issue"
    value = "letsencrypt.org"  # ← content renamed to value
  }
}
```

**What Changed:**
- `data.content` → `data.value` (CAA-specific rename)
- State includes special flags handling: `{"type": "string", "value": "0"}`
- State includes generated `content`: `"0 issue letsencrypt.org"`

---

### Example 4: SRV Record (Priority in Both Locations)

**v4 Configuration:**
```hcl
resource "cloudflare_record" "srv" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "_sip._tcp.example.com"
  type    = "SRV"
  data {
    priority = 10
    weight   = 60
    port     = 5060
    target   = "sipserver.example.com"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_dns_record" "srv" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  name     = "_sip._tcp.example.com"
  type     = "SRV"
  ttl      = 1
  priority = 10  # ← Hoisted to root
  data = {
    priority = 10  # ← Also kept in data (SRV exception)
    weight   = 60
    port     = 5060
    target   = "sipserver.example.com"
  }
}
```

**What Changed:**
- `priority` exists in BOTH root AND data (SRV-specific behavior)
- All numeric fields in data converted to float64

---

### Example 5: CNAME Record

**v4 Configuration:**
```hcl
resource "cloudflare_record" "alias" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "alias"
  type    = "CNAME"
  value   = "example.com"
  proxied = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_dns_record" "alias" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "alias"
  type    = "CNAME"
  content = "example.com"  # ← value renamed to content
  ttl     = 1
  proxied = true
}
```

---

